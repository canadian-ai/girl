package redact

import (
	"strings"
	"testing"
)

// Synthetic credentials only. They are assembled at runtime so this file never
// contains a literal token that resembles a real secret.
func syn(parts ...string) string { return strings.Join(parts, "") }

var (
	synOpenAI    = syn("sk-", "proj-", "SYNTHETICabcdefghijklmnop0123456789")
	synAnthropic = syn("sk-", "ant-", "api03-SYNTHETICabcdefghij0123456789")
	synGitHub    = syn("ghp", "_", "SYNTHETICabcdefghijklmnop0123456789AB")
	synAWS       = syn("AKIA", "SYNTHETIC0000000")
	synSlack     = syn("xox", "b-", "000000000000-SYNTHETICabcdef")
	synJWT       = syn("eyJ", "hbGciOiJIUzI1NiJ9", ".", "eyJ", "zdWIiOiJzeW50aGV0aWMifQ", ".", "c3ludGhldGljc2lnbmF0dXJl")
	synPassword  = "Synthetic-Pa55word-Value"
	synEntropy   = syn("Zx9", "Qm4LpT7vWc2RbN8yKd5HsJ3gFa6EuXo1")
)

func TestStringRedactsCommonSyntaxes(t *testing.T) {
	cases := map[string]string{
		"env":            "DB_PASSWORD=" + synPassword,
		"export":         "export API_TOKEN=" + synPassword,
		"yaml":           "client_secret: " + synPassword,
		"yaml quoted":    `client_secret: "` + synPassword + `"`,
		"json":           `{"apiKey": "` + synPassword + `"}`,
		"toml":           `secret_key = '` + synPassword + `'`,
		"go":             `signingKey := "` + synPassword + `"`,
		"python":         `PASSWORD = "` + synPassword + `"`,
		"url creds":      "postgres://admin:" + synPassword + "@db.internal:5432/app",
		"bearer":         "Authorization: Bearer " + synPassword + "abcdefgh",
		"openai":         "key " + synOpenAI,
		"anthropic":      "key " + synAnthropic,
		"github":         "token " + synGitHub,
		"aws":            "id " + synAWS,
		"slack":          "slack " + synSlack,
		"jwt":            "jwt " + synJWT,
		"high entropy":   "blob " + synEntropy,
		"triple quoted":  "SECRET = \"\"\"\n" + synPassword + "\nmore\n\"\"\"",
		"yaml block":     "private_key: |\n  " + synPassword + "\n  " + synPassword + "\nnext: 1\n",
		"pem block":      syn("-----BEGIN ", "RSA PRIVATE KEY-----\n") + synEntropy + "\n" + synEntropy + syn("\n-----END ", "RSA PRIVATE KEY-----"),
		"unterminated":   syn("-----BEGIN ", "PRIVATE KEY-----\n") + synEntropy,
		"personal email": "contact someone@example.org",
		"prose":          "use password " + synPassword,
	}
	for name, input := range cases {
		t.Run(name, func(t *testing.T) {
			got := String(input)
			for _, secret := range []string{synPassword, synOpenAI, synAnthropic, synGitHub, synAWS, synSlack, synJWT, synEntropy, "someone@example.org"} {
				if strings.Contains(input, secret) && strings.Contains(got, secret) {
					t.Fatalf("secret survived: %q", got)
				}
			}
			if f := Scan(got); len(f) > 0 {
				t.Fatalf("independent scan still fires on sanitized output: %+v in %q", f, got)
			}
		})
	}
}

func TestStringPreservesNonSensitiveControls(t *testing.T) {
	for _, input := range []string{
		"func add(a, b int) int { return a + b }",
		"const maxRetries = 5",
		"commit 3f786850e387550fdab836ed7e6dc881de23001b", // hex hashes are not secrets
		"// contentHash: 1a2b3c4d5e6f7a8b",
		"if user.isAdmin { return nil }",
		"import { useState } from 'react'",
		"implement the password reset flow",
	} {
		if got := String(input); got != input {
			t.Errorf("modified harmless text: %q -> %q", input, got)
		}
	}
}

type nested struct {
	Name  string
	Meta  map[string]string
	List  []string
	Inner *nested
	Any   map[string]interface{}
	priv  string
}

func TestValueSanitizesEveryReachableString(t *testing.T) {
	v := &nested{
		Name:  "API_KEY=" + synPassword,
		Meta:  map[string]string{"k": "token: " + synPassword},
		List:  []string{"ok", synGitHub},
		Inner: &nested{Name: synOpenAI},
		Any:   map[string]interface{}{"x": synAWS},
		priv:  synPassword,
	}
	Value(v)
	if findings := ScanValue(v); len(findings) > 0 {
		t.Fatalf("remaining findings: %+v", findings)
	}
	if v.List[0] != "ok" {
		t.Fatal("harmless value modified")
	}
	if v.Any["x"] == synAWS {
		t.Fatal("interface map value not sanitized")
	}
}

func TestFindingsNeverCarryValues(t *testing.T) {
	for _, f := range ScanValue(struct{ S string }{S: "pw=" + synPassword}) {
		if strings.Contains(f.Path+f.Detector, synPassword) {
			t.Fatal("finding leaked the secret")
		}
	}
}

func TestIsSensitiveFile(t *testing.T) {
	for _, p := range []string{".env", "app/.env.production", "deploy/id_rsa", "certs/server.pem", "config/credentials.json", "infra/prod.tfvars"} {
		if !IsSensitiveFile(p) {
			t.Errorf("%s should be sensitive", p)
		}
	}
	for _, p := range []string{"src/env.ts", "src/secretary.go", "README.md"} {
		if IsSensitiveFile(p) {
			t.Errorf("%s should not be sensitive", p)
		}
	}
}
