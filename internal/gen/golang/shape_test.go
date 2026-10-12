package golang

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/tolelom/daemok/internal/model"
)

func TestShapes(t *testing.T) {
	dir := t.TempDir()
	src := `version: 0.5
go_module: example.com/s
tiers: [a, b]
modules:
  model: { tier: a, path: internal/model }
  post:  { tier: b, path: internal/post, depends_on: [model] }
types:
  post_row: { in: model, gen: false, go: { name: Post }, fields: { title: string, tags: { ref: tag, many: true }, series_id: { type: uint, optional: true }, raw: json.RawMessage, kind: Local } }
  tag: { in: model, gen: false, fields: { name: string } }
contracts:
  service:
    in: post
    gen: false
    methods:
      get: { params: { id: uint, user: "*uint" }, returns: [post_row], errors: [] }
      list: { params: { page: int }, returns: ["[]post_row", int64] }
      delete: { params: { id: uint } }
`
	if err := os.WriteFile(filepath.Join(dir, "d.yaml"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	m, ds := model.Load(dir)
	if ds.HasErrors() {
		t.Fatal(ds)
	}
	gotM := ContractShape(m, m.Contract("service"))
	wantM := []MethodShape{
		{"Get", []string{"uint", "*uint"}, []string{"*model.Post", "error"}},
		{"List", []string{"int"}, []string{"[]model.Post", "int64", "error"}},
		{"Delete", []string{"uint"}, []string{"error"}},
	}
	if !reflect.DeepEqual(gotM, wantM) {
		t.Fatalf("contract:\n got %+v\nwant %+v", gotM, wantM)
	}
	gotF := TypeShape(m, m.Type("post_row"))
	wantF := []FieldShape{{"Title", "string"}, {"Tags", "[]Tag"}, {"SeriesID", "*uint"}, {"Raw", "json.RawMessage"}, {"Kind", "Local"}}
	if !reflect.DeepEqual(gotF, wantF) {
		t.Fatalf("type:\n got %+v\nwant %+v", gotF, wantF)
	}
}
