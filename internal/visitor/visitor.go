package visitor

import "github.com/canadian-ai/girl/internal/ir"

type Visitor interface {
	VisitFile(file *ir.FileIR) error
	VisitComponent(comp *ir.ComponentIR) error
	VisitHook(hook *ir.HookIR) error
	GetResult() interface{}
}

type BaseVisitor struct{}

func (v *BaseVisitor) VisitFile(file *ir.FileIR) error           { return nil }
func (v *BaseVisitor) VisitComponent(comp *ir.ComponentIR) error { return nil }
func (v *BaseVisitor) VisitHook(hook *ir.HookIR) error           { return nil }
func (v *BaseVisitor) GetResult() interface{}                    { return nil }

type VisitorPipeline struct {
	Visitors []Visitor
}

func NewPipeline(visitors ...Visitor) *VisitorPipeline {
	return &VisitorPipeline{Visitors: visitors}
}

func (p *VisitorPipeline) ProcessFile(file *ir.FileIR) error {
	for _, v := range p.Visitors {
		if err := v.VisitFile(file); err != nil {
			return err
		}
	}
	for i := range file.Components {
		for _, v := range p.Visitors {
			if err := v.VisitComponent(&file.Components[i]); err != nil {
				return err
			}
		}
	}
	for i := range file.Hooks {
		for _, v := range p.Visitors {
			if err := v.VisitHook(&file.Hooks[i]); err != nil {
				return err
			}
		}
	}
	return nil
}

type ResponsibilityVisitor struct {
	BaseVisitor
	Responsibilities map[string][]string
}

func NewResponsibilityVisitor() *ResponsibilityVisitor {
	return &ResponsibilityVisitor{
		Responsibilities: make(map[string][]string),
	}
}

func (v *ResponsibilityVisitor) VisitComponent(comp *ir.ComponentIR) error {
	var resp []string
	resp = append(resp, hookResponsibilities(comp)...)
	if len(comp.StateVars) > 3 {
		resp = append(resp, "local-state-management")
	}
	if len(comp.Effects) > 2 {
		resp = append(resp, "side-effects")
	}
	if comp.HasKeyDown {
		resp = append(resp, "keyboard-events")
	}
	if comp.HasAnalytics {
		resp = append(resp, "analytics")
	}
	if comp.ConditionalCount > 5 {
		resp = append(resp, "complex-rendering-logic")
	}
	if comp.LoopCount > 3 {
		resp = append(resp, "list-rendering")
	}

	v.Responsibilities[comp.Name] = resp
	return nil
}

func hookResponsibilities(comp *ir.ComponentIR) []string {
	hasForm := false
	hasData := false
	for _, h := range comp.Hooks {
		switch h.Name {
		case "useForm", "useController", "useFieldArray":
			hasForm = true
		case "useQuery", "useMutation":
			hasData = true
		}
	}

	var resp []string
	if hasForm {
		resp = append(resp, "form-logic")
	}
	if hasData {
		resp = append(resp, "data-fetching")
	}
	return resp
}

func (v *ResponsibilityVisitor) GetResult() interface{} {
	return v.Responsibilities
}
