package jsonmigration

import (
	"bytes"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"strings"
	"testing"
)

func TestMigrate(t *testing.T) {
	source := []byte(`package example
import j /* import comment */ "encoding/json" // retain me
// Keep this comment.
func encode(v any) ([]byte,error) { return j.Marshal(v) }
func decode(b []byte,v any) error {return j.Unmarshal(
b,
v, // final argument comment
)}
func shadow() {j := struct{Marshal func(int) int}{func(i int)int{return i}}; _=j.Marshal(1)}
`)
	r, err := Migrate("example.go", source)
	if err != nil {
		t.Fatal(err)
	}
	if !r.Changed || r.Review != "" {
		t.Fatalf("result: %+v", r)
	}
	for _, want := range []string{`j /* import comment */ "encoding/json/v2"`, `jsonv1 "encoding/json"`, `j.Marshal(v, jsonv1.DefaultOptionsV1())`, `jsonv1.DefaultOptionsV1(),`, `_=j.Marshal(1)`, `retain me`, `import comment`, `Keep this comment.`} {
		if !strings.Contains(strings.ReplaceAll(string(r.Source), " ", ""), strings.ReplaceAll(want, " ", "")) {
			t.Errorf("missing %q in %s", want, r.Source)
		}
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "example.go", r.Source, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	config := types.Config{Importer: importer.Default()}
	if _, err := config.Check("example", fset, []*ast.File{file}, nil); err != nil {
		t.Fatalf("migrated file does not type check: %v\n%s", err, r.Source)
	}
	again, err := Migrate("example.go", r.Source)
	if err != nil {
		t.Fatal(err)
	}
	if again.Changed || !bytes.Equal(again.Source, r.Source) {
		t.Fatal("migration is not idempotent")
	}
}

func TestManualReviewLeavesFileUnchanged(t *testing.T) {
	for _, body := range []string{
		`var f=json.Marshal`,
		`func f(){ _=json.NewDecoder(nil) }`,
		`var v json.RawMessage`,
		`func f(){ _=json.Marshal() }`,
	} {
		t.Run(body, func(t *testing.T) {
			src := []byte("package p\nimport \"encoding/json\"\n" + body)
			r, err := Migrate("p.go", src)
			if err != nil {
				t.Fatal(err)
			}
			if r.Changed || r.Review == "" || !bytes.Equal(src, r.Source) {
				t.Fatalf("unsafe edit: %+v", r)
			}
		})
	}
}

func TestDotImportAndMalformedSource(t *testing.T) {
	r, err := Migrate("p.go", []byte("package p\nimport . \"encoding/json\"\nvar f=Marshal"))
	if err != nil || r.Changed || r.Review == "" {
		t.Fatalf("%+v %v", r, err)
	}
	if _, err := Migrate("p.go", []byte("package")); err == nil {
		t.Fatal("invalid source accepted")
	}
}

func TestAliasCollisionAndNestedCalls(t *testing.T) {
	src := []byte(`package p
import "encoding/json"
func f(jsonv1 any) ([]byte,error) {return json.Marshal(jsonv1)}
`)
	r, err := Migrate("p.go", src)
	if err != nil {
		t.Fatal(err)
	}
	if !r.Changed || !strings.Contains(string(r.Source), "jsonv12.DefaultOptionsV1()") {
		t.Fatalf("alias collision: %s", r.Source)
	}
}

func TestNoJSON(t *testing.T) {
	src := []byte("package p\n")
	r, err := Migrate("p.go", src)
	if err != nil || r.Changed || r.Review != "" || !bytes.Equal(src, r.Source) {
		t.Fatalf("%+v %v", r, err)
	}
}
