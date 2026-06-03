package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/canadian-ai/girl/internal/packer"
	"github.com/canadian-ai/girl/internal/planner"
	"github.com/canadian-ai/girl/pkg/grp"
)

func TestPackGRPContextUsesCanonicalPlanID(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example\n\ngo 1.25\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0644); err != nil {
		t.Fatal(err)
	}

	lang := "go"
	goal := "Improve Go code quality"
	result, err := analyzePath(dir, lang)
	if err != nil {
		t.Fatal(err)
	}
	plan := planner.NewPlanner().GeneratePlan(planner.PlanRequest{
		Target:      dir,
		Goal:        goal,
		Diagnostics: result.Diagnostics,
		Files:       result.Files,
		Lang:        lang,
	})

	gp := grp.FromIRPlan(plan)
	gp.Language = lang
	grp.NormalizePlan(gp)
	gp.ID = grp.ComputePlanID(gp)

	pack, err := packer.NewPacker(4000).Pack(packer.PackRequest{
		Goal:        plan.Goal,
		Files:       result.Files,
		Diagnostics: plan.Diagnostics,
		Steps:       plan.Steps,
		TargetPath:  dir,
	})
	if err != nil {
		t.Fatal(err)
	}
	gcp := pack.ToGrpContextPack(canonicalGRPPlanID(plan, lang))

	if gcp.PlanID == "" {
		t.Fatal("grp context pack planId is empty")
	}
	if gcp.PlanID != gp.ID {
		t.Fatalf("context pack planId = %q, want plan grp-json ID %q", gcp.PlanID, gp.ID)
	}
}
