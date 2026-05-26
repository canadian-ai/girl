package parsertsx

import (
	"strings"
	"testing"

	"github.com/canadian-ai/girl/internal/node"
)

func TestBuildComponentFromBody(t *testing.T) {
	content := `import { useState } from "react";

export default function MyComponent() {
  const [count, setCount] = useState(0);
  return <div>{count}</div>;
}
`
	g := node.NewNodeGraph()
	lines := strings.Split(content, "\n")
	id := buildComponentFromBody(g, "test.tsx", content, lines, "MyComponent")
	if id == "" {
		t.Fatal("expected non-empty component ID")
	}
	comps := g.AllNodesOfKind(node.KindComponent)
	if len(comps) != 1 {
		t.Fatalf("expected 1 component, got %d", len(comps))
	}
	if comps[0].Name() != "MyComponent" {
		t.Errorf("expected MyComponent, got %s", comps[0].Name())
	}
}

func TestBuildComponentFromBody_ArrowFn(t *testing.T) {
	content := `const Greeting: React.FC<{ name: string }> = ({ name }) => {
  return <h1>Hello, {name}!</h1>;
};
`
	g := node.NewNodeGraph()
	lines := strings.Split(content, "\n")
	id := buildComponentFromBody(g, "test.tsx", content, lines, "Greeting")
	if id == "" {
		t.Fatal("expected non-empty component ID")
	}
	comps := g.AllNodesOfKind(node.KindComponent)
	if len(comps) != 1 {
		t.Fatalf("expected 1 component, got %d", len(comps))
	}
	if comps[0].Name() != "Greeting" {
		t.Errorf("expected Greeting, got %s", comps[0].Name())
	}
}

func TestFindFunctionBody(t *testing.T) {
	content := `function Foo() {
  return <div>hello</div>;
}`
	braceStart := strings.Index(content, "{")
	if braceStart < 0 {
		t.Fatal("expected opening brace")
	}
	end := findFunctionBody(content, braceStart)
	if end <= braceStart {
		t.Fatal("expected end after start")
	}
	body := content[braceStart:end]
	if !strings.Contains(body, "return") {
		t.Error("expected body to contain 'return'")
	}
	if body[len(body)-1] != '}' {
		t.Error("expected body to end with '}'")
	}
}

func TestFindFunctionBody_NestedBraces(t *testing.T) {
	content := `function Outer() {
  if (true) {
    return <div>nested</div>;
  }
}`
	braceStart := strings.Index(content, "{")
	end := findFunctionBody(content, braceStart)
	body := content[braceStart:end]
	if !strings.HasSuffix(body, "}\n}") && !strings.HasSuffix(body, "}") {
		t.Error("expected body to close outermost brace")
	}
	braceCount := strings.Count(body, "{") - strings.Count(body, "}")
	if braceCount != 0 {
		t.Errorf("expected balanced braces, got delta %d", braceCount)
	}
}

func TestLooksLikeComponentInitializer_Arrow(t *testing.T) {
	tz := newTokenizer("= () => {")
	result := looksLikeComponentInitializer(tz)
	if !result {
		t.Error("expected true for arrow function initializer")
	}
}

func TestLooksLikeComponentInitializer_Function(t *testing.T) {
	tz := newTokenizer("= function() {")
	result := looksLikeComponentInitializer(tz)
	if !result {
		t.Error("expected true for function initializer")
	}
}

func TestLooksLikeComponentInitializer_Lowercase(t *testing.T) {
	tz := newTokenizer("= 42")
	result := looksLikeComponentInitializer(tz)
	if result {
		t.Error("expected false for non-component initializer")
	}
}

func TestLooksLikeComponentInitializer_Memo(t *testing.T) {
	tz := newTokenizer("= React.memo(function() {")
	result := looksLikeComponentInitializer(tz)
	if !result {
		t.Error("expected true for React.memo initializer")
	}
}

func TestLooksLikeComponentInitializer_NoEquals(t *testing.T) {
	tz := newTokenizer("something else")
	result := looksLikeComponentInitializer(tz)
	if result {
		t.Error("expected false when no '=' sign")
	}
}

func TestIsComponentName_Uppercase(t *testing.T) {
	if !isComponentName("MyComponent") {
		t.Error("expected MyComponent to be a component name")
	}
	if !isComponentName("Header") {
		t.Error("expected Header to be a component name")
	}
}

func TestIsComponentName_Lowercase(t *testing.T) {
	if isComponentName("helper") {
		t.Error("expected helper to NOT be a component name")
	}
	if isComponentName("myComponent") {
		t.Error("expected myComponent to NOT be a component name")
	}
}

func TestIsComponentName_Empty(t *testing.T) {
	if isComponentName("") {
		t.Error("expected empty string to NOT be a component name")
	}
}

func TestIsComponentName_AllUpper(t *testing.T) {
	if isComponentName("API") {
		t.Error("expected all-uppercase API to NOT be a component name")
	}
	if isComponentName("GLOBAL") {
		t.Error("expected all-uppercase GLOBAL to NOT be a component name")
	}
}
