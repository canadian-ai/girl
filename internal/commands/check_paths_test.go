package commands

import (
    "os"
    "path/filepath"
    "testing"
)

func TestRelativeAnalyzedPath(t *testing.T) {
    root := t.TempDir()
    file := filepath.Join(root, "runtime", "runtime-gdb.py")
    if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil { t.Fatal(err) }
    if err := os.WriteFile(file, []byte("print('ok')\n"), 0o644); err != nil { t.Fatal(err) }

    cwd, err := os.Getwd()
    if err != nil { t.Fatal(err) }
    relative, err := filepath.Rel(cwd, file)
    if err != nil { t.Fatal(err) }

    for _, candidate := range []string{file, relative} {
        got, err := relativeAnalyzedPath(root, candidate)
        if err != nil { t.Fatalf("normalize %q: %v", candidate, err) }
        if got != "runtime/runtime-gdb.py" {
            t.Errorf("normalize %q = %q", candidate, got)
        }
    }

    outside := filepath.Join(t.TempDir(), "outside.py")
    if _, err := relativeAnalyzedPath(root, outside); err == nil {
        t.Fatal("expected outside-root file to be rejected")
    }
}
