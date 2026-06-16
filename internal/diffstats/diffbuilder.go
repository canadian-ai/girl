package diffstats

import "fmt"

// DiffLineType represents the type of a line in a diff hunk.
type DiffLineType int

const (
	LineAdded   DiffLineType = iota
	LineDeleted DiffLineType = iota
	LineContext DiffLineType = iota
)

// DiffLine represents a single line in a diff hunk.
type DiffLine struct {
	Type    DiffLineType
	Content string
}

// DiffHunk represents a single hunk in a diff file.
type DiffHunk struct {
	OldStart int
	OldCount int
	NewStart int
	NewCount int
	Lines    []DiffLine
}

// DiffFile represents a single file entry in a diff.
type DiffFile struct {
	Path    string
	OldPath string
	IsNew   bool
	IsBinary bool
	Hunks   []DiffHunk
}

// Diff generates a unified diff string from human-readable parameters.
func Diff(files ...DiffFile) string {
	var out string
	for _, f := range files {
		out += diffFile(f)
	}
	return out
}

func diffFile(f DiffFile) string {
	oldPath := f.OldPath
	if oldPath == "" {
		oldPath = f.Path
	}
	out := fmt.Sprintf("diff --git a/%s b/%s\n", oldPath, f.Path)

	if f.IsNew {
		out += "new file mode 100644\n"
	}
	out += fmt.Sprintf("index 0000000..0000000 100644\n")

	if f.IsBinary {
		out += fmt.Sprintf("Binary files a/%s and b/%s differ\n", oldPath, f.Path)
		return out
	}

	out += fmt.Sprintf("--- a/%s\n", oldPath)
	out += fmt.Sprintf("+++ b/%s\n", f.Path)

	for _, h := range f.Hunks {
		out += diffHunk(h)
	}
	return out
}

func diffHunk(h DiffHunk) string {
	oldStart := h.OldStart
	if oldStart == 0 {
		oldStart = 1
	}
	newStart := h.NewStart
	if newStart == 0 {
		newStart = 1
	}
	oldCount := h.OldCount
	if oldCount == 0 {
		oldCount = len(h.Lines)
	}
	newCount := h.NewCount
	if newCount == 0 {
		newCount = len(h.Lines)
	}

	out := fmt.Sprintf("@@ -%d,%d +%d,%d @@\n", oldStart, oldCount, newStart, newCount)
	for _, line := range h.Lines {
		switch line.Type {
		case LineAdded:
			out += "+" + line.Content + "\n"
		case LineDeleted:
			out += "-" + line.Content + "\n"
		case LineContext:
			out += " " + line.Content + "\n"
		}
	}
	return out
}

// Test builders for common diff patterns.

// DiffOneFile creates a diff with a single file and one hunk.
func DiffOneFile(path string, added, deleted int) string {
	var lines []DiffLine
	for i := 0; i < deleted; i++ {
		lines = append(lines, DiffLine{Type: LineDeleted, Content: fmt.Sprintf("old%d", i)})
	}
	for i := 0; i < added; i++ {
		lines = append(lines, DiffLine{Type: LineAdded, Content: fmt.Sprintf("new%d", i)})
	}
	return Diff(DiffFile{
		Path: path,
		Hunks: []DiffHunk{{
			OldStart: 1,
			NewStart: 1,
			Lines:    lines,
		}},
	})
}

// DiffWithDeletedLines creates a diff where only lines are deleted (no additions).
func DiffWithDeletedLines(path string, count int) string {
	var lines []DiffLine
	for i := 0; i < count; i++ {
		lines = append(lines, DiffLine{Type: LineDeleted, Content: fmt.Sprintf("line%d", i)})
	}
	return Diff(DiffFile{
		Path: path,
		Hunks: []DiffHunk{{
			OldStart: 1,
			NewStart: 1,
			Lines:    lines,
		}},
	})
}

// DiffWithAddedLines creates a diff where only lines are added (no deletions).
func DiffWithAddedLines(path string, count int) string {
	var lines []DiffLine
	for i := 0; i < count; i++ {
		lines = append(lines, DiffLine{Type: LineAdded, Content: fmt.Sprintf("line%d", i)})
	}
	return Diff(DiffFile{
		Path: path,
		Hunks: []DiffHunk{{
			OldStart: 1,
			NewStart: 1,
			Lines:    lines,
		}},
	})
}

// DiffBinaryFile creates a diff for a binary file.
func DiffBinaryFile(path string) string {
	return Diff(DiffFile{
		Path:     path,
		IsBinary: true,
	})
}

// DiffNewFile creates a diff that adds a new file.
func DiffNewFile(path string, lines []string) string {
	dlines := make([]DiffLine, len(lines))
	for i, l := range lines {
		dlines[i] = DiffLine{Type: LineAdded, Content: l}
	}
	return Diff(DiffFile{
		Path:  path,
		IsNew: true,
		Hunks: []DiffHunk{{
			OldStart: 0,
			OldCount: 0,
			NewStart: 1,
			Lines:    dlines,
		}},
	})
}

// DiffMultipleHunks creates a diff with multiple hunks for one file.
func DiffMultipleHunks(path string, hunkCount, linesPerHunk int) string {
	hunks := make([]DiffHunk, hunkCount)
	for i := 0; i < hunkCount; i++ {
		var lines []DiffLine
		for j := 0; j < linesPerHunk; j++ {
			lines = append(lines, DiffLine{Type: LineAdded, Content: fmt.Sprintf("hunk%d_line%d", i, j)})
		}
		hunks[i] = DiffHunk{
			OldStart: i*10 + 1,
			NewStart: i*10 + 1,
			Lines:    lines,
		}
	}
	return Diff(DiffFile{
		Path:  path,
		Hunks: hunks,
	})
}
