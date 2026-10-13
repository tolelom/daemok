package coverage

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/tolelom/daemok/internal/model"
)

const design = `version: 0.5
go_module: example.com/cov
tiers: [a, b]
modules:
  core: { tier: a }
  vm: { tier: a, depends_on: [core] }
groups:
  mods: { tier: b, kind: h, path: vm/mods, depends_on: [core, vm], members: { pay: { handles: [send] } } }
enums:
  kind: { in: core, gen: false, go: { type: Kind, prefix: Kind }, values: [send] }
handler_kinds:
  h: { in: vm, gen: false, go: { name: Handler }, params: { x: int }, returns: [error], key: kind }
`

func write(t *testing.T, root, rel, src string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	os.MkdirAll(filepath.Dir(p), 0o755)
	if err := os.WriteFile(p, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestCount(t *testing.T) {
	root := t.TempDir()
	write(t, root, "core/kind.go", `package core

type Kind string

const KindSend Kind = "send"

func KindValues() []Kind { return []Kind{KindSend} }

func (k Kind) Valid() bool { return true }

var other, _ = 1, 2

func helper() {}

func init() {}
`)
	write(t, root, "core/kind_test.go", "package core\n\nvar t1 = 1\n")
	write(t, root, "vm/h.go", "package vm\n\ntype Handler func(x int) error\n\nfunc local() {}\n")
	write(t, root, "vm/mods/pay/pay.go", "package pay\n\nfunc handleSend(x int) error { return nil }\n\nfunc extra() {}\n")
	m, diags := model.LoadSources(map[string][]byte{"d.yaml": []byte(design)})
	if diags.HasErrors() {
		t.Fatal(diags)
	}
	rows, _, err := Count(m, root)
	if err != nil {
		t.Fatal(err)
	}
	want := []Row{{"core", 5, 3, 2, 1}, {"vm", 2, 1, 1, 0}, {"vm/mods/pay", 2, 1, 2, 1}}
	if !reflect.DeepEqual(rows, want) {
		t.Fatalf("%+v", rows)
	}
	if m.Enums[0].Gen {
		t.Fatal("Count가 모델을 바꿨다")
	}
}

// 일반 함수는 Go 이름으로 센다.
func TestCountFreeFunc(t *testing.T) {
	root := t.TempDir()
	write(t, root, "core/c.go", "package core\n\nfunc SafeAdd() {}\n\nfunc CheckIT() {}\n\nfunc Other() {}\n")
	src := "version: 0.5\ngo_module: example.com/cov\ntiers: [a]\nmodules:\n  core: { tier: a }\nfuncs:\n  safe_add: { in: core, gen: false }\n  check_it: { in: core, go: { name: CheckIT } }\n"
	m, diags := model.LoadSources(map[string][]byte{"d.yaml": []byte(src)})
	if diags.HasErrors() {
		t.Fatal(diags)
	}
	rows, _, err := Count(m, root)
	if err != nil {
		t.Fatal(err)
	}
	if want := []Row{{"core", 3, 2, 3, 2}}; !reflect.DeepEqual(rows, want) {
		t.Fatalf("%+v", rows)
	}
}
