package parsertsx

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/canadian-ai/girl/internal/ir"
)

func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestFunctionComponent(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "CompA.tsx", `function MyComp() { return <div/>; }`)

	fir, err := New().ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(fir.Components) != 1 {
		t.Fatalf("expected 1 component, got %d", len(fir.Components))
	}
	comp := fir.Components[0]
	if comp.Name != "MyComp" {
		t.Errorf("expected MyComp, got %s", comp.Name)
	}
	if comp.Kind != ir.ComponentKindFunction {
		t.Errorf("expected function kind, got %s", comp.Kind)
	}
}

func TestArrowComponent(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "CompB.tsx", `const MyComp = () => { return <div/>; }`)

	fir, err := New().ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(fir.Components) != 1 {
		t.Fatalf("expected 1 component, got %d", len(fir.Components))
	}
	comp := fir.Components[0]
	if comp.Name != "MyComp" {
		t.Errorf("expected MyComp, got %s", comp.Name)
	}
	if comp.Kind != ir.ComponentKindArrow {
		t.Errorf("expected arrow kind, got %s", comp.Kind)
	}
}

func TestExportedComponent(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "CompC.tsx", `export function ExportedComp() { return <div/>; }`)

	fir, err := New().ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(fir.Components) != 1 {
		t.Fatalf("expected 1 component, got %d", len(fir.Components))
	}
	comp := fir.Components[0]
	if comp.Name != "ExportedComp" {
		t.Errorf("expected ExportedComp, got %s", comp.Name)
	}
	if len(comp.Exports) != 1 {
		t.Fatalf("expected 1 export, got %d", len(comp.Exports))
	}
	if comp.Exports[0].Name != "ExportedComp" || comp.Exports[0].Default {
		t.Errorf("unexpected export: %+v", comp.Exports[0])
	}
}

func TestDefaultExport(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "CompD.tsx", `export default function DefaultComp() { return <div/>; }`)

	fir, err := New().ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(fir.Components) != 1 {
		t.Fatalf("expected 1 component, got %d", len(fir.Components))
	}
	comp := fir.Components[0]
	if comp.Name != "DefaultComp" {
		t.Errorf("expected DefaultComp, got %s", comp.Name)
	}
	if len(comp.Exports) != 1 {
		t.Fatalf("expected 1 export, got %d", len(comp.Exports))
	}
	if comp.Exports[0].Name != "DefaultComp" || !comp.Exports[0].Default {
		t.Errorf("unexpected default export: %+v", comp.Exports[0])
	}
}

func TestHooksAndState(t *testing.T) {
	dir := t.TempDir()
	content := `import { useState, useEffect, useRef } from 'react';
function HookedComp() {
  const [count, setCount] = useState(0);
  useEffect(() => {}, []);
  const ref = useRef(null);
  return <div>{count}</div>;
}`
	path := writeFile(t, dir, "hooks.tsx", content)

	fir, err := New().ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(fir.Components) != 1 {
		t.Fatalf("expected 1 component, got %d", len(fir.Components))
	}

	comp := fir.Components[0]
	if comp.Name != "HookedComp" {
		t.Errorf("expected HookedComp, got %s", comp.Name)
	}
	if len(comp.Hooks) != 3 {
		t.Errorf("expected 3 hooks, got %d: %+v", len(comp.Hooks), comp.Hooks)
	}
	if len(comp.StateVars) != 1 {
		t.Fatalf("expected 1 state var, got %d", len(comp.StateVars))
	}
	if comp.StateVars[0].Name != "count" {
		t.Errorf("expected state name 'count', got %s", comp.StateVars[0].Name)
	}
	if !comp.StateVars[0].HasUpdater {
		t.Error("expected state to have updater")
	}
	if comp.StateVars[0].Line != 3 {
		t.Errorf("expected state var line 3, got %d", comp.StateVars[0].Line)
	}
	if len(comp.Effects) != 1 {
		t.Fatalf("expected 1 effect, got %d", len(comp.Effects))
	}
	if comp.Effects[0].Name != "useEffect" {
		t.Errorf("expected effect name useEffect, got %s", comp.Effects[0].Name)
	}
}

func TestJSXChildComponents(t *testing.T) {
	dir := t.TempDir()
	content := `function ParentComp() {
  return <div><Child /><OtherChild></OtherChild></div>;
}`
	path := writeFile(t, dir, "jsx.tsx", content)

	fir, err := New().ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(fir.Components) != 1 {
		t.Fatalf("expected 1 component, got %d", len(fir.Components))
	}
	comp := fir.Components[0]
	if len(comp.JSXBlocks) != 2 {
		t.Errorf("expected 2 JSX blocks, got %d: %+v", len(comp.JSXBlocks), comp.JSXBlocks)
	}
}

func TestEventHandlers(t *testing.T) {
	dir := t.TempDir()
	content := `function FormComp() {
  const handleClick = () => {};
  const onSubmit = (e) => { e.preventDefault(); };
  const submitHandler = () => {};
  return <form onSubmit={onSubmit}><button onClick={handleClick}>Submit</button></form>;
}`
	path := writeFile(t, dir, "events.tsx", content)

	fir, err := New().ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(fir.Components) != 1 {
		t.Fatalf("expected 1 component, got %d", len(fir.Components))
	}
	comp := fir.Components[0]
	if len(comp.EventHandlers) != 3 {
		t.Errorf("expected 3 event handlers, got %d: %+v", len(comp.EventHandlers), comp.EventHandlers)
	}
}

func TestImports(t *testing.T) {
	dir := t.TempDir()
	content := `import React, { useState, useEffect } from 'react';
import { BrowserRouter as Router, Route } from 'react-router-dom';
import * as Utils from './utils';
import type { SomeType } from './types';
function ImportComp() { return <div/>; }`
	path := writeFile(t, dir, "imports.tsx", content)

	fir, err := New().ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(fir.Imports) != 4 {
		t.Fatalf("expected 4 import groups, got %d: %+v", len(fir.Imports), fir.Imports)
	}
}

func TestImportDetails(t *testing.T) {
	dir := t.TempDir()
	content := `import React, { useState, useEffect } from 'react';
import _ from 'lodash';
import * as d3 from 'd3';
function ImportCheck() { return <div/>; }`
	path := writeFile(t, dir, "importcheck.tsx", content)

	fir, err := New().ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}

	var reactImp, lodashImp, d3Imp bool
	for _, imp := range fir.Imports {
		switch imp.Source {
		case "react":
			reactImp = true
			if imp.Default != "React" {
				t.Errorf("react import: expected default 'React', got %q", imp.Default)
			}
			if len(imp.Names) != 2 {
				t.Errorf("react import: expected 2 named imports, got %d: %v", len(imp.Names), imp.Names)
			}
		case "lodash":
			lodashImp = true
			if imp.Default != "_" {
				t.Errorf("lodash import: expected default '_', got %q", imp.Default)
			}
		case "d3":
			d3Imp = true
		}
	}
	if !reactImp {
		t.Error("import from 'react' not found")
	}
	if !lodashImp {
		t.Error("import from 'lodash' not found")
	}
	if !d3Imp {
		t.Error("import from 'd3' not found")
	}
}

func TestReactMemo(t *testing.T) {
	dir := t.TempDir()
	content := `const Memoized = React.memo(() => { return <div/>; });`
	path := writeFile(t, dir, "memo.tsx", content)

	fir, err := New().ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(fir.Components) != 1 {
		t.Fatalf("expected 1 component, got %d", len(fir.Components))
	}
	if fir.Components[0].Name != "Memoized" {
		t.Errorf("expected Memoized, got %s", fir.Components[0].Name)
	}
	if fir.Components[0].Kind != ir.ComponentKindArrow {
		t.Errorf("expected arrow kind, got %s", fir.Components[0].Kind)
	}
}

func TestReactMemoNamedFunction(t *testing.T) {
	dir := t.TempDir()
	content := `const MemoizedNamed = React.memo(function Inner() {
  return <section/>;
});`
	path := writeFile(t, dir, "memo_named.tsx", content)

	fir, err := New().ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(fir.Components) != 1 {
		t.Fatalf("expected 1 component, got %d", len(fir.Components))
	}
	if fir.Components[0].Name != "MemoizedNamed" {
		t.Errorf("expected MemoizedNamed, got %s", fir.Components[0].Name)
	}
}

func TestGenericArrowComponent(t *testing.T) {
	dir := t.TempDir()
	content := `const MyComp = <T,>(props: { data: T }) => { return <div/>; }`
	path := writeFile(t, dir, "generic.tsx", content)

	fir, err := New().ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(fir.Components) != 1 {
		t.Fatalf("expected 1 component, got %d", len(fir.Components))
	}
	if fir.Components[0].Name != "MyComp" {
		t.Errorf("expected MyComp, got %s", fir.Components[0].Name)
	}
}

func TestNestedComponents(t *testing.T) {
	dir := t.TempDir()
	content := `function OuterComp() {
  return <div/>;
  function InnerComp() { return <span/>; }
}`
	path := writeFile(t, dir, "nested.tsx", content)

	fir, err := New().ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(fir.Components) != 2 {
		t.Fatalf("expected 2 components (Outer, Inner), got %d: %+v", len(fir.Components), fir.Components)
	}
}

func TestHasKeyDown(t *testing.T) {
	dir := t.TempDir()
	content := `function KeyboardComp() {
  const handleKeyDown = (e) => { console.log(e.key); };
  return <div onKeyDown={handleKeyDown}/>;
}`
	path := writeFile(t, dir, "keydown.tsx", content)

	fir, err := New().ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(fir.Components) != 1 {
		t.Fatalf("expected 1 component, got %d", len(fir.Components))
	}
	if !fir.Components[0].HasKeyDown {
		t.Error("expected HasKeyDown to be true")
	}
}

func TestHasAnalytics(t *testing.T) {
	dir := t.TempDir()
	content := `function AnalyticsComp() {
  useEffect(() => { analytics.track('page_view'); }, []);
  return <div/>;
}`
	path := writeFile(t, dir, "analytics.tsx", content)

	fir, err := New().ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(fir.Components) != 1 {
		t.Fatalf("expected 1 component, got %d", len(fir.Components))
	}
	if !fir.Components[0].HasAnalytics {
		t.Error("expected HasAnalytics to be true")
	}
}

func TestConditionalCount(t *testing.T) {
	dir := t.TempDir()
	content := `function CondComp() {
  const x = true;
  if (x) { return <div/>; }
  return null;
}`
	path := writeFile(t, dir, "cond.tsx", content)

	fir, err := New().ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(fir.Components) != 1 {
		t.Fatalf("expected 1 component, got %d", len(fir.Components))
	}
	if fir.Components[0].ConditionalCount != 1 {
		t.Errorf("expected ConditionalCount=1, got %d", fir.Components[0].ConditionalCount)
	}
}

func TestLoopCount(t *testing.T) {
	dir := t.TempDir()
	content := `function LoopComp() {
  const items = [1, 2, 3];
  return <div>{items.map(i => <span key={i}/>)}</div>;
}`
	path := writeFile(t, dir, "loop.tsx", content)

	fir, err := New().ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(fir.Components) != 1 {
		t.Fatalf("expected 1 component, got %d", len(fir.Components))
	}
	if fir.Components[0].LoopCount < 1 {
		t.Errorf("expected LoopCount >= 1 (items.map), got %d", fir.Components[0].LoopCount)
	}
}

func TestFunctionExpressionComponent(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "fnExpr.tsx", `const MyComp = function() { return <div/>; }`)

	fir, err := New().ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(fir.Components) != 1 {
		t.Fatalf("expected 1 component, got %d", len(fir.Components))
	}
	if fir.Components[0].Name != "MyComp" {
		t.Errorf("expected MyComp, got %s", fir.Components[0].Name)
	}
	if fir.Components[0].Kind != ir.ComponentKindFunction {
		t.Errorf("expected function kind, got %s", fir.Components[0].Kind)
	}
}

func TestLowerCaseIgnored(t *testing.T) {
	dir := t.TempDir()
	content := `function helper() { return 42; }
function MyComp() { return <div/>; }`
	path := writeFile(t, dir, "mixed.tsx", content)

	fir, err := New().ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(fir.Components) != 1 {
		t.Fatalf("expected 1 component, got %d", len(fir.Components))
	}
	if fir.Components[0].Name != "MyComp" {
		t.Errorf("expected MyComp, got %s", fir.Components[0].Name)
	}
}

func TestJSFile(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "comp.js", `function MyComp() { return <div/>; }`)

	fir, err := New().ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if fir.Language != "javascript" {
		t.Errorf("expected language javascript, got %s", fir.Language)
	}
	if len(fir.Components) != 1 {
		t.Fatalf("expected 1 component, got %d", len(fir.Components))
	}
	if fir.Components[0].Name != "MyComp" {
		t.Errorf("expected MyComp, got %s", fir.Components[0].Name)
	}
}

func TestTSFile(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "comp.ts", `function MyComp() { return null; }`)

	fir, err := New().ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if fir.Language != "typescript" {
		t.Errorf("expected language typescript, got %s", fir.Language)
	}
	if len(fir.Components) != 1 {
		t.Fatalf("expected 1 component, got %d", len(fir.Components))
	}
	if fir.Components[0].Name != "MyComp" {
		t.Errorf("expected MyComp, got %s", fir.Components[0].Name)
	}
}

func TestJSXFile(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "comp.jsx", `function MyComp() { return <div/>; }`)

	fir, err := New().ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if fir.Language != "javascriptreact" {
		t.Errorf("expected language javascriptreact, got %s", fir.Language)
	}
	if len(fir.Components) != 1 {
		t.Fatalf("expected 1 component, got %d", len(fir.Components))
	}
	if fir.Components[0].Name != "MyComp" {
		t.Errorf("expected MyComp, got %s", fir.Components[0].Name)
	}
}

func TestFileLines(t *testing.T) {
	dir := t.TempDir()
	content := "import React from 'react';\n\nfunction LinesComp() {\n  return <div/>;\n}\n"
	path := writeFile(t, dir, "lines.tsx", content)

	fir, err := New().ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if fir.Lines != 6 {
		t.Errorf("expected 6 lines (5 newlines + 1), got %d", fir.Lines)
	}
	if len(fir.Components) != 1 {
		t.Fatalf("expected 1 component, got %d", len(fir.Components))
	}
	comp := fir.Components[0]
	if comp.Lines != 3 {
		t.Errorf("expected component body to be 3 lines, got %d", comp.Lines)
	}
}

func TestEmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "empty.tsx", "")

	fir, err := New().ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(fir.Components) != 0 {
		t.Errorf("expected 0 components, got %d", len(fir.Components))
	}
}

func TestUnsupportedExtension(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "file.css", ".foo { color: red; }")

	_, err := New().ParseFile(path)
	if err == nil {
		t.Fatal("expected error for unsupported extension, got nil")
	}
}

func TestTopLevelHooks(t *testing.T) {
	dir := t.TempDir()
	content := `import { useTheme } from './theme';
const theme = useTheme('dark');
function ThemedComp() { return <div/>; }`
	path := writeFile(t, dir, "toplevel.tsx", content)

	fir, err := New().ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	// useTheme('dark') is outside the component — should be a top-level hook
	if len(fir.Hooks) != 1 {
		t.Fatalf("expected 1 top-level hook, got %d: %+v", len(fir.Hooks), fir.Hooks)
	}
	if fir.Hooks[0].Name != "useTheme" {
		t.Errorf("expected useTheme, got %s", fir.Hooks[0].Name)
	}
}

func TestOptionalChainingAttr(t *testing.T) {
	dir := t.TempDir()
	content := `function OptChainComp(props: { data?: { name?: string } }) {
  return <div title={props.data?.name ?? 'default'}>{props.data?.name}</div>;
}`
	path := writeFile(t, dir, "optchain.tsx", content)

	fir, err := New().ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(fir.Components) != 1 {
		t.Fatalf("expected 1 component, got %d", len(fir.Components))
	}
	if fir.Components[0].Name != "OptChainComp" {
		t.Errorf("expected OptChainComp, got %s", fir.Components[0].Name)
	}
}

func TestTemplateLiteralAttr(t *testing.T) {
	dir := t.TempDir()
	content := `function ThemeComp({ active }: { active: boolean }) {
  return <div className={` + "`bg-blue-${active ? '500' : '300'}`" + `}/>;
}`
	path := writeFile(t, dir, "templit.tsx", content)

	fir, err := New().ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(fir.Components) != 1 {
		t.Fatalf("expected 1 component, got %d", len(fir.Components))
	}
	if fir.Components[0].Name != "ThemeComp" {
		t.Errorf("expected ThemeComp, got %s", fir.Components[0].Name)
	}
	if len(fir.Components[0].Props) != 1 || fir.Components[0].Props[0].Name != "active" {
		t.Errorf("expected prop 'active', got %+v", fir.Components[0].Props)
	}
}

func TestForwardRefComponent(t *testing.T) {
	dir := t.TempDir()
	content := `const Input = forwardRef<HTMLInputElement, Props>((props, ref) => {
  return <input ref={ref} />;
});`
	path := writeFile(t, dir, "forwardref.tsx", content)

	fir, err := New().ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(fir.Components) != 1 {
		t.Fatalf("expected 1 component, got %d", len(fir.Components))
	}
	if fir.Components[0].Name != "Input" {
		t.Errorf("expected Input, got %s", fir.Components[0].Name)
	}
}

func TestForwardRefReactComponent(t *testing.T) {
	dir := t.TempDir()
	content := `const Input = React.forwardRef<HTMLInputElement, Props>(function InputInner(props, ref) {
  return <input ref={ref} />;
});`
	path := writeFile(t, dir, "forwardref_react.tsx", content)

	fir, err := New().ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(fir.Components) != 1 {
		t.Fatalf("expected 1 component, got %d", len(fir.Components))
	}
	if fir.Components[0].Name != "Input" {
		t.Errorf("expected Input, got %s", fir.Components[0].Name)
	}
	if fir.Components[0].Kind != ir.ComponentKindFunction {
		t.Errorf("expected function kind, got %s", fir.Components[0].Kind)
	}
}

func TestOptionalChainingJSX(t *testing.T) {
	dir := t.TempDir()
	content := `function OptionalComp({ user }) {
  return <div>{user?.profile?.name}</div>;
}`
	path := writeFile(t, dir, "optchain_jsx.tsx", content)

	fir, err := New().ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(fir.Components) != 1 {
		t.Fatalf("expected 1 component, got %d", len(fir.Components))
	}
	if fir.Components[0].Name != "OptionalComp" {
		t.Errorf("expected OptionalComp, got %s", fir.Components[0].Name)
	}
	if fir.Components[0].EndLine <= fir.Components[0].StartLine {
		t.Error("component body bounds are not sane: EndLine <= StartLine")
	}
}

func TestTemplateLiteralJSX(t *testing.T) {
	dir := t.TempDir()
	content := "function TemplateAttrComp({ id }) {\n  return <div className={`card-${id}`} data-id={`${id}`} />;\n}"
	path := writeFile(t, dir, "templit_jsx.tsx", content)

	fir, err := New().ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(fir.Components) != 1 {
		t.Fatalf("expected 1 component, got %d", len(fir.Components))
	}
	if fir.Components[0].Name != "TemplateAttrComp" {
		t.Errorf("expected TemplateAttrComp, got %s", fir.Components[0].Name)
	}
	if fir.Components[0].EndLine <= fir.Components[0].StartLine {
		t.Error("component body bounds are not sane: EndLine <= StartLine")
	}
}

func TestPropsFromDestructuring(t *testing.T) {
	dir := t.TempDir()
	content := `function DestructuredComp({ name, age }: { name: string; age: number }) {
  return <div>{name} is {age}</div>;
}`
	path := writeFile(t, dir, "props.tsx", content)

	fir, err := New().ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(fir.Components) != 1 {
		t.Fatalf("expected 1 component, got %d", len(fir.Components))
	}
	comp := fir.Components[0]
	if len(comp.Props) != 2 {
		t.Fatalf("expected 2 props, got %d: %+v", len(comp.Props), comp.Props)
	}
}
