package complexity

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/javascript"
	"github.com/smacker/go-tree-sitter/typescript/tsx"
	"github.com/smacker/go-tree-sitter/typescript/typescript"

	"github.com/canadian-ai/girl/internal/shared"
)

// Analyze measures every JavaScript/TypeScript function, method, arrow
// function, callback, and React function component under path.
func Analyze(path string, opts Options) (*Report, error) {
	if opts.Threshold <= 0 {
		opts.Threshold = 10
	}
	files, root, err := collectFiles(path, opts)
	if err != nil {
		return nil, err
	}

	report := &Report{
		SchemaVersion: SchemaVersion,
		Metric:        "cyclomatic-complexity",
		Root:          displayRoot(path),
		Threshold:     opts.Threshold,
		Functions:     []FunctionMetric{},
	}
	for _, file := range files {
		rel, relErr := filepath.Rel(root, file)
		if relErr != nil {
			rel = filepath.Base(file)
		}
		rel = filepath.ToSlash(rel)
		metrics, parseErr := analyzeFile(file, rel, opts.Threshold)
		if parseErr != nil {
			report.ParseErrors = append(report.ParseErrors, ParseError{File: rel, Message: parseErr.Error()})
			continue
		}
		report.Functions = append(report.Functions, metrics...)
	}

	sort.Slice(report.Functions, func(i, j int) bool {
		if report.Functions[i].File != report.Functions[j].File {
			return report.Functions[i].File < report.Functions[j].File
		}
		if report.Functions[i].StartLine != report.Functions[j].StartLine {
			return report.Functions[i].StartLine < report.Functions[j].StartLine
		}
		return report.Functions[i].Symbol < report.Functions[j].Symbol
	})
	report.Summary = summarize(report.Functions, len(files)-len(report.ParseErrors))
	return report, nil
}

func collectFiles(path string, opts Options) ([]string, string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, "", fmt.Errorf("cannot access path %s: %w", path, err)
	}
	if !info.IsDir() {
		if !matchesLanguage(path, opts.Language) {
			return nil, "", fmt.Errorf("unsupported source file %s", path)
		}
		return []string{path}, filepath.Dir(path), nil
	}

	excluded := make(map[string]bool, len(opts.Exclude))
	for _, item := range opts.Exclude {
		excluded[item] = true
	}
	var files []string
	err = filepath.Walk(path, func(current string, entry os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if current != path && (shared.ShouldSkipDir(entry.Name()) || excluded[entry.Name()]) {
				return filepath.SkipDir
			}
			return nil
		}
		if matchesLanguage(current, opts.Language) {
			files = append(files, current)
		}
		return nil
	})
	sort.Strings(files)
	return files, path, err
}

func matchesLanguage(path, language string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	language = strings.ToLower(language)
	switch language {
	case "ts", "typescript":
		return ext == ".ts" || ext == ".tsx"
	case "tsx", "react":
		return ext == ".tsx" || ext == ".jsx"
	case "js", "javascript":
		return ext == ".js" || ext == ".jsx"
	case "jsx":
		return ext == ".jsx"
	default:
		return ext == ".ts" || ext == ".tsx" || ext == ".js" || ext == ".jsx"
	}
}

func displayRoot(path string) string {
	clean := filepath.Clean(path)
	if filepath.IsAbs(clean) {
		return filepath.Base(clean)
	}
	return filepath.ToSlash(clean)
}

func languageFor(path string) (string, *sitter.Language, error) {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".ts":
		return "typescript", typescript.GetLanguage(), nil
	case ".tsx":
		return "typescript-react", tsx.GetLanguage(), nil
	case ".js":
		return "javascript", javascript.GetLanguage(), nil
	case ".jsx":
		return "javascript-react", javascript.GetLanguage(), nil
	default:
		return "", nil, fmt.Errorf("unsupported source file %s", path)
	}
}

func analyzeFile(path, rel string, threshold int) ([]FunctionMetric, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	language, grammar, err := languageFor(path)
	if err != nil {
		return nil, err
	}
	parser := sitter.NewParser()
	parser.SetLanguage(grammar)
	tree, err := parser.ParseCtx(context.Background(), nil, data)
	if err != nil {
		return nil, fmt.Errorf("parse source: %w", err)
	}
	root := tree.RootNode()
	if root.HasError() {
		return nil, fmt.Errorf("source contains syntax errors")
	}

	collector := functionCollector{
		data:       data,
		file:       rel,
		language:   language,
		threshold:  threshold,
		anonymous:  map[string]int{},
		identities: map[string]int{},
	}
	collector.walk(root, nil, "")
	return collector.metrics, nil
}

type functionCollector struct {
	data       []byte
	file       string
	language   string
	threshold  int
	metrics    []FunctionMetric
	anonymous  map[string]int
	identities map[string]int
}

func (c *functionCollector) walk(node *sitter.Node, ancestors []*sitter.Node, scope string) {
	nextScope := scope
	if isFunctionNode(node) && functionBody(node) != nil {
		name := functionName(node, ancestors, c.data)
		kind := functionKind(node, name)
		if name == "" {
			key := scope + "::callback"
			c.anonymous[key]++
			name = fmt.Sprintf("<callback#%d>", c.anonymous[key])
		}
		symbol := name
		if scope != "" {
			symbol = scope + "/" + name
		}
		identity := c.file + "::" + symbol
		c.identities[identity]++
		if c.identities[identity] > 1 {
			symbol = fmt.Sprintf("%s#%d", symbol, c.identities[identity])
			identity = c.file + "::" + symbol
		}
		value := CyclomaticForFunction(node, c.data)
		start := int(node.StartPoint().Row) + 1
		end := int(node.EndPoint().Row) + 1
		c.metrics = append(c.metrics, FunctionMetric{
			ID:             identity,
			File:           c.file,
			Symbol:         symbol,
			Kind:           kind,
			Language:       c.language,
			StartLine:      start,
			EndLine:        end,
			Lines:          end - start + 1,
			Complexity:     value,
			DecisionPoints: value - 1,
			OverThreshold:  value > c.threshold,
		})
		nextScope = symbol
	}
	nextAncestors := append(ancestors, node)
	for i := 0; i < int(node.NamedChildCount()); i++ {
		child := node.NamedChild(i)
		c.walk(child, nextAncestors, nextScope)
	}
}

func isFunctionNode(node *sitter.Node) bool {
	switch node.Type() {
	case "function_declaration", "generator_function_declaration", "function_expression",
		"generator_function", "arrow_function", "method_definition":
		return true
	default:
		return false
	}
}

func functionBody(node *sitter.Node) *sitter.Node {
	return node.ChildByFieldName("body")
}

func functionName(node *sitter.Node, ancestors []*sitter.Node, data []byte) string {
	if named := node.ChildByFieldName("name"); named != nil {
		return nodeText(named, data)
	}
	for i := len(ancestors) - 1; i >= 0; i-- {
		parent := ancestors[i]
		if isFunctionNode(parent) {
			break
		}
		switch parent.Type() {
		case "variable_declarator":
			return nodeText(parent.ChildByFieldName("name"), data)
		case "pair", "pair_pattern", "public_field_definition":
			return nodeText(parent.ChildByFieldName("key"), data)
		case "export_statement":
			if strings.HasPrefix(strings.TrimSpace(nodeText(parent, data)), "export default") {
				return "<default-export>"
			}
		}
	}
	return ""
}

func functionKind(node *sitter.Node, name string) string {
	if node.Type() == "method_definition" {
		return "method"
	}
	if (startsUpper(name) || name == "<default-export>") && containsJSX(functionBody(node)) {
		return "react-component"
	}
	if name == "" {
		return "callback"
	}
	return "function"
}

func startsUpper(value string) bool {
	return len(value) > 0 && value[0] >= 'A' && value[0] <= 'Z'
}

func containsJSX(node *sitter.Node) bool {
	if node == nil {
		return false
	}
	found := false
	var walk func(*sitter.Node)
	walk = func(current *sitter.Node) {
		if found {
			return
		}
		if current != node && isFunctionNode(current) {
			return
		}
		if strings.HasPrefix(current.Type(), "jsx_") {
			found = true
			return
		}
		for i := 0; i < int(current.NamedChildCount()); i++ {
			walk(current.NamedChild(i))
		}
	}
	walk(node)
	return found
}

func nodeText(node *sitter.Node, data []byte) string {
	if node == nil {
		return ""
	}
	return strings.TrimSpace(string(data[node.StartByte():node.EndByte()]))
}

func summarize(metrics []FunctionMetric, files int) Summary {
	summary := Summary{Files: files, Functions: len(metrics)}
	for _, metric := range metrics {
		summary.Total += metric.Complexity
		if metric.Complexity > summary.Maximum {
			summary.Maximum = metric.Complexity
		}
		if metric.OverThreshold {
			summary.OverThreshold++
		}
	}
	if summary.Functions > 0 {
		summary.Average = float64(summary.Total) / float64(summary.Functions)
	}
	return summary
}
