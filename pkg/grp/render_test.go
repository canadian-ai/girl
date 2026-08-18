package grp

import (
	"strings"
	"testing"
)

func TestRenderPlanMarkdownReductionExplainsSafety(t *testing.T) {
	p := loadPlanFixture(t, "reduction-duplicate")
	out := RenderPlanMarkdown(p)

	required := []string{
		"# GRP Plan: grp_reduction_notification_001",
		"## Reduction",
		"### cap_notification",
		"### cap_notifier_member",
		"**Classification:** `duplicate`",
		"**Canonical target:** `cap_notification`",
		"safe to collect only after references migrate",
		"**Canonical for:** cap_booking_email, cap_notifier_member, cap_notify_member",
		"**Evidence:**",
		"`cap_notifier_member` -> `cap_notification` (duplicate-of) at notifications/member_notifier.go:12",
		"### Standard Blocks",
		"`blk_notification`",
		"## Steps",
		"this collect step is gated on reference migration + verification",
		"(migration gate)",
		"## Verification",
	}
	for _, want := range required {
		if !strings.Contains(out, want) {
			t.Errorf("rendered plan missing %q\n--- got:\n%s", want, out)
		}
	}
}

func TestRenderPlanMarkdownReductionContractOpaqueIDs(t *testing.T) {
	p := loadPlanFixture(t, "reduction-contract")
	out := RenderPlanMarkdown(p)

	required := []string{
		"# GRP Plan: grp_reduction_booking_001",
		"## Reduction",
		"### cap.booking",
		"### booking.create",
		"**Classification:** `duplicate`",
		"**Canonical target:** `cap.booking`",
		"safe to collect only after references migrate",
		"**Evidence:**",
		"`booking.create` -> `cap.booking` (duplicate-of) at booking/create.go:12",
		"`blk_booking`",
		"## Steps",
		"this collect step is gated on reference migration + verification",
	}
	for _, want := range required {
		if !strings.Contains(out, want) {
			t.Errorf("rendered plan missing %q\n--- got:\n%s", want, out)
		}
	}
}

func TestRenderPlanMarkdownNil(t *testing.T) {
	if got := RenderPlanMarkdown(nil); got != "" {
		t.Errorf("nil plan should render empty, got %q", got)
	}
}

func TestRenderPlanMarkdownDeterministic(t *testing.T) {
	p1 := loadPlanFixture(t, "reduction-duplicate")
	p2 := loadPlanFixture(t, "reduction-duplicate")
	if a, b := RenderPlanMarkdown(p1), RenderPlanMarkdown(p2); a != b {
		t.Errorf("rendered plan is not deterministic")
	}
}
