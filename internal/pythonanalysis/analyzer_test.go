package pythonanalysis

import (
 "os"
 "os/exec"
 "path/filepath"
 "strings"
 "testing"
)

func requirePython(t *testing.T) { t.Helper(); if _,err:=exec.LookPath("python3");err!=nil { t.Skip("python3 unavailable") } }
func TestAnalyzePythonFunctions(t *testing.T) {
 requirePython(t)
 dir:=t.TempDir()
 source:="def complicated(a,b,c,d,e,f):\n" +
  "    if a:\n        if b:\n            if c:\n                if d:\n                    if e:\n                        return f\n" +
  "    return None\n"
 if err:=os.WriteFile(filepath.Join(dir,"main.py"),[]byte(source),0600);err!=nil {t.Fatal(err)}
 cfg:=DefaultConfig();cfg.MaxFunctionLines=4;cfg.MaxNesting=2;cfg.MaxComplexity=3
 result,err:=AnalyzePath(dir,cfg);if err!=nil {t.Fatal(err)}
 codes:=map[string]bool{}
 for _,d:=range result.Diagnostics {
  codes[d.Code]=true
  if d.Line<1 || d.EndLine<d.Line || d.Span==nil {t.Fatalf("invalid span: %+v",d)}
 }
 for _,code:=range []string{"py.long-function","py.high-complexity","py.deep-nesting","py.too-many-params"} {
  if !codes[code] {t.Errorf("missing %s: %+v",code,result.Diagnostics)}
 }
}
func TestMalformedSourceDoesNotCrash(t *testing.T) {
 requirePython(t)
 dir:=t.TempDir()
 if err:=os.WriteFile(filepath.Join(dir,"bad.py"),[]byte("def bad(:\n"),0600);err!=nil {t.Fatal(err)}
 result,err:=AnalyzePath(dir,nil);if err!=nil {t.Fatal(err)}
 if len(result.Diagnostics)!=1 || result.Diagnostics[0].Code!="py.syntax-error" {t.Fatalf("unexpected result: %+v",result.Diagnostics)}
}
func TestSkipsVirtualenvAndHandlesNestedSymbols(t *testing.T) {
 requirePython(t)
 dir:=t.TempDir()
 if err:=os.Mkdir(filepath.Join(dir,".venv"),0700);err!=nil {t.Fatal(err)}
 if err:=os.WriteFile(filepath.Join(dir,".venv","ignored.py"),[]byte(strings.Repeat("x=1\n",600)),0600);err!=nil {t.Fatal(err)}
 src:="class Example:\n    def method(self, x, y):\n        return x + y\n"
 if err:=os.WriteFile(filepath.Join(dir,"main.py"),[]byte(src),0600);err!=nil {t.Fatal(err)}
 r,err:=AnalyzePath(dir,nil);if err!=nil {t.Fatal(err)}
 if len(r.Files)!=1 || len(r.Diagnostics)!=0 {t.Fatalf("unexpected findings: %+v",r)}
}
