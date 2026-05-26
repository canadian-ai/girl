package node

import "reflect"

type VisitContext struct {
	Graph     *NodeGraph
	Current   Node
	Parent    Node
	Depth     int
	Path      []NodeID
}

type EnterExitVisitor interface {
	EnterNode(ctx *VisitContext, n Node) error
	ExitNode(ctx *VisitContext, n Node) error
}

type TypedVisitor interface {
	VisitVariable(ctx *VisitContext, n *VariableNode) error
	VisitArray(ctx *VisitContext, n *ArrayNode) error
	VisitObject(ctx *VisitContext, n *ObjectNode) error
	VisitFunction(ctx *VisitContext, n *FunctionNode) error
	VisitComponent(ctx *VisitContext, n *ComponentNode) error
	VisitHook(ctx *VisitContext, n *HookNode) error
	VisitState(ctx *VisitContext, n *StateNode) error
	VisitEffect(ctx *VisitContext, n *EffectNode) error
	VisitJSX(ctx *VisitContext, n *JSXNode) error
	VisitReference(ctx *VisitContext, n *ReferenceNode) error
	VisitCall(ctx *VisitContext, n *CallNode) error
	VisitImport(ctx *VisitContext, n *ImportNode) error
	VisitExport(ctx *VisitContext, n *ExportNode) error
	VisitConditional(ctx *VisitContext, n *ConditionalNode) error
	VisitLoop(ctx *VisitContext, n *LoopNode) error
	VisitEvent(ctx *VisitContext, n *EventNode) error
	VisitProp(ctx *VisitContext, n *PropNode) error
}

type BaseTypedVisitor struct{}

func (b *BaseTypedVisitor) VisitVariable(*VisitContext, *VariableNode) error { return nil }
func (b *BaseTypedVisitor) VisitArray(*VisitContext, *ArrayNode) error { return nil }
func (b *BaseTypedVisitor) VisitObject(*VisitContext, *ObjectNode) error { return nil }
func (b *BaseTypedVisitor) VisitFunction(*VisitContext, *FunctionNode) error { return nil }
func (b *BaseTypedVisitor) VisitComponent(*VisitContext, *ComponentNode) error { return nil }
func (b *BaseTypedVisitor) VisitHook(*VisitContext, *HookNode) error { return nil }
func (b *BaseTypedVisitor) VisitState(*VisitContext, *StateNode) error { return nil }
func (b *BaseTypedVisitor) VisitEffect(*VisitContext, *EffectNode) error { return nil }
func (b *BaseTypedVisitor) VisitJSX(*VisitContext, *JSXNode) error { return nil }
func (b *BaseTypedVisitor) VisitReference(*VisitContext, *ReferenceNode) error { return nil }
func (b *BaseTypedVisitor) VisitCall(*VisitContext, *CallNode) error { return nil }
func (b *BaseTypedVisitor) VisitImport(*VisitContext, *ImportNode) error { return nil }
func (b *BaseTypedVisitor) VisitExport(*VisitContext, *ExportNode) error { return nil }
func (b *BaseTypedVisitor) VisitConditional(*VisitContext, *ConditionalNode) error { return nil }
func (b *BaseTypedVisitor) VisitLoop(*VisitContext, *LoopNode) error { return nil }
func (b *BaseTypedVisitor) VisitEvent(*VisitContext, *EventNode) error { return nil }
func (b *BaseTypedVisitor) VisitProp(*VisitContext, *PropNode) error { return nil }

type visitorEntry struct {
	enterExit EnterExitVisitor
	typed     *TypedVisitorReflect
}

type TypedVisitorReflect struct {
	visitor interface{}
}

func newTypedReflect(v interface{}) *TypedVisitorReflect {
	return &TypedVisitorReflect{visitor: v}
}

type Walker struct {
	graph    *NodeGraph
	visitors []visitorEntry
}

func NewWalker(g *NodeGraph) *Walker {
	return &Walker{
		graph:    g,
		visitors: []visitorEntry{},
	}
}

func (w *Walker) Register(v interface{}) {
	entry := visitorEntry{}
	if ee, ok := v.(EnterExitVisitor); ok {
		entry.enterExit = ee
	}
	entry.typed = newTypedReflect(v)
	w.visitors = append(w.visitors, entry)
}

func (w *Walker) Walk() error {
	for _, fileID := range w.graph.AllFiles() {
		root := w.graph.Node(w.graph.FileNodeFor(fileID))
		if root != nil {
			if err := w.walkNode(root, nil, 0); err != nil {
				return err
			}
		}
	}
	for _, id := range w.graph.AllFiles() {
		rootID := w.graph.FileNodeFor(id)
		if rootID == "" {
			root := w.graph.Node(NodeID(id))
			if root != nil {
				if err := w.walkNode(root, nil, 0); err != nil {
					return err
				}
			}
		}
	}
	orphans := w.walkOrphans()
	for _, n := range orphans {
		if err := w.walkNode(n, nil, 0); err != nil {
			return err
		}
	}
	return nil
}

func (w *Walker) walkOrphans() []Node {
	visited := map[NodeID]bool{}
	for _, fileID := range w.graph.AllFiles() {
		root := w.graph.Node(w.graph.FileNodeFor(fileID))
		if root != nil {
			collectIDs(root, visited)
		}
	}
	var orphans []Node
	for _, n := range w.graph.AllNodes() {
		if !visited[n.ID()] {
			orphans = append(orphans, n)
		}
	}
	return orphans
}

func collectIDs(n Node, visited map[NodeID]bool) {
	if visited[n.ID()] {
		return
	}
	visited[n.ID()] = true
	for _, c := range n.Children() {
		if child := n; child != nil {
			_ = c
		}
	}
}

func (w *Walker) walkNode(n Node, parent Node, depth int) error {
	if n == nil {
		return nil
	}
	ctx := &VisitContext{
		Graph:   w.graph,
		Current: n,
		Parent:  parent,
		Depth:   depth,
	}

	for _, v := range w.visitors {
		if v.enterExit != nil {
			if err := v.enterExit.EnterNode(ctx, n); err != nil {
				return err
			}
		}
		if err := dispatchTyped(v.typed.visitor, ctx, n); err != nil {
			return err
		}
	}

	for _, cid := range n.Children() {
		child := w.graph.Node(cid)
		if child != nil {
			if err := w.walkNode(child, n, depth+1); err != nil {
				return err
			}
		}
	}

	for _, v := range w.visitors {
		if v.enterExit != nil {
			if err := v.enterExit.ExitNode(ctx, n); err != nil {
				return err
			}
		}
	}

	return nil
}

func dispatchTyped(v interface{}, ctx *VisitContext, n Node) error {
	if v == nil {
		return nil
	}
	val := reflect.ValueOf(v)
	if val.Kind() != reflect.Ptr || val.IsNil() {
		return nil
	}
	methodName := visitMethodName(n.Kind())
	method := val.MethodByName(methodName)
	if !method.IsValid() {
		return nil
	}
	nodeVal := reflect.ValueOf(n)
	if !nodeVal.IsValid() || nodeVal.IsNil() {
		return nil
	}
	ctxVal := reflect.ValueOf(ctx)
	method.Call([]reflect.Value{ctxVal, nodeVal})
	return nil
}

func visitMethodName(kind NodeKind) string {
	switch kind {
	case KindVariable:
		return "VisitVariable"
	case KindArray:
		return "VisitArray"
	case KindObject:
		return "VisitObject"
	case KindFunction:
		return "VisitFunction"
	case KindComponent:
		return "VisitComponent"
	case KindHook:
		return "VisitHook"
	case KindState:
		return "VisitState"
	case KindEffect:
		return "VisitEffect"
	case KindJSX:
		return "VisitJSX"
	case KindReference:
		return "VisitReference"
	case KindCall:
		return "VisitCall"
	case KindImport:
		return "VisitImport"
	case KindExport:
		return "VisitExport"
	case KindConditional:
		return "VisitConditional"
	case KindLoop:
		return "VisitLoop"
	case KindEvent:
		return "VisitEvent"
	case KindProp:
		return "VisitProp"
	default:
		return ""
	}
}
