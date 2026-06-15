package recipes

import (
	"testing"

	"github.com/canadian-ai/girl/internal/ir"
)

func TestThresholds_DefaultNoMatch(t *testing.T) {
	comp := &ir.ComponentIR{
		Name:      "TestComponent",
		Lines:     150,
		Hooks:     make([]ir.HookIR, 3),
		StateVars: make([]ir.StateVarIR, 3),
		Effects:   make([]ir.EffectIR, 1),
	}
	engine := NewEngine()
	matches := engine.Match(comp)
	if len(matches) != 0 {
		t.Errorf("expected 0 matches with default thresholds, got %d", len(matches))
		for _, m := range matches {
			t.Logf("  matched: %s", m.RecipeID)
		}
	}
}

func TestThresholds_LoweredMatches(t *testing.T) {
	comp := &ir.ComponentIR{
		Name:      "TestComponent",
		Lines:     150,
		Hooks:     make([]ir.HookIR, 3),
		StateVars: make([]ir.StateVarIR, 3),
		Effects:   make([]ir.EffectIR, 1),
	}
	low := &Thresholds{
		LargeComponentLines:  100,
		RepeatedJSXCount:     3,
		MaxHooksPerComponent: 2,
		MaxStateVars:         2,
		MaxEffects:           0,
	}
	engine := NewEngine(low)
	matches := engine.Match(comp)

	expected := map[string]bool{
		"react.split-large-component":  true,
		"react.extract-custom-hook":    true,
		"react.reduce-state-vars":      true,
		"react.consolidate-effects":    true,
	}
	for _, m := range matches {
		if !expected[m.RecipeID] {
			t.Errorf("unexpected match: %s", m.RecipeID)
		}
		delete(expected, m.RecipeID)
	}
	if len(expected) > 0 {
		for id := range expected {
			t.Errorf("expected but not matched: %s", id)
		}
	}
}

func TestThresholds_CustomValuesUsed(t *testing.T) {
	large := &Thresholds{
		LargeComponentLines:  50,
		RepeatedJSXCount:     3,
		MaxHooksPerComponent: 5,
		MaxStateVars:         4,
		MaxEffects:           2,
	}
	small := &Thresholds{
		LargeComponentLines:  200,
		RepeatedJSXCount:     3,
		MaxHooksPerComponent: 5,
		MaxStateVars:         4,
		MaxEffects:           2,
	}

	comp := &ir.ComponentIR{
		Name:  "TestComponent",
		Lines: 60,
	}

	largeEngine := NewEngine(large)
	smallEngine := NewEngine(small)

	if len(largeEngine.Match(comp)) == 0 {
		t.Error("expected match with LargeComponentLines=50 and comp.Lines=60")
	}
	if len(smallEngine.Match(comp)) != 0 {
		t.Error("expected no match with LargeComponentLines=200 and comp.Lines=60")
	}
}

func TestAddPropTypes_DefaultThreshold_Match(t *testing.T) {
	comp := &ir.ComponentIR{
		Name:  "UntypedBig",
		Lines: 31,
		Props: []ir.PropIR{
			{Name: "title", Type: ""},
		},
	}
	engine := NewEngine()
	matches := engine.Match(comp)
	found := false
	for _, m := range matches {
		if m.RecipeID == "react.add-prop-types" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected add-prop-types match for untyped 31-line component with props")
	}
}

func TestAddPropTypes_DefaultThreshold_NoMatchBelow(t *testing.T) {
	comp := &ir.ComponentIR{
		Name:  "UntypedSmall",
		Lines: 30,
		Props: []ir.PropIR{
			{Name: "title", Type: ""},
		},
	}
	engine := NewEngine()
	matches := engine.Match(comp)
	for _, m := range matches {
		if m.RecipeID == "react.add-prop-types" {
			t.Error("expected no add-prop-types match for 30-line untyped component")
		}
	}
}

func TestAddPropTypes_CustomThreshold(t *testing.T) {
	comp := &ir.ComponentIR{
		Name:  "CustomThreshold",
		Lines: 11,
		Props: []ir.PropIR{
			{Name: "title", Type: ""},
		},
	}
	low := &Thresholds{
		LargeComponentLines:  200,
		RepeatedJSXCount:     3,
		MaxHooksPerComponent: 5,
		MaxStateVars:         4,
		MaxEffects:           2,
		UntypedPropsMinLines: 10,
	}
	engine := NewEngine(low)
	matches := engine.Match(comp)
	found := false
	for _, m := range matches {
		if m.RecipeID == "react.add-prop-types" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected add-prop-types match with UntypedPropsMinLines=10 and comp.Lines=11")
	}
}

func TestAddPropTypes_TypedPropsNoMatch(t *testing.T) {
	comp := &ir.ComponentIR{
		Name:  "TypedBig",
		Lines: 100,
		Props: []ir.PropIR{
			{Name: "title", Type: "string"},
		},
	}
	engine := NewEngine()
	matches := engine.Match(comp)
	for _, m := range matches {
		if m.RecipeID == "react.add-prop-types" {
			t.Error("expected no add-prop-types match for typed props even above threshold")
		}
	}
}

func TestAddPropTypes_NoPropsNoMatch(t *testing.T) {
	comp := &ir.ComponentIR{
		Name:  "NoProps",
		Lines: 200,
		Props: []ir.PropIR{},
	}
	engine := NewEngine()
	matches := engine.Match(comp)
	for _, m := range matches {
		if m.RecipeID == "react.add-prop-types" {
			t.Error("expected no add-prop-types match for component with no props")
		}
	}
}

func TestThresholds_NilDefaults(t *testing.T) {
	comp := &ir.ComponentIR{
		Name:      "TestComponent",
		Lines:     250,
		Hooks:     make([]ir.HookIR, 6),
		StateVars: make([]ir.StateVarIR, 5),
		Effects:   make([]ir.EffectIR, 3),
	}
	engine := NewEngine(nil)
	matches := engine.Match(comp)

	expected := map[string]bool{
		"react.split-large-component": true,
		"react.extract-custom-hook":   true,
		"react.reduce-state-vars":     true,
		"react.consolidate-effects":   true,
	}
	if len(matches) != len(expected) {
		t.Errorf("expected %d matches with nil thresholds, got %d", len(expected), len(matches))
	}
	for _, m := range matches {
		if !expected[m.RecipeID] {
			t.Errorf("unexpected match: %s", m.RecipeID)
		}
	}
}
