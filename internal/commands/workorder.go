package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/canadian-ai/girl/internal/ir"
	"github.com/urfave/cli/v2"
)

const workOrderSpecVersion = "girl.io/workorder/v1"
const workOrderType = "workorder"
const workOrderSetType = "workorder-set"

type WorkOrder struct {
	SpecVersion    string      `json:"specversion"`
	ID             string      `json:"id"`
	Type           string      `json:"type"`
	Goal           string      `json:"goal"`
	Risk           string      `json:"risk"`
	AllowedFiles   []string    `json:"allowedFiles,omitempty"`
	ForbiddenFiles []string    `json:"forbiddenFiles,omitempty"`
	MaxDiffLines   int         `json:"maxDiffLines,omitempty"`
	Parallelizable bool        `json:"parallelizable"`
	DependsOn      []string    `json:"dependsOn,omitempty"`
	Verification   []string    `json:"verification,omitempty"`
	SourcePlanID   string      `json:"sourcePlanId,omitempty"`
	Steps          []ir.GrpStep `json:"steps,omitempty"`
	CreatedAt      string      `json:"createdAt"`
}

type WorkOrderSet struct {
	SpecVersion string      `json:"specversion"`
	ID          string      `json:"id"`
	Type        string      `json:"type"`
	Tasks       []WorkOrder `json:"tasks"`
	CreatedAt   string      `json:"createdAt"`
}

func WorkOrderCommand() *cli.Command {
	return &cli.Command{
		Name:      "workorder",
		Usage:     "Generate agent-ready task work orders from a plan or decomposition",
		ArgsUsage: "",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "plan",
				Aliases: []string{"p"},
				Usage:   "Path to GRP plan JSON file",
			},
			&cli.StringFlag{
				Name:    "decomposition",
				Aliases: []string{"d"},
				Usage:   "Path to decomposition JSON file",
			},
			&cli.StringFlag{
				Name:    "task",
				Aliases: []string{"t"},
				Usage:   "Specific task ID to generate workorder for",
			},
			&cli.StringFlag{
				Name:    "output",
				Aliases: []string{"o"},
				Usage:   "Output format: json (default), markdown",
				Value:   "json",
			},
			&cli.StringFlag{
				Name:    "output-file",
				Aliases: []string{"f"},
				Usage:   "Write workorder to file (e.g., .grp/workorder.json)",
			},
		},
		Action: func(c *cli.Context) error {
			planPath := c.String("plan")
			decompPath := c.String("decomposition")
			taskID := c.String("task")
			outputFmt := stringFlag(c, "output", "o")
			outputFile := c.String("output-file")

			if planPath == "" && decompPath == "" {
				return fmt.Errorf("either --plan or --decomposition must be provided")
			}
			if planPath != "" && decompPath != "" {
				return fmt.Errorf("only one of --plan or --decomposition may be provided")
			}

			var (
				decomp *ir.Decomposition
				plan   *ir.GrpPlan
				setID  string
			)

			if planPath != "" {
				plan = &ir.GrpPlan{}
				data, err := os.ReadFile(planPath)
				if err != nil {
					return fmt.Errorf("read plan file: %w", err)
				}
				if err := json.Unmarshal(data, plan); err != nil {
					return fmt.Errorf("parse plan file: %w", err)
				}
				if plan.Decomposition == nil {
					return fmt.Errorf("plan has no decomposition; run 'girl decompose' first or use --decomposition directly")
				}
				decomp = plan.Decomposition
				setID = plan.PlanID
			} else {
				decomp = &ir.Decomposition{}
				data, err := os.ReadFile(decompPath)
				if err != nil {
					return fmt.Errorf("read decomposition file: %w", err)
				}
				if err := json.Unmarshal(data, decomp); err != nil {
					return fmt.Errorf("parse decomposition file: %w", err)
				}
				setID = decomp.ParentPlan
			}

			allWorkOrders := generateWorkOrders(decomp, plan)

			if taskID != "" {
				filtered := make([]WorkOrder, 0)
				for _, wo := range allWorkOrders {
					if wo.ID == taskID {
						filtered = append(filtered, wo)
					}
				}
				if len(filtered) == 0 {
					return fmt.Errorf("task %q not found in %s", taskID, map[bool]string{true: "plan", false: "decomposition"}[planPath != ""])
				}
				allWorkOrders = filtered
			}

			if outputFile != "" {
				set := WorkOrderSet{
					SpecVersion: workOrderSpecVersion,
					ID:          setID,
					Type:        workOrderSetType,
					Tasks:       allWorkOrders,
					CreatedAt:   nowISO(),
				}
				if err := writeJSONFile(outputFile, set); err != nil {
					return fmt.Errorf("write workorder file: %w", err)
				}
			}

			switch outputFmt {
			case "markdown":
				if len(allWorkOrders) == 1 {
					printWorkOrderMarkdown(allWorkOrders[0])
				} else {
					set := WorkOrderSet{
						SpecVersion: workOrderSpecVersion,
						ID:          setID,
						Type:        workOrderSetType,
						Tasks:       allWorkOrders,
						CreatedAt:   nowISO(),
					}
					printWorkOrderSetMarkdown(set)
				}
			default:
				if len(allWorkOrders) == 1 {
					printJSON(allWorkOrders[0])
				} else {
					set := WorkOrderSet{
						SpecVersion: workOrderSpecVersion,
						ID:          setID,
						Type:        workOrderSetType,
						Tasks:       allWorkOrders,
						CreatedAt:   nowISO(),
					}
					printJSON(set)
				}
			}

			return nil
		},
	}
}

func generateWorkOrders(decomp *ir.Decomposition, plan *ir.GrpPlan) []WorkOrder {
	now := nowISO()
	var sourcePlanID string
	var risk string

	if plan != nil {
		sourcePlanID = plan.PlanID
		risk = strings.ToUpper(string(plan.Risk))
	} else {
		risk = "MEDIUM"
	}

	if risk == "" {
		risk = "MEDIUM"
	}

	orders := make([]WorkOrder, 0, len(decomp.Tasks))
	for _, task := range decomp.Tasks {
		taskSteps := filterStepsForTask(plan, task)
		wo := WorkOrder{
			SpecVersion:    workOrderSpecVersion,
			ID:             task.ID,
			Type:           workOrderType,
			Goal:           task.Goal,
			Risk:           risk,
			AllowedFiles:   task.AllowedFiles,
			ForbiddenFiles: task.ForbiddenFiles,
			MaxDiffLines:   task.MaxDiffLines,
			Parallelizable: task.Parallelizable,
			DependsOn:      task.DependsOn,
			Verification:   task.Verification,
			SourcePlanID:   sourcePlanID,
			Steps:          taskSteps,
			CreatedAt:      now,
		}
		if wo.AllowedFiles == nil {
			wo.AllowedFiles = []string{}
		}
		if wo.DependsOn == nil {
			wo.DependsOn = []string{}
		}
		if wo.Verification == nil {
			wo.Verification = []string{}
		}
		if wo.Steps == nil {
			wo.Steps = []ir.GrpStep{}
		}
		orders = append(orders, wo)
	}
	return orders
}

func filterStepsForTask(plan *ir.GrpPlan, task ir.DecompositionTask) []ir.GrpStep {
	if plan == nil || len(plan.Steps) == 0 {
		return []ir.GrpStep{}
	}
	allowed := make(map[string]bool, len(task.AllowedFiles))
	for _, f := range task.AllowedFiles {
		allowed[f] = true
	}
	if len(allowed) == 0 {
		return plan.Steps
	}
	var filtered []ir.GrpStep
	for _, s := range plan.Steps {
		if allowed[s.File] {
			filtered = append(filtered, s)
		}
	}
	if filtered == nil {
		filtered = []ir.GrpStep{}
	}
	return filtered
}

func printWorkOrderMarkdown(wo WorkOrder) {
	fmt.Printf("# WorkOrder: %s\n\n", wo.ID)
	fmt.Printf("**Goal:** %s\n\n", wo.Goal)
	fmt.Printf("**Risk:** %s\n", wo.Risk)
	fmt.Printf("**Spec version:** %s\n", wo.SpecVersion)
	fmt.Printf("**Created at:** %s\n\n", wo.CreatedAt)

	if wo.SourcePlanID != "" {
		fmt.Printf("**Source plan:** `%s`\n\n", wo.SourcePlanID)
	}

	if wo.Parallelizable {
		fmt.Printf("- **Parallelizable:** Yes\n")
	}
	if wo.MaxDiffLines > 0 {
		fmt.Printf("- **Max diff lines:** %d\n", wo.MaxDiffLines)
	}
	if wo.AllowedFiles != nil && len(wo.AllowedFiles) > 0 {
		fmt.Printf("\n## Allowed Files\n\n")
		for _, f := range wo.AllowedFiles {
			fmt.Printf("- `%s`\n", f)
		}
	}
	if wo.ForbiddenFiles != nil && len(wo.ForbiddenFiles) > 0 {
		fmt.Printf("\n## Forbidden Files\n\n")
		for _, f := range wo.ForbiddenFiles {
			fmt.Printf("- `%s`\n", f)
		}
	}
	if wo.DependsOn != nil && len(wo.DependsOn) > 0 {
		fmt.Printf("\n## Depends On\n\n")
		for _, d := range wo.DependsOn {
			fmt.Printf("- `%s`\n", d)
		}
	}
	if wo.Verification != nil && len(wo.Verification) > 0 {
		fmt.Printf("\n## Verification\n\n")
		for _, v := range wo.Verification {
			fmt.Printf("```bash\n%s\n```\n\n", v)
		}
	}
	if wo.Steps != nil && len(wo.Steps) > 0 {
		fmt.Printf("\n## Steps\n\n")
		for _, s := range wo.Steps {
			fmt.Printf("### %s: %s\n\n", s.ID, s.Action)
			fmt.Printf("- **Recipe:** `%s`\n", s.Recipe)
			fmt.Printf("- **File:** `%s`\n", s.File)
			fmt.Printf("- **Risk:** %s\n", s.Risk)
			if len(s.Verify) > 0 {
				fmt.Printf("- **Verify:** %s\n", strings.Join(s.Verify, ", "))
			}
		}
	}
}

func printWorkOrderSetMarkdown(set WorkOrderSet) {
	fmt.Printf("# WorkOrder Set\n\n")
	fmt.Printf("**ID:** `%s`\n\n", set.ID)
	fmt.Printf("**Tasks:** %d\n\n", len(set.Tasks))
	fmt.Printf("**Created at:** %s\n\n", set.CreatedAt)

	for _, wo := range set.Tasks {
		fmt.Printf("---\n\n")
		printWorkOrderMarkdown(wo)
	}
}

func nowISO() string {
	return time.Now().UTC().Format(time.RFC3339)
}
