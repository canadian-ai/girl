package lang

import "testing"

func TestResolve(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"go", Go},
		{"ts", TypeScript},
		{"typescript", TypeScript},
		{"tsx", TypeScriptReact},
		{"typescriptreact", TypeScriptReact},
		{"js", JavaScript},
		{"javascript", JavaScript},
		{"jsx", JavaScriptReact},
		{"javascriptreact", JavaScriptReact},
		{"unknown", "unknown"},
		{"auto", "auto"},
		{"", ""},
	}
	for _, tt := range tests {
		got := Resolve(tt.input)
		if got != tt.want {
			t.Errorf("Resolve(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestIsGo(t *testing.T) {
	if !IsGo(Go) {
		t.Errorf("IsGo(%q) should be true", Go)
	}
	if IsGo(TypeScript) {
		t.Errorf("IsGo(%q) should be false", TypeScript)
	}
	if IsGo("") {
		t.Error("IsGo(\"\") should be false")
	}
}

func TestIsTypeScript(t *testing.T) {
	if !IsTypeScript(TypeScript) {
		t.Errorf("IsTypeScript(%q) should be true", TypeScript)
	}
	if !IsTypeScript(TypeScriptReact) {
		t.Errorf("IsTypeScript(%q) should be true", TypeScriptReact)
	}
	if IsTypeScript(Go) {
		t.Errorf("IsTypeScript(%q) should be false", Go)
	}
	if IsTypeScript(JavaScript) {
		t.Errorf("IsTypeScript(%q) should be false", JavaScript)
	}
	if IsTypeScript("") {
		t.Error("IsTypeScript(\"\") should be false")
	}
}
