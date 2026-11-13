package extensions

import (
 "strings"
 "testing"
)

func TestSARIFImport(t *testing.T){
 input:=`{"version":"2.1.0","runs":[{"results":[{"ruleId":"F401","level":"warning","message":{"text":"unused import"},"locations":[{"physicalLocation":{"artifactLocation":{"uri":"src/app.py"},"region":{"startLine":4}}}]}]}]}`
 found,err:=ImportSARIF([]byte(input),"ruff")
 if err!=nil{t.Fatal(err)}
 if len(found)!=1||found[0].File!="src/app.py"||found[0].Fingerprint==""||found[0].Severity!="medium"{t.Fatalf("unexpected normalized result: %+v",found)}
 bad:=strings.Replace(input,"src/app.py","../escape.py",1)
 if _,err:=ImportSARIF([]byte(bad),"ruff");err==nil{t.Fatal("accepted traversal")}
 if _,err:=ImportSARIF([]byte(input),"");err==nil{t.Fatal("accepted unnamed provider")}
}
