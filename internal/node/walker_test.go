package node

import (
	"testing"
)

type testSpyVisitor struct {
	entered []string
	exited  []string
}

func newSpyVisitor() *testSpyVisitor {
	return &testSpyVisitor{
		entered: []string{},
		exited:  []string{},
	}
}

func (v *testSpyVisitor) EnterNode(ctx *VisitContext, n Node) error {
	v.entered = append(v.entered, string(n.Kind())+":"+n.Name())
	return nil
}

func (v *testSpyVisitor) ExitNode(ctx *VisitContext, n Node) error {
	v.exited = append(v.exited, string(n.Kind())+":"+n.Name())
	return nil
}

func TestWalkerVisitsAllNodes(t *testing.T) {
	g := NewNodeGraph()
	root := NewRootNode("root")
	g.AddNode(root)
	v1 := NewVariableNode("v_1", "x")
	v2 := NewVariableNode("v_2", "y")
	fn := NewFunctionNode("fn_1", "doIt")
	g.AddNode(v1)
	g.AddNode(v2)
	g.AddNode(fn)

	g.SetChildren("root", []NodeID{"v_1", "v_2", "fn_1"})

	spy := newSpyVisitor()
	walker := NewWalker(g)
	walker.Register(spy)
	err := walker.Walk()
	if err != nil {
		t.Fatalf("walk error: %v", err)
	}

	expectedEntries := []string{
		"root:<file>",
		"variable:x",
		"variable:y",
		"function:doIt",
	}

	if len(spy.entered) != len(expectedEntries) {
		t.Fatalf("expected %d enters, got %d: %v", len(expectedEntries), len(spy.entered), spy.entered)
	}
	for i, e := range expectedEntries {
		if spy.entered[i] != e {
			t.Errorf("enter[%d]: expected %q, got %q", i, e, spy.entered[i])
		}
	}
}

func TestWalkerComponentWithChildren(t *testing.T) {
	g := NewNodeGraph()
	root := NewRootNode("root")
	comp := NewComponentNode("comp_1", "UserForm")
	hook := NewHookNode("hook_1", "useState")
	state := NewStateNode("state_1", "name")
	jsx := NewJSXNode("jsx_1", "div")

	g.AddNode(root)
	g.AddNode(comp)
	g.AddNode(hook)
	g.AddNode(state)
	g.AddNode(jsx)

	g.SetChildren("root", []NodeID{"comp_1"})
	g.SetChildren("comp_1", []NodeID{"hook_1", "state_1", "jsx_1"})

	spy := newSpyVisitor()
	walker := NewWalker(g)
	walker.Register(spy)
	err := walker.Walk()
	if err != nil {
		t.Fatalf("walk error: %v", err)
	}

	if len(spy.entered) < 3 {
		t.Fatalf("expected at least 3 entries, got %d: %v", len(spy.entered), spy.entered)
	}
	if spy.entered[1] != "component:UserForm" {
		t.Errorf("expected component:UserForm, got %q", spy.entered[1])
	}
	if spy.entered[2] != "hook:useState" {
		t.Errorf("expected hook:useState, got %q", spy.entered[2])
	}
}

type typedSpy struct {
	variables []string
	functions []string
	components []string
	hooks []string
	jsxs []string
	references []string
}

func (v *typedSpy) VisitVariable(ctx *VisitContext, n *VariableNode) error {
	v.variables = append(v.variables, n.Name())
	return nil
}
func (v *typedSpy) VisitFunction(ctx *VisitContext, n *FunctionNode) error {
	v.functions = append(v.functions, n.Name())
	return nil
}
func (v *typedSpy) VisitComponent(ctx *VisitContext, n *ComponentNode) error {
	v.components = append(v.components, n.Name())
	return nil
}
func (v *typedSpy) VisitHook(ctx *VisitContext, n *HookNode) error {
	v.hooks = append(v.hooks, n.Name())
	return nil
}
func (v *typedSpy) VisitJSX(ctx *VisitContext, n *JSXNode) error {
	v.jsxs = append(v.jsxs, n.ElementType)
	return nil
}
func (v *typedSpy) VisitReference(ctx *VisitContext, n *ReferenceNode) error {
	v.references = append(v.references, n.Name())
	return nil
}

func TestTypedVisitorDispatch(t *testing.T) {
	g := NewNodeGraph()
	root := NewRootNode("root")
	comp := NewComponentNode("c_1", "ProfileCard")
	hook := NewHookNode("h_1", "useQuery")
	jsx := NewJSXNode("j_1", "div")
	ref := NewReferenceNode("r_1", "data", "h_1", UsageRead)
	fn := NewFunctionNode("fn_1", "formatDate")

	g.AddNode(root)
	g.AddNode(comp)
	g.AddNode(hook)
	g.AddNode(jsx)
	g.AddNode(ref)
	g.AddNode(fn)
	g.SetChildren("root", []NodeID{"c_1", "fn_1"})
	g.SetChildren("c_1", []NodeID{"h_1", "j_1", "r_1"})

	spy := &typedSpy{}
	walker := NewWalker(g)
	walker.Register(spy)
	err := walker.Walk()
	if err != nil {
		t.Fatalf("walk error: %v", err)
	}

	if len(spy.components) != 1 || spy.components[0] != "ProfileCard" {
		t.Errorf("components: expected [ProfileCard], got %v", spy.components)
	}
	if len(spy.hooks) != 1 || spy.hooks[0] != "useQuery" {
		t.Errorf("hooks: expected [useQuery], got %v", spy.hooks)
	}
	if len(spy.functions) != 1 || spy.functions[0] != "formatDate" {
		t.Errorf("functions: expected [formatDate], got %v", spy.functions)
	}
}

func TestVisitContext(t *testing.T) {
	g := NewNodeGraph()
	root := NewRootNode("root")
	comp := NewComponentNode("c_1", "UserForm")
	hook := NewHookNode("h_1", "useState")

	g.AddNode(root)
	g.AddNode(comp)
	g.AddNode(hook)
	g.SetChildren("root", []NodeID{"c_1"})
	g.SetChildren("c_1", []NodeID{"h_1"})

	depthCheck := func() *typedSpy {
		return &typedSpy{}
	}

	spy := depthCheck
	walker := NewWalker(g)
	walker.Register(spy)

	_ = walker.Walk()

	_ = spy
}
