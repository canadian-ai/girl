package grp

import (
	"fmt"
	"strings"
)

type ValidationError struct {
	Field    string `json:"field"`
	Message  string `json:"message"`
	Severity string `json:"severity"`
}

type ValidationResult struct {
	Valid  bool              `json:"valid"`
	Errors []ValidationError `json:"errors"`
}

func ValidatePlan(p *Plan) *ValidationResult {
	result := &ValidationResult{Valid: true}
	if p == nil {
		result.Errors = append(result.Errors, ValidationError{
			Field: "plan", Message: "plan is nil", Severity: "error",
		})
		result.Valid = false
		return result
	}
	validateBasicFields(p, result)
	diagIDs := validateDiagnostics(p.Diagnostics, result)
	validateSteps(p.Steps, diagIDs, result)
	validateReduction(p.Reduction, result)
	validateVerification(p.Verification, result)
	validateReviewability(p.Reviewability, result)
	validateDecomposition(p.Decomposition, result)
	result.Valid = len(result.Errors) == 0
	return result
}

func validateBasicFields(p *Plan, result *ValidationResult) {
	requiredStr(result, "specversion", p.SpecVersion, func(v string) bool { return v == "0.1" })
	if p.ID == "" {
		result.Errors = append(result.Errors, err("id", "must not be empty"))
	} else if !strings.HasPrefix(p.ID, "grp_") {
		result.Errors = append(result.Errors, err("id", `must start with "grp_"`))
	}
	requiredNonEmpty(result, "type", p.Type)
	requiredNonEmpty(result, "source", p.Source)
	requiredNonEmpty(result, "subject", p.Subject)
	if p.Subject != "" && isAbsolute(p.Subject) {
		result.Errors = append(result.Errors, err("subject", "must not be an absolute path"))
	}
	requiredNonEmpty(result, "language", p.Language)
	requiredNonEmpty(result, "goal", p.Goal)
	enumCheck(result, "risk", string(p.Risk), validRisk)
}

func validateDiagnostics(diags []Diagnostic, result *ValidationResult) map[string]bool {
	ids := make(map[string]bool, len(diags))
	for i, d := range diags {
		prefix := fmt.Sprintf("diagnostics[%d]", i)
		if d.ID == "" {
			result.Errors = append(result.Errors, err(prefix+".id", "must not be empty"))
		} else {
			if !strings.HasPrefix(d.ID, "diag_") {
				result.Errors = append(result.Errors, err(prefix+".id", `must start with "diag_"`))
			}
			if ids[d.ID] {
				result.Errors = append(result.Errors, err(prefix+".id", fmt.Sprintf("duplicate diagnostic ID %q", d.ID)))
			}
			ids[d.ID] = true
		}
		requiredNonEmpty(result, prefix+".code", d.Code)
		enumCheck(result, prefix+".severity", string(d.Severity), validSeverity)
		enumCheck(result, prefix+".confidence", string(d.Confidence), validConfidence)
		requiredNonEmpty(result, prefix+".message", d.Message)
		if d.File == "" {
			result.Errors = append(result.Errors, err(prefix+".file", "must not be empty"))
		} else if isAbsolute(d.File) {
			result.Errors = append(result.Errors, err(prefix+".file", "must not be an absolute path"))
		}
	}
	return ids
}

func validateSteps(steps []Step, diagIDs map[string]bool, result *ValidationResult) {
	ids := make(map[string]bool, len(steps))
	for i, s := range steps {
		prefix := fmt.Sprintf("steps[%d]", i)
		if s.ID == "" {
			result.Errors = append(result.Errors, err(prefix+".id", "must not be empty"))
		} else {
			if !strings.HasPrefix(s.ID, "step_") {
				result.Errors = append(result.Errors, err(prefix+".id", `must start with "step_"`))
			}
			if ids[s.ID] {
				result.Errors = append(result.Errors, err(prefix+".id", fmt.Sprintf("duplicate step ID %q", s.ID)))
			}
			ids[s.ID] = true
		}
		requiredNonEmpty(result, prefix+".title", s.Title)
		requiredNonEmpty(result, prefix+".action", s.Action)
		requiredNonEmpty(result, prefix+".target.file", s.Target.File)
		if s.Target.File != "" && isAbsolute(s.Target.File) {
			result.Errors = append(result.Errors, err(prefix+".target.file", "must not be an absolute path"))
		}
		enumCheck(result, prefix+".risk", string(s.Risk), validRisk)
		for j, v := range s.Verify {
			vp := fmt.Sprintf("%s.verify[%d]", prefix, j)
			requiredNonEmpty(result, vp+".command", v.Command)
			enumCheck(result, vp+".confidence", v.Confidence, validConfidence)
		}
	}

	// Requires may reference diagnostics or sibling steps (e.g. a collect step
	// must require its migration steps), so validate in a second pass after the
	// full step-ID set is known.
	for i, s := range steps {
		prefix := fmt.Sprintf("steps[%d]", i)
		for _, req := range s.Requires {
			if !diagIDs[req] && !ids[req] {
				result.Errors = append(result.Errors, err(prefix+".requires", fmt.Sprintf("references unknown diagnostic or step ID %q", req)))
			}
		}
	}

	for i, s := range steps {
		if !isReductionStep(s, ActionCollect) {
			continue
		}
		prefix := fmt.Sprintf("steps[%d]", i)
		if !hasMigratePrereq(s, steps) {
			result.Errors = append(result.Errors, err(prefix+".requires",
				fmt.Sprintf("collect step %q must require at least one %q migration step", s.ID, ActionMigrate)))
		}
		if len(s.Verify) == 0 {
			result.Errors = append(result.Errors, err(prefix+".verify",
				fmt.Sprintf("collect step %q must carry a verification gate before collection", s.ID)))
		}
	}

	for i, s := range steps {
		if !isReductionStep(s, ActionMigrate) {
			continue
		}
		prefix := fmt.Sprintf("steps[%d]", i)
		if len(s.Verify) == 0 {
			result.Errors = append(result.Errors, err(prefix+".verify",
				fmt.Sprintf("migration step %q must carry a verification gate after migrating references", s.ID)))
		}
	}
}

// isReductionStep reports whether a step belongs to a reduction lifecycle phase.
// The machine-readable marker lives in recipe (e.g. "grp.reduction.collect");
// action is the freeform human description, but either field may carry it.
func isReductionStep(s Step, prefix string) bool {
	return strings.HasPrefix(s.Recipe, prefix) || strings.HasPrefix(s.Action, prefix)
}

func hasMigratePrereq(s Step, steps []Step) bool {
	migrateIDs := make(map[string]bool, len(steps))
	for _, other := range steps {
		if isReductionStep(other, ActionMigrate) {
			migrateIDs[other.ID] = true
		}
	}
	for _, req := range s.Requires {
		if migrateIDs[req] {
			return true
		}
	}
	return false
}

func validateReduction(r *Reduction, result *ValidationResult) {
	if r == nil {
		return
	}
	nodeIDs := make(map[string]bool, len(r.Nodes))
	for _, n := range r.Nodes {
		if n.ID != "" {
			if nodeIDs[n.ID] {
				// duplicate handled in the per-node loop below
				continue
			}
			nodeIDs[n.ID] = true
		}
	}
	seen := make(map[string]bool, len(r.Nodes))
	for i, n := range r.Nodes {
		prefix := fmt.Sprintf("reduction.nodes[%d]", i)
		if n.ID == "" {
			result.Errors = append(result.Errors, err(prefix+".id", "must not be empty"))
		} else {
			if !strings.HasPrefix(n.ID, "cap_") {
				result.Errors = append(result.Errors, err(prefix+".id", `must start with "cap_"`))
			}
			if seen[n.ID] {
				result.Errors = append(result.Errors, err(prefix+".id", fmt.Sprintf("duplicate node ID %q", n.ID)))
			}
			seen[n.ID] = true
		}
		if n.GarbageClass != "" && !validGarbageClass(string(n.GarbageClass)) {
			result.Errors = append(result.Errors, err(prefix+".garbageClass",
				fmt.Sprintf("invalid value %q; must be one of unreachable, duplicate, obsolete, redundant, dead-api, dead-policy, dead-schema-field, dead-dependency-adapter", n.GarbageClass)))
		}
		if n.CanonicalID != "" {
			if !nodeIDs[n.CanonicalID] {
				result.Errors = append(result.Errors, err(prefix+".canonicalID",
					fmt.Sprintf("references unknown canonical node %q", n.CanonicalID)))
			}
			if n.CanonicalID == n.ID {
				result.Errors = append(result.Errors, err(prefix+".canonicalID", "must not reference itself"))
			}
		}
		if requiresCanonical(n.GarbageClass) && n.CanonicalID == "" {
			result.Errors = append(result.Errors, err(prefix+".canonicalID",
				fmt.Sprintf("node with garbageClass %q must declare a canonical target", n.GarbageClass)))
		}
		if n.Reachable && n.GarbageClass == GarbageUnreachable {
			result.Errors = append(result.Errors, err(prefix+".reachable", "unreachable node must not be marked reachable"))
		}
		for j, ref := range n.References {
			rp := fmt.Sprintf("%s.references[%d]", prefix, j)
			if ref.From == "" {
				result.Errors = append(result.Errors, err(rp+".from", "must not be empty"))
			}
			if ref.To == "" {
				result.Errors = append(result.Errors, err(rp+".to", "must not be empty"))
			}
			if ref.From != "" && !nodeIDs[ref.From] {
				result.Errors = append(result.Errors, err(rp+".from",
					fmt.Sprintf("references unknown node %q", ref.From)))
			}
			if ref.To != "" && !nodeIDs[ref.To] {
				result.Errors = append(result.Errors, err(rp+".to",
					fmt.Sprintf("references unknown node %q", ref.To)))
			}
		}
	}

	canonical := canonicalNodes(r.Nodes)
	for i, n := range r.Nodes {
		if n.CanonicalID == "" {
			continue
		}
		if target := canonical[n.CanonicalID]; target != nil && !target.Reachable {
			result.Errors = append(result.Errors, err(
				fmt.Sprintf("reduction.nodes[%d].canonicalID", i),
				fmt.Sprintf("canonical target %q is not reachable", n.CanonicalID)))
		}
	}

	blockIDs := make(map[string]bool, len(r.Blocks))
	for i, b := range r.Blocks {
		prefix := fmt.Sprintf("reduction.blocks[%d]", i)
		if b.ID == "" {
			result.Errors = append(result.Errors, err(prefix+".id", "must not be empty"))
		} else {
			if !strings.HasPrefix(b.ID, "blk_") {
				result.Errors = append(result.Errors, err(prefix+".id", `must start with "blk_"`))
			}
			if blockIDs[b.ID] {
				result.Errors = append(result.Errors, err(prefix+".id", fmt.Sprintf("duplicate block ID %q", b.ID)))
			}
			blockIDs[b.ID] = true
		}
		if b.CapabilityID != "" && !nodeIDs[b.CapabilityID] {
			result.Errors = append(result.Errors, err(prefix+".capabilityId",
				fmt.Sprintf("references unknown node %q", b.CapabilityID)))
		}
		for j, nid := range b.Nodes {
			if !nodeIDs[nid] {
				result.Errors = append(result.Errors, err(
					fmt.Sprintf("%s.nodes[%d]", prefix, j),
					fmt.Sprintf("references unknown node %q", nid)))
			}
		}
	}
}

func validGarbageClass(s string) bool {
	switch GarbageClass(s) {
	case GarbageUnreachable, GarbageDuplicate, GarbageObsolete, GarbageRedundant,
		GarbageDeadAPI, GarbageDeadPolicy, GarbageDeadSchemaField, GarbageDeadDependencyAdapter:
		return true
	}
	return false
}

func requiresCanonical(g GarbageClass) bool {
	return g == GarbageDuplicate || g == GarbageObsolete || g == GarbageRedundant
}

func canonicalNodes(nodes []ReductionNode) map[string]*ReductionNode {
	m := make(map[string]*ReductionNode, len(nodes))
	for i := range nodes {
		m[nodes[i].ID] = &nodes[i]
	}
	return m
}

func validateVerification(verifications []Verification, result *ValidationResult) {
	for i, v := range verifications {
		prefix := fmt.Sprintf("verification[%d]", i)
		requiredNonEmpty(result, prefix+".command", v.Command)
		requiredNonEmpty(result, prefix+".source", v.Source)
		enumCheck(result, prefix+".confidence", v.Confidence, validConfidence)
	}
}

func validateReviewability(r *Reviewability, result *ValidationResult) {
	if r == nil {
		return
	}
	prefix := "reviewability"
	enumCheck(result, prefix+".status", r.Status, func(s string) bool {
		return s == "pass" || s == "warn" || s == "fail" || s == "unknown"
	})
	if r.Recommendation != "" {
		enumCheck(result, prefix+".recommendation", r.Recommendation, func(s string) bool {
			return s == "review" || s == "decompose" || s == "reject" || s == "unknown"
		})
	}
	if r.Budget.MaxDiffLines < 0 {
		result.Errors = append(result.Errors, err(prefix+".budget.maxDiffLines", "must be non-negative"))
	}
	if r.Budget.MaxTouchedFiles < 0 {
		result.Errors = append(result.Errors, err(prefix+".budget.maxTouchedFiles", "must be non-negative"))
	}
	if r.Observed.AddedLines < 0 {
		result.Errors = append(result.Errors, err(prefix+".observed.addedLines", "must be non-negative"))
	}
	if r.Observed.DeletedLines < 0 {
		result.Errors = append(result.Errors, err(prefix+".observed.deletedLines", "must be non-negative"))
	}
	if r.Observed.ChangedFiles < 0 {
		result.Errors = append(result.Errors, err(prefix+".observed.changedFiles", "must be non-negative"))
	}
}

func validateDecomposition(d *Decomposition, result *ValidationResult) {
	if d == nil {
		return
	}
	requiredNonEmpty(result, "decomposition.strategy", d.Strategy)
	if len(d.Tasks) == 0 {
		result.Errors = append(result.Errors, err("decomposition.tasks", "must not be empty"))
	}
	ids := map[string]bool{}
	for i, task := range d.Tasks {
		prefix := fmt.Sprintf("decomposition.tasks[%d]", i)
		requiredNonEmpty(result, prefix+".id", task.ID)
		if task.ID != "" && !strings.HasPrefix(task.ID, "task_") {
			result.Errors = append(result.Errors, err(prefix+".id", `must start with "task_"`))
		}
		if task.ID != "" {
			if ids[task.ID] {
				result.Errors = append(result.Errors, err(prefix+".id", fmt.Sprintf("duplicate task ID %q", task.ID)))
			}
			ids[task.ID] = true
		}
		requiredNonEmpty(result, prefix+".goal", task.Goal)
		if task.MaxDiffLines < 0 {
			result.Errors = append(result.Errors, err(prefix+".maxDiffLines", "must be non-negative"))
		}
		for _, dep := range task.DependsOn {
			if task.ID != "" && !ids[dep] && dep != task.ID {
				found := false
				for j := 0; j < i; j++ {
					if d.Tasks[j].ID == dep {
						found = true
						break
					}
				}
				if !found {
					result.Errors = append(result.Errors, err(prefix+".dependsOn", fmt.Sprintf("references unknown task ID %q", dep)))
				}
			}
		}
	}
}

func requiredNonEmpty(result *ValidationResult, field, value string) {
	if value != "" {
		return
	}
	result.Errors = append(result.Errors, err(field, "must not be empty"))
}

func requiredStr(result *ValidationResult, field, value string, check func(string) bool) {
	if check(value) {
		return
	}
	result.Errors = append(result.Errors, err(field, fmt.Sprintf("invalid value %q", value)))
}

func enumCheck(result *ValidationResult, field, value string, valid func(string) bool) {
	if value == "" || valid(value) {
		return
	}
	result.Errors = append(result.Errors, err(field, `must be one of "low", "medium", "high"`))
}

func err(field, msg string) ValidationError {
	return ValidationError{Field: field, Message: msg, Severity: "error"}
}

func validRisk(s string) bool       { return s == "low" || s == "medium" || s == "high" }
func validSeverity(s string) bool   { return s == "low" || s == "medium" || s == "high" }
func validConfidence(s string) bool { return s == "low" || s == "medium" || s == "high" }
func isAbsolute(s string) bool {
	if len(s) == 0 {
		return false
	}
	if s[0] == '/' {
		return true
	}
	if len(s) >= 3 && s[0] == '\\' && s[1] == '\\' {
		return true
	}
	if len(s) >= 3 && isAlpha(s[0]) && s[1] == ':' && (s[2] == '\\' || s[2] == '/') {
		return true
	}
	return false
}

func isAlpha(c byte) bool { return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') }
