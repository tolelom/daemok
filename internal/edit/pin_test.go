package edit

import (
	"errors"
	"testing"

	"github.com/tolelom/daemok/internal/yaml"
)

func pinCase(t *testing.T, src string, path []string, key, sub, value, want string) {
	t.Helper()
	out, err := Pin([]byte(src), "d.yaml", path, key, sub, value)
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != want {
		t.Fatalf("got\n%s\nwant\n%s", out, want)
	}
	if _, err := yaml.Parse("d.yaml", out); err != nil {
		t.Fatal(err)
	}
}

func TestPinFlow(t *testing.T) {
	pinCase(t, "types:\n  article: { in: core, fields: { a: int } }\n", []string{"types", "article"}, "go", "name", "Post",
		"types:\n  article: { in: core, fields: { a: int }, go: { name: Post } }\n")
}

func TestPinFlowEmpty(t *testing.T) {
	pinCase(t, "groups:\n  g: {}\n", []string{"groups", "g"}, "path", "", "old",
		"groups:\n  g: { path: old }\n")
}

func TestPinFlowExistingGoMap(t *testing.T) {
	pinCase(t, "enums:\n  kind2: { in: core, values: [a], go: { prefix: K } }\n", []string{"enums", "kind2"}, "go", "type", "Kind",
		"enums:\n  kind2: { in: core, values: [a], go: { prefix: K, type: Kind } }\n")
}

func TestPinBlock(t *testing.T) {
	pinCase(t, "modules:\n  base:\n    tier: a\n    # 주석\n    depends_on: [x]\nother: 1\n", []string{"modules", "base"}, "path", "", "core",
		"modules:\n  base:\n    tier: a\n    # 주석\n    depends_on: [x]\n    path: core\nother: 1\n")
}

func TestPinBlockExistingGoMap(t *testing.T) {
	pinCase(t, "types:\n  a:\n    in: core\n    go:\n      register: R\n", []string{"types", "a"}, "go", "name", "B",
		"types:\n  a:\n    in: core\n    go:\n      register: R\n      name: B\n")
}

func TestPinCRLF(t *testing.T) {
	pinCase(t, "modules:\r\n  base:\r\n    tier: a\r\n", []string{"modules", "base"}, "path", "", "core",
		"modules:\r\n  base:\r\n    tier: a\r\n    path: core\r\n")
}

func TestPinMember(t *testing.T) {
	pinCase(t, "groups:\n  g:\n    tier: a\n    members:\n      m2: { handles: [x] }\n", []string{"groups", "g", "members", "m2"}, "go", "name", "M",
		"groups:\n  g:\n    tier: a\n    members:\n      m2: { handles: [x], go: { name: M } }\n")
}

func TestPinNotInFile(t *testing.T) {
	if _, err := Pin([]byte("types: {}\n"), "d.yaml", []string{"types", "x"}, "go", "name", "X"); !errors.Is(err, ErrNodeNotInFile) {
		t.Fatalf("%v", err)
	}
}

func TestPinRefusesExistingKey(t *testing.T) {
	if _, err := Pin([]byte("types:\n  a: { go: { name: A } }\n"), "d.yaml", []string{"types", "a"}, "go", "name", "B"); err == nil {
		t.Fatal("이미 있는 키를 덮었다")
	}
}
