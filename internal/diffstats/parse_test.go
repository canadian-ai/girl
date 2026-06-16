package diffstats

import (
	"strings"
	"testing"
)

func TestParseDiffSingleFile(t *testing.T) {
	input := `diff --git a/main.go b/main.go
index abc..def 100644
--- a/main.go
+++ b/main.go
@@ -1,5 +1,7 @@
 package main

+import "fmt"
+
 func main() {
-	println("hello")
+	fmt.Println("hello world")
 }
`
	stats, err := ParseDiff(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if stats.TotalFiles != 1 {
		t.Errorf("expected 1 file, got %d", stats.TotalFiles)
	}
	if len(stats.Files) != 1 {
		t.Fatalf("expected 1 file stat, got %d", len(stats.Files))
	}
	if stats.TotalAdded != 3 {
		t.Errorf("expected 3 added (including blank line), got %d", stats.TotalAdded)
	}
	if stats.TotalDeleted != 1 {
		t.Errorf("expected 1 deleted, got %d", stats.TotalDeleted)
	}
	if stats.TotalChanged != 4 {
		t.Errorf("expected 4 changed (including blank line), got %d", stats.TotalChanged)
	}
}

func TestParseDiffMultiFile(t *testing.T) {
	input := `diff --git a/a.go b/a.go
index abc..def 100644
--- a/a.go
+++ b/a.go
@@ -1 +1,2 @@
 a
+b
diff --git a/b.go b/b.go
index abc..def 100644
--- a/b.go
+++ b/b.go
@@ -1 +1,3 @@
 b
+c
+d
`
	stats, err := ParseDiff(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if stats.TotalFiles != 2 {
		t.Errorf("expected 2 files, got %d", stats.TotalFiles)
	}
	if stats.TotalAdded != 3 {
		t.Errorf("expected 3 added, got %d", stats.TotalAdded)
	}
	if stats.TotalDeleted != 0 {
		t.Errorf("expected 0 deleted, got %d", stats.TotalDeleted)
	}
	if stats.TotalChanged != 3 {
		t.Errorf("expected 3 changed, got %d", stats.TotalChanged)
	}
}

func TestParseDiffBinary(t *testing.T) {
	input := `diff --git a/image.png b/image.png
index abc..def 100644
Binary files /dev/null and b/image.png differ
`
	stats, err := ParseDiff(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if stats.TotalFiles != 1 {
		t.Errorf("expected 1 file, got %d", stats.TotalFiles)
	}
	if !stats.Files[0].IsBinary {
		t.Error("expected binary file")
	}
}

func TestParseDiffGenerated(t *testing.T) {
	input := `diff --git a/go.sum b/go.sum
index abc..def 100644
--- a/go.sum
+++ b/go.sum
@@ -1 +1,3 @@
 oldhash
+newhash1
+newhash2
`
	stats, err := ParseDiff(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if !stats.Files[0].IsGenerated {
		t.Error("expected generated file")
	}
	if !stats.Files[0].IsLockfile {
		t.Error("expected lockfile")
	}
}

func TestParseDiffEmpty(t *testing.T) {
	stats, err := ParseDiff(strings.NewReader(""))
	if err != nil {
		t.Fatal(err)
	}
	if stats.TotalFiles != 0 {
		t.Errorf("expected 0 files, got %d", stats.TotalFiles)
	}
}

func TestParseDiffBytes(t *testing.T) {
	data := []byte("diff --git a/main.go b/main.go\nindex abc..def 100644\n--- a/main.go\n+++ b/main.go\n@@ -1 +1,2 @@\n a\n+b\n")
	stats, err := ParseDiffBytes(data)
	if err != nil {
		t.Fatal(err)
	}
	if stats.TotalFiles != 1 {
		t.Errorf("expected 1 file, got %d", stats.TotalFiles)
	}
	if stats.TotalAdded != 1 {
		t.Errorf("expected 1 added, got %d", stats.TotalAdded)
	}
}

func TestParseDiffNewFile(t *testing.T) {
	input := `diff --git a/newfile.go b/newfile.go
new file mode 100644
index 000..abc 100644
--- /dev/null
+++ b/newfile.go
@@ -0,0 +1,3 @@
+package main
+
+func New() {}
`
	stats, err := ParseDiff(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if stats.TotalFiles != 1 {
		t.Errorf("expected 1 file, got %d", stats.TotalFiles)
	}
	if stats.TotalAdded != 3 {
		t.Errorf("expected 3 added, got %d", stats.TotalAdded)
	}
}

func TestParseDiffDeletedFile(t *testing.T) {
	input := `diff --git a/old.go b/old.go
deleted file mode 100644
index abc..000 100644
--- a/old.go
+++ /dev/null
@@ -1,3 +0,0 @@
-package main
-
-func Old() {}
`
	stats, err := ParseDiff(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if stats.TotalFiles != 1 {
		t.Errorf("expected 1 file, got %d", stats.TotalFiles)
	}
	if stats.TotalDeleted != 3 {
		t.Errorf("expected 3 deleted, got %d", stats.TotalDeleted)
	}
}

func TestParseDiffMultipleHunks(t *testing.T) {
	input := `diff --git a/main.go b/main.go
index abc..def 100644
--- a/main.go
+++ b/main.go
@@ -1,5 +1,6 @@
 package main

+import "fmt"
+
 func foo() {
-	println("old")
+	fmt.Println("new")
 }
@@ -10,7 +11,7 @@
 func bar() {
 	// comment
-	println("bar old")
+	fmt.Println("bar new")
 }
`
	stats, err := ParseDiff(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if stats.TotalAdded != 4 {
		t.Errorf("expected 4 added (including blank + line), got %d", stats.TotalAdded)
	}
	if stats.TotalDeleted != 2 {
		t.Errorf("expected 2 deleted, got %d", stats.TotalDeleted)
	}
}

func TestParseDiffLargestDelta(t *testing.T) {
	input := `diff --git a/small.go b/small.go
--- a/small.go
+++ b/small.go
@@ -1 +1,2 @@
 a
+b
diff --git a/large.go b/large.go
--- a/large.go
+++ b/large.go
@@ -1 +1,11 @@
 a
+b
+c
+d
+e
+f
+g
+h
+i
+j
+k
`
	stats, err := ParseDiff(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if stats.LargestDelta != 10 {
		t.Errorf("expected largest delta 10, got %d", stats.LargestDelta)
	}
	if stats.TotalChanged != 11 {
		t.Errorf("expected 11 total changed, got %d", stats.TotalChanged)
	}
}

func TestParseDiffGeneratedNodeModules(t *testing.T) {
	input := `diff --git a/node_modules/pkg/index.js b/node_modules/pkg/index.js
new file mode 100644
index 000..abc
--- /dev/null
+++ b/node_modules/pkg/index.js
@@ -0,0 +1,2 @@
+module.exports = {}
`
	stats, err := ParseDiff(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if !stats.Files[0].IsGenerated {
		t.Error("expected generated file for node_modules path")
	}
}

func TestParseDiffCategories(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"main.go", "go"},
		{"internal/server/handler.go", "library"},
		{"pkg/api/types.go", "library"},
		{"frontend/App.tsx", "typescript"},
		{"src/index.js", "typescript"},
		{"style.css", "style"},
		{"config.yaml", "config"},
		{"README.md", "documentation"},
		{"test/integration_test.go", "go"},
		{"somefile.rs", "other"},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			input := "diff --git a/" + tt.path + " b/" + tt.path + "\n--- a/" + tt.path + "\n+++ b/" + tt.path + "\n@@ -1 +1,2 @@\n old\n+new\n"
			stats, err := ParseDiff(strings.NewReader(input))
			if err != nil {
				t.Fatal(err)
			}
			if len(stats.Categories) == 0 {
				t.Fatalf("expected categories, got none")
			}
			got := stats.Categories[0]
			if got != tt.expected {
				t.Errorf("categoryForFile(%q) = %q, want %q", tt.path, got, tt.expected)
			}
		})
	}
}

func TestParseDiffWithRename(t *testing.T) {
	input := `diff --git a/old.go b/new.go
similarity index 100%
rename from old.go
rename to new.go
`
	stats, err := ParseDiff(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if stats.TotalFiles != 1 {
		t.Errorf("expected 1 file, got %d", stats.TotalFiles)
	}
	if !stats.Files[0].IsRename {
		t.Error("expected rename")
	}
}
