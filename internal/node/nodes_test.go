package node

import (
	"testing"
)

func TestNodeBase(t *testing.T) {
	n := NewNodeBase("test_1", KindVariable, "myVar")
	if n.ID() != "test_1" {
		t.Errorf("expected id test_1, got %s", n.ID())
	}
	if n.Kind() != KindVariable {
		t.Errorf("expected kind variable, got %s", n.Kind())
	}
	if n.Name() != "myVar" {
		t.Errorf("expected name myVar, got %s", n.Name())
	}
	r := Range{StartLine: 1, EndLine: 5}
	n.SetRange(r)
	if n.Range() != r {
		t.Errorf("expected range %+v, got %+v", r, n.Range())
	}
	n.SetFile("test.tsx")
	if n.File() != "test.tsx" {
		t.Errorf("expected file test.tsx, got %s", n.File())
	}
	if len(n.Children()) != 0 {
		t.Errorf("expected 0 children initially")
	}
	n.AddChild("child_1")
	if len(n.Children()) != 1 || n.Children()[0] != "child_1" {
		t.Errorf("expected 1 child child_1, got %v", n.Children())
	}
	if n.Parent() != "" {
		t.Errorf("expected empty parent")
	}
	n.SetParent("parent_1")
	if n.Parent() != "parent_1" {
		t.Errorf("expected parent parent_1")
	}
}

func TestNodeTypes(t *testing.T) {
	t.Run("VariableNode", func(t *testing.T) {
		n := NewVariableNode("var_1", "userName")
		n.DeclaredType = "string"
		n.IsConst = true
		if n.Kind() != KindVariable {
			t.Fatal("wrong kind")
		}
		if n.Name() != "userName" {
			t.Fatal("wrong name")
		}
		if n.DeclaredType != "string" {
			t.Fatal("wrong type")
		}
		if !n.IsConst {
			t.Fatal("expected const")
		}
	})

	t.Run("ArrayNode", func(t *testing.T) {
		n := NewArrayNode("arr_1", "users")
		n.ElementType = "User[]"
		n.Elements = []NodeID{"elem_1", "elem_2"}
		if n.Kind() != KindArray {
			t.Fatal("wrong kind")
		}
		if len(n.Elements) != 2 {
			t.Fatal("wrong element count")
		}
	})

	t.Run("ObjectNode", func(t *testing.T) {
		n := NewObjectNode("obj_1", "config")
		n.Properties["key"] = "val_1"
		if len(n.Properties) != 1 {
			t.Fatal("wrong prop count")
		}
		if n.Properties["key"] != "val_1" {
			t.Fatal("wrong prop")
		}
	})

	t.Run("FunctionNode", func(t *testing.T) {
		n := NewFunctionNode("fn_1", "handleSubmit")
		n.IsAsync = true
		n.IsArrow = false
		n.Params = []NodeID{"p_1"}
		if n.Kind() != KindFunction {
			t.Fatal("wrong kind")
		}
		if !n.IsAsync {
			t.Fatal("expected async")
		}
	})

	t.Run("ComponentNode", func(t *testing.T) {
		n := NewComponentNode("comp_1", "UserForm")
		n.Lines = 120
		n.IsExport = true
		n.Hooks = []NodeID{"h_1", "h_2"}
		if n.Kind() != KindComponent {
			t.Fatal("wrong kind")
		}
		if n.Lines != 120 {
			t.Fatal("wrong lines")
		}
		if !n.IsExport {
			t.Fatal("expected export")
		}
		if len(n.Hooks) != 2 {
			t.Fatal("wrong hook count")
		}
	})

	t.Run("HookNode", func(t *testing.T) {
		n := NewHookNode("hook_1", "useState")
		n.Deps = []NodeID{"d_1"}
		if n.Kind() != KindHook {
			t.Fatal("wrong kind")
		}
		if n.Name() != "useState" {
			t.Fatal("wrong name")
		}
	})

	t.Run("StateNode", func(t *testing.T) {
		n := NewStateNode("state_1", "selectedId")
		n.Value = "val_1"
		n.Setter = "set_1"
		if n.Kind() != KindState {
			t.Fatal("wrong kind")
		}
	})

	t.Run("JSXNode", func(t *testing.T) {
		n := NewJSXNode("jsx_1", "div")
		n.IsComponent = false
		n.Depth = 3
		n.Props["className"] = "expr_1"
		if n.Kind() != KindJSX {
			t.Fatal("wrong kind")
		}
		if n.ElementType != "div" {
			t.Fatal("wrong element")
		}
		if n.Depth != 3 {
			t.Fatal("wrong depth")
		}
	})

	t.Run("ReferenceNode", func(t *testing.T) {
		n := NewReferenceNode("ref_1", "userName", "var_1", UsageRead)
		if n.Kind() != KindReference {
			t.Fatal("wrong kind")
		}
		if n.Target != "var_1" {
			t.Fatal("wrong target")
		}
		if n.Usage != UsageRead {
			t.Fatal("wrong usage")
		}
	})

	t.Run("CallNode", func(t *testing.T) {
		n := NewCallNode("call_1", "useQuery")
		n.Arguments = []NodeID{"arg_1"}
		if n.Kind() != KindCall {
			t.Fatal("wrong kind")
		}
		if len(n.Arguments) != 1 {
			t.Fatal("wrong args")
		}
	})

	t.Run("ImportNode", func(t *testing.T) {
		n := NewImportNode("imp_1", "react")
		n.Default = "React"
		n.Named = []string{"useState", "useEffect"}
		if n.Kind() != KindImport {
			t.Fatal("wrong kind")
		}
		if n.Source != "react" {
			t.Fatal("wrong source")
		}
		if len(n.Named) != 2 {
			t.Fatal("wrong named count")
		}
	})

	t.Run("ExportNode", func(t *testing.T) {
		n := NewExportNode("exp_1", "UserForm")
		n.IsDefault = true
		if n.Kind() != KindExport {
			t.Fatal("wrong kind")
		}
		if !n.IsDefault {
			t.Fatal("expected default")
		}
	})

	t.Run("ConditionalNode", func(t *testing.T) {
		n := NewConditionalNode("cond_1")
		n.Condition = "cond_expr"
		n.Consequent = "then_block"
		if n.Kind() != KindConditional {
			t.Fatal("wrong kind")
		}
	})

	t.Run("EventNode", func(t *testing.T) {
		n := NewEventNode("evt_1", "handleClick")
		n.EventType = "onClick"
		if n.Kind() != KindEvent {
			t.Fatal("wrong kind")
		}
	})
}

func TestNewNodeID(t *testing.T) {
	id := NewNodeID("var", 42)
	if id != "var_42" {
		t.Errorf("expected var_42, got %s", id)
	}
}
