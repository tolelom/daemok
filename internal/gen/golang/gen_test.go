package golang

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tolelom/daemok/internal/check"
	"github.com/tolelom/daemok/internal/model"
)

func genSrc(t *testing.T, src string) map[string]string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "d.yaml"), []byte("version: 0.4\ngo_module: example.com/x\n"+src), 0o644); err != nil {
		t.Fatal(err)
	}
	m, ds := model.Load(dir)
	if ds.HasErrors() {
		t.Fatalf("load:\n%v", ds)
	}
	if cds := check.Run(m); cds.HasErrors() {
		t.Fatalf("check:\n%v", cds)
	}
	files, err := Generate(m)
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]string{}
	for _, f := range files {
		out[f.Path] = string(f.Content)
	}
	return out
}

func TestEmptyModel(t *testing.T) {
	if files := genSrc(t, "tiers: [a]\nmodules:\n  core: { tier: a }\n"); len(files) != 0 {
		t.Fatalf("%v", files)
	}
}

func TestGenFalse(t *testing.T) {
	files := genSrc(t, `
tiers: [a]
modules:
  core: { tier: a }
enums:
  e1: { in: core, gen: false, values: [x] }
  e2: { in: core, values: [y] }
errors:
  ErrA: { in: core, gen: false, message: "a" }
`)
	if _, ok := files["core/e1.gen.go"]; ok {
		t.Fatal("gen: false enum이 생성됐다")
	}
	if _, ok := files["core/e2.gen.go"]; !ok {
		t.Fatal("e2가 없다")
	}
	if _, ok := files["core/errors.gen.go"]; ok {
		t.Fatal("모든 에러가 gen: false인데 파일이 생겼다")
	}
}

func TestImportAliasOnSameBase(t *testing.T) {
	files := genSrc(t, `
executables:
  app: { composes: [a, b, user] }
tiers: [base, top]
modules:
  a:    { tier: base }
  b:    { tier: base }
  user: { tier: top, needs: [ra, rb] }
contracts:
  ra: { in: a, kind: query, methods: { x: { errors: [] } } }
  rb: { in: b, kind: query, methods: { y: { errors: [] } } }
`)
	got := files["cmd/app/contracts.gen.go"]
	for _, want := range []string{`"example.com/x/a/query"`, `bquery "example.com/x/b/query"`, "(*query.Ra)(nil)", "(*bquery.Rb)(nil)"} {
		if !strings.Contains(got, want) {
			t.Fatalf("%q 없음:\n%s", want, got)
		}
	}
}
