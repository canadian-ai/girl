package parsertsx

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/canadian-ai/girl/internal/node"
)

func parseStr(t *testing.T, content string) *node.NodeGraph {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.tsx")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	p := New()
	g, err := p.ParseFile(path)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	return g
}

func TestParseBasicComponent(t *testing.T) {
	src := `import { useState } from "react";

export default function MyComponent() {
  const [count, setCount] = useState(0);
  return <div>{count}</div>;
}
`
	g := parseStr(t, src)
	if g == nil {
		t.Fatal("expected graph")
	}
	comps := g.AllNodesOfKind(node.KindComponent)
	if len(comps) == 0 {
		t.Fatal("expected at least one component")
	}
	if comps[0].Name() != "MyComponent" {
		t.Errorf("expected MyComponent, got %s", comps[0].Name())
	}
}

func TestParseComponentWithHooks(t *testing.T) {
	src := `import { useState, useEffect } from "react";

export default function DataFetcher() {
  const [data, setData] = useState(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetch("/api/data").then(setData).finally(() => setLoading(false));
  }, []);

  return <div>{loading ? "Loading..." : data}</div>;
}
`
	g := parseStr(t, src)
	comps := g.AllNodesOfKind(node.KindComponent)
	if len(comps) != 1 {
		t.Fatalf("expected 1 component, got %d", len(comps))
	}
	comp := comps[0].(*node.ComponentNode)
	if len(comp.Hooks) < 3 {
		t.Errorf("expected at least 3 hooks (2 useState + 1 useEffect), got %d", len(comp.Hooks))
	}
	if len(comp.StateVars) < 2 {
		t.Errorf("expected at least 2 state vars, got %d", len(comp.StateVars))
	}
	if len(comp.Effects) < 1 {
		t.Errorf("expected at least 1 effect, got %d", len(comp.Effects))
	}
}

func TestParseArrowComponent(t *testing.T) {
	src := `interface Props { name: string; }
const Greeting: React.FC<Props> = ({ name }) => {
  return <h1>Hello, {name}!</h1>;
};
`
	g := parseStr(t, src)
	comps := g.AllNodesOfKind(node.KindComponent)
	if len(comps) == 0 {
		t.Fatal("expected at least one component")
	}
	comp := comps[0].(*node.ComponentNode)
	if comp.Name() != "Greeting" {
		t.Errorf("expected Greeting, got %s", comp.Name())
	}
}

func TestParseComponentWithConvexHooks(t *testing.T) {
	src := `import { useQuery, useMutation } from "convex/react";
import { api } from "../convex/_generated/api";

export default function Dashboard() {
  const data = useQuery(api.dashboard.get);
  const update = useMutation(api.dashboard.update);

  return (
    <div>
      {data?.map(item => <div key={item.id}>{item.name}</div>)}
    </div>
  );
}
`
	g := parseStr(t, src)
	comps := g.AllNodesOfKind(node.KindComponent)
	if len(comps) != 1 {
		t.Fatalf("expected 1 component, got %d", len(comps))
	}
	comp := comps[0].(*node.ComponentNode)
	if len(comp.Hooks) < 2 {
		t.Errorf("expected at least 2 hooks (useQuery + useMutation), got %d", len(comp.Hooks))
	}
}

func TestParseExportConstComponent(t *testing.T) {
	src := `export const UserProfile = () => {
  return <div className="profile">User</div>;
};
`
	g := parseStr(t, src)
	comps := g.AllNodesOfKind(node.KindComponent)
	if len(comps) == 0 {
		t.Fatal("expected a component")
	}
	if comps[0].Name() != "UserProfile" {
		t.Errorf("expected UserProfile, got %s", comps[0].Name())
	}
}

func TestParseMultipleComponents(t *testing.T) {
	src := `function Header() { return <header>Head</header>; }
function Footer() { return <footer>Foot</footer>; }
function Layout() {
  return (
    <div>
      <Header />
      <Footer />
    </div>
  );
}
`
	g := parseStr(t, src)
	comps := g.AllNodesOfKind(node.KindComponent)
	if len(comps) != 3 {
		t.Fatalf("expected 3 components, got %d", len(comps))
	}
}

func TestParseReferenceTracking(t *testing.T) {
	src := `const helper = (x: number) => x * 2;

export function Calculator({ value }: { value: number }) {
  const result = helper(value);
  return <div>{result}</div>;
}
`
	g := parseStr(t, src)
	comps := g.AllNodesOfKind(node.KindComponent)
	if len(comps) == 0 {
		t.Fatal("expected component")
	}
	comp := comps[0].(*node.ComponentNode)
	kids := g.ChildrenOf(comp.ID())
	if len(kids) == 0 {
		t.Error("expected children")
	}
}

func TestParseRealFixture(t *testing.T) {
	path := "../../testdata/real/insurance-review-panel.tsx"
	data, err := os.ReadFile(path)
	if err != nil {
		t.Skip("fixture not found:", err)
	}
	dir := t.TempDir()
	fixturePath := filepath.Join(dir, "test.tsx")
	os.WriteFile(fixturePath, data, 0644)

	p := New()
	g, err := p.ParseFile(fixturePath)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	comps := g.AllNodesOfKind(node.KindComponent)
	if len(comps) == 0 {
		t.Fatal("expected at least 1 component in insurance review fixture")
	}
	comp := comps[0].(*node.ComponentNode)
	t.Logf("Component: %s, Hooks: %d, States: %d, Effects: %d, Events: %d, JSX: %d, Lines: %d",
		comp.Name(), len(comp.Hooks), len(comp.StateVars), len(comp.Effects),
		len(comp.Events),
		len(g.AllNodesOfKind(node.KindJSX)),
		comp.Lines)
}

func TestParseBrokerWorkspaceFixture(t *testing.T) {
	path := "../../testdata/real/broker-workspace.tsx"
	data, err := os.ReadFile(path)
	if err != nil {
		t.Skip("fixture not found:", err)
	}
	dir := t.TempDir()
	fixturePath := filepath.Join(dir, "test.tsx")
	os.WriteFile(fixturePath, data, 0644)

	p := New()
	g, err := p.ParseFile(fixturePath)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	comps := g.AllNodesOfKind(node.KindComponent)
	if len(comps) == 0 {
		t.Fatal("expected at least 1 component")
	}
}

func TestParseGraphVisitors(t *testing.T) {
	src := `import { useState } from "react";

function Counter() {
  const [count, setCount] = useState(0);
  return <button onClick={() => setCount(count + 1)}>{count}</button>;
}
`
	g := parseStr(t, src)

	type visitorResult struct {
		components int
		hooks      int
		states     int
		jsxs       int
	}

	result := &visitCounts{}
	visitor := &sampleVisitor{result: result}
	walker := node.NewWalker(g)
	walker.Register(visitor)

	if err := walker.Walk(); err != nil {
		t.Fatalf("walk: %v", err)
	}
	if result.components == 0 {
		t.Error("expected at least 1 component visit")
	}
	if result.hooks == 0 {
		t.Error("expected at least 1 hook visit")
	}
	t.Logf("Visited: components=%d hooks=%d states=%d jsx=%d",
		result.components, result.hooks, result.states, result.jsxs)
}

type visitCounts struct {
	components int
	hooks      int
	states     int
	jsxs       int
}

type sampleVisitor struct {
	result *visitCounts
}

func (v *sampleVisitor) VisitComponent(ctx *node.VisitContext, n *node.ComponentNode) error {
	v.result.components++
	return nil
}
func (v *sampleVisitor) VisitHook(ctx *node.VisitContext, n *node.HookNode) error {
	v.result.hooks++
	return nil
}
func (v *sampleVisitor) VisitState(ctx *node.VisitContext, n *node.StateNode) error {
	v.result.states++
	return nil
}
func (v *sampleVisitor) VisitJSX(ctx *node.VisitContext, n *node.JSXNode) error {
	v.result.jsxs++
	return nil
}

func TestParseThenBuildIR(t *testing.T) {
	src := `export function SimpleCard({ title }: { title: string }) {
  return <div className="card"><h2>{title}</h2></div>;
}
`
	g := parseStr(t, src)
	comps := g.AllNodesOfKind(node.KindComponent)
	if len(comps) != 1 {
		t.Fatalf("expected 1 component, got %d", len(comps))
	}
	if comps[0].Name() != "SimpleCard" {
		t.Errorf("expected SimpleCard, got %s", comps[0].Name())
	}
	if !comps[0].(*node.ComponentNode).IsExport {
		t.Error("expected IsExport true")
	}
}
