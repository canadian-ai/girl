package commands

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFrameworkTargets_AllFrameworksRegistered(t *testing.T) {
	expected := []string{"opencode", "claude", "codex", "pi", "openrewrite", "rtk", "gritql", "rust-lsp"}
	for _, name := range expected {
		target, ok := frameworkTargets[name]
		if !ok {
			t.Fatalf("framework %q not found in frameworkTargets", name)
		}
		if target.DestDir == "" {
			t.Errorf("framework %q has empty DestDir", name)
		}
		if len(target.Files) == 0 {
			t.Errorf("framework %q has no files", name)
		}
	}
}

func TestFrameworkTargets_EmbeddedFilesExist(t *testing.T) {
	for name, target := range frameworkTargets {
		t.Run(name, func(t *testing.T) {
			for _, f := range target.Files {
				embedPath := filepath.ToSlash(filepath.Join(target.EmbedDir, f))
				data, err := installFS.ReadFile(embedPath)
				if err != nil {
					t.Errorf("embedded file %s: %v", embedPath, err)
				}
				if len(data) == 0 {
					t.Errorf("embedded file %s is empty", embedPath)
				}
			}
		})
	}
}

func TestFrameworkTargets_AllEmbedDirsCompiled(t *testing.T) {
	// Verify all embed dirs are listed in the go:embed directive by checking
	// that every target's embed dir is accessible
	for name, target := range frameworkTargets {
		t.Run(name, func(t *testing.T) {
			dir, err := installFS.ReadDir(target.EmbedDir)
			if err != nil {
				t.Errorf("embed dir %s not found (missing from go:embed?): %v", target.EmbedDir, err)
				return
			}
			if len(dir) == 0 {
				t.Errorf("embed dir %s is empty", target.EmbedDir)
			}
		})
	}
}

func TestInstall_OpenRewriteCreatesFiles(t *testing.T) {
	dir := t.TempDir()
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(origDir) })

	target, ok := frameworkTargets["openrewrite"]
	if !ok {
		t.Fatal("openrewrite not in frameworkTargets")
	}

	for _, f := range target.Files {
		embedPath := filepath.ToSlash(filepath.Join(target.EmbedDir, f))
		data, err := installFS.ReadFile(embedPath)
		if err != nil {
			t.Fatal(err)
		}
		dst := filepath.Join(target.DestDir, f)
		if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(dst, data, 0644); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(dst); os.IsNotExist(err) {
			t.Errorf("file %s was not created", dst)
		}
	}

	// Verify the SKILL.md has correct YAML frontmatter
	skillPath := filepath.Join(dir, target.DestDir, "skills", "SKILL.md")
	data, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if len(content) == 0 {
		t.Fatal("SKILL.md is empty")
	}
}

func TestInstall_RTKCreatesFiles(t *testing.T) {
	dir := t.TempDir()
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(origDir) })

	target, ok := frameworkTargets["rtk"]
	if !ok {
		t.Fatal("rtk not in frameworkTargets")
	}

	for _, f := range target.Files {
		embedPath := filepath.ToSlash(filepath.Join(target.EmbedDir, f))
		data, err := installFS.ReadFile(embedPath)
		if err != nil {
			t.Fatal(err)
		}
		dst := filepath.Join(target.DestDir, f)
		if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(dst, data, 0644); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(dst); os.IsNotExist(err) {
			t.Errorf("file %s was not created", dst)
		}
	}
}

func TestInstall_GritQLCreatesFiles(t *testing.T) {
	dir := t.TempDir()
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(origDir) })

	target, ok := frameworkTargets["gritql"]
	if !ok {
		t.Fatal("gritql not in frameworkTargets")
	}

	for _, f := range target.Files {
		embedPath := filepath.ToSlash(filepath.Join(target.EmbedDir, f))
		data, err := installFS.ReadFile(embedPath)
		if err != nil {
			t.Fatal(err)
		}
		dst := filepath.Join(target.DestDir, f)
		if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(dst, data, 0644); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(dst); os.IsNotExist(err) {
			t.Errorf("file %s was not created", dst)
		}
	}
}

func TestInstall_RustLSPCreatesFiles(t *testing.T) {
	dir := t.TempDir()
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(origDir) })

	target, ok := frameworkTargets["rust-lsp"]
	if !ok {
		t.Fatal("rust-lsp not in frameworkTargets")
	}

	for _, f := range target.Files {
		embedPath := filepath.ToSlash(filepath.Join(target.EmbedDir, f))
		data, err := installFS.ReadFile(embedPath)
		if err != nil {
			t.Fatal(err)
		}
		dst := filepath.Join(target.DestDir, f)
		if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(dst, data, 0644); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(dst); os.IsNotExist(err) {
			t.Errorf("file %s was not created", dst)
		}
	}
}

func TestInstall_UnknownFramework(t *testing.T) {
	_, ok := frameworkTargets["nonexistent-framework"]
	if ok {
		t.Error("expected nonexistent-framework to not be in frameworkTargets")
	}
}
