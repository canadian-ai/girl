package extensions

import "testing"

func TestGovernedActions(t *testing.T){
 f,err:=Normalize(Finding{Provider:"sonarqube",ExternalID:"issue-1",RuleID:"python:S1",File:"src/app.py",Line:10,Severity:"high"});if err!=nil{t.Fatal(err)}
 a:=Action{ID:"a",Finding:f,Recipe:"py.extract-function",Files:[]string{"src/app.py"},ChangedLines:20,Risk:"low"}
 p:=Policy{AllowedPaths:[]string{"src/**"},MaxFiles:2,MaxChangedLines:100,ReviewCategories:[]string{"security"}}
 if d:=Evaluate(a,p);d.Status!=Allow{t.Fatalf("allow: %+v",d)}
 a.Files=[]string{"../secrets"};if d:=Evaluate(a,p);d.Status!=Deny{t.Fatalf("traversal: %+v",d)}
 a.Files=[]string{"src/app.py"};a.Category="security";if d:=Evaluate(a,p);d.Status!=Review{t.Fatalf("review: %+v",d)}
 a.Category="";a.ChangedLines=101;if d:=Evaluate(a,p);d.Status!=Deny{t.Fatalf("budget: %+v",d)}
 if d:=Evaluate(a,Policy{});d.Status!=Deny{t.Fatalf("missing policy: %+v",d)}
}
func TestNormalize(t *testing.T){
 f:=Finding{Provider:"ruff",RuleID:"F401",File:"src/x.py",Line:1,Severity:"low"}
 a,err:=Normalize(f);if err!=nil{t.Fatal(err)}
 b,_:=Normalize(f);if a.Fingerprint!=b.Fingerprint{t.Fatal("unstable fingerprint")}
 for _,bad:=range []string{"../x","/etc/passwd","src/../x","C:/x","src\\x"}{
  f.File=bad;if _,err:=Normalize(f);err==nil{t.Errorf("accepted %q",bad)}
 }
}
