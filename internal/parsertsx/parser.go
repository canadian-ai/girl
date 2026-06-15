package parsertsx

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/javascript"
	"github.com/smacker/go-tree-sitter/typescript/tsx"
	"github.com/smacker/go-tree-sitter/typescript/typescript"

	"github.com/canadian-ai/girl/internal/ir"
)

type Parser struct {
	initOnce sync.Once
	tsxLang  *sitter.Language
	tsLang   *sitter.Language
	jsLang   *sitter.Language

	importQ    *sitter.Query
	importDefQ *sitter.Query
	importNsQ  *sitter.Query
	importTypeQ *sitter.Query
	compFuncQ  *sitter.Query
	compArrowQ *sitter.Query
	compFnExprQ *sitter.Query
	compMemoQ  *sitter.Query
	hookQ      *sitter.Query
	stateVarQ  *sitter.Query
	jsxElemQ   *sitter.Query
	jsxSelfQ   *sitter.Query
	handlerQ   *sitter.Query
	exportQ    *sitter.Query
	exportDefQ *sitter.Query
}

func New() *Parser {
	return &Parser{}
}

func (p *Parser) lazyInit() *sitter.Query {
	p.initOnce.Do(func() {
		p.tsxLang = tsx.GetLanguage()
		p.tsLang = typescript.GetLanguage()
		p.jsLang = javascript.GetLanguage()

		lang := p.tsxLang
		p.importQ = p.comp(lang, `(import_statement (import_clause (named_imports (import_specifier name: (identifier) @import_name)) (identifier)? @import_default) source: (string (string_fragment) @import_source)) @import_full`)
		p.importDefQ = p.comp(lang, `(import_statement (import_clause (identifier) @import_default) source: (string (string_fragment) @import_source)) @import_full`)
		p.importNsQ = p.comp(lang, `(import_statement (import_clause (namespace_import) @import_ns) source: (string (string_fragment) @import_source)) @import_full`)
		p.importTypeQ = p.comp(lang, `(import_statement "type" (import_clause (named_imports (import_specifier name: (identifier) @import_name))) source: (string (string_fragment) @import_source)) @import_full`)
		p.compFuncQ = p.comp(lang, `(function_declaration name: (identifier) @comp_name) @comp_func`)
		p.compArrowQ = p.comp(lang, `(lexical_declaration (variable_declarator name: (identifier) @comp_name value: (arrow_function))) @comp_arrow`)
		p.compFnExprQ = p.comp(lang, `(lexical_declaration (variable_declarator name: (identifier) @comp_name value: (function_expression))) @comp_fn_expr`)
		p.compMemoQ = p.comp(lang, `(lexical_declaration (variable_declarator name: (identifier) @comp_name value: (call_expression function: (_) @memo_func))) @comp_memo`)
		p.hookQ = p.comp(lang, `(call_expression function: (identifier) @hook_name arguments: (arguments) @hook_args) @hook_call`)
		p.stateVarQ = p.comp(lang, `(variable_declarator name: (array_pattern (identifier) @state_name (identifier) @state_setter) value: (call_expression function: (identifier) @state_fn arguments: (arguments (_)?))) @state_decl`)
		p.jsxElemQ = p.comp(lang, `(jsx_element open_tag: (jsx_opening_element (identifier) @jsx_tag)) @jsx_elem`)
		p.jsxSelfQ = p.comp(lang, `(jsx_self_closing_element (identifier) @jsx_tag) @jsx_self`)
		p.handlerQ = p.comp(lang, `(variable_declarator name: (identifier) @handler_name) @handler_decl`)
		p.exportQ = p.comp(lang, `(export_statement declaration: (_) @export_decl) @export_stmt`)
		p.exportDefQ = p.comp(lang, `(export_statement "default" declaration: (_) @export_default_decl) @export_default`)
	})
	return p.importQ
}

func (p *Parser) comp(lang *sitter.Language, pattern string) *sitter.Query {
	q, err := sitter.NewQuery([]byte(pattern), lang)
	if err != nil {
		panic(fmt.Sprintf("invalid tree-sitter query: %v", err))
	}
	return q
}

func (p *Parser) grammarFor(path string) (*sitter.Language, error) {
	p.lazyInit()
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".tsx":
		return p.tsxLang, nil
	case ".ts":
		return p.tsLang, nil
	case ".jsx", ".js":
		return p.jsLang, nil
	default:
		return nil, fmt.Errorf("unsupported file extension: %s", ext)
	}
}

func (p *Parser) ParseFile(path string) (*ir.FileIR, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	lang, err := p.grammarFor(path)
	if err != nil {
		return nil, err
	}

	sp := sitter.NewParser()
	sp.SetLanguage(lang)
	tree, err := sp.ParseCtx(context.Background(), nil, data)
	if err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}
	root := tree.RootNode()

	fir := &ir.FileIR{
		Path:       path,
		Language:   languageTag(path),
		Lines:      bytes.Count(data, []byte{'\n'}) + 1,
		Components: nil,
		Hooks:      nil,
		Imports:    nil,
	}

	fir.Imports = p.extractImports(root, data)
	comps, hooks := p.extractComponents(root, data, path)
	fir.Components = comps
	fir.Hooks = hooks

	return fir, nil
}

func (p *Parser) ParseDir(dir string, excludeDirs []string) ([]*ir.FileIR, error) {
	p.lazyInit()
	excludeMap := make(map[string]bool)
	for _, d := range excludeDirs {
		excludeMap[d] = true
	}
	var results []*ir.FileIR
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() && path != dir {
			base := filepath.Base(path)
			if isSkippedDir(base) || excludeMap[base] {
				return filepath.SkipDir
			}
			return nil
		}
		if info.IsDir() {
			return nil
		}
		if !isParseableExt(path) {
			return nil
		}
		fir, err := p.ParseFile(path)
		if err != nil {
			return nil
		}
		results = append(results, fir)
		return nil
	})
	return results, err
}

func isSkippedDir(name string) bool {
	if strings.HasPrefix(name, ".") {
		return true
	}
	switch name {
	case "node_modules", "dist", "build", ".next", "coverage":
		return true
	}
	return false
}

func isParseableExt(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".tsx", ".ts", ".jsx", ".js":
		return true
	}
	return false
}

func languageTag(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".tsx":
		return "typescriptreact"
	case ".ts":
		return "typescript"
	case ".jsx":
		return "javascriptreact"
	case ".js":
		return "javascript"
	}
	return "unknown"
}

func (p *Parser) execQuery(q *sitter.Query, node *sitter.Node, data []byte) []*sitter.QueryMatch {
	cursor := sitter.NewQueryCursor()
	cursor.Exec(q, node)
	var matches []*sitter.QueryMatch
	for {
		m, ok := cursor.NextMatch()
		if !ok {
			break
		}
		m = cursor.FilterPredicates(m, data)
		if len(m.Captures) > 0 {
			matches = append(matches, m)
		}
	}
	return matches
}

func captureByName(m *sitter.QueryMatch, q *sitter.Query, name string) *sitter.Node {
	for _, c := range m.Captures {
		if q.CaptureNameForId(c.Index) == name {
			return c.Node
		}
	}
	return nil
}

func captureContent(m *sitter.QueryMatch, q *sitter.Query, name string, data []byte) string {
	n := captureByName(m, q, name)
	if n == nil {
		return ""
	}
	return string(data[n.StartByte():n.EndByte()])
}

func (p *Parser) extractImports(root *sitter.Node, data []byte) []ir.ImportIR {
	p.lazyInit()
	imports := []ir.ImportIR{}
	seen := map[string]int{}

	merge := func(source string, name string, named string) {
		if source == "" {
			return
		}
		idx, ok := seen[source]
		if !ok {
			idx = len(imports)
			seen[source] = idx
			imports = append(imports, ir.ImportIR{Source: source})
		}
		if name != "" && imports[idx].Default == "" {
			imports[idx].Default = name
		}
		if named != "" {
			found := false
			for _, n := range imports[idx].Names {
				if n == named {
					found = true
					break
				}
			}
			if !found {
				imports[idx].Names = append(imports[idx].Names, named)
			}
		}
	}

	for _, m := range p.execQuery(p.importQ, root, data) {
		source := captureContent(m, p.importQ, "import_source", data)
		def := captureContent(m, p.importQ, "import_default", data)
		name := captureContent(m, p.importQ, "import_name", data)
		merge(source, def, name)
	}

	for _, m := range p.execQuery(p.importDefQ, root, data) {
		source := captureContent(m, p.importDefQ, "import_source", data)
		def := captureContent(m, p.importDefQ, "import_default", data)
		merge(source, def, "")
	}

	for _, m := range p.execQuery(p.importNsQ, root, data) {
		source := captureContent(m, p.importNsQ, "import_source", data)
		merge(source, "", "")
	}

	for _, m := range p.execQuery(p.importTypeQ, root, data) {
		source := captureContent(m, p.importTypeQ, "import_source", data)
		name := captureContent(m, p.importTypeQ, "import_name", data)
		merge(source, "", name)
	}

	return imports
}

type componentMatch struct {
	name    string
	body    *sitter.Node
	params  *sitter.Node
	isArrow bool
}

func (p *Parser) findComponentMatches(root *sitter.Node, data []byte) []componentMatch {
	var matches []componentMatch

	for _, m := range p.execQuery(p.compFuncQ, root, data) {
		name := captureContent(m, p.compFuncQ, "comp_name", data)
		if !isComponentName(name) {
			continue
		}
		fn := captureByName(m, p.compFuncQ, "comp_func")
		body := fn.ChildByFieldName("body")
		params := fn.ChildByFieldName("parameters")
		if body == nil || params == nil {
			continue
		}
		matches = append(matches, componentMatch{
			name:    name,
			body:    body,
			params:  params,
			isArrow: false,
		})
	}

	for _, m := range p.execQuery(p.compArrowQ, root, data) {
		name := captureContent(m, p.compArrowQ, "comp_name", data)
		if !isComponentName(name) {
			continue
		}
		decl := captureByName(m, p.compArrowQ, "comp_arrow")
		arrow := findChildByType(decl, "arrow_function")
		if arrow == nil {
			continue
		}
		body := arrow.ChildByFieldName("body")
		params := arrow.ChildByFieldName("parameters")
		if body == nil || params == nil {
			continue
		}
		if body.Type() == "statement_block" {
			matches = append(matches, componentMatch{
				name:    name,
				body:    body,
				params:  params,
				isArrow: true,
			})
		}
	}

	for _, m := range p.execQuery(p.compFnExprQ, root, data) {
		name := captureContent(m, p.compFnExprQ, "comp_name", data)
		if !isComponentName(name) {
			continue
		}
		decl := captureByName(m, p.compFnExprQ, "comp_fn_expr")
		fn := findChildByType(decl, "function_expression")
		if fn == nil {
			continue
		}
		body := fn.ChildByFieldName("body")
		params := fn.ChildByFieldName("parameters")
		if body == nil || params == nil {
			continue
		}
		matches = append(matches, componentMatch{
			name:    name,
			body:    body,
			params:  params,
			isArrow: false,
		})
	}

	for _, m := range p.execQuery(p.compMemoQ, root, data) {
		name := captureContent(m, p.compMemoQ, "comp_name", data)
		if !isComponentName(name) {
			continue
		}
		decl := captureByName(m, p.compMemoQ, "comp_memo")
		vd := decl.NamedChild(0)
		if vd == nil || vd.Type() != "variable_declarator" {
			continue
		}
		val := vd.ChildByFieldName("value")
		if val == nil || val.Type() != "call_expression" {
			continue
		}
		funcArg := val.NamedChild(0)
		if funcArg == nil {
			continue
		}
		if funcArg.Type() == "arrow_function" || funcArg.Type() == "function_expression" {
			body := funcArg.ChildByFieldName("body")
			params := funcArg.ChildByFieldName("parameters")
			if body != nil && params != nil && body.Type() == "statement_block" {
				matches = append(matches, componentMatch{
					name:    name,
					body:    body,
					params:  params,
					isArrow: funcArg.Type() == "arrow_function",
				})
			}
		}
	}

	return matches
}

func isComponentName(name string) bool {
	if len(name) == 0 {
		return false
	}
	first := name[0]
	return first >= 'A' && first <= 'Z'
}

func isJSXComponentName(name string) bool {
	if len(name) == 0 {
		return false
	}
	if name[0] >= 'a' && name[0] <= 'z' {
		return false
	}
	return true
}

func findChildByType(n *sitter.Node, typ string) *sitter.Node {
	for i := 0; i < int(n.NamedChildCount()); i++ {
		c := n.NamedChild(i)
		if c.Type() == typ {
			return c
		}
		r := findChildByType(c, typ)
		if r != nil {
			return r
		}
	}
	return nil
}

func (p *Parser) extractComponents(root *sitter.Node, data []byte, path string) ([]ir.ComponentIR, []ir.HookIR) {
	matches := p.findComponentMatches(root, data)
	exportedNames := p.exportedFunctionNames(root, data)

	components := make([]ir.ComponentIR, 0, len(matches))
	var topHooks []ir.HookIR

	for _, m := range matches {
		kind := ir.ComponentKindArrow
		if !m.isArrow {
			kind = ir.ComponentKindFunction
		}

		isExport := false
		for _, e := range exportedNames {
			if e == m.name {
				isExport = true
				break
			}
		}

		comp := ir.ComponentIR{
			Name:           m.name,
			FilePath:       path,
			Kind:           kind,
			StartLine:      int(m.body.StartPoint().Row) + 1,
			EndLine:        int(m.body.EndPoint().Row) + 1,
			Lines:          int(m.body.EndPoint().Row-m.body.StartPoint().Row) + 1,
			Hooks:          nil,
			JSXBlocks:      nil,
			Props:          nil,
			StateVars:      nil,
			Effects:        nil,
			EventHandlers:  nil,
			Imports:        nil,
			Exports:        nil,
			HasKeyDown:     false,
			HasAnalytics:   false,
			ConditionalCount: 0,
			LoopCount:      0,
		}

		if isExport {
			comp.Exports = append(comp.Exports, ir.ExportIR{
				Name:    m.name,
				Default: false,
			})
		}

		if m.params != nil {
			comp.Props = extractProps(m.params, data)
		}

		bodyData := data[m.body.StartByte():m.body.EndByte()]

		comp.Hooks = p.extractHooksInRange(m.body, data)
		comp.StateVars = p.extractStateVarsInRange(m.body, data)
		comp.Effects = extractEffectsFromHooks(comp.Hooks)
		comp.JSXBlocks = p.extractJSXInRange(m.body, data)
		comp.EventHandlers = p.extractEventHandlersInRange(m.body, data)
		comp.HasKeyDown = hasKeyDownPattern(bodyData)
		comp.HasAnalytics = hasAnalyticsPattern(bodyData)
		comp.ConditionalCount = countConditionalsInNode(m.body, data)
		comp.LoopCount = countLoopsInNode(m.body, data)

		components = append(components, comp)
	}

	topHooks = p.extractTopLevelHooks(root, data, components)

	return components, topHooks
}

func (p *Parser) extractHooksInRange(body *sitter.Node, data []byte) []ir.HookIR {
	hooks := []ir.HookIR{}
	for _, m := range p.execQuery(p.hookQ, body, data) {
		nameNode := captureByName(m, p.hookQ, "hook_name")
		if nameNode == nil {
			continue
		}
		name := string(data[nameNode.StartByte():nameNode.EndByte()])
		if !strings.HasPrefix(name, "use") {
			continue
		}
		line := int(nameNode.StartPoint().Row) + 1
		argsNode := captureByName(m, p.hookQ, "hook_args")
		var args []string
		if argsNode != nil && argsNode.NamedChildCount() > 0 {
			argStr := strings.TrimSpace(string(data[argsNode.StartByte()+1 : argsNode.EndByte()-1]))
			if argStr != "" {
				args = []string{argStr}
			}
		}
		hooks = append(hooks, ir.HookIR{
			Name: name,
			Line: line,
			Args: args,
		})
	}
	return hooks
}

func (p *Parser) extractStateVarsInRange(body *sitter.Node, data []byte) []ir.StateVarIR {
	vars := []ir.StateVarIR{}
	for _, m := range p.execQuery(p.stateVarQ, body, data) {
		fnName := captureContent(m, p.stateVarQ, "state_fn", data)
		if fnName != "useState" {
			continue
		}
		name := captureContent(m, p.stateVarQ, "state_name", data)
		setter := captureContent(m, p.stateVarQ, "state_setter", data)
		nameNode := captureByName(m, p.stateVarQ, "state_name")
		if nameNode == nil {
			continue
		}
		line := int(nameNode.StartPoint().Row) + 1
		vars = append(vars, ir.StateVarIR{
			Name:       name,
			Line:       line,
			HasUpdater: setter != "",
		})
	}
	return vars
}

func extractEffectsFromHooks(hooks []ir.HookIR) []ir.EffectIR {
	var effects []ir.EffectIR
	for _, h := range hooks {
		if h.Name == "useEffect" {
			effects = append(effects, ir.EffectIR{
				Name: "useEffect",
				Line: h.Line,
			})
		}
	}
	return effects
}

func (p *Parser) extractJSXInRange(body *sitter.Node, data []byte) []ir.JSXBlockIR {
	blocks := []ir.JSXBlockIR{}

	for _, m := range p.execQuery(p.jsxElemQ, body, data) {
		tag := captureContent(m, p.jsxElemQ, "jsx_tag", data)
		if !isJSXComponentName(tag) {
			continue
		}
		line := int(captureByName(m, p.jsxElemQ, "jsx_tag").StartPoint().Row) + 1
		blocks = append(blocks, ir.JSXBlockIR{
			Element: tag,
			Line:    line,
		})
	}

	for _, m := range p.execQuery(p.jsxSelfQ, body, data) {
		tag := captureContent(m, p.jsxSelfQ, "jsx_tag", data)
		if !isJSXComponentName(tag) {
			continue
		}
		line := int(captureByName(m, p.jsxSelfQ, "jsx_tag").StartPoint().Row) + 1
		blocks = append(blocks, ir.JSXBlockIR{
			Element: tag,
			Line:    line,
		})
	}

	return blocks
}

func (p *Parser) extractEventHandlersInRange(body *sitter.Node, data []byte) []ir.EventHandlerIR {
	handlers := []ir.EventHandlerIR{}
	for _, m := range p.execQuery(p.handlerQ, body, data) {
		name := captureContent(m, p.handlerQ, "handler_name", data)
		if name == "" {
			continue
		}
		if strings.HasPrefix(name, "handle") || strings.HasPrefix(name, "on") ||
			strings.HasSuffix(name, "Handler") {
			line := int(captureByName(m, p.handlerQ, "handler_name").StartPoint().Row) + 1
			handlers = append(handlers, ir.EventHandlerIR{
				Name: name,
				Line: line,
			})
		}
	}
	return handlers
}

func extractProps(params *sitter.Node, data []byte) []ir.PropIR {
	if params == nil {
		return nil
	}

	props := []ir.PropIR{}

	for i := 0; i < int(params.NamedChildCount()); i++ {
		param := params.NamedChild(i)
		if param.Type() != "required_parameter" && param.Type() != "optional_parameter" {
			continue
		}
		pattern := param.ChildByFieldName("pattern")
		if pattern == nil || pattern.Type() != "object_pattern" {
			continue
		}

		for j := 0; j < int(pattern.NamedChildCount()); j++ {
			prop := pattern.NamedChild(j)
			name := ""
			propType := ""
			required := true

			switch prop.Type() {
			case "shorthand_property_identifier_pattern":
				name = string(data[prop.StartByte():prop.EndByte()])
				ann := findTypeAnnotation(prop)
				if ann != nil {
					raw := strings.TrimSpace(string(data[ann.StartByte():ann.EndByte()]))
					propType = strings.TrimSpace(strings.TrimPrefix(raw, ":"))
					propType = strings.TrimSpace(propType)
				}
			case "pair_pattern":
				val := prop.ChildByFieldName("value")
				if val != nil && val.Type() == "identifier" {
					name = string(data[val.StartByte():val.EndByte()])
				}
				ann := findTypeAnnotation(prop)
				if ann != nil {
					raw := strings.TrimSpace(string(data[ann.StartByte():ann.EndByte()]))
					propType = strings.TrimSpace(strings.TrimPrefix(raw, ":"))
					propType = strings.TrimSpace(propType)
				}
			case "property_identifier_pattern":
				name = string(data[prop.StartByte():prop.EndByte()])
			}

			if name == "" {
				continue
			}

			if param.Type() == "optional_parameter" {
				required = false
			}

			props = append(props, ir.PropIR{
				Name:     name,
				Type:     propType,
				Required: required,
				Line:     int(prop.StartPoint().Row) + 1,
			})
		}
	}

	return props
}

func findTypeAnnotation(n *sitter.Node) *sitter.Node {
	for i := 0; i < int(n.NamedChildCount()); i++ {
		c := n.NamedChild(i)
		if c.Type() == "type_annotation" {
			return c
		}
		r := findTypeAnnotation(c)
		if r != nil {
			return r
		}
	}
	return nil
}

func countConditionalsInNode(body *sitter.Node, data []byte) int {
	count := 0
	walkTree(body, func(n *sitter.Node) {
		switch n.Type() {
		case "if_statement":
			count++
		case "ternary_expression":
			count++
		}
	})
	return count
}

func countLoopsInNode(body *sitter.Node, data []byte) int {
	count := 0
	walkTree(body, func(n *sitter.Node) {
		switch n.Type() {
		case "for_statement", "for_in_statement", "while_statement", "do_statement":
			count++
		}
		if n.Type() == "call_expression" {
			fn := n.ChildByFieldName("function")
			if fn != nil {
				switch string(data[fn.StartByte():fn.EndByte()]) {
				case "map", "forEach", "filter", "reduce":
					count++
				}
			}
		}
	})
	return count
}

func walkTree(n *sitter.Node, fn func(*sitter.Node)) {
	fn(n)
	for i := 0; i < int(n.NamedChildCount()); i++ {
		walkTree(n.NamedChild(i), fn)
	}
}

func hasKeyDownPattern(content []byte) bool {
	lower := bytes.ToLower(content)
	for _, p := range [][]byte{
		[]byte("onkeydown"), []byte("usekeydown"), []byte("keyboard"), []byte("keydown"),
	} {
		if bytes.Contains(lower, p) {
			return true
		}
	}
	return false
}

func hasAnalyticsPattern(content []byte) bool {
	lower := bytes.ToLower(content)
	for _, p := range [][]byte{
		[]byte("analytics"), []byte("track"), []byte("gtag"),
		[]byte("posthog"), []byte("amplitude"), []byte("mixpanel"),
	} {
		if bytes.Contains(lower, p) {
			return true
		}
	}
	return false
}

func (p *Parser) extractTopLevelHooks(root *sitter.Node, data []byte, components []ir.ComponentIR) []ir.HookIR {
	var hooks []ir.HookIR

	compLines := make(map[int]bool)
	for _, c := range components {
		for line := c.StartLine; line <= c.EndLine; line++ {
			compLines[line] = true
		}
	}

	seen := make(map[string]bool)

	for _, m := range p.execQuery(p.hookQ, root, data) {
		nameNode := captureByName(m, p.hookQ, "hook_name")
		if nameNode == nil {
			continue
		}
		name := string(data[nameNode.StartByte():nameNode.EndByte()])
		if !strings.HasPrefix(name, "use") {
			continue
		}

		line := int(nameNode.StartPoint().Row) + 1
		if compLines[line] {
			continue
		}

		key := fmt.Sprintf("%s:%d", name, line)
		if seen[key] {
			continue
		}
		seen[key] = true

		argsNode := captureByName(m, p.hookQ, "hook_args")
		var args []string
		if argsNode != nil && argsNode.NamedChildCount() > 0 {
			argStr := strings.TrimSpace(string(data[argsNode.StartByte()+1 : argsNode.EndByte()-1]))
			if argStr != "" {
				args = []string{argStr}
			}
		}

		hooks = append(hooks, ir.HookIR{
			Name: name,
			Line: line,
			Args: args,
		})
	}

	return hooks
}

func (p *Parser) exportedFunctionNames(root *sitter.Node, data []byte) []string {
	var names []string

	for _, m := range p.execQuery(p.exportQ, root, data) {
		decl := captureByName(m, p.exportQ, "export_decl")
		if decl == nil {
			continue
		}
		nameNode := decl.ChildByFieldName("name")
		if nameNode != nil {
			names = append(names, string(data[nameNode.StartByte():nameNode.EndByte()]))
			continue
		}
		if decl.NamedChildCount() > 0 {
			first := decl.NamedChild(0)
			if first.Type() == "lexical_declaration" {
				for i := 0; i < int(first.NamedChildCount()); i++ {
					vd := first.NamedChild(i)
					if vd.Type() == "variable_declarator" {
						n := vd.ChildByFieldName("name")
						if n != nil {
							names = append(names, string(data[n.StartByte():n.EndByte()]))
						}
					}
				}
			}
		}
	}

	for _, m := range p.execQuery(p.exportDefQ, root, data) {
		decl := captureByName(m, p.exportDefQ, "export_default_decl")
		if decl == nil {
			continue
		}
		nameNode := decl.ChildByFieldName("name")
		if nameNode != nil {
			names = append(names, string(data[nameNode.StartByte():nameNode.EndByte()]))
			continue
		}
		if decl.NamedChildCount() > 0 {
			first := decl.NamedChild(0)
			if first.Type() == "lexical_declaration" {
				for i := 0; i < int(first.NamedChildCount()); i++ {
					vd := first.NamedChild(i)
					if vd.Type() == "variable_declarator" {
						n := vd.ChildByFieldName("name")
						if n != nil {
							names = append(names, string(data[n.StartByte():n.EndByte()]))
						}
					}
				}
			}
		}
	}

	return names
}
