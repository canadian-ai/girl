package commands

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/canadian-ai/girl/internal/verification"
)

const girlProjectConfigVersion = "1"

type GirlProjectConfig struct {
	Version       string
	Profiles      []string
	Workspaces    []string
	Analysis      GirlAnalysisConfig
	Complexity    GirlComplexityConfig
	Reviewability GirlReviewabilityConfig
	Verify        map[string][]string
	Agents        map[string]bool
	Decisions     GirlDecisionsConfig

	// VerifySource is "config" when verification commands came from
	// .girl/config.yaml and "detected" when inferred from repo-native files.
	VerifySource string
	// VerifyDiscoveryError is non-empty when repo-native detection failed.
	// Detection failure is evidence of nothing; it must not look like
	// "no commands needed".
	VerifyDiscoveryError string
	// VerifyProvenance maps a detected command to where it was found.
	VerifyProvenance map[string]verification.Command
	// Problems lists invalid configuration values. Strict mode rejects them.
	Problems []string

	sourcePath string
}

type GirlAnalysisConfig struct {
	ChangedOnly bool
	Exclude     []string
}

type GirlComplexityConfig struct {
	Max      int
	Baseline string
	FailOn   string
}

// GirlDecisionsConfig configures the opt-in decision-model extension. Zero
// values mean "use the default"; Mode defaults to shadow.
type GirlDecisionsConfig struct {
	Mode                     string // off, shadow (default) or enforce
	Provider                 string
	Model                    string
	MaxFindings              int
	Consistency              *bool
	MinRecipeConfidence      float64
	ReviewProbability        float64
	HighRiskProbability      float64
	FalsePositiveProbability float64
}

type GirlReviewabilityConfig struct {
	MaxDiffLines    int
	MaxTouchedFiles int
	MaxRisk         string
}

type projectPackageJSON struct {
	Dependencies    map[string]string `json:"dependencies"`
	DevDependencies map[string]string `json:"devDependencies"`
	Scripts         map[string]string `json:"scripts"`
	Workspaces      json.RawMessage   `json:"workspaces"`
}

func defaultGirlProjectConfig(path string) (*GirlProjectConfig, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}

	cfg := &GirlProjectConfig{
		Version:    girlProjectConfigVersion,
		Profiles:   []string{},
		Workspaces: []string{"."},
		Analysis: GirlAnalysisConfig{
			ChangedOnly: true,
			Exclude:     defaultAnalysisExclude(),
		},
		Complexity: GirlComplexityConfig{
			Max:      10,
			Baseline: ".girl/complexity-baseline.json",
			FailOn:   "regression",
		},
		Reviewability: GirlReviewabilityConfig{
			MaxDiffLines:    1500,
			MaxTouchedFiles: 12,
			MaxRisk:         "medium",
		},
		Verify: map[string][]string{},
		Agents: map[string]bool{},
	}

	pkg, _ := readProjectPackageJSON(abs)
	if pkg != nil {
		cfg.Workspaces = packageWorkspaces(pkg)
		if len(cfg.Workspaces) == 0 {
			cfg.Workspaces = []string{"."}
		}
	}

	profiles := map[string]bool{}
	if dirExists(filepath.Join(abs, ".cai")) || fileContains(filepath.Join(abs, "AGENTS.md"), "Canadian AI") {
		profiles["cai"] = true
	}
	if fileExistsAny(abs, "next.config.js", "next.config.mjs", "next.config.ts") || packageHas(pkg, "next") {
		profiles["next"] = true
	}
	if dirExists(filepath.Join(abs, "convex")) || packageHas(pkg, "convex") {
		profiles["convex"] = true
	}
	if packageHas(pkg, "@clerk/nextjs") || packageHas(pkg, "@clerk/clerk-react") {
		profiles["clerk"] = true
	}
	if fileExistsAny(abs, "vercel.json") || dirExists(filepath.Join(abs, ".vercel")) {
		profiles["vercel"] = true
	}
	if fileExistsAny(abs, "wrangler.toml", "wrangler.json", "wrangler.jsonc") {
		profiles["cloudflare"] = true
	}
	if fileExistsAny(abs, "railway.toml", "railway.json") {
		profiles["railway"] = true
	}
	if len(cfg.Workspaces) > 1 || (len(cfg.Workspaces) == 1 && cfg.Workspaces[0] != ".") {
		profiles["monorepo"] = true
	}

	for p := range profiles {
		cfg.Profiles = append(cfg.Profiles, p)
	}
	sort.Strings(cfg.Profiles)

	cfg.Agents["opencode"] = dirExists(filepath.Join(abs, ".opencode"))
	cfg.Agents["codex"] = dirExists(filepath.Join(abs, ".codex"))
	cfg.Agents["claude"] = dirExists(filepath.Join(abs, ".claude"))

	cfg.VerifySource = "detected"
	cfg.VerifyProvenance = map[string]verification.Command{}
	if detected, err := verification.Detect(abs); err == nil {
		for _, cmd := range detected.Commands {
			role := commandRole(cmd)
			cfg.Verify[role] = appendUnique(cfg.Verify[role], cmd.Command)
			cfg.VerifyProvenance[cmd.Command] = cmd
		}
	} else {
		cfg.VerifyDiscoveryError = err.Error()
	}

	return cfg, nil
}

func commandRole(cmd verification.Command) string {
	if cmd.Type != "" {
		return cmd.Type
	}
	name := strings.ToLower(cmd.Name)
	switch {
	case strings.Contains(name, "type"):
		return "typecheck"
	case strings.Contains(name, "lint"):
		return "lint"
	case strings.Contains(name, "test"):
		return "test"
	case strings.Contains(name, "build"):
		return "build"
	default:
		return "other"
	}
}

func renderGirlProjectConfig(cfg *GirlProjectConfig) string {
	var b strings.Builder
	fmt.Fprintf(&b, "version: %q\n", cfg.Version)
	b.WriteString("profiles:\n")
	for _, p := range cfg.Profiles {
		fmt.Fprintf(&b, "  - %s\n", p)
	}
	b.WriteString("workspaces:\n")
	for _, ws := range cfg.Workspaces {
		fmt.Fprintf(&b, "  - %s\n", ws)
	}
	b.WriteString("analysis:\n")
	fmt.Fprintf(&b, "  changed_only: %t\n", cfg.Analysis.ChangedOnly)
	b.WriteString("  exclude:\n")
	for _, ex := range cfg.Analysis.Exclude {
		fmt.Fprintf(&b, "    - %s\n", ex)
	}
	b.WriteString("complexity:\n")
	fmt.Fprintf(&b, "  max: %d\n", cfg.Complexity.Max)
	fmt.Fprintf(&b, "  baseline: %s\n", cfg.Complexity.Baseline)
	fmt.Fprintf(&b, "  fail_on: %s\n", cfg.Complexity.FailOn)
	b.WriteString("reviewability:\n")
	fmt.Fprintf(&b, "  max_diff_lines: %d\n", cfg.Reviewability.MaxDiffLines)
	fmt.Fprintf(&b, "  max_touched_files: %d\n", cfg.Reviewability.MaxTouchedFiles)
	fmt.Fprintf(&b, "  max_risk: %s\n", cfg.Reviewability.MaxRisk)
	b.WriteString("verify:\n")
	keys := sortedStringKeys(cfg.Verify)
	for _, role := range keys {
		fmt.Fprintf(&b, "  %s:\n", role)
		for _, cmd := range cfg.Verify[role] {
			fmt.Fprintf(&b, "    - %s\n", cmd)
		}
	}
	b.WriteString("agents:\n")
	for _, name := range []string{"opencode", "codex", "claude"} {
		fmt.Fprintf(&b, "  %s: %t\n", name, cfg.Agents[name])
	}
	// Decision models are opt-in (`girl decide`). Shadow records answers
	// without letting them change anything.
	mode := cfg.Decisions.Mode
	if mode == "" {
		mode = "shadow"
	}
	fmt.Fprintf(&b, "decisions:\n  mode: %s\n", mode)
	return b.String()
}

func loadGirlProjectConfig(path string) (*GirlProjectConfig, error) {
	configPath := filepath.Join(path, ".girl", "config.yaml")
	data, err := os.ReadFile(configPath)
	if os.IsNotExist(err) {
		return defaultGirlProjectConfig(path)
	}
	if err != nil {
		return nil, err
	}

	cfg := &GirlProjectConfig{
		Version:      girlProjectConfigVersion,
		Verify:       map[string][]string{},
		Agents:       map[string]bool{},
		VerifySource: "config",
		sourcePath:   configPath,
	}
	excludeSeen := false
	problem := func(format string, args ...any) {
		cfg.Problems = append(cfg.Problems, fmt.Sprintf(format, args...))
	}
	parseInt := func(key, v string) int {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			problem("%s: expected a non-negative integer, got %q", key, v)
			return 0
		}
		return n
	}
	parseBool := func(key, v string) bool {
		b, err := strconv.ParseBool(v)
		if err != nil {
			problem("%s: expected true or false, got %q", key, v)
		}
		return b
	}
	section := ""
	subsection := ""
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		raw := scanner.Text()
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		indent := len(raw) - len(strings.TrimLeft(raw, " "))
		if indent == 0 {
			subsection = ""
			if strings.HasSuffix(trimmed, ":") {
				section = strings.TrimSuffix(trimmed, ":")
				continue
			}
			if k, v, ok := splitYAMLKV(trimmed); ok && k == "version" {
				cfg.Version = trimYAMLValue(v)
			}
			continue
		}

		if indent == 2 && strings.HasSuffix(trimmed, ":") {
			subsection = strings.TrimSuffix(trimmed, ":")
			continue
		}
		if strings.HasPrefix(trimmed, "- ") {
			v := trimYAMLValue(strings.TrimSpace(strings.TrimPrefix(trimmed, "- ")))
			switch section {
			case "profiles":
				cfg.Profiles = append(cfg.Profiles, v)
			case "workspaces":
				cfg.Workspaces = append(cfg.Workspaces, v)
			case "analysis":
				if subsection == "exclude" {
					excludeSeen = true
					cfg.Analysis.Exclude = append(cfg.Analysis.Exclude, v)
				}
			case "verify":
				if subsection != "" {
					cfg.Verify[subsection] = append(cfg.Verify[subsection], v)
				}
			}
			continue
		}

		k, v, ok := splitYAMLKV(trimmed)
		if !ok {
			continue
		}
		v = trimYAMLValue(v)
		switch section {
		case "analysis":
			switch k {
			case "changed_only":
				cfg.Analysis.ChangedOnly = parseBool("analysis.changed_only", v)
			case "exclude":
				excludeSeen = true
				if v == "[]" {
					cfg.Analysis.Exclude = []string{}
				}
			}
		case "complexity":
			switch k {
			case "max":
				cfg.Complexity.Max = parseInt("complexity.max", v)
			case "baseline":
				cfg.Complexity.Baseline = v
			case "fail_on":
				switch strings.ToLower(v) {
				case "regression", "increase", "threshold", "never", "none":
				default:
					problem("complexity.fail_on: unknown policy %q", v)
				}
				cfg.Complexity.FailOn = v
			}
		case "reviewability":
			switch k {
			case "max_diff_lines":
				cfg.Reviewability.MaxDiffLines = parseInt("reviewability.max_diff_lines", v)
			case "max_touched_files":
				cfg.Reviewability.MaxTouchedFiles = parseInt("reviewability.max_touched_files", v)
			case "max_risk":
				cfg.Reviewability.MaxRisk = v
			}
		case "agents":
			cfg.Agents[k] = parseBool("agents."+k, v)
		case "decisions":
			parseProb := func(key string) float64 {
				f, err := strconv.ParseFloat(v, 64)
				if err != nil || f < 0 || f > 1 {
					problem("decisions.%s: expected a probability between 0 and 1, got %q", key, v)
					return 0
				}
				return f
			}
			switch k {
			case "mode":
				switch strings.ToLower(v) {
				case "off", "shadow", "enforce":
					cfg.Decisions.Mode = strings.ToLower(v)
				default:
					problem("decisions.mode: expected off, shadow or enforce, got %q", v)
				}
			case "provider":
				cfg.Decisions.Provider = v
			case "model":
				cfg.Decisions.Model = v
			case "max_findings":
				cfg.Decisions.MaxFindings = parseInt("decisions.max_findings", v)
			case "consistency":
				b := parseBool("decisions.consistency", v)
				cfg.Decisions.Consistency = &b
			case "min_recipe_confidence":
				cfg.Decisions.MinRecipeConfidence = parseProb(k)
			case "review_probability":
				cfg.Decisions.ReviewProbability = parseProb(k)
			case "high_risk_probability":
				cfg.Decisions.HighRiskProbability = parseProb(k)
			case "false_positive_probability":
				cfg.Decisions.FalsePositiveProbability = parseProb(k)
			default:
				problem("decisions.%s: unknown setting", k)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if len(cfg.Workspaces) == 0 {
		cfg.Workspaces = []string{"."}
	}
	if !excludeSeen {
		// A config without an exclude list keeps GIRL's default exclusions
		// rather than silently scanning dependency and generated trees.
		cfg.Analysis.Exclude = defaultAnalysisExclude()
	}
	for _, pat := range cfg.Analysis.Exclude {
		if _, err := newScopePolicy(path, []string{pat}); err != nil {
			problem("analysis.exclude: %v", err)
		}
	}
	if cfg.Complexity.Max == 0 {
		cfg.Complexity.Max = 10
	}
	if cfg.Complexity.FailOn == "" {
		cfg.Complexity.FailOn = "regression"
	}
	if cfg.Complexity.Baseline == "" {
		cfg.Complexity.Baseline = ".girl/complexity-baseline.json"
	}
	if cfg.Reviewability.MaxDiffLines == 0 {
		cfg.Reviewability.MaxDiffLines = 1500
	}
	if cfg.Reviewability.MaxTouchedFiles == 0 {
		cfg.Reviewability.MaxTouchedFiles = 12
	}
	if cfg.Reviewability.MaxRisk == "" {
		cfg.Reviewability.MaxRisk = "medium"
	}
	return cfg, nil
}

func defaultAnalysisExclude() []string {
	return []string{".next", "node_modules", "dist", "build", "coverage", "generated"}
}

func splitYAMLKV(line string) (string, string, bool) {
	idx := strings.Index(line, ":")
	if idx < 0 {
		return "", "", false
	}
	return strings.TrimSpace(line[:idx]), strings.TrimSpace(line[idx+1:]), true
}

func trimYAMLValue(v string) string {
	return strings.Trim(strings.TrimSpace(v), "\"'")
}

func readProjectPackageJSON(path string) (*projectPackageJSON, error) {
	data, err := os.ReadFile(filepath.Join(path, "package.json"))
	if err != nil {
		return nil, err
	}
	var pkg projectPackageJSON
	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil, err
	}
	return &pkg, nil
}

func packageWorkspaces(pkg *projectPackageJSON) []string {
	if pkg == nil || len(pkg.Workspaces) == 0 {
		return nil
	}
	var list []string
	if json.Unmarshal(pkg.Workspaces, &list) == nil {
		return list
	}
	var obj struct {
		Packages []string `json:"packages"`
	}
	if json.Unmarshal(pkg.Workspaces, &obj) == nil {
		return obj.Packages
	}
	return nil
}

func packageHas(pkg *projectPackageJSON, name string) bool {
	if pkg == nil {
		return false
	}
	_, dep := pkg.Dependencies[name]
	_, dev := pkg.DevDependencies[name]
	return dep || dev
}

func fileExistsAny(path string, names ...string) bool {
	for _, name := range names {
		if checkFileExists(filepath.Join(path, name)) {
			return true
		}
	}
	return false
}

func fileContains(path, needle string) bool {
	data, err := os.ReadFile(path)
	return err == nil && strings.Contains(string(data), needle)
}

func appendUnique(items []string, item string) []string {
	for _, existing := range items {
		if existing == item {
			return items
		}
	}
	return append(items, item)
}

func sortedStringKeys(m map[string][]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
