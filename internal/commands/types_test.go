package commands

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveLang_GoMod(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test"), 0644)
	lang := resolveLang(dir, "auto")
	if lang != "go" {
		t.Errorf("expected 'go', got %q", lang)
	}
}

func TestResolveLang_PackageJSON(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "package.json"), []byte("{}"), 0644)
	lang := resolveLang(dir, "auto")
	if lang != "ts" {
		t.Errorf("expected 'ts', got %q", lang)
	}
}

func TestResolveLang_Both(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test"), 0644)
	os.WriteFile(filepath.Join(dir, "package.json"), []byte("{}"), 0644)
	lang := resolveLang(dir, "auto")
	if lang != "go" {
		t.Errorf("expected 'go' (go.mod takes priority), got %q", lang)
	}
}

func TestResolveLang_MixedExplicit(t *testing.T) {
	lang := resolveLang(".", "mixed")
	if lang != "mixed" {
		t.Errorf("expected 'mixed', got %q", lang)
	}
}

func TestResolveLang_AutoOnGoFile(t *testing.T) {
	dir := t.TempDir()
	goFile := filepath.Join(dir, "main.go")
	os.WriteFile(goFile, []byte("package main"), 0644)
	lang := resolveLang(goFile, "auto")
	if lang != "go" {
		t.Errorf("expected 'go' for .go file, got %q", lang)
	}
}

func TestResolveLang_AutoOnTSFile(t *testing.T) {
	dir := t.TempDir()
	tsFile := filepath.Join(dir, "app.ts")
	os.WriteFile(tsFile, []byte("const x = 1"), 0644)
	lang := resolveLang(tsFile, "auto")
	if lang != "ts" {
		t.Errorf("expected 'ts' for .ts file, got %q", lang)
	}
}

func TestResolveLang_ExplicitLang(t *testing.T) {
	lang := resolveLang(".", "go")
	if lang != "go" {
		t.Errorf("expected 'go' (explicit), got %q", lang)
	}
}

func TestResolveLang_ExplicitTS(t *testing.T) {
	lang := resolveLang(".", "ts")
	if lang != "ts" {
		t.Errorf("expected 'ts' (explicit), got %q", lang)
	}
}

func TestHasGoMod(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test"), 0644)
	if !HasGoMod(dir) {
		t.Error("HasGoMod should be true")
	}
}

func TestHasGoMod_Missing(t *testing.T) {
	dir := t.TempDir()
	if HasGoMod(dir) {
		t.Error("HasGoMod should be false without go.mod")
	}
}

func TestHasPackageJSON(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "package.json"), []byte("{}"), 0644)
	if !HasPackageJSON(dir) {
		t.Error("HasPackageJSON should be true")
	}
}

func TestHasPackageJSON_Missing(t *testing.T) {
	dir := t.TempDir()
	if HasPackageJSON(dir) {
		t.Error("HasPackageJSON should be false without package.json")
	}
}
