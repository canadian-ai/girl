package node

import (
	"sort"
	"strings"
)

type NodeGraph struct {
	nodes      map[NodeID]Node
	children   map[NodeID][]NodeID
	parent     map[NodeID]NodeID
	references map[NodeID][]NodeID
	symbols    map[string]NodeID
	nextID     int
	fileNodes  map[string]NodeID
}

func NewNodeGraph() *NodeGraph {
	return &NodeGraph{
		nodes:      make(map[NodeID]Node),
		children:   make(map[NodeID][]NodeID),
		parent:     make(map[NodeID]NodeID),
		references: make(map[NodeID][]NodeID),
		symbols:    make(map[string]NodeID),
		fileNodes:  make(map[string]NodeID),
	}
}

func (g *NodeGraph) NextID(prefix string) NodeID {
	g.nextID++
	return NewNodeID(prefix, g.nextID)
}

func (g *NodeGraph) FileNodeFor(path string) NodeID {
	return g.fileNodes[path]
}

func (g *NodeGraph) AddNode(n Node) {
	g.nodes[n.ID()] = n
	base := g.getBase(n)
	if base != nil {
		for _, c := range base.children {
			g.addChild(n.ID(), c)
		}
	}
}

func (g *NodeGraph) addChild(parent, child NodeID) {
	g.children[parent] = append(g.children[parent], child)
	g.parent[child] = parent
}

func (g *NodeGraph) SetChildren(parent NodeID, children []NodeID) {
	for _, c := range children {
		g.addChild(parent, c)
	}
}

func (g *NodeGraph) Node(id NodeID) Node {
	return g.nodes[id]
}

func (g *NodeGraph) ChildrenOf(id NodeID) []Node {
	var result []Node
	for _, cid := range g.children[id] {
		if n := g.nodes[cid]; n != nil {
			result = append(result, n)
		}
	}
	return result
}

func (g *NodeGraph) ParentOf(id NodeID) Node {
	if pid, ok := g.parent[id]; ok {
		return g.nodes[pid]
	}
	return nil
}

func (g *NodeGraph) AddSymbol(name string, id NodeID) {
	if existing, ok := g.symbols[name]; ok {
		if g.nodes[existing] != nil {
			return
		}
	}
	g.symbols[name] = id
}

func (g *NodeGraph) LookupSymbol(name string) NodeID {
	if id, ok := g.symbols[name]; ok {
		return id
	}
	return ""
}

func (g *NodeGraph) AddReference(from, to NodeID) {
	g.references[to] = append(g.references[to], from)
}

func (g *NodeGraph) ReferencesTo(id NodeID) []Node {
	var result []Node
	for _, rid := range g.references[id] {
		if n := g.nodes[rid]; n != nil {
			result = append(result, n)
		}
	}
	return result
}

func (g *NodeGraph) AllSymbols() map[string]NodeID {
	return g.symbols
}

func (g *NodeGraph) AllNodes() []Node {
	var result []Node
	for _, n := range g.nodes {
		result = append(result, n)
	}
	return result
}

func (g *NodeGraph) AllNodesOfKind(kind NodeKind) []Node {
	var result []Node
	for _, n := range g.nodes {
		if n.Kind() == kind {
			result = append(result, n)
		}
	}
	return result
}

func (g *NodeGraph) NodesByFile(file string) []Node {
	fileID := g.fileNodes[file]
	if fileID == "" {
		return nil
	}
	return g.ChildrenOf(fileID)
}

func (g *NodeGraph) SetFileNode(file string, id NodeID) {
	g.fileNodes[file] = id
}

func (g *NodeGraph) AllFiles() []string {
	var files []string
	for f := range g.fileNodes {
		files = append(files, f)
	}
	sort.Strings(files)
	return files
}

func (g *NodeGraph) FindByName(name string) []Node {
	var result []Node
	name = strings.TrimPrefix(name, "default_export_")
	for _, n := range g.nodes {
		if n.Name() == name || strings.HasSuffix(n.Name(), "."+name) {
			result = append(result, n)
		}
	}
	return result
}

func (g *NodeGraph) getBase(n Node) *NodeBase {
	switch v := n.(type) {
	case *RootNode:
		return &v.NodeBase
	case *VariableNode:
		return &v.NodeBase
	case *ArrayNode:
		return &v.NodeBase
	case *ObjectNode:
		return &v.NodeBase
	case *FunctionNode:
		return &v.NodeBase
	case *ComponentNode:
		return &v.NodeBase
	case *HookNode:
		return &v.NodeBase
	case *StateNode:
		return &v.NodeBase
	case *EffectNode:
		return &v.NodeBase
	case *JSXNode:
		return &v.NodeBase
	case *ReferenceNode:
		return &v.NodeBase
	case *CallNode:
		return &v.NodeBase
	case *ImportNode:
		return &v.NodeBase
	case *ExportNode:
		return &v.NodeBase
	case *ConditionalNode:
		return &v.NodeBase
	case *LoopNode:
		return &v.NodeBase
	case *EventNode:
		return &v.NodeBase
	case *PropNode:
		return &v.NodeBase
	default:
		return nil
	}
}

func (g *NodeGraph) Merge(other *NodeGraph) {
	for id, n := range other.nodes {
		g.nodes[id] = n
	}
	for p, kids := range other.children {
		g.children[p] = append(g.children[p], kids...)
	}
	for c, p := range other.parent {
		g.parent[c] = p
	}
	for sym, id := range other.symbols {
		if _, ok := g.symbols[sym]; !ok {
			g.symbols[sym] = id
		}
	}
	for f, id := range other.fileNodes {
		g.fileNodes[f] = id
	}
	if other.nextID > g.nextID {
		g.nextID = other.nextID
	}
}
