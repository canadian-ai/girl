package node

import "fmt"

type NodeID string

func NewNodeID(prefix string, id int) NodeID {
	return NodeID(fmt.Sprintf("%s_%d", prefix, id))
}

type NodeKind string

const (
	KindRoot        NodeKind = "root"
	KindVariable    NodeKind = "variable"
	KindArray       NodeKind = "array"
	KindObject      NodeKind = "object"
	KindFunction    NodeKind = "function"
	KindComponent   NodeKind = "component"
	KindHook        NodeKind = "hook"
	KindState       NodeKind = "state"
	KindEffect      NodeKind = "effect"
	KindReducer     NodeKind = "reducer"
	KindJSX         NodeKind = "jsx"
	KindReference   NodeKind = "reference"
	KindCall        NodeKind = "call"
	KindImport      NodeKind = "import"
	KindExport      NodeKind = "export"
	KindConditional NodeKind = "conditional"
	KindLoop        NodeKind = "loop"
	KindProp        NodeKind = "prop"
	KindEvent       NodeKind = "event"
	KindLitString   NodeKind = "lit_string"
	KindLitNumber   NodeKind = "lit_number"
	KindLitBool     NodeKind = "lit_bool"
	KindSpread      NodeKind = "spread"
	KindTemplate    NodeKind = "template"
	KindMember      NodeKind = "member_access"
)

type ReferenceUsage string

const (
	UsageRead    ReferenceUsage = "read"
	UsageWrite   ReferenceUsage = "write"
	UsageCall    ReferenceUsage = "call"
	UsageCapture ReferenceUsage = "capture"
	UsageDeclare ReferenceUsage = "declare"
	UsageProp    ReferenceUsage = "jsx-prop"
	UsageDep     ReferenceUsage = "dependency"
	UsageAssign  ReferenceUsage = "assign"
)

type Range struct {
	StartLine int `json:"startLine"`
	EndLine   int `json:"endLine"`
	StartCol  int `json:"startCol"`
	EndCol    int `json:"endCol"`
}

type NodeBase struct {
	id       NodeID
	kind     NodeKind
	name     string
	loc      Range
	file     string
	children []NodeID
	parent   NodeID
}

func NewNodeBase(id NodeID, kind NodeKind, name string) NodeBase {
	return NodeBase{
		id:       id,
		kind:     kind,
		name:     name,
		children: []NodeID{},
	}
}

func (b *NodeBase) ID() NodeID           { return b.id }
func (b *NodeBase) Kind() NodeKind       { return b.kind }
func (b *NodeBase) Name() string         { return b.name }
func (b *NodeBase) Range() Range         { return b.loc }
func (b *NodeBase) File() string         { return b.file }
func (b *NodeBase) Children() []NodeID   { return b.children }
func (b *NodeBase) Parent() NodeID       { return b.parent }
func (b *NodeBase) SetRange(r Range)     { b.loc = r }
func (b *NodeBase) SetFile(f string)     { b.file = f }
func (b *NodeBase) SetParent(p NodeID)   { b.parent = p }
func (b *NodeBase) AddChild(c NodeID)    { b.children = append(b.children, c) }

type Node interface {
	ID() NodeID
	Kind() NodeKind
	Name() string
	Range() Range
	File() string
	Children() []NodeID
	Parent() NodeID
}
