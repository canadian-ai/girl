package tokens

import "testing"

func TestHeuristicEstimator_ASCII(t *testing.T) {
	e := NewHeuristicEstimator()
	got := e.Estimate("The quick brown fox jumps over the lazy dog")
	if got != 14 {
		t.Errorf("expected 14, got %d", got)
	}
}

func TestHeuristicEstimator_Code(t *testing.T) {
	e := NewHeuristicEstimator()
	code := `func main() {
	fmt.Println("hello world")
}`
	got := e.Estimate(code)
	if got != 14 {
		t.Errorf("expected 14, got %d", got)
	}
}

func TestHeuristicEstimator_LongIdentifiers(t *testing.T) {
	e := NewHeuristicEstimator()
	got := e.Estimate("veryLongIdentifierNameThatGoesOnForAWhile")
	if got != 13 {
		t.Errorf("expected 13, got %d", got)
	}
}

func TestHeuristicEstimator_NonASCII(t *testing.T) {
	e := NewHeuristicEstimator()
	got := e.Estimate("Hello, 世界")
	if got != 4 {
		t.Errorf("expected 4 (len(bytes)/3), got %d", got)
	}
}

func TestHeuristicEstimator_Empty(t *testing.T) {
	e := NewHeuristicEstimator()
	got := e.Estimate("")
	if got != 0 {
		t.Errorf("expected 0, got %d", got)
	}
}

func TestHeuristicEstimator_Short(t *testing.T) {
	e := NewHeuristicEstimator()
	got := e.Estimate("ab")
	if got != 0 {
		t.Errorf("expected 0, got %d", got)
	}
}

func TestHeuristicEstimator_Bytes(t *testing.T) {
	e := NewHeuristicEstimator()
	got := e.EstimateBytes([]byte("hello"))
	if got != 1 {
		t.Errorf("expected 1, got %d", got)
	}
}

func TestHeuristicEstimator_EmptyBytes(t *testing.T) {
	e := NewHeuristicEstimator()
	got := e.EstimateBytes([]byte{})
	if got != 0 {
		t.Errorf("expected 0, got %d", got)
	}
}
