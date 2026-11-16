package grp

import "fmt"

// ReadinessResult separates execution readiness from structural validity.
// A plan can be a perfectly valid document and still not be safe to hand to
// an autonomous agent: findings it was built from may have no step.
type ReadinessResult struct {
	Ready bool `json:"ready"`
	// Actionable counts medium/high (or blocking) diagnostics that need a step.
	Actionable int `json:"actionable"`
	// Covered counts actionable diagnostics referenced by a step's requires.
	Covered int `json:"covered"`
	// Unresolved lists actionable diagnostic IDs with no step.
	Unresolved []string `json:"unresolved,omitempty"`
	Reasons    []string `json:"reasons,omitempty"`
}

func diagnosticIsBlocking(d Diagnostic) bool {
	for _, t := range d.Tags {
		if t == "blocking" {
			return true
		}
	}
	return false
}

// CheckExecutionReadiness reports whether every actionable diagnostic in the
// plan is addressed by at least one step. A plan with no actionable
// diagnostics and no steps is a legitimate clean no-op and is ready.
func CheckExecutionReadiness(p *Plan) *ReadinessResult {
	r := &ReadinessResult{Ready: true}
	if p == nil {
		r.Ready = false
		r.Reasons = append(r.Reasons, "plan is nil")
		return r
	}
	required := map[string]bool{}
	for _, s := range p.Steps {
		for _, req := range s.Requires {
			required[req] = true
		}
	}
	for _, d := range p.Diagnostics {
		if d.Severity == SeverityLow && !diagnosticIsBlocking(d) {
			continue
		}
		r.Actionable++
		if d.ID != "" && required[d.ID] {
			r.Covered++
			continue
		}
		r.Unresolved = append(r.Unresolved, d.ID)
	}
	for _, u := range p.Unplanned {
		if u.Disposition == "blocked" {
			r.Reasons = append(r.Reasons, fmt.Sprintf("%s in %s has no recipe and is blocking", u.Code, u.File))
		}
	}
	if len(r.Unresolved) > 0 {
		r.Ready = false
		r.Reasons = append(r.Reasons, fmt.Sprintf("%d actionable diagnostic(s) have no step: %v", len(r.Unresolved), r.Unresolved))
	}
	if len(r.Reasons) > 0 {
		r.Ready = false
	}
	return r
}
