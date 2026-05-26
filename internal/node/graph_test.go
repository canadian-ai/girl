package node

import (
	"testing"
)

func TestNodeGraphBasic(t *testing.T) {
	g := NewNodeGraph()
	if g == nil {
		t.Fatal("expected non-nil graph")
	}
	n := NewVariableNode("v_1", "testVar")
	g.AddNode(n)
	if got := g.Node("v_1"); got == nil {
		t.Fatal("expected node v_1")
	}
}

func TestNodeGraphChildren(t *testing.T) {
	g := NewNodeGraph()
	root := NewRootNode("root_1")
	child := NewVariableNode("v_1", "childVar")
	g.AddNode(root)
	g.AddNode(child)
	g.SetChildren("root_1", []NodeID{"v_1"})
	kids := g.ChildrenOf("root_1")
	if len(kids) != 1 || kids[0].ID() != "v_1" {
		t.Fatalf("expected 1 child v_1, got %v", ids(kids))
	}
	parent := g.ParentOf("v_1")
	if parent == nil || parent.ID() != "root_1" {
		t.Fatalf("expected parent root_1, got %v", parentID(parent))
	}
}

func TestNodeGraphSymbols(t *testing.T) {
	g := NewNodeGraph()
	comp := NewComponentNode("comp_1", "UserForm")
	g.AddNode(comp)
	g.AddSymbol("UserForm", "comp_1")
	if id := g.LookupSymbol("UserForm"); id != "comp_1" {
		t.Errorf("expected comp_1, got %s", id)
	}
	all := g.AllSymbols()
	if len(all) != 1 {
		t.Errorf("expected 1 symbol, got %d", len(all))
	}
}

func TestNodeGraphReferences(t *testing.T) {
	g := NewNodeGraph()
	g.AddNode(NewVariableNode("v_1", "name"))
	g.AddNode(NewReferenceNode("r_1", "name", "v_1", UsageRead))
	g.AddReference("r_1", "v_1")
	refs := g.ReferencesTo("v_1")
	if len(refs) != 1 || refs[0].ID() != "r_1" {
		t.Fatalf("expected 1 reference r_1, got %v", ids(refs))
	}
}

func TestNodeGraphFindByName(t *testing.T) {
	g := NewNodeGraph()
	g.AddNode(NewComponentNode("c_1", "UserForm"))
	g.AddNode(NewComponentNode("c_2", "ProjectList"))
	nodes := g.FindByName("UserForm")
	if len(nodes) != 1 || nodes[0].ID() != "c_1" {
		t.Fatalf("expected 1 UserForm match, got %v", ids(nodes))
	}
}

func TestNodeGraphFileNodes(t *testing.T) {
	g := NewNodeGraph()
	root := NewRootNode("root_f1")
	g.AddNode(root)
	g.SetFileNode("src/App.tsx", "root_f1")
	if g.FileNodeFor("src/App.tsx") != "root_f1" {
		t.Fatal("wrong file node")
	}
	files := g.AllFiles()
	if len(files) != 1 || files[0] != "src/App.tsx" {
		t.Fatalf("expected [src/App.tsx], got %v", files)
	}
}

func TestNodeGraphAllNodesOfKind(t *testing.T) {
	g := NewNodeGraph()
	g.AddNode(NewComponentNode("c_1", "A"))
	g.AddNode(NewComponentNode("c_2", "B"))
	g.AddNode(NewVariableNode("v_1", "x"))
	comps := g.AllNodesOfKind(KindComponent)
	if len(comps) != 2 {
		t.Fatalf("expected 2 components, got %d", len(comps))
	}
}

func TestNodeGraphMerge(t *testing.T) {
	g1 := NewNodeGraph()
	g1.AddNode(NewComponentNode("c_1", "A"))
	g1.AddSymbol("A", "c_1")

	g2 := NewNodeGraph()
	g2.AddNode(NewComponentNode("c_2", "B"))
	g2.AddSymbol("B", "c_2")

	g1.Merge(g2)

	if len(g1.AllNodes()) != 2 {
		t.Fatalf("expected 2 nodes after merge, got %d", len(g1.AllNodes()))
	}
	if g1.LookupSymbol("A") != "c_1" || g1.LookupSymbol("B") != "c_2" {
		t.Fatal("symbols not merged correctly")
	}
}

func TestNodeGraphNextID(t *testing.T) {
	g := NewNodeGraph()
	id1 := g.NextID("var")
	id2 := g.NextID("var")
	if id1 == id2 {
		t.Fatal("expected different IDs")
	}
	if id1 != "var_1" {
		t.Fatalf("expected var_1, got %s", id1)
	}
	if id2 != "var_2" {
		t.Fatalf("expected var_2, got %s", id2)
	}
}

func TestNodeGraphAllNodes(t *testing.T) {
	g := NewNodeGraph()
	g.AddNode(NewVariableNode("v_1", "x"))
	g.AddNode(NewVariableNode("v_2", "y"))
	all := g.AllNodes()
	if len(all) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(all))
	}
}

func parentID(n Node) string {
	if n == nil {
		return "<nil>"
	}
	return string(n.ID())
}

func ids(nodes []Node) []string {
	var s []string
	for _, n := range nodes {
		if n != nil {
			s = append(s, string(n.ID()))
		}
	}
	return s
}
