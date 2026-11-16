package packer

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/canadian-ai/girl/internal/ir"
)

// Synthetic credentials assembled at runtime; no real secret is used.
var (
	privPassword = strings.Join([]string{"Synthetic", "-Pa55word-", "Value"}, "")
	privToken    = strings.Join([]string{"ghp", "_", "SYNTHETICabcdefghijklmnop0123456789AB"}, "")
)

func privacyRequest(t *testing.T, mode string) PackRequest {
	t.Helper()
	dir := t.TempDir()
	src := "export function connect() {\n  const password = \"" + privPassword + "\"\n  const gh = \"" + privToken + "\"\n  return password + gh\n}\n"
	file := filepath.Join(dir, "db.ts")
	if err := os.WriteFile(file, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	envFile := filepath.Join(dir, ".env")
	if err := os.WriteFile(envFile, []byte("HARMLESS=1\nOTHER=2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	diag := ir.Diagnostic{
		Code: "ts.long-function", Severity: ir.SeverityMedium, File: file, Line: 1, EndLine: 5,
		Message:    "token=" + privToken + " found in connect",
		Suggestion: "rotate password: " + privPassword,
		Metadata:   map[string]string{"note": "api_key=" + privPassword},
		Related:    []ir.RelatedInfo{{Message: "see " + privToken}},
	}
	envDiag := ir.Diagnostic{Code: "x", Severity: ir.SeverityLow, File: envFile, Line: 1, EndLine: 2, Message: "env"}
	return PackRequest{
		Goal:        "Refactor connect; do not leak " + privToken,
		Files:       []*ir.FileIR{{Path: file, Language: "ts", Lines: 5}},
		Diagnostics: []ir.Diagnostic{diag, envDiag},
		Steps:       []ir.GrpStep{{Recipe: "ts.extract-function", Action: "use password " + privPassword, File: file, Verify: []string{"API_TOKEN=" + privPassword + " npm test"}}},
		TokenBudget: 3000,
		TargetPath:  dir,
		PrivacyMode: mode,
	}
}

func TestPrivacyModesRemoveSyntheticSecretsFromSerializedOutput(t *testing.T) {
	for _, mode := range []string{PrivacyRedacted, PrivacyPublic} {
		t.Run(mode, func(t *testing.T) {
			pack, err := NewPacker(3000).Pack(privacyRequest(t, mode))
			if err != nil {
				t.Fatal(err)
			}
			for _, out := range []any{pack, pack.ToGrpContextPack("grp_test")} {
				data, err := json.Marshal(out)
				if err != nil {
					t.Fatal(err)
				}
				s := string(data)
				for _, secret := range []string{privPassword, privToken} {
					if i := strings.Index(s, secret); i >= 0 {
						lo := i - 80
						if lo < 0 {
							lo = 0
						}
						t.Fatalf("%s export contains a synthetic secret near %q", mode, strings.ReplaceAll(s[lo:i], secret, "X"))
					}
				}
				if i := strings.Index(s, "HARMLESS=1"); i >= 0 {
					lo := i - 150
					if lo < 0 {
						lo = 0
					}
					t.Fatalf("%s export included .env contents near %q", mode, s[lo:i])
				}
			}
			if !strings.Contains(pack.Goal, "Refactor connect") {
				t.Fatalf("non-sensitive text lost: %q", pack.Goal)
			}
			if len(pack.SelectedSnippets) == 0 || !strings.Contains(strings.Join(snippetContents(pack), "\n"), "export function connect") {
				t.Fatalf("safe code context was dropped entirely: %+v", pack.SelectedSnippets)
			}
		})
	}
}

func snippetContents(pack *ir.ContextPack) []string {
	var out []string
	for _, s := range pack.SelectedSnippets {
		out = append(out, s.Content)
	}
	return out
}

func TestPrivateModeIsUnchangedAndUnknownModesAreRejected(t *testing.T) {
	pack, err := NewPacker(3000).Pack(privacyRequest(t, PrivacyPrivate))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(pack.Goal, privToken) {
		t.Fatal("private mode must not alter content")
	}
	for _, bad := range []string{"Public", "none", "redact"} {
		_, err := NewPacker(3000).Pack(privacyRequest(t, bad))
		if err == nil {
			t.Fatalf("mode %q accepted", bad)
		}
		if strings.Contains(err.Error(), privToken) || strings.Contains(err.Error(), privPassword) {
			t.Fatal("error leaked a secret")
		}
	}
}
