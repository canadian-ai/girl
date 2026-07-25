package commands

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/canadian-ai/girl/internal/diffstats"
	"github.com/canadian-ai/girl/internal/verifier"
	"github.com/urfave/cli/v2"
)

type ReceiptFile struct {
	Path         string `json:"path"`
	LinesAdded   int    `json:"linesAdded"`
	LinesRemoved int    `json:"linesRemoved"`
}

type Receipt struct {
	SpecVersion string `json:"specversion"`
	ID          string `json:"id"`
	Type        string `json:"type"`
	CreatedAt   string `json:"createdAt"`
	Source      string `json:"source,omitempty"`

	Diff struct {
		TotalLines   int           `json:"totalLines"`
		AddedLines   int           `json:"addedLines"`
		DeletedLines int           `json:"deletedLines"`
		FilesChanged int           `json:"filesChanged"`
		Files        []ReceiptFile `json:"files,omitempty"`
		ContentHash  string        `json:"contentHash"`
	} `json:"diff"`

	PlanRef  string            `json:"planRef,omitempty"`
	Verify   *verifier.VerifyResult `json:"verify,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

func ReceiptCommand() *cli.Command {
	return &cli.Command{
		Name:      "receipt",
		Usage:     "Generate an agent-change receipt for a diff",
		ArgsUsage: "",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "diff-file",
				Usage: "Path to unified diff file",
			},
			&cli.BoolFlag{
				Name:  "stdin",
				Usage: "Read diff from stdin",
			},
			&cli.StringFlag{
				Name:    "plan",
				Aliases: []string{"p"},
				Usage:   "Path to GRP plan JSON file",
			},
			&cli.StringFlag{
				Name:    "verify",
				Aliases: []string{"v"},
				Usage:   "Path to verification result JSON file",
			},
			&cli.StringFlag{
				Name:    "output",
				Aliases: []string{"o"},
				Usage:   "Output format: json (default), text, markdown",
				Value:   "json",
			},
			&cli.StringFlag{
				Name:    "output-file",
				Aliases: []string{"f"},
				Usage:   "Write receipt to file",
			},
		},
		Action: func(c *cli.Context) error {
			in, err := readDiffFromFlags(c)
			if err != nil {
				return err
			}
			stats := in.Stats
			data := in.Raw

			var planRef string
			if planPath := c.String("plan"); planPath != "" {
				planData, err := os.ReadFile(planPath)
				if err != nil {
					return fmt.Errorf("read plan file: %w", err)
				}
				var plan struct {
					PlanID string `json:"planId"`
				}
				if err := json.Unmarshal(planData, &plan); err != nil {
					return fmt.Errorf("parse plan file: %w", err)
				}
				planRef = plan.PlanID
			}

			var verifyResult *verifier.VerifyResult
			if verifyPath := c.String("verify"); verifyPath != "" {
				vData, err := os.ReadFile(verifyPath)
				if err != nil {
					return fmt.Errorf("read verify file: %w", err)
				}
				var vr verifier.VerifyResult
				if err := json.Unmarshal(vData, &vr); err != nil {
					return fmt.Errorf("parse verify file: %w", err)
				}
				verifyResult = &vr
			}

			contentHash := computeContentHash(data)
			fileManifest := extractFileManifest(stats)

			r := &Receipt{
				SpecVersion: "1.0",
				ID:          generateReceiptID(),
				Type:        "cai.girl.agent-change",
				CreatedAt:   time.Now().UTC().Format(time.RFC3339),
				Source:      "girl",
			}
			r.Diff.TotalLines = stats.TotalChanged
			r.Diff.AddedLines = stats.TotalAdded
			r.Diff.DeletedLines = stats.TotalDeleted
			r.Diff.FilesChanged = stats.TotalFiles
			r.Diff.Files = fileManifest
			r.Diff.ContentHash = contentHash
			r.PlanRef = planRef
			r.Verify = verifyResult

			outputFile := c.String("output-file")
			if outputFile != "" {
				data, err := json.MarshalIndent(r, "", "  ")
				if err != nil {
					return fmt.Errorf("marshal receipt: %w", err)
				}
				if err := os.WriteFile(outputFile, data, 0644); err != nil {
					return fmt.Errorf("write receipt: %w", err)
				}
				fmt.Fprintf(os.Stderr, "Written to %s\n", outputFile)
			}

			switch stringFlag(c, "output", "o") {
			case "text":
				printReceiptText(r)
			case "markdown":
				printReceiptMarkdown(r)
			default:
				printJSON(r)
			}

			return nil
		},
	}
}

func generateReceiptID() string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 8)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return fmt.Sprintf("receipt_%d_%s", time.Now().Unix(), string(b))
}

func computeContentHash(data []byte) string {
	if len(data) == 0 {
		return ""
	}
	h := sha256.Sum256(data)
	return fmt.Sprintf("%x", h)
}

func extractFileManifest(stats *diffstats.DiffStats) []ReceiptFile {
	files := make([]ReceiptFile, len(stats.Files))
	for i, f := range stats.Files {
		files[i] = ReceiptFile{
			Path:         f.Path,
			LinesAdded:   f.AddedLines,
			LinesRemoved: f.DeletedLines,
		}
	}
	return files
}

func printReceiptText(r *Receipt) {
	fmt.Printf("Receipt: %s\n", r.ID)
	fmt.Printf("Spec:    %s\n", r.SpecVersion)
	fmt.Printf("Type:    %s\n", r.Type)
	fmt.Printf("Created: %s\n", r.CreatedAt)
	fmt.Println()
	fmt.Println("Diff:")
	fmt.Printf("  Total lines:   %d\n", r.Diff.TotalLines)
	fmt.Printf("  Added:         %d\n", r.Diff.AddedLines)
	fmt.Printf("  Deleted:       %d\n", r.Diff.DeletedLines)
	fmt.Printf("  Files changed: %d\n", r.Diff.FilesChanged)
	fmt.Printf("  Content hash:  %s\n", r.Diff.ContentHash)
	if len(r.Diff.Files) > 0 {
		fmt.Println()
		fmt.Println("Files:")
		for _, f := range r.Diff.Files {
			fmt.Printf("  %s (+%d, -%d)\n", f.Path, f.LinesAdded, f.LinesRemoved)
		}
	}
	if r.PlanRef != "" {
		fmt.Printf("\nPlan: %s\n", r.PlanRef)
	}
	if r.Verify != nil {
		fmt.Printf("\nVerification:\n")
		fmt.Printf("  Package manager: %s\n", r.Verify.PackageManager)
		fmt.Printf("  Commands:        %d\n", len(r.Verify.Commands))
	}
	if len(r.Metadata) > 0 {
		fmt.Println("\nMetadata:")
		for k, v := range r.Metadata {
			fmt.Printf("  %s: %s\n", k, v)
		}
	}
	fmt.Println()
}

func printReceiptMarkdown(r *Receipt) {
	fmt.Printf("# Agent Change Receipt\n\n")
	fmt.Printf("| Field | Value |\n")
	fmt.Printf("|-------|-------|\n")
	fmt.Printf("| ID | `%s` |\n", r.ID)
	fmt.Printf("| Spec | %s |\n", r.SpecVersion)
	fmt.Printf("| Type | %s |\n", r.Type)
	fmt.Printf("| Created | %s |\n", r.CreatedAt)
	fmt.Println()
	fmt.Printf("## Diff\n\n")
	fmt.Printf("| Metric | Value |\n")
	fmt.Printf("|--------|-------|\n")
	fmt.Printf("| Total lines | %d |\n", r.Diff.TotalLines)
	fmt.Printf("| Added | %d |\n", r.Diff.AddedLines)
	fmt.Printf("| Deleted | %d |\n", r.Diff.DeletedLines)
	fmt.Printf("| Files changed | %d |\n", r.Diff.FilesChanged)
	fmt.Printf("| Content hash | `%s` |\n", r.Diff.ContentHash)
	if len(r.Diff.Files) > 0 {
		fmt.Println()
		fmt.Printf("## Files\n\n")
		fmt.Printf("| File | Added | Removed |\n")
		fmt.Printf("|------|-------|--------|\n")
		for _, f := range r.Diff.Files {
			fmt.Printf("| `%s` | %d | %d |\n", f.Path, f.LinesAdded, f.LinesRemoved)
		}
	}
	if r.PlanRef != "" {
		fmt.Printf("\n**Plan:** `%s`\n\n", r.PlanRef)
	}
	if r.Verify != nil {
		fmt.Printf("\n## Verification\n\n")
		fmt.Printf("| Check | Value |\n")
		fmt.Printf("|-------|-------|\n")
		fmt.Printf("| Package manager | %s |\n", r.Verify.PackageManager)
		fmt.Printf("| Commands found | %d |\n", len(r.Verify.Commands))
	}
	if len(r.Metadata) > 0 {
		fmt.Println()
		fmt.Printf("## Metadata\n\n")
		for k, v := range r.Metadata {
			fmt.Printf("- **%s:** %s\n", k, v)
		}
	}
	fmt.Println()
}
