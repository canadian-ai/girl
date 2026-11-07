package commands

import (
    "strings"
    "testing"
)

func TestRepeatedDomainLiterals(t *testing.T) {
    cases := []struct {
        name string
        source string
        want string
    }{
        {"state", `if (status === "failed") return
setStatus("failed")
const statusLabel = "failed"
status = "failed"
`, "failed"},
        {"identity", `const item = { id: "intro", role: "assistant" }
if (item.id === "intro") return
if (message.id === "intro") return
`, "intro"},
        {"not just formatting", `const x = "wide"
const y = "wide"
const z = "wide"
const q = "wide"
`, ""},
        {"not protocol literals without comparisons", `const extensions = ["jpeg", "jpeg", "jpeg"]
`, ""},
        {"not comments", `// if (state === "failed")
const example = "failed"
// status: "failed"
`, ""},
        {"not only two occurrences", `if (state === "failed") return
const fallback = "failed"
`, ""},
        {"ignore imports", `import "failed"
import "failed"
if (status === "failed") return
`, ""},
    }
    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            got := scanRepeatedDomainLiterals("example.ts", []byte(tc.source))
            if tc.want == "" && len(got) != 0 {
                t.Fatalf("unexpected diagnostics: %+v", got)
            }
            if tc.want != "" && (len(got) != 1 || !strings.Contains(got[0].Message, tc.want) || got[0].Code != "ts.repeated-domain-literal") {
                t.Fatalf("unexpected diagnostics: %+v", got)
            }
        })
    }
}
