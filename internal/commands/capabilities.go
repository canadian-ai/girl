package commands

import (
	"fmt"
	"sort"
	"strings"

	"github.com/urfave/cli/v2"
)

// CapabilitiesSchema identifies the `girl capabilities` JSON contract.
const CapabilitiesSchema = "girl.capabilities/v1"

// Capabilities describes the CLI for agents that discover tools at runtime.
// It is generated from the live command tree, so it cannot drift from --help.
type Capabilities struct {
	Schema    string              `json:"schema"`
	Name      string              `json:"name"`
	Version   string              `json:"version"`
	Summary   string              `json:"summary"`
	AgentLoop []AgentLoopStep     `json:"agentLoop"`
	Languages []string            `json:"languages"`
	Privacy   string              `json:"privacy"`
	Env       map[string]string   `json:"env"`
	Commands  []CommandCapability `json:"commands"`
}

type AgentLoopStep struct {
	Step    string `json:"step"`
	Command string `json:"command"`
	Why     string `json:"why"`
}

type CommandCapability struct {
	Name        string              `json:"name"`
	Usage       string              `json:"usage"`
	Args        string              `json:"args,omitempty"`
	Flags       []FlagCapability    `json:"flags,omitempty"`
	Subcommands []CommandCapability `json:"subcommands,omitempty"`
}

type FlagCapability struct {
	Name    string   `json:"name"`
	Aliases []string `json:"aliases,omitempty"`
	Usage   string   `json:"usage,omitempty"`
	Default string   `json:"default,omitempty"`
	Value   bool     `json:"takesValue"`
}

// DefaultAgentLoop is the recommended agent workflow, shared by
// `girl capabilities` and the generated agent instructions.
var DefaultAgentLoop = []AgentLoopStep{
	{Step: "orient", Command: "girl next -o json", Why: "Top-ranked fixes with excerpts, recipe, and verify commands; bounded output"},
	{Step: "plan", Command: "girl plan <path> --output grp-json", Why: "Ordered, risk-scored GRP steps when the change spans many locations"},
	{Step: "context", Command: "girl pack <path> --budget 8000 --output markdown", Why: "Token-budgeted context for a sub-agent or a fresh session"},
	{Step: "change", Command: "(edit code)", Why: "Fix one item; keep the diff small"},
	{Step: "check", Command: "girl check --changed -o json", Why: "Analysis, complexity ratchet, reviewability, and repo verification on what changed"},
	{Step: "repeat", Command: "girl next --changed -o json", Why: "Confirm the fix and pick up anything the change introduced"},
}

func CapabilitiesCommand() *cli.Command {
	return &cli.Command{
		Name:    "capabilities",
		Aliases: []string{"caps"},
		Usage:   "Describe every command and flag as JSON for agent tool discovery",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "output", Aliases: []string{"o"}, Value: "json", Usage: "Output format: json or text"},
		},
		Action: func(c *cli.Context) error {
			caps := BuildCapabilities(c.App)
			if stringFlag(c, "output", "o") == "text" {
				fmt.Print(renderCapabilitiesText(caps))
				return nil
			}
			printJSON(caps)
			return nil
		},
	}
}

func BuildCapabilities(app *cli.App) Capabilities {
	caps := Capabilities{
		Schema:    CapabilitiesSchema,
		Name:      app.Name,
		Version:   app.Version,
		Summary:   app.Description,
		AgentLoop: DefaultAgentLoop,
		Languages: []string{"go", "typescript", "tsx", "javascript", "jsx", "rust"},
		Privacy:   "local-only: source never leaves the machine",
		Env: map[string]string{
			"GIRL_JOBS": "Number of parallel analysis workers (default: CPU count)",
		},
	}
	for _, cmd := range app.Commands {
		if cmd.Hidden || cmd.Name == "help" {
			continue
		}
		caps.Commands = append(caps.Commands, describeCommand(cmd))
	}
	return caps
}

func describeCommand(cmd *cli.Command) CommandCapability {
	out := CommandCapability{Name: cmd.Name, Usage: cmd.Usage, Args: cmd.ArgsUsage}
	for _, f := range cmd.Flags {
		out.Flags = append(out.Flags, describeFlag(f))
	}
	sort.Slice(out.Flags, func(i, j int) bool { return out.Flags[i].Name < out.Flags[j].Name })
	for _, sub := range cmd.Subcommands {
		if sub.Hidden || sub.Name == "help" {
			continue
		}
		out.Subcommands = append(out.Subcommands, describeCommand(sub))
	}
	return out
}

func describeFlag(f cli.Flag) FlagCapability {
	names := f.Names()
	fc := FlagCapability{Name: names[0]}
	if len(names) > 1 {
		fc.Aliases = names[1:]
	}
	if doc, ok := f.(cli.DocGenerationFlag); ok {
		fc.Usage = doc.GetUsage()
		fc.Value = doc.TakesValue()
		if fc.Value {
			fc.Default = doc.GetValue()
		}
	}
	return fc
}

func renderCapabilitiesText(caps Capabilities) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s %s — %s\n\nAgent loop:\n", caps.Name, caps.Version, caps.Summary)
	for _, s := range caps.AgentLoop {
		fmt.Fprintf(&b, "  %-8s %s\n", s.Step, s.Command)
	}
	b.WriteString("\nCommands:\n")
	for _, c := range caps.Commands {
		fmt.Fprintf(&b, "  %-14s %s\n", c.Name, c.Usage)
	}
	return b.String()
}
