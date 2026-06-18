package commands

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/urfave/cli/v2"
)

func TestBenchmarkCommandSmoke(t *testing.T) {
	app := cli.NewApp()
	app.Commands = []*cli.Command{BenchmarkCommand()}

	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	defer func() { os.Stdout = oldStdout }()

	err = app.Run([]string{"girl", "benchmark", "../../examples/real-refactors/go-before", "--lang", "go", "--output", "text"})
	_ = w.Close()
	var out bytes.Buffer
	_, _ = io.Copy(&out, r)
	if err != nil {
		t.Fatalf("benchmark failed: %v", err)
	}
	if !strings.Contains(out.String(), "GIRL Benchmark") {
		t.Fatalf("missing benchmark header: %s", out.String())
	}
}
