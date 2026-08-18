package grp

import (
	"fmt"
	"sort"
	"strings"
)

// RenderPlanMarkdown renders a GRP plan as human-readable markdown. For
// reduction plans it explains, per garbage-classified node, why collection is
// (or is not) safe: canonical target, reachability evidence, and the migration
// and verification gates a collect step must link.
func RenderPlanMarkdown(p *Plan) string {
	if p == nil {
		return ""
	}
	var b strings.Builder

	fmt.Fprintf(&b, "# GRP Plan: %s\n\n", p.ID)
	fmt.Fprintf(&b, "**Goal:** %s\n\n", p.Goal)
	fmt.Fprintf(&b, "**Type:** %s  \n", p.Type)
	fmt.Fprintf(&b, "**Language:** %s  \n", p.Language)
	fmt.Fprintf(&b, "**Risk:** %s\n\n", strings.ToUpper(string(p.Risk)))

	if len(p.Diagnostics) > 0 {
		fmt.Fprintf(&b, "## Diagnostics\n\n")
		for _, d := range p.Diagnostics {
			loc := d.File
			if d.Line > 0 {
				loc = fmt.Sprintf("%s:%d", d.File, d.Line)
			}
			fmt.Fprintf(&b, "- [`%s`] %s (`%s`)\n", strings.ToUpper(string(d.Severity)), d.Message, loc)
		}
		b.WriteString("\n")
	}

	if p.Reduction != nil {
		renderReduction(&b, p.Reduction)
	}

	if len(p.Steps) > 0 {
		renderSteps(&b, p.Steps)
	}

	if len(p.Verification) > 0 {
		fmt.Fprintf(&b, "## Verification\n\n")
		for _, v := range p.Verification {
			fmt.Fprintf(&b, "```bash\n%s\n```\n\n", v.Command)
		}
	}

	return b.String()
}

func renderReduction(b *strings.Builder, r *Reduction) {
	if len(r.Nodes) > 0 {
		fmt.Fprintf(b, "## Reduction\n\n")
		sorted := make([]ReductionNode, len(r.Nodes))
		copy(sorted, r.Nodes)
		sort.Slice(sorted, func(i, j int) bool { return sorted[i].ID < sorted[j].ID })

		canonicalFor := make(map[string][]string)
		for _, n := range r.Nodes {
			if n.CanonicalID != "" {
				canonicalFor[n.CanonicalID] = append(canonicalFor[n.CanonicalID], n.ID)
			}
		}

		for _, n := range sorted {
			reach := "unreachable"
			if n.Reachable {
				reach = fmt.Sprintf("reachable (%d refs)", n.RefCount)
			}
			fmt.Fprintf(b, "### %s — %s\n\n", n.ID, n.Kind)
			fmt.Fprintf(b, "- **Reachability:** %s\n", reach)
			if n.Symbol != "" {
				fmt.Fprintf(b, "- **Symbol:** `%s` (%s)\n", n.Symbol, n.File)
			}
			if n.CanonicalID != "" {
				fmt.Fprintf(b, "- **Canonical target:** `%s`\n", n.CanonicalID)
			}
			if n.GarbageClass != "" {
				fmt.Fprintf(b, "- **Classification:** `%s`\n", n.GarbageClass)
				if requiresCanonical(n.GarbageClass) {
					fmt.Fprintf(b, "- **Safety:** safe to collect only after references migrate to the canonical capability **and** verification passes.\n")
				} else if isCollectable(n.GarbageClass) {
					fmt.Fprintf(b, "- **Safety:** safe to collect only after reachability/behavior is verified; no canonical migration required.\n")
				}
			}
			if deps := canonicalFor[n.ID]; len(deps) > 0 {
				sort.Strings(deps)
				fmt.Fprintf(b, "- **Canonical for:** %s\n", strings.Join(deps, ", "))
			}
			if len(n.References) > 0 {
				fmt.Fprintf(b, "- **Evidence:**\n")
				for _, ref := range n.References {
					loc := ref.File
					if ref.Line > 0 {
						loc = fmt.Sprintf("%s:%d", ref.File, ref.Line)
					}
					fmt.Fprintf(b, "  - `%s` -> `%s` (%s) at %s\n", ref.From, ref.To, ref.Kind, loc)
				}
			}
			b.WriteString("\n")
		}
	}

	if len(r.Blocks) > 0 {
		fmt.Fprintf(b, "### Standard Blocks\n\n")
		sorted := make([]ReductionBlock, len(r.Blocks))
		copy(sorted, r.Blocks)
		sort.Slice(sorted, func(i, j int) bool { return sorted[i].ID < sorted[j].ID })
		for _, blk := range sorted {
			standard := ""
			if blk.Standard {
				standard = ", standard"
			}
			fmt.Fprintf(b, "- `%s` (%s%s): inputs `%s`, outputs `%s`\n",
				blk.ID, blk.CapabilityID, standard,
				strings.Join(blk.Inputs, ", "), strings.Join(blk.Outputs, ", "))
		}
		b.WriteString("\n")
	}
}

func renderSteps(b *strings.Builder, steps []Step) {
	fmt.Fprintf(b, "## Steps\n\n")
	stepByID := make(map[string]Step, len(steps))
	for _, s := range steps {
		stepByID[s.ID] = s
	}
	for _, s := range steps {
		fmt.Fprintf(b, "### %s: %s\n\n", s.ID, s.Action)
		fmt.Fprintf(b, "- **Title:** %s\n", s.Title)
		fmt.Fprintf(b, "- **Target:** `%s`\n", s.Target.File)
		fmt.Fprintf(b, "- **Risk:** %s\n", s.Risk)
		if s.Recipe != "" {
			fmt.Fprintf(b, "- **Recipe:** `%s`\n", s.Recipe)
		}
		if len(s.Requires) > 0 {
			fmt.Fprintf(b, "- **Prerequisites:**\n")
			for _, req := range s.Requires {
				fmt.Fprintf(b, "  - `%s` %s\n", req, stepSummary(stepByID[req]))
			}
		}
		if isReductionStep(s, ActionCollect) {
			fmt.Fprintf(b, "- **Safety:** this collect step is gated on reference migration + verification; it never runs blind.\n")
		}
		if len(s.Verify) > 0 {
			fmt.Fprintf(b, "- **Verification:**\n")
			for _, v := range s.Verify {
				fmt.Fprintf(b, "  - `%s`\n", v.Command)
			}
		}
		b.WriteString("\n")
	}
}

func stepSummary(s Step) string {
	if s.ID == "" {
		return ""
	}
	switch {
	case isReductionStep(s, ActionMigrate):
		return "(migration gate)"
	case isReductionStep(s, ActionCollect):
		return "(collect step)"
	case isReductionStep(s, ActionCanonicalize):
		return "(canonicalization)"
	}
	return ""
}

func isCollectable(g GarbageClass) bool {
	switch g {
	case GarbageUnreachable, GarbageDuplicate, GarbageObsolete, GarbageRedundant,
		GarbageDeadAPI, GarbageDeadPolicy, GarbageDeadSchemaField, GarbageDeadDependencyAdapter:
		return true
	}
	return false
}
