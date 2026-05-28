package parser

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/canadian-ai/girl/internal/ir"
	"github.com/canadian-ai/girl/internal/shared"
)

type Parser interface {
	ParseFile(path string) (*ir.FileIR, error)
	ParseDir(dir string, excludeDirs []string) ([]*ir.FileIR, error)
}

type SimpleParser struct {
	ExcludeDirs []string
	ExcludeExts []string
}

func NewSimpleParser() *SimpleParser {
	return &SimpleParser{
		ExcludeExts: []string{".png", ".jpg", ".jpeg", ".gif", ".ico", ".pdf", ".zip", ".lock"},
	}
}

func (p *SimpleParser) isExcludedDir(name string) bool {
	if shared.ShouldSkipDir(name) {
		return true
	}
	for _, d := range p.ExcludeDirs {
		if name == d {
			return true
		}
	}
	return false
}

func (p *SimpleParser) isExcludedExt(path string) bool {
	ext := filepath.Ext(path)
	for _, e := range p.ExcludeExts {
		if ext == e {
			return true
		}
	}
	return false
}

func (p *SimpleParser) ParseDir(dir string, excludeDirs []string) ([]*ir.FileIR, error) {
	if excludeDirs != nil {
		p.ExcludeDirs = append(p.ExcludeDirs, excludeDirs...)
	}

	files, err := p.collectParseFiles(dir)
	if err != nil {
		return nil, err
	}

	var results []*ir.FileIR
	for _, f := range files {
		fir, err := p.ParseFile(f)
		if err != nil {
			continue
		}
		results = append(results, fir)
	}
	return results, nil
}

func (p *SimpleParser) collectParseFiles(dir string) ([]string, error) {
	var files []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if shouldSkipWalkDir(path, dir, info, p) {
			return filepath.SkipDir
		}
		if info.IsDir() {
			return nil
		}
		if p.shouldParseFile(path) {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

func shouldSkipWalkDir(path, root string, info os.FileInfo, p *SimpleParser) bool {
	return info.IsDir() && path != root && p.isExcludedDir(info.Name())
}

func (p *SimpleParser) shouldParseFile(path string) bool {
	switch filepath.Ext(path) {
	case ".tsx", ".ts", ".jsx", ".js":
		return !p.isExcludedExt(path)
	}
	return false
}

func (p *SimpleParser) ParseFile(path string) (*ir.FileIR, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	content := string(data)
	lines := strings.Split(content, "\n")

	fir := &ir.FileIR{
		Path:       path,
		Language:   detectLanguage(path),
		Lines:      len(lines),
		Components: []ir.ComponentIR{},
		Hooks:      []ir.HookIR{},
		Imports:    []ir.ImportIR{},
	}

	fir.Imports = parseImports(content)
	fir.Components = parseComponents(path, content, lines)
	fir.Hooks = parseTopLevelHooks(content, lines)

	return fir, nil
}

func detectLanguage(path string) string {
	ext := filepath.Ext(path)
	switch ext {
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

var importRe = regexp.MustCompile(`import\s+(?:(?:\{[^}]*\})\s+)?(?:(\w+)\s+)?(?:from\s+)?["']([^"']+)["']`)

func parseImports(content string) []ir.ImportIR {
	imports := []ir.ImportIR{}
	matches := importRe.FindAllStringSubmatch(content, -1)
	seen := map[string]bool{}
	for _, m := range matches {
		source := m[2]
		if seen[source] {
			continue
		}
		seen[source] = true
		imports = append(imports, ir.ImportIR{
			Source:  source,
			Default: m[1],
		})
	}
	return imports
}

var componentRe = regexp.MustCompile(`(?:export\s+)?(?:default\s+)?(?:function\s+(\w+)|const\s+(\w+)\s*[=:]\s*(?:React\.)?memo\s*(?:<[^>]*>)?\s*\(?\s*(?:\()?(?:props|[\w\s,{}:;]+)\)?\s*=>|const\s+(\w+)\s*(?:[=:])\s*(?:\()?(?:props|[\w\s,{}:;]+)\)?\s*=>)`)

func isCapitalized(s string) bool {
	if len(s) == 0 {
		return false
	}
	return s[0] >= 'A' && s[0] <= 'Z'
}

func parseComponents(path, content string, lines []string) []ir.ComponentIR {
	components := []ir.ComponentIR{}
	matches := componentRe.FindAllStringSubmatchIndex(content, -1)

	for _, match := range matches {
		name := ""
		if match[2] >= 0 {
			name = content[match[2]:match[3]]
		} else if match[4] >= 0 {
			name = content[match[4]:match[5]]
		} else if match[6] >= 0 {
			name = content[match[6]:match[7]]
		}
		if name == "" || !isCapitalized(name) {
			continue
		}

		startLine := countLines(content[:match[0]])
		comp := ir.ComponentIR{
			Name:      name,
			FilePath:  path,
			Kind:      ir.ComponentKindFunction,
			StartLine: startLine,
			EndLine:   startLine,
			Lines:     0,
			Hooks:     []ir.HookIR{},
			JSXBlocks: []ir.JSXBlockIR{},
			Props:     []ir.PropIR{},
		}

		endLine := findComponentEnd(content, match[1], lines)
		comp.EndLine = endLine
		comp.Lines = endLine - startLine + 1

		parseComponentBody(&comp, content, lines, startLine, endLine)

		components = append(components, comp)
	}

	return components
}

func countLines(s string) int {
	if s == "" {
		return 1
	}
	return strings.Count(s, "\n") + 1
}

func findComponentEnd(content string, startIdx int, lines []string) int {
	depth := 0
	inJSX := false
	foundBrace := false
	lineNum := countLines(content[:startIdx])

	for i := startIdx; i < len(content); i++ {
		ch := content[i]
		if ch == '\n' {
			lineNum++
			continue
		}
		if inJSX {
			inJSX = !endsJSXTag(ch)
			continue
		}
		if startsJSXTag(content, i) {
			inJSX = true
			continue
		}
		if ch == '{' {
			depth++
			foundBrace = true
			continue
		}
		if ch == '}' {
			depth--
			if foundBrace && depth == 0 {
				return lineNum
			}
		}
	}
	return lineNum
}

func startsJSXTag(content string, pos int) bool {
	if content[pos] != '<' || pos+1 >= len(content) {
		return false
	}
	next := content[pos+1]
	return isLetter(next) || next == '/' || next == '>'
}

func endsJSXTag(ch byte) bool {
	return ch == '>'
}

func isLetter(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z')
}

func parseComponentBody(comp *ir.ComponentIR, content string, lines []string, startLine, endLine int) {
	body := getLineRange(lines, startLine, endLine)
	bodyStr := strings.Join(body, "\n")

	comp.Hooks = parseHooksInRange(bodyStr, startLine)
	comp.Props = parsePropsInRange(bodyStr, startLine)
	comp.JSXBlocks = parseJSXBlocks(bodyStr, startLine)
	comp.StateVars = parseStateVarsInRange(bodyStr, startLine)
	comp.Effects = parseEffectsInRange(bodyStr, startLine)
	comp.EventHandlers = parseEventHandlersInRange(bodyStr, startLine)
	comp.HasKeyDown = hasKeyDown(bodyStr)
	comp.HasAnalytics = hasAnalytics(bodyStr)
	comp.ConditionalCount = countConditionals(bodyStr)
	comp.LoopCount = countLoops(bodyStr)
}

func getLineRange(lines []string, start, end int) []string {
	if start < 1 {
		start = 1
	}
	if end > len(lines) {
		end = len(lines)
	}
	if start > end {
		return []string{}
	}
	return lines[start-1 : end]
}

var hookRe = regexp.MustCompile(`(use\w+)\s*\(`)

func parseHooksInRange(content string, offset int) []ir.HookIR {
	hooks := []ir.HookIR{}
	matches := hookRe.FindAllStringSubmatchIndex(content, -1)
	knownHooks := map[string]bool{
		"useState": true, "useEffect": true, "useCallback": true,
		"useMemo": true, "useRef": true, "useContext": true,
		"useReducer": true, "useLayoutEffect": true, "useImperativeHandle": true,
		"useForm": true, "useController": true, "useFieldArray": true,
		"useWatch": true, "useNavigate": true, "useParams": true,
		"useSearchParams": true, "useRouter": true, "useDisclosure": true,
		"useQuery": true, "useMutation": true, "useKeyDown": true,
		"useToast": true, "useTheme": true, "useAuth": true, "useUser": true,
	}
	for _, m := range matches {
		if m[2] < 0 {
			continue
		}
		name := content[m[2]:m[3]]
		if knownHooks[name] || strings.HasPrefix(name, "use") {
			line := countLines(content[:m[0]]) + offset - 1
			args := extractArgs(content, m[3])
			hooks = append(hooks, ir.HookIR{
				Name: name,
				Line: line,
				Args: args,
			})
		}
	}
	return hooks
}

func extractArgs(content string, fromIdx int) []string {
	depth := 1
	i := fromIdx
	for i < len(content) && content[i] != '(' {
		i++
	}
	if i >= len(content) {
		return nil
	}
	i++
	start := i
	for i < len(content) && depth > 0 {
		switch content[i] {
		case '(':
			depth++
		case ')':
			depth--
		}
		i++
	}
	argStr := content[start : i-1]
	if argStr == "" {
		return nil
	}
	return []string{strings.TrimSpace(argStr)}
}

var propRe = regexp.MustCompile(`(\w+)\s*(?:[?:]\s*(\w+(?:<[^>]*>)?))\s*(?:[,)}])`)

func parsePropsInRange(content string, offset int) []ir.PropIR {
	props := []ir.PropIR{}
	matches := propRe.FindAllStringSubmatch(content, -1)
	seen := map[string]bool{}
	for _, m := range matches {
		if seen[m[1]] {
			continue
		}
		seen[m[1]] = true
		props = append(props, ir.PropIR{
			Name: m[1],
			Type: m[2],
		})
	}
	return props
}

var jsxBlockRe = regexp.MustCompile(`<(\w+)`)

func parseJSXBlocks(content string, offset int) []ir.JSXBlockIR {
	blocks := []ir.JSXBlockIR{}
	matches := jsxBlockRe.FindAllStringSubmatchIndex(content, -1)
	for _, m := range matches {
		if m[2] < 0 {
			continue
		}
		elem := content[m[2]:m[3]]
		if isComponentName(elem) {
			line := countLines(content[:m[0]]) + offset - 1
			blocks = append(blocks, ir.JSXBlockIR{
				Element: elem,
				Line:    line,
			})
		}
	}
	return blocks
}

func isComponentName(name string) bool {
	if len(name) == 0 {
		return false
	}
	lower := strings.ToLower(name)
	if name == lower {
		return false
	}
	return true
}

var stateRe = regexp.MustCompile(`(?:const\s+)?\[(\w+),\s*(\w+)\]\s*=\s*useState`)

func parseStateVarsInRange(content string, offset int) []ir.StateVarIR {
	vars := []ir.StateVarIR{}
	matches := stateRe.FindAllStringSubmatch(content, -1)
	for _, m := range matches {
		vars = append(vars, ir.StateVarIR{
			Name:       m[1],
			HasUpdater: m[2] != "",
		})
	}
	return vars
}

var effectRe = regexp.MustCompile(`useEffect\s*\(\s*(?:\(\)\s*=>\s*\{|function\s*\(\))`)

func parseEffectsInRange(content string, offset int) []ir.EffectIR {
	effects := []ir.EffectIR{}
	matches := effectRe.FindAllStringSubmatchIndex(content, -1)
	for _, m := range matches {
		if m[0] < 0 {
			continue
		}
		effects = append(effects, ir.EffectIR{
			Name: "useEffect",
			Line: countLines(content[:m[0]]) + offset - 1,
		})
	}
	return effects
}

var handlerRe = regexp.MustCompile(`(?:const\s+)?(\w+)\s*(?:[=:])\s*(?:\([^)]*\)\s*=>|async\s*\([^)]*\)\s*=>|function\s*\([^)]*\))`)

func parseEventHandlersInRange(content string, offset int) []ir.EventHandlerIR {
	handlers := []ir.EventHandlerIR{}
	matches := handlerRe.FindAllStringSubmatch(content, -1)
	for _, m := range matches {
		name := m[1]
		if name == "" {
			continue
		}
		if strings.HasPrefix(name, "handle") || strings.HasPrefix(name, "on") ||
			strings.HasSuffix(name, "Handler") || strings.HasPrefix(name, "set") {
			handlers = append(handlers, ir.EventHandlerIR{
				Name: name,
			})
		}
	}
	return handlers
}

var keyDownRe = regexp.MustCompile(`(?i)(onKeyDown|useKeyDown|keyboard|keydown)`)

func hasKeyDown(content string) bool {
	return keyDownRe.MatchString(content)
}

var analyticsRe = regexp.MustCompile(`(?i)(analytics|track|gtag|posthog|amplitude|mixpanel)`)

func hasAnalytics(content string) bool {
	return analyticsRe.MatchString(content)
}

func countConditionals(content string) int {
	ifRe := regexp.MustCompile(`(?m)^\s*if\s*\(`)
	ternaryRe := regexp.MustCompile(`\?\s*[^:]+`)
	andRe := regexp.MustCompile(`\{\s*\w+\s*&&\s*`)
	count := 0
	count += len(ifRe.FindAllString(content, -1))
	count += len(ternaryRe.FindAllString(content, -1))
	count += len(andRe.FindAllString(content, -1))
	return count
}

func countLoops(content string) int {
	loops := regexp.MustCompile(`(?i)(for\s*\(|while\s*\(|\.map\(|\.forEach\(|\.filter\(|\.reduce\()`)
	return len(loops.FindAllString(content, -1))
}

func parseTopLevelHooks(content string, lines []string) []ir.HookIR {
	return parseHooksInRange(content, 1)
}
