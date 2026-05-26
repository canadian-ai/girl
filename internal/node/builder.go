package node

import (
	"github.com/canadian-ai/girl/internal/ir"
)

func BuildFromIR(files []*ir.FileIR) *NodeGraph {
	g := NewNodeGraph()
	for _, f := range files {
		buildFile(g, f)
	}
	return g
}

func buildFile(g *NodeGraph, f *ir.FileIR) {
	root := NewRootNode(g.NextID("root"))
	root.SetFile(f.Path)
	g.AddNode(root)
	g.SetFileNode(f.Path, root.ID())

	var rootChildren []NodeID

	for _, imp := range f.Imports {
		importNode := NewImportNode(g.NextID("import"), imp.Source)
		importNode.Default = imp.Default
		importNode.Named = imp.Names
		importNode.SetFile(f.Path)
		g.AddNode(importNode)
		g.AddSymbol(imp.Source, importNode.ID())
		rootChildren = append(rootChildren, importNode.ID())
	}

	for _, comp := range f.Components {
		compID := buildComponent(g, f.Path, comp)
		rootChildren = append(rootChildren, compID)
	}

	for _, hook := range f.Hooks {
		hookNode := NewHookNode(g.NextID("hook"), hook.Name)
		hookNode.SetFile(f.Path)
		g.AddNode(hookNode)
		rootChildren = append(rootChildren, hookNode.ID())
	}

	g.SetChildren(root.ID(), rootChildren)
}

func buildComponent(g *NodeGraph, filePath string, comp ir.ComponentIR) NodeID {
	compNode := NewComponentNode(g.NextID("comp"), comp.Name)
	compNode.Lines = comp.Lines
	compNode.SetFile(filePath)
	g.AddNode(compNode)
	g.AddSymbol(comp.Name, compNode.ID())

	var children []NodeID

	for _, h := range comp.Hooks {
		hookNode := NewHookNode(g.NextID("hook"), h.Name)
		hookNode.SetFile(filePath)
		g.AddNode(hookNode)
		compNode.Hooks = append(compNode.Hooks, hookNode.ID())
		children = append(children, hookNode.ID())

		if h.Name == "useState" || h.Name == "useReducer" {
			stateName := "state:" + h.Name
			stateNode := NewStateNode(g.NextID("state"), stateName)
			stateNode.SetFile(filePath)
			g.AddNode(stateNode)
			compNode.StateVars = append(compNode.StateVars, stateNode.ID())
			children = append(children, stateNode.ID())
		}
	}

	for range comp.Effects {
		effectNode := NewEffectNode(g.NextID("effect"))
		effectNode.SetFile(filePath)
		effectNode.Deps = []NodeID{}
		g.AddNode(effectNode)
		compNode.Effects = append(compNode.Effects, effectNode.ID())
		children = append(children, effectNode.ID())
	}

	for _, jsx := range comp.JSXBlocks {
		jsxNode := NewJSXNode(g.NextID("jsx"), jsx.Element)
		jsxNode.SetFile(filePath)
		jsxNode.Depth = 1
		g.AddNode(jsxNode)
		children = append(children, jsxNode.ID())
	}

	for _, eh := range comp.EventHandlers {
		eventNode := NewEventNode(g.NextID("event"), eh.Name)
		eventNode.SetFile(filePath)
		g.AddNode(eventNode)
		compNode.Events = append(compNode.Events, eventNode.ID())
		children = append(children, eventNode.ID())
	}

	for _, p := range comp.Props {
		propNode := NewPropNode(g.NextID("prop"), p.Name)
		propNode.PropType = p.Type
		propNode.Required = p.Required
		propNode.SetFile(filePath)
		g.AddNode(propNode)
		children = append(children, propNode.ID())
	}

	compNode.IsExport = len(comp.Exports) > 0

	for _, export := range comp.Exports {
		exportNode := NewExportNode(g.NextID("export"), export.Name)
		exportNode.IsDefault = export.Default
		exportNode.SetFile(filePath)
		g.AddNode(exportNode)
		children = append(children, exportNode.ID())
	}

	g.SetChildren(compNode.ID(), children)
	return compNode.ID()
}

func BuildFromIRAndResolve(files []*ir.FileIR) *NodeGraph {
	g := BuildFromIR(files)
	resolveReferences(g)
	return g
}

func resolveReferences(g *NodeGraph) {
	for _, n := range g.AllNodes() {
		if ref, ok := n.(*ReferenceNode); ok {
			target := g.LookupSymbol(ref.Name())
			if target != "" {
				ref.Target = target
				g.AddReference(ref.ID(), target)
			}
		}
	}
}
