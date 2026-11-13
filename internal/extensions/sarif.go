package extensions

import (
 "encoding/json"
 "errors"
 "fmt"
 "net/url"
 "strings"
)

// ImportSARIF imports a constrained SARIF 2.1.0 result subset from tools such
// as Clippy and Ruff. No commands, fixes or external URIs are executed.
func ImportSARIF(data []byte, provider string) ([]Finding,error) {
 if len(data)>16<<20 {return nil,errors.New("SARIF exceeds 16 MiB")}
 var doc struct {
  Version string `json:"version"`
  Runs []struct {
   Results []struct {
    RuleID string `json:"ruleId"`
    Level string `json:"level"`
    Message struct {Text string `json:"text"`} `json:"message"`
    PartialFingerprints map[string]string `json:"partialFingerprints"`
    Locations []struct {PhysicalLocation struct {
     ArtifactLocation struct {URI string `json:"uri"`} `json:"artifactLocation"`
     Region struct {StartLine int `json:"startLine"`} `json:"region"`
    } `json:"physicalLocation"`} `json:"locations"`
   } `json:"results"`
  } `json:"runs"`
 }
 if err:=json.Unmarshal(data,&doc);err!=nil{return nil,err}
 if doc.Version!="2.1.0"||provider==""{return nil,errors.New("SARIF 2.1.0 and provider required")}
 findings:=[]Finding{}
 for _,run:=range doc.Runs{for _,r:=range run.Results{
  if len(r.Locations)==0{continue}
  loc:=r.Locations[0].PhysicalLocation
  uri,err:=url.Parse(loc.ArtifactLocation.URI)
  if err!=nil||uri.IsAbs()||uri.Host!=""||strings.ContainsAny(loc.ArtifactLocation.URI,"?#"){return nil,fmt.Errorf("unsafe artifact URI %q",loc.ArtifactLocation.URI)}
  severity:="medium"
  switch r.Level{case "error":severity="high";case "warning":severity="medium";case "note","none":severity="low"}
  f,err:=Normalize(Finding{Provider:provider,RuleID:r.RuleID,ExternalID:r.PartialFingerprints["primaryLocationLineHash"],File:loc.ArtifactLocation.URI,Line:loc.Region.StartLine,Severity:severity,Message:r.Message.Text})
  if err!=nil{return nil,fmt.Errorf("invalid external finding: %w",err)}
  findings=append(findings,f)
 }}
 return findings,nil
}
