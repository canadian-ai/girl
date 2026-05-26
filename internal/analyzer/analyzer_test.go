package analyzer

import (
	"testing"

	"github.com/canadian-ai/girl/internal/ir"
)

func makeFileIR(components ...ir.ComponentIR) *ir.FileIR {
	return &ir.FileIR{
		Path:       "/test/file.tsx",
		Language:   "tsx",
		Lines:      500,
		Components: components,
	}
}

func makeComponent(name string, overrides func(*ir.ComponentIR)) ir.ComponentIR {
	c := ir.ComponentIR{
		Name:             name,
		Kind:             ir.ComponentKindFunction,
		StartLine:        1,
		EndLine:          10,
		Lines:            10,
		HasKeyDown:       false,
		HasAnalytics:     false,
		ConditionalCount: 0,
		LoopCount:        0,
	}
	if overrides != nil {
		overrides(&c)
	}
	return c
}

func TestDetectLargeComponents_SetsStructuredFields(t *testing.T) {
	a := NewAnalyzer(&Config{MaxComponentLines: 5, MinRepeatedJSX: 3, MaxHooksPerComponent: 5, MaxStateVars: 4, MaxEffects: 2, MaxConditionals: 5, MaxLoops: 3})
	f := makeFileIR(makeComponent("LargeOne", func(c *ir.ComponentIR) {
		c.Lines = 50
		c.StartLine = 10
		c.EndLine = 60
	}))
	diags := a.detectLargeComponents(f)
	if len(diags) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d", len(diags))
	}
	d := diags[0]
	if d.Kind != ir.NodeKindComponent {
		t.Errorf("Kind = %q, want %q", d.Kind, ir.NodeKindComponent)
	}
	if d.Symbol != "LargeOne" {
		t.Errorf("Symbol = %q, want %q", d.Symbol, "LargeOne")
	}
	if d.Component != "LargeOne" {
		t.Errorf("Component = %q, want %q", d.Component, "LargeOne")
	}
	if d.EndLine != 60 {
		t.Errorf("EndLine = %d, want %d", d.EndLine, 60)
	}
	if d.Span == nil {
		t.Fatal("Span is nil")
	}
	if d.Span.StartLine != 10 {
		t.Errorf("Span.StartLine = %d, want %d", d.Span.StartLine, 10)
	}
	if d.Span.EndLine != 60 {
		t.Errorf("Span.EndLine = %d, want %d", d.Span.EndLine, 60)
	}
}

func TestDetectRepeatedJSX_SetsStructuredFields(t *testing.T) {
	a := NewAnalyzer(&Config{MaxComponentLines: 200, MinRepeatedJSX: 2, MaxHooksPerComponent: 5, MaxStateVars: 4, MaxEffects: 2, MaxConditionals: 5, MaxLoops: 3})
	f := makeFileIR(makeComponent("Repeater", func(c *ir.ComponentIR) {
		c.JSXBlocks = []ir.JSXBlockIR{
			{Element: "div", Line: 1},
			{Element: "div", Line: 2},
			{Element: "div", Line: 3},
		}
	}))
	diags := a.detectRepeatedJSX(f)
	if len(diags) == 0 {
		t.Fatal("expected at least 1 diagnostic")
	}
	d := diags[0]
	if d.Kind != ir.NodeKindComponent {
		t.Errorf("Kind = %q, want %q", d.Kind, ir.NodeKindComponent)
	}
	if d.Symbol != "Repeater" {
		t.Errorf("Symbol = %q, want %q", d.Symbol, "Repeater")
	}
	if d.Component != "Repeater" {
		t.Errorf("Component = %q, want %q", d.Component, "Repeater")
	}
}

func TestDetectTooManyHooks_SetsStructuredFields(t *testing.T) {
	a := NewAnalyzer(&Config{MaxComponentLines: 200, MinRepeatedJSX: 3, MaxHooksPerComponent: 1, MaxStateVars: 4, MaxEffects: 2, MaxConditionals: 5, MaxLoops: 3})
	f := makeFileIR(makeComponent("TooManyHooks", func(c *ir.ComponentIR) {
		c.Hooks = []ir.HookIR{
			{Name: "useState", Line: 1},
			{Name: "useEffect", Line: 2},
			{Name: "useMemo", Line: 3},
		}
	}))
	diags := a.detectTooManyHooks(f)
	if len(diags) == 0 {
		t.Fatal("expected at least 1 diagnostic")
	}
	d := diags[0]
	if d.Kind != ir.NodeKindComponent {
		t.Errorf("Kind = %q, want %q", d.Kind, ir.NodeKindComponent)
	}
	if d.Symbol != "TooManyHooks" {
		t.Errorf("Symbol = %q, want %q", d.Symbol, "TooManyHooks")
	}
	if d.Component != "TooManyHooks" {
		t.Errorf("Component = %q, want %q", d.Component, "TooManyHooks")
	}
}

func TestDetectTooManyStateVars_SetsStructuredFields(t *testing.T) {
	a := NewAnalyzer(&Config{MaxComponentLines: 200, MinRepeatedJSX: 3, MaxHooksPerComponent: 5, MaxStateVars: 1, MaxEffects: 2, MaxConditionals: 5, MaxLoops: 3})
	f := makeFileIR(makeComponent("TooManyState", func(c *ir.ComponentIR) {
		c.StateVars = []ir.StateVarIR{
			{Name: "a", Line: 1},
			{Name: "b", Line: 2},
			{Name: "c", Line: 3},
		}
	}))
	diags := a.detectTooManyStateVars(f)
	if len(diags) == 0 {
		t.Fatal("expected at least 1 diagnostic")
	}
	d := diags[0]
	if d.Kind != ir.NodeKindComponent {
		t.Errorf("Kind = %q, want %q", d.Kind, ir.NodeKindComponent)
	}
	if d.Symbol != "TooManyState" {
		t.Errorf("Symbol = %q, want %q", d.Symbol, "TooManyState")
	}
	if d.Component != "TooManyState" {
		t.Errorf("Component = %q, want %q", d.Component, "TooManyState")
	}
}

func TestDetectTooManyEffects_SetsStructuredFields(t *testing.T) {
	a := NewAnalyzer(&Config{MaxComponentLines: 200, MinRepeatedJSX: 3, MaxHooksPerComponent: 5, MaxStateVars: 4, MaxEffects: 1, MaxConditionals: 5, MaxLoops: 3})
	f := makeFileIR(makeComponent("TooManyEffects", func(c *ir.ComponentIR) {
		c.Effects = []ir.EffectIR{
			{Name: "useEffect", Line: 1},
			{Name: "useEffect", Line: 2},
			{Name: "useEffect", Line: 3},
		}
	}))
	diags := a.detectTooManyEffects(f)
	if len(diags) == 0 {
		t.Fatal("expected at least 1 diagnostic")
	}
	d := diags[0]
	if d.Kind != ir.NodeKindComponent {
		t.Errorf("Kind = %q, want %q", d.Kind, ir.NodeKindComponent)
	}
	if d.Symbol != "TooManyEffects" {
		t.Errorf("Symbol = %q, want %q", d.Symbol, "TooManyEffects")
	}
	if d.Component != "TooManyEffects" {
		t.Errorf("Component = %q, want %q", d.Component, "TooManyEffects")
	}
}

func TestDetectComplexConditionals_SetsStructuredFields(t *testing.T) {
	a := NewAnalyzer(&Config{MaxComponentLines: 200, MinRepeatedJSX: 3, MaxHooksPerComponent: 5, MaxStateVars: 4, MaxEffects: 2, MaxConditionals: 1, MaxLoops: 3})
	f := makeFileIR(makeComponent("ComplexCond", func(c *ir.ComponentIR) {
		c.ConditionalCount = 5
	}))
	diags := a.detectComplexConditionals(f)
	if len(diags) == 0 {
		t.Fatal("expected at least 1 diagnostic")
	}
	d := diags[0]
	if d.Kind != ir.NodeKindComponent {
		t.Errorf("Kind = %q, want %q", d.Kind, ir.NodeKindComponent)
	}
	if d.Symbol != "ComplexCond" {
		t.Errorf("Symbol = %q, want %q", d.Symbol, "ComplexCond")
	}
	if d.Component != "ComplexCond" {
		t.Errorf("Component = %q, want %q", d.Component, "ComplexCond")
	}
}

func TestDetectMixedResponsibilities_SetsStructuredFields(t *testing.T) {
	a := NewAnalyzer(&Config{MaxComponentLines: 200, MinRepeatedJSX: 3, MaxHooksPerComponent: 5, MaxStateVars: 4, MaxEffects: 2, MaxConditionals: 5, MaxLoops: 3})
	f := makeFileIR(makeComponent("MixedResp", func(c *ir.ComponentIR) {
		c.StateVars = []ir.StateVarIR{{Name: "a"}, {Name: "b"}, {Name: "c"}}
		c.Effects = []ir.EffectIR{{Name: "useEffect"}, {Name: "useEffect"}}
		c.Hooks = []ir.HookIR{{Name: "useState"}, {Name: "useEffect"}, {Name: "useMemo"}, {Name: "useCallback"}}
		c.HasKeyDown = true
		c.HasAnalytics = true
	}))
	diags := a.detectMixedResponsibilities(f)
	if len(diags) == 0 {
		t.Fatal("expected at least 1 diagnostic")
	}
	d := diags[0]
	if d.Kind != ir.NodeKindComponent {
		t.Errorf("Kind = %q, want %q", d.Kind, ir.NodeKindComponent)
	}
	if d.Symbol != "MixedResp" {
		t.Errorf("Symbol = %q, want %q", d.Symbol, "MixedResp")
	}
	if d.Component != "MixedResp" {
		t.Errorf("Component = %q, want %q", d.Component, "MixedResp")
	}
}

func TestDetectHardcodedData_SetsStructuredFields(t *testing.T) {
	a := NewAnalyzer(&Config{MaxComponentLines: 200, MinRepeatedJSX: 3, MaxHooksPerComponent: 5, MaxStateVars: 4, MaxEffects: 2, MaxConditionals: 5, MaxLoops: 3})
	f := makeFileIR(makeComponent("HardData", func(c *ir.ComponentIR) {
		c.Lines = 50
	}))
	diags := a.detectHardcodedData(f)
	if len(diags) != 0 {
		t.Fatalf("expected 0 diagnostics (getComponentBody doesn't match hardcoded array pattern), got %d", len(diags))
	}
}

func TestDetectMissingPropTypes_SetsStructuredFields(t *testing.T) {
	a := NewAnalyzer(&Config{MaxComponentLines: 200, MinRepeatedJSX: 3, MaxHooksPerComponent: 5, MaxStateVars: 4, MaxEffects: 2, MaxConditionals: 5, MaxLoops: 3})
	f := makeFileIR(makeComponent("NoProps", func(c *ir.ComponentIR) {
		c.Props = []ir.PropIR{{Name: "title", Type: ""}}
		c.JSXBlocks = []ir.JSXBlockIR{{Element: "div"}}
	}))
	diags := a.detectMissingPropTypes(f)
	if len(diags) == 0 {
		t.Fatal("expected at least 1 diagnostic")
	}
	d := diags[0]
	if d.Kind != ir.NodeKindComponent {
		t.Errorf("Kind = %q, want %q", d.Kind, ir.NodeKindComponent)
	}
	if d.Symbol != "NoProps" {
		t.Errorf("Symbol = %q, want %q", d.Symbol, "NoProps")
	}
	if d.Component != "NoProps" {
		t.Errorf("Component = %q, want %q", d.Component, "NoProps")
	}
}
