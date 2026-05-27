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
	validateVerification(p.Verification, result)
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
		enumCheck(result, prefix+".risk", string(s.Risk), validRisk)
		for _, req := range s.Requires {
			if !diagIDs[req] {
				result.Errors = append(result.Errors, err(prefix+".requires", fmt.Sprintf("references unknown diagnostic ID %q", req)))
			}
		}
		for j, v := range s.Verify {
			vp := fmt.Sprintf("%s.verify[%d]", prefix, j)
			requiredNonEmpty(result, vp+".command", v.Command)
			enumCheck(result, vp+".confidence", v.Confidence, validConfidence)
		}
	}
}

func validateVerification(verifications []Verification, result *ValidationResult) {
	for i, v := range verifications {
		prefix := fmt.Sprintf("verification[%d]", i)
		requiredNonEmpty(result, prefix+".command", v.Command)
		requiredNonEmpty(result, prefix+".source", v.Source)
		enumCheck(result, prefix+".confidence", v.Confidence, validConfidence)
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

func validRisk(s string) bool    { return s == "low" || s == "medium" || s == "high" }
func validSeverity(s string) bool { return s == "low" || s == "medium" || s == "high" }
func validConfidence(s string) bool { return s == "low" || s == "medium" || s == "high" }
func isAbsolute(s string) bool   { return strings.HasPrefix(s, "/") }
