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
	if err := os.WriteFile(filepath.Join(dir, "d.yaml"), []byte("version: 0.5\ngo_module: example.com/x\n"+src), 0o644); err != nil {
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

func TestCmdPathModuleIsNotMain(t *testing.T) {
	files := genSrc(t, `
executables:
  tool: { composes: [cli] }
tiers: [a]
modules:
  cli: { tier: a, path: cmd/tool/cli }
enums:
  k: { in: cli, values: [x] }
`)
	if !strings.Contains(files["cmd/tool/cli/k.gen.go"], "package cli") {
		t.Fatalf("%s", files["cmd/tool/cli/k.gen.go"])
	}
}

func TestLocalNamesDoNotShadowImports(t *testing.T) {
	files := genSrc(t, `
tiers: [a, b]
modules:
  m: { tier: a }
  d: { tier: a }
  ev: { tier: b, depends_on: [m, d] }
enums:
  mk: { in: m, values: [x] }
  event_type: { in: ev, values: [e1] }
types:
  dt: { in: d, fields: { n: int } }
event_kinds:
  kind: { in: ev, key: event_type }
events:
  e1: { kind: kind, publishers: [ev], data: { a: mk, b: { ref: dt } } }
`)
	got := files["ev/kind.gen.go"]
	for _, bad := range []string{"\t\"example.com/x/m\"", "\t\"example.com/x/d\""} {
		if strings.Contains(got, bad) {
			t.Fatalf("지역 이름과 같은 import 이름:\n%s", got)
		}
	}
}

func TestErrorGoName(t *testing.T) {
	files := genSrc(t, `
tiers: [a, b]
modules:
  core: { tier: a }
  api:  { tier: b, depends_on: [core] }
errors:
  core_nf: { in: core, message: "nf", http: 404, go: { name: ErrNotFound } }
  api_nf:  { in: api, message: "nf", go: { name: ErrNotFound } }
contracts:
  store:
    in: api
    methods:
      get: { params: { id: uint }, returns: [], errors: [core_nf, api_nf] }
funcs:
  get:
    in: api
    of: store.get
    errors: [core_nf]
    cases:
      - { name: missing, given: { id: 1 }, error: core_nf }
`)
	all := ""
	for _, c := range files {
		all += c
	}
	for _, s := range []string{
		`ErrNotFound = errors.New("nf")`,
		"case errors.Is(err, ErrNotFound):",
		"core.ErrNotFound",
		"// core.ErrNotFound, ErrNotFound",
	} {
		if !strings.Contains(all, s) {
			t.Errorf("생성 코드에 %q가 없다", s)
		}
	}
	if strings.Contains(all, "core_nf") {
		t.Errorf("생성 코드에 에러 ID가 남았다:\n%s", all)
	}
}

func TestGroupWithoutKindGeneratesNothing(t *testing.T) {
	files := genSrc(t, `
system: { id: s, interfaces: { api: { protocol: http } } }
executables:
  server: { exposes: [api], composes: [router, domains] }
tiers: [a, b, c]
modules:
  model:  { tier: a }
  router: { tier: c, depends_on: [domains], implements: [api] }
groups:
  domains:
    tier: b
    path: internal
    depends_on: [model]
    rule: no_sibling_deps
    members: { post: {}, comment: {} }
enums:
  post_status: { in: post, values: [draft, published] }
`)
	for p := range files {
		if strings.HasSuffix(p, "register.gen.go") || strings.HasSuffix(p, "modules.gen.go") {
			t.Errorf("kind 없는 그룹이 %s를 만들었다", p)
		}
	}
	if _, ok := files["internal/post/post_status.gen.go"]; !ok {
		t.Errorf("멤버의 L3 노드는 생성한다: %v", files)
	}
}
