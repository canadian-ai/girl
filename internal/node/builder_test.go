package node

import (
	"testing"

	"github.com/canadian-ai/girl/internal/ir"
)

func TestBuildFromIR(t *testing.T) {
	files := []*ir.FileIR{
		{
			Path:     "test.tsx",
			Language: "typescriptreact",
			Lines:    50,
			Components: []ir.ComponentIR{
				{
					Name:      "UserForm",
					StartLine: 1,
					EndLine:   40,
					Lines:     40,
					Hooks: []ir.HookIR{
						{Name: "useState", Line: 3},
						{Name: "useEffect", Line: 10},
					},
					StateVars: []ir.StateVarIR{
						{Name: "name", HasUpdater: true},
					},
					Effects: []ir.EffectIR{
						{Name: "useEffect", Line: 10, DepsCount: 1},
					},
					JSXBlocks: []ir.JSXBlockIR{
						{Element: "div", Line: 20},
						{Element: "span", Line: 25},
					},
					EventHandlers: []ir.EventHandlerIR{
						{Name: "handleSubmit", Line: 15},
					},
					Props: []ir.PropIR{
						{Name: "initialName", Type: "string", Required: true},
					},
					Exports: []ir.ExportIR{
						{Name: "UserForm", Default: true},
					},
				},
			},
			Imports: []ir.ImportIR{
				{Source: "react", Default: "React", Names: []string{"useState", "useEffect"}},
			},
			Hooks: []ir.HookIR{},
		},
	}

	g := BuildFromIR(files)

	if g == nil {
		t.Fatal("expected non-nil graph")
	}

	comps := g.AllNodesOfKind(KindComponent)
	if len(comps) != 1 {
		t.Fatalf("expected 1 component, got %d", len(comps))
	}
	comp := comps[0].(*ComponentNode)
	if comp.Name() != "UserForm" {
		t.Errorf("expected UserForm, got %s", comp.Name())
	}
	if comp.Lines != 40 {
		t.Errorf("expected 40 lines, got %d", comp.Lines)
	}

	hooks := g.AllNodesOfKind(KindHook)
	if len(hooks) != 2 {
		t.Fatalf("expected 2 hooks, got %d", len(hooks))
	}

	states := g.AllNodesOfKind(KindState)
	if len(states) != 1 {
		t.Fatalf("expected 1 state var, got %d", len(states))
	}

	jsxs := g.AllNodesOfKind(KindJSX)
	if len(jsxs) != 2 {
		t.Fatalf("expected 2 JSX nodes, got %d", len(jsxs))
	}

	imports := g.AllNodesOfKind(KindImport)
	if len(imports) != 1 {
		t.Fatalf("expected 1 import, got %d", len(imports))
	}

	sym := g.LookupSymbol("UserForm")
	if sym == "" {
		t.Fatal("expected symbol UserForm")
	}
}

func TestBuildFromIR_MultipleFiles(t *testing.T) {
	files := []*ir.FileIR{
		{
			Path: "comp-a.tsx",
			Components: []ir.ComponentIR{
				{Name: "ComponentA", StartLine: 1, EndLine: 10, Lines: 10},
			},
		},
		{
			Path: "comp-b.tsx",
			Components: []ir.ComponentIR{
				{Name: "ComponentB", StartLine: 1, EndLine: 20, Lines: 20},
			},
		},
	}

	g := BuildFromIR(files)
	comps := g.AllNodesOfKind(KindComponent)
	if len(comps) != 2 {
		t.Fatalf("expected 2 components across files, got %d", len(comps))
	}

	fileNodes := g.AllFiles()
	if len(fileNodes) != 2 {
		t.Fatalf("expected 2 files, got %d (%v)", len(fileNodes), fileNodes)
	}
}

func TestBuildFromIR_Empty(t *testing.T) {
	g := BuildFromIR([]*ir.FileIR{})
	if g == nil {
		t.Fatal("expected non-nil graph even with empty input")
	}
	all := g.AllNodes()
	if len(all) != 0 {
		t.Errorf("expected 0 nodes for empty input, got %d", len(all))
	}
}

func TestBuildFromIRAndResolve(t *testing.T) {
	files := []*ir.FileIR{
		{
			Path: "app.tsx",
			Components: []ir.ComponentIR{
				{
					Name:      "App",
					StartLine: 1,
					EndLine:   15,
					Lines:     15,
					Hooks: []ir.HookIR{
						{Name: "useQuery", Line: 3},
					},
				},
			},
		},
	}

	g := BuildFromIRAndResolve(files)
	if g == nil {
		t.Fatal("expected non-nil graph")
	}
	comps := g.AllNodesOfKind(KindComponent)
	if len(comps) != 1 {
		t.Fatalf("expected 1 component, got %d", len(comps))
	}
	if comps[0].Name() != "App" {
		t.Errorf("expected App, got %s", comps[0].Name())
	}
}

func TestBuilderWithWalker(t *testing.T) {
	files := []*ir.FileIR{
		{
			Path: "test.tsx",
			Components: []ir.ComponentIR{
				{
					Name:      "ProfileCard",
					StartLine: 1,
					EndLine:   30,
					Lines:     30,
					Hooks: []ir.HookIR{
						{Name: "useState", Line: 3},
						{Name: "useEffect", Line: 8},
						{Name: "useQuery", Line: 12},
					},
					StateVars: []ir.StateVarIR{
						{Name: "user", HasUpdater: true},
					},
					Effects: []ir.EffectIR{
						{Name: "useEffect", Line: 8, DepsCount: 1},
					},
				},
			},
		},
	}

	g := BuildFromIR(files)

	type walkResult struct {
		components int
		hooks      int
		states     int
		effects    int
	}

	result := &walkResult{}

	visitor := struct {
		*BaseTypedVisitor
		onComponent func(*VisitContext, *ComponentNode)
		onHook      func(*VisitContext, *HookNode)
		onState     func(*VisitContext, *StateNode)
		onEffect    func(*VisitContext, *EffectNode)
	}{}

	visitor.onComponent = func(ctx *VisitContext, n *ComponentNode) {
		result.components++
		if n.Name() != "ProfileCard" {
			t.Errorf("expected ProfileCard, got %s", n.Name())
		}
	}

	walker := NewWalker(g)

	_ = visitor.onComponent
	_ = visitor.onHook
	_ = visitor.onState
	_ = visitor.onEffect

	walker.Register(struct{}{})

	err := walker.Walk()
	if err != nil {
		t.Fatalf("walk error: %v", err)
	}
	_ = result
}

func TestBuilderCrossFileReferences(t *testing.T) {
	files := []*ir.FileIR{
		{
			Path: "shared.tsx",
			Components: []ir.ComponentIR{
				{Name: "SharedButton", StartLine: 1, EndLine: 5, Lines: 5},
			},
		},
		{
			Path: "main.tsx",
			Imports: []ir.ImportIR{
				{Source: "./shared", Names: []string{"SharedButton"}},
			},
		},
	}

	g := BuildFromIR(files)
	if g.LookupSymbol("SharedButton") == "" {
		t.Fatal("expected SharedButton symbol to be registered")
	}

	root1 := g.FileNodeFor("shared.tsx")
	root2 := g.FileNodeFor("main.tsx")
	if root1 == "" {
		t.Fatal("expected file node for shared.tsx")
	}
	if root2 == "" {
		t.Fatal("expected file node for main.tsx")
	}
}

func TestGraphFromRealFixture(t *testing.T) {
	files := []*ir.FileIR{
		{
			Path:     "insurance-review-panel.tsx",
			Language: "typescriptreact",
			Lines:    120,
			Components: []ir.ComponentIR{
				{
					Name:      "InsuranceReviewPanel",
					StartLine: 12,
					EndLine:   120,
					Lines:     108,
					Hooks: []ir.HookIR{
						{Name: "useState", Line: 14},
						{Name: "useState", Line: 15},
						{Name: "useState", Line: 16},
						{Name: "useQuery", Line: 18},
						{Name: "useQuery", Line: 21},
						{Name: "useMutation", Line: 26},
						{Name: "useEffect", Line: 29},
						{Name: "useEffect", Line: 40},
						{Name: "useCallback", Line: 50},
						{Name: "useCallback", Line: 62},
						{Name: "useCallback", Line: 84},
					},
					StateVars: []ir.StateVarIR{
						{Name: "review", HasUpdater: true},
						{Name: "selectedDoc", HasUpdater: true},
						{Name: "isSubmitting", HasUpdater: true},
					},
					Effects: []ir.EffectIR{
						{Name: "useEffect", Line: 29},
						{Name: "useEffect", Line: 40},
					},
					JSXBlocks: []ir.JSXBlockIR{
						{Element: "div", Line: 97},
						{Element: "h2", Line: 98},
						{Element: "div", Line: 100},
						{Element: "div", Line: 105},
						{Element: "div", Line: 118},
					},
					EventHandlers: []ir.EventHandlerIR{
						{Name: "handleStatusChange", Line: 50},
						{Name: "handleSubmit", Line: 62},
						{Name: "handleKeyDown", Line: 84},
					},
					Props: []ir.PropIR{
						{Name: "tenantId", Type: "string", Required: true},
						{Name: "workspaceId", Type: "string", Required: true},
						{Name: "onComplete", Type: "(reviewId: string) => void", Required: true},
					},
					Exports: []ir.ExportIR{
						{Name: "InsuranceReviewPanel", Default: true},
					},
				},
			},
			Imports: []ir.ImportIR{
				{Source: "react", Names: []string{"useState", "useEffect", "useCallback"}},
				{Source: "convex/react", Names: []string{"useQuery", "useMutation"}},
				{Source: "../convex/_generated/api", Default: "api"},
			},
		},
	}

	g := BuildFromIRAndResolve(files)

	t.Run("component", func(t *testing.T) {
		comps := g.AllNodesOfKind(KindComponent)
		if len(comps) != 1 {
			t.Fatalf("expected 1 component, got %d", len(comps))
		}
		comp := comps[0].(*ComponentNode)
		if comp.Name() != "InsuranceReviewPanel" {
			t.Errorf("wrong name: %s", comp.Name())
		}
		if comp.Lines != 108 {
			t.Errorf("wrong lines: %d", comp.Lines)
		}
	})

	t.Run("hooks", func(t *testing.T) {
		hooks := g.AllNodesOfKind(KindHook)
		if len(hooks) != 11 {
			t.Errorf("expected 11 hooks, got %d", len(hooks))
		}
	})

	t.Run("states", func(t *testing.T) {
		states := g.AllNodesOfKind(KindState)
		if len(states) != 3 {
			t.Errorf("expected 3 state vars, got %d", len(states))
		}
	})

	t.Run("effects", func(t *testing.T) {
		effects := g.AllNodesOfKind(KindEffect)
		if len(effects) != 2 {
			t.Errorf("expected 2 effects, got %d", len(effects))
		}
	})

	t.Run("jsx", func(t *testing.T) {
		jsxs := g.AllNodesOfKind(KindJSX)
		if len(jsxs) != 5 {
			t.Errorf("expected 5 JSX nodes, got %d", len(jsxs))
		}
	})

	t.Run("events", func(t *testing.T) {
		events := g.AllNodesOfKind(KindEvent)
		if len(events) != 3 {
			t.Errorf("expected 3 events, got %d", len(events))
		}
	})

	t.Run("exports", func(t *testing.T) {
		exports := g.AllNodesOfKind(KindExport)
		if len(exports) != 1 {
			t.Errorf("expected 1 export, got %d", len(exports))
		}
	})

	t.Run("files", func(t *testing.T) {
		files := g.AllFiles()
		if len(files) != 1 {
			t.Errorf("expected 1 file, got %d", len(files))
		}
	})

	t.Run("symbols", func(t *testing.T) {
		if g.LookupSymbol("InsuranceReviewPanel") == "" {
			t.Error("expected InsuranceReviewPanel symbol")
		}
	})
}

func TestBuilderLinesCount(t *testing.T) {
	files := []*ir.FileIR{
		{
			Path: "small.tsx",
			Components: []ir.ComponentIR{
				{Name: "SmallComp", StartLine: 1, EndLine: 5, Lines: 5},
				{Name: "LargeComp", StartLine: 10, EndLine: 250, Lines: 240},
			},
		},
	}

	g := BuildFromIR(files)
	comps := g.AllNodesOfKind(KindComponent)
	if len(comps) != 2 {
		t.Fatalf("expected 2 components, got %d", len(comps))
	}

	for _, n := range comps {
		comp := n.(*ComponentNode)
		if comp.Name() == "SmallComp" && comp.Lines != 5 {
			t.Errorf("expected SmallComp 5 lines, got %d", comp.Lines)
		}
		if comp.Name() == "LargeComp" && comp.Lines != 240 {
			t.Errorf("expected LargeComp 240 lines, got %d", comp.Lines)
		}
	}
}
