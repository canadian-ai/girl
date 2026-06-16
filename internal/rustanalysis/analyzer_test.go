package rustanalysis

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTempRust(t *testing.T, name, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestParseRustFile(t *testing.T) {
	src := `pub fn add(a: i32, b: i32) -> i32 {
    a + b
}

pub fn long_function(x: i32) -> i32 {
    let mut result = 0;
    if x > 0 {
        result += x;
    }
    if x > 10 {
        result += 10;
    }
    if x > 20 {
        result += 20;
    }
    if x > 30 {
        result += 30;
    }
    result
}

impl Counter {
    pub fn increment(&mut self) {
        self.value += 1;
    }

    pub fn add_many(&mut self, a: i32, b: i32, c: i32, d: i32, e: i32, f: i32) {
        self.value += a + b + c + d + e + f;
    }
}
`
	path := writeTempRust(t, "test.rs", src)
	rf, err := ParseRustFile(path)
	if err != nil {
		t.Fatalf("ParseRustFile error: %v", err)
	}
	if len(rf.Functions) != 4 {
		t.Fatalf("expected 4 functions, got %d", len(rf.Functions))
	}

	// Check add function
	add := rf.Functions[0]
	if add.Name != "add" {
		t.Errorf("expected first fn 'add', got %q", add.Name)
	}
	if add.Params != 2 {
		t.Errorf("expected add to have 2 params, got %d", add.Params)
	}
	if !add.IsPub {
		t.Error("expected add to be pub")
	}

	// Check long_function
	lf := rf.Functions[1]
	if lf.Name != "long_function" {
		t.Errorf("expected second fn 'long_function', got %q", lf.Name)
	}
	if lf.Complexity < 4 {
		t.Errorf("expected long_function complexity >= 4, got %d", lf.Complexity)
	}

	// Check impl methods
	increment := rf.Functions[2]
	if increment.Name != "increment" {
		t.Errorf("expected third fn 'increment', got %q", increment.Name)
	}
	if increment.Receiver != "Counter" {
		t.Errorf("expected increment receiver 'Counter', got %q", increment.Receiver)
	}

	addMany := rf.Functions[3]
	if addMany.Params != 7 { // &mut self + 6 params
		t.Errorf("expected add_many to have 7 params, got %d", addMany.Params)
	}
}

func TestAnalyzePath(t *testing.T) {
	src := `pub fn massive_function(a: i32, b: i32, c: i32, d: i32, e: i32, f: i32, g: i32) -> i32 {
    let mut result = 0;
    if a > 0 {
        if b > 0 {
            if c > 0 {
                if d > 0 {
                    result += 1;
                }
            }
        }
    }
    if e > 0 {
        result += e;
    }
    if f > 0 {
        result += f;
    }
    if g > 0 {
        result += g;
    }
    result
}
`
	path := writeTempRust(t, "big.rs", src)

	cfg := &Config{
		MaxFunctionLines: 20,
		MaxComplexity:    5,
		MaxNesting:       3,
		MaxFileLines:     10,
		MaxParams:        4,
	}

	result, err := AnalyzePath(path, cfg)
	if err != nil {
		t.Fatalf("AnalyzePath error: %v", err)
	}
	if len(result.Diagnostics) == 0 {
		t.Fatal("expected diagnostics, got none")
	}

	codes := map[string]bool{}
	for _, d := range result.Diagnostics {
		codes[d.Code] = true
	}

	expected := []string{
		"rust.long-function",
		"rust.high-complexity",
		"rust.deep-nesting",
		"rust.too-many-params",
		"rust.large-file",
	}
	for _, code := range expected {
		if !codes[code] {
			t.Errorf("expected diagnostic %s not found", code)
		}
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.MaxFunctionLines != 80 {
		t.Errorf("expected MaxFunctionLines 80, got %d", cfg.MaxFunctionLines)
	}
	if cfg.MaxComplexity != 10 {
		t.Errorf("expected MaxComplexity 10, got %d", cfg.MaxComplexity)
	}
	if cfg.MaxNesting != 4 {
		t.Errorf("expected MaxNesting 4, got %d", cfg.MaxNesting)
	}
}
