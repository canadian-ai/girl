// Package extensions provides a fail-closed boundary for third-party findings
// and proposed cloud-agent refactorings. It does not execute source tool output.
package extensions

import (
 "crypto/sha256"
 "encoding/hex"
 "errors"
 "fmt"
 "path"
 "strings"
)

type Finding struct {
 Provider string `json:"provider"`
 ExternalID string `json:"externalId"`
 RuleID string `json:"ruleId"`
 File string `json:"file"`
 Line int `json:"line"`
 Severity string `json:"severity"`
 Message string `json:"message"`
 Fingerprint string `json:"fingerprint"`
}
type Action struct {
 ID string `json:"id"`
 Finding Finding `json:"finding"`
 Recipe string `json:"recipe"`
 Files []string `json:"files"`
 ChangedLines int `json:"changedLines"`
 Risk string `json:"risk"`
 Category string `json:"category"`
}
type Policy struct {
 AllowedPaths []string `json:"allowedPaths"`
 MaxFiles int `json:"maxFiles"`
 MaxChangedLines int `json:"maxChangedLines"`
 ReviewCategories []string `json:"reviewCategories"`
}
type Decision struct {Status string `json:"status"`; Reasons []string `json:"reasons"`}
const (Deny="deny";Review="review";Allow="allow")

func NormalizePath(p string)(string,error){
 if p==""||strings.Contains(p,"\\")||strings.HasPrefix(p,"/")||strings.Contains(p,":") {return "",errors.New("invalid repository path")}
 clean:=path.Clean(p)
 if clean=="."||clean==".."||strings.HasPrefix(clean,"../")||clean!=p{return "",errors.New("noncanonical repository path")}
 return clean,nil
}
func Normalize(f Finding)(Finding,error){
 if f.Provider==""||f.RuleID==""||f.Line<1{return Finding{},errors.New("provider, rule and positive line required")}
 p,err:=NormalizePath(f.File);if err!=nil{return Finding{},err};f.File=p
 switch f.Severity{case "info","low","medium","high","critical":default:return Finding{},fmt.Errorf("invalid severity %q",f.Severity)}
 key:=fmt.Sprintf("%s\x00%s\x00%s\x00%s\x00%d",f.Provider,f.ExternalID,f.RuleID,f.File,f.Line)
 sum:=sha256.Sum256([]byte(key));f.Fingerprint=hex.EncodeToString(sum[:]);return f,nil
}
func Evaluate(a Action,p Policy)Decision{
 deny:=func(reason string)Decision{return Decision{Status:Deny,Reasons:[]string{reason}}}
 if a.ID==""||a.Recipe==""||len(a.Files)==0||a.ChangedLines<0||p.MaxFiles<=0||p.MaxChangedLines<=0||len(p.AllowedPaths)==0{return deny("missing action or explicit policy budget")}
 if _,err:=Normalize(a.Finding);err!=nil{return deny("invalid originating finding")}
 if len(a.Files)>p.MaxFiles||a.ChangedLines>p.MaxChangedLines{return deny("change budget exceeded")}
 for _,file:=range a.Files{
  f,err:=NormalizePath(file);if err!=nil{return deny("invalid action path")}
  matched:=false
  for _,pattern:=range p.AllowedPaths{
   if strings.HasSuffix(pattern,"/**"){prefix:=strings.TrimSuffix(pattern,"/**");if prefix!=""&&strings.HasPrefix(f,prefix+"/"){matched=true}
   }else if ok,_:=path.Match(pattern,f);ok{matched=true}
  }
  if !matched{return deny("file outside approved paths")}
 }
 if a.Risk=="high"||a.Risk=="critical"||a.Finding.Severity=="critical"{return Decision{Status:Review,Reasons:[]string{"high-risk change requires human approval"}}}
 for _,c:=range p.ReviewCategories{if c==a.Category{return Decision{Status:Review,Reasons:[]string{"category requires approval"}}}}
 if a.Risk!="low"&&a.Risk!="medium"{return deny("unknown risk")}
 return Decision{Status:Allow,Reasons:[]string{"within explicit budget"}}
}
