package node

type RootNode struct {
	NodeBase
	Imports    []NodeID
	Components []NodeID
	Hooks      []NodeID
	Exports    []NodeID
}

func NewRootNode(id NodeID) *RootNode {
	return &RootNode{NodeBase: NewNodeBase(id, KindRoot, "<file>")}
}

type VariableNode struct {
	NodeBase
	DeclaredType   string
	InferredKind   NodeKind
	Initializer    NodeID
	IsConst        bool
	IsDestructured bool
}

func NewVariableNode(id NodeID, name string) *VariableNode {
	return &VariableNode{NodeBase: NewNodeBase(id, KindVariable, name)}
}

type ArrayNode struct {
	NodeBase
	ElementType string
	Elements    []NodeID
}

func NewArrayNode(id NodeID, name string) *ArrayNode {
	return &ArrayNode{NodeBase: NewNodeBase(id, KindArray, name), Elements: []NodeID{}}
}

type ObjectNode struct {
	NodeBase
	Properties map[string]NodeID
}

func NewObjectNode(id NodeID, name string) *ObjectNode {
	return &ObjectNode{NodeBase: NewNodeBase(id, KindObject, name), Properties: map[string]NodeID{}}
}

type FunctionNode struct {
	NodeBase
	Params   []NodeID
	Returns  []NodeID
	Calls    []NodeID
	Captures []NodeID
	Body     []NodeID
	IsAsync  bool
	IsArrow  bool
	IsExport bool
}

func NewFunctionNode(id NodeID, name string) *FunctionNode {
	return &FunctionNode{
		NodeBase: NewNodeBase(id, KindFunction, name),
		Params:   []NodeID{},
		Returns:  []NodeID{},
		Calls:    []NodeID{},
		Captures: []NodeID{},
		Body:     []NodeID{},
	}
}

type ReferenceNode struct {
	NodeBase
	Target NodeID
	Usage  ReferenceUsage
}

func NewReferenceNode(id NodeID, name string, target NodeID, usage ReferenceUsage) *ReferenceNode {
	return &ReferenceNode{
		NodeBase: NewNodeBase(id, KindReference, name),
		Target:   target,
		Usage:    usage,
	}
}

type CallNode struct {
	NodeBase
	Target    NodeID
	Arguments []NodeID
	IsNew     bool
}

func NewCallNode(id NodeID, name string) *CallNode {
	return &CallNode{
		NodeBase:  NewNodeBase(id, KindCall, name),
		Arguments: []NodeID{},
	}
}

type ImportNode struct {
	NodeBase
	Source    string
	Default   string
	Named     []string
	IsType    bool
}

func NewImportNode(id NodeID, source string) *ImportNode {
	return &ImportNode{
		NodeBase: NewNodeBase(id, KindImport, source),
		Source:   source,
		Named:    []string{},
	}
}

type ExportNode struct {
	NodeBase
	LocalName  string
	ExportedAs string
	IsDefault  bool
	IsType     bool
}

func NewExportNode(id NodeID, name string) *ExportNode {
	return &ExportNode{
		NodeBase:   NewNodeBase(id, KindExport, name),
		IsDefault:  false,
	}
}

type ComponentNode struct {
	NodeBase
	Props        NodeID
	Hooks        []NodeID
	JSX          NodeID
	StateVars    []NodeID
	Effects      []NodeID
	Events       []NodeID
	IsMemoized   bool
	IsForwardRef bool
	IsExport     bool
	Lines        int
}

func NewComponentNode(id NodeID, name string) *ComponentNode {
	return &ComponentNode{
		NodeBase: NewNodeBase(id, KindComponent, name),
		Hooks:    []NodeID{},
		StateVars: []NodeID{},
		Effects:  []NodeID{},
		Events:   []NodeID{},
	}
}

type HookNode struct {
	NodeBase
	Args      []NodeID
	Deps      []NodeID
}

func NewHookNode(id NodeID, name string) *HookNode {
	return &HookNode{
		NodeBase: NewNodeBase(id, KindHook, name),
		Args:     []NodeID{},
		Deps:     []NodeID{},
	}
}

type StateNode struct {
	NodeBase
	Value  NodeID
	Setter NodeID
}

func NewStateNode(id NodeID, name string) *StateNode {
	return &StateNode{NodeBase: NewNodeBase(id, KindState, name)}
}

type EffectNode struct {
	NodeBase
	Deps       []NodeID
	IsAsync    bool
	HasCleanup bool
}

func NewEffectNode(id NodeID) *EffectNode {
	return &EffectNode{
		NodeBase: NewNodeBase(id, KindEffect, "useEffect"),
		Deps:     []NodeID{},
	}
}

type JSXNode struct {
	NodeBase
	ElementType string
	Props       map[string]NodeID
	ChildrenIds []NodeID
	IsFragment  bool
	IsComponent bool
	Depth       int
}

func NewJSXNode(id NodeID, element string) *JSXNode {
	return &JSXNode{
		NodeBase:    NewNodeBase(id, KindJSX, element),
		ElementType: element,
		Props:       map[string]NodeID{},
		ChildrenIds: []NodeID{},
	}
}

type ConditionalNode struct {
	NodeBase
	Condition  NodeID
	Consequent NodeID
	Alternate  NodeID
}

func NewConditionalNode(id NodeID) *ConditionalNode {
	return &ConditionalNode{NodeBase: NewNodeBase(id, KindConditional, "conditional")}
}

type LoopNode struct {
	NodeBase
	Iterable NodeID
	Body     []NodeID
}

func NewLoopNode(id NodeID, kind string) *LoopNode {
	return &LoopNode{
		NodeBase: NewNodeBase(id, KindLoop, kind),
		Body:     []NodeID{},
	}
}

type EventNode struct {
	NodeBase
	EventType string
	Handler   NodeID
}

func NewEventNode(id NodeID, name string) *EventNode {
	return &EventNode{
		NodeBase:  NewNodeBase(id, KindEvent, name),
	}
}

type PropNode struct {
	NodeBase
	PropType string
	Required bool
}

func NewPropNode(id NodeID, name string) *PropNode {
	return &PropNode{NodeBase: NewNodeBase(id, KindProp, name)}
}
