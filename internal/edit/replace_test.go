package edit

import (
	"strings"
	"testing"

	"github.com/tolelom/daemok/internal/yaml"
)

func at(line, col int) yaml.Pos { return yaml.Pos{File: "d.yaml", Line: line, Column: col} }

func TestReplaceKeepsQuotes(t *testing.T) {
	src := "types:\n  post: { fields: { a: \"[]*post\", b: post_status } }\nmodules:\n  web: { depends_on: [\"core\"] }\n"
	out, err := Replace([]byte(src), "d.yaml", []Replacement{
		{Pos: at(2, 3), Old: "post", New: "article"},
		{Pos: at(2, 24), Old: "[]*post", New: "[]*article"},
		{Pos: at(4, 23), Old: "core", New: "base"},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := "types:\n  article: { fields: { a: \"[]*article\", b: post_status } }\nmodules:\n  web: { depends_on: [\"base\"] }\n"
	if string(out) != want {
		t.Fatalf("got\n%s", out)
	}
}

func TestReplaceBlockPlainWithBrackets(t *testing.T) {
	src := "types:\n  a:\n    fields:\n      m: map[string]post # 주석\n"
	out, err := Replace([]byte(src), "d.yaml", []Replacement{{Pos: at(4, 10), Old: "map[string]post", New: "map[string]article"}})
	if err != nil {
		t.Fatal(err)
	}
	if want := "types:\n  a:\n    fields:\n      m: map[string]article # 주석\n"; string(out) != want {
		t.Fatalf("got\n%s", out)
	}
}

func TestReplaceRefusesPrefixOfLongerToken(t *testing.T) {
	_, err := Replace([]byte("a: post_status\n"), "d.yaml", []Replacement{{Pos: at(1, 4), Old: "post", New: "x"}})
	if err == nil || !strings.Contains(err.Error(), "원문") {
		t.Fatalf("%v", err)
	}
}

func TestReplaceRefusesMismatch(t *testing.T) {
	_, err := Replace([]byte("a: x\n"), "d.yaml", []Replacement{{Pos: at(1, 4), Old: "y", New: "z"}})
	if err == nil || !strings.Contains(err.Error(), "원문") {
		t.Fatalf("%v", err)
	}
}

func TestReplaceRefusesEscapes(t *testing.T) {
	_, err := Replace([]byte("a: \"x\\ty\"\n"), "d.yaml", []Replacement{{Pos: at(1, 4), Old: "x\ty", New: "z"}})
	if err == nil {
		t.Fatal("이스케이프가 든 토큰을 바꿨다")
	}
}

func TestReplaceRefusesConflictAtSamePos(t *testing.T) {
	_, err := Replace([]byte("a: x\n"), "d.yaml", []Replacement{{Pos: at(1, 4), Old: "x", New: "y"}, {Pos: at(1, 4), Old: "x", New: "z"}})
	if err == nil {
		t.Fatal("같은 위치의 다른 교체를 버렸다")
	}
}
