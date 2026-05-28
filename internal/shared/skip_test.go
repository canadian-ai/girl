package shared

import "testing"

func TestShouldSkipDir_Exact(t *testing.T) {
	for _, name := range []string{".git", ".grp", "node_modules", "vendor", "dist", "build", ".next", ".turbo", ".vercel", "out", ".cache", "coverage"} {
		if !ShouldSkipDir(name) {
			t.Errorf("ShouldSkipDir(%q) should be true", name)
		}
	}
}

func TestShouldSkipDir_Normal(t *testing.T) {
	for _, name := range []string{"src", "internal", "pkg", "testdata", "docs"} {
		if ShouldSkipDir(name) {
			t.Errorf("ShouldSkipDir(%q) should be false", name)
		}
	}
}

func TestShouldSkipDir_DotPrefix(t *testing.T) {
	if !ShouldSkipDir(".config") {
		t.Error("ShouldSkipDir('.config') should be true (dot prefix)")
	}
}

func TestShouldSkipDir_Empty(t *testing.T) {
	if ShouldSkipDir("") {
		t.Error("ShouldSkipDir('') should be false")
	}
}
