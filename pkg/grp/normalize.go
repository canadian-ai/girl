package grp

import (
	"crypto/sha256"
	"fmt"
	"io"
	"sort"
	"strings"
)

var severityOrder = map[Severity]int{
	SeverityHigh: 0, SeverityMedium: 1, SeverityLow: 2,
}

func NormalizePlan(p *Plan) {
	if p == nil {
		return
	}

	sort.SliceStable(p.Diagnostics, diagLess(p.Diagnostics))
	oldToNew := remapDiagIDs(p.Diagnostics)
	sort.SliceStable(p.Steps, stepLess(p.Steps))
	renumberSteps(p.Steps, p.Diagnostics, oldToNew)

	if p.Decomposition != nil {
		sort.SliceStable(p.Decomposition.Tasks, func(i, j int) bool {
			return p.Decomposition.Tasks[i].ID < p.Decomposition.Tasks[j].ID
		})
		for i := range p.Decomposition.Tasks {
			sort.Strings(p.Decomposition.Tasks[i].AllowedFiles)
			sort.Strings(p.Decomposition.Tasks[i].ForbiddenFiles)
			sort.Strings(p.Decomposition.Tasks[i].DependsOn)
			sort.Strings(p.Decomposition.Tasks[i].Verification)
		}
	}
}

func diagLess(diags []Diagnostic) func(i, j int) bool {
	return func(i, j int) bool {
		a, b := diags[i], diags[j]
		if a.Severity != b.Severity {
			return severityOrder[a.Severity] < severityOrder[b.Severity]
		}
		if a.File != b.File {
			return a.File < b.File
		}
		if a.Line != b.Line {
			if a.Line == 0 {
				return false
			}
			if b.Line == 0 {
				return true
			}
			return a.Line < b.Line
		}
		if a.Code != b.Code {
			return a.Code < b.Code
		}
		aSym, bSym := symName(a), symName(b)
		if aSym != bSym {
			return aSym < bSym
		}
		return a.Message < b.Message
	}
}

func remapDiagIDs(diags []Diagnostic) map[string]string {
	oldToNew := make(map[string]string, len(diags))
	for i := range diags {
		old := diags[i].ID
		newID := fmt.Sprintf("diag_%03d", i+1)
		diags[i].ID = newID
		oldToNew[old] = newID
	}
	return oldToNew
}

func stepLess(steps []Step) func(i, j int) bool {
	return func(i, j int) bool {
		a, b := steps[i], steps[j]
		if a.Risk != b.Risk {
			return severityOrder[a.Risk] < severityOrder[b.Risk]
		}
		if a.Target.File != b.Target.File {
			return a.Target.File < b.Target.File
		}
		if a.Recipe != b.Recipe {
			return a.Recipe < b.Recipe
		}
		return a.Action < b.Action
	}
}

func renumberSteps(steps []Step, diags []Diagnostic, oldToNew map[string]string) {
	for i := range steps {
		for j, req := range steps[i].Requires {
			if newID, ok := oldToNew[req]; ok {
				steps[i].Requires[j] = newID
			}
		}
		steps[i].ID = fmt.Sprintf("step_%03d_%s_%s", i+1, diagCodeForStep(steps[i], diags), stepTargetSlug(steps[i], diags))
	}
}

func ComputePlanID(p *Plan) string {
	h := sha256.New()
	io.WriteString(h, p.SpecVersion)
	io.WriteString(h, p.Source)
	io.WriteString(h, p.Subject)
	io.WriteString(h, p.Language)
	io.WriteString(h, p.Goal)
	for _, d := range p.Diagnostics {
		io.WriteString(h, d.Code)
		io.WriteString(h, d.File)
		fmt.Fprintf(h, "%d", d.Line)
		if d.Symbol != nil {
			io.WriteString(h, d.Symbol.Name)
		}
	}
	for _, s := range p.Steps {
		io.WriteString(h, s.Recipe)
		io.WriteString(h, s.Action)
		io.WriteString(h, s.Target.File)
	}
	sum := h.Sum(nil)
	return fmt.Sprintf("grp_%08x", sum[:4])
}

func diagCodeForStep(s Step, diags []Diagnostic) string {
	if len(s.Requires) > 0 {
		for _, d := range diags {
			if d.ID == s.Requires[0] {
				return d.Code
			}
		}
	}
	code := s.Recipe
	if code == "" {
		code = "unknown"
	}
	return code
}

func stepTargetSlug(s Step, diags []Diagnostic) string {
	for _, req := range s.Requires {
		for _, d := range diags {
			if d.ID == req && d.Symbol != nil && d.Symbol.Name != "" {
				return slugString(d.Symbol.Name)
			}
		}
	}
	file := s.Target.File
	if idx := strings.LastIndex(file, "/"); idx >= 0 {
		file = file[idx+1:]
	}
	if idx := strings.LastIndex(file, "."); idx >= 0 {
		file = file[:idx]
	}
	slug := slugString(file)
	if slug == "" {
		slug = "target"
	}
	return slug
}

func symName(d Diagnostic) string {
	if d.Symbol != nil {
		return d.Symbol.Name
	}
	return ""
}

func slugString(s string) string {
	var result strings.Builder
	for _, r := range strings.ToLower(s) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			result.WriteRune(r)
		} else if r == ' ' || r == '-' || r == '_' {
			if !strings.HasSuffix(result.String(), "-") {
				result.WriteRune('-')
			}
		}
	}
	slug := strings.TrimRight(result.String(), "-")
	if len(slug) > 40 {
		slug = slug[:40]
	}
	return slug
}
