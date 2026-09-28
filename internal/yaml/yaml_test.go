package yaml

import (
	"strings"
	"testing"
)

func parse(t *testing.T, src string) *Node {
	t.Helper()
	n, err := Parse("d.yaml", []byte(src))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	return n
}

func parseErr(t *testing.T, src string) *Error {
	t.Helper()
	_, err := Parse("d.yaml", []byte(src))
	if err == nil {
		t.Fatal("expected error")
	}
	e, ok := err.(*Error)
	if !ok {
		t.Fatalf("error type %T", err)
	}
	return e
}

func wantErr(t *testing.T, e *Error, line, col int, substr string) {
	t.Helper()
	if e.Pos.Line != line || e.Pos.Col != col || !strings.Contains(e.Msg, substr) {
		t.Fatalf("got %v, want %d:%d %q", e, line, col, substr)
	}
}

func TestBlockMapScalars(t *testing.T) {
	n := parse(t, "go_module: github.com/x/y\nts_out: web/gen.ts\n")
	if n.Kind != Map || len(n.Entries) != 2 {
		t.Fatalf("kind %v entries %d", n.Kind, len(n.Entries))
	}
	e := n.Entries[1]
	if e.Key != "ts_out" || e.Pos.Line != 2 || e.Pos.Col != 1 {
		t.Fatalf("entry %+v", e)
	}
	if e.Value.Value != "web/gen.ts" || e.Value.Pos.Col != 9 {
		t.Fatalf("value %+v", e.Value)
	}
}

func TestNestedBlockMap(t *testing.T) {
	n := parse(t, "modules:\n  core:\n    tier: domain\n  vm:\n    tier: execution\n")
	mods := n.Get("modules")
	if mods == nil || mods.Kind != Map || len(mods.Entries) != 2 {
		t.Fatalf("modules %+v", mods)
	}
	if got := mods.Get("vm").Get("tier").Value; got != "execution" {
		t.Fatalf("tier %q", got)
	}
	if mods.Get("vm").Pos.Line != 5 || mods.Get("vm").Pos.Col != 5 {
		t.Fatalf("pos %v", mods.Get("vm").Pos)
	}
}

func TestBlockSeq(t *testing.T) {
	n := parse(t, "tiers:\n  - foundation\n  - domain\n")
	s := n.Get("tiers")
	if s.Kind != Seq || len(s.Items) != 2 || s.Items[1].Value != "domain" {
		t.Fatalf("seq %+v", s)
	}
	if s.Items[1].Pos.Line != 3 || s.Items[1].Pos.Col != 5 {
		t.Fatalf("pos %v", s.Items[1].Pos)
	}
}

func TestSeqAtSameIndentAsKey(t *testing.T) {
	n := parse(t, "tiers:\n- a\n- b\nnext: 1\n")
	if len(n.Get("tiers").Items) != 2 || n.Get("next").Value != "1" {
		t.Fatalf("%+v", n)
	}
}

func TestSeqOfInlineMaps(t *testing.T) {
	src := "exceptions:\n  - depends_on: asset\n    reason: 재사용\n  - depends_on: core\n"
	s := parse(t, src).Get("exceptions")
	if len(s.Items) != 2 {
		t.Fatalf("items %d", len(s.Items))
	}
	first := s.Items[0]
	if first.Kind != Map || len(first.Entries) != 2 || first.Get("reason").Value != "재사용" {
		t.Fatalf("first %+v", first)
	}
	if first.Get("depends_on").Pos.Col != 17 {
		t.Fatalf("value col %d", first.Get("depends_on").Pos.Col)
	}
}

func TestNullValues(t *testing.T) {
	n := parse(t, "a:\nb: null\nc: x\n")
	if !n.Get("a").IsNull() || !n.Get("b").IsNull() || n.Get("c").IsNull() {
		t.Fatalf("%+v", n)
	}
}

func TestComments(t *testing.T) {
	n := parse(t, "# 머리\na: 1 # 뒤\nb: /x#y\nc: \"a#b\"\n")
	if n.Get("a").Value != "1" || n.Get("b").Value != "/x#y" || n.Get("c").Value != "a#b" {
		t.Fatalf("%+v", n)
	}
}

func TestEmptyFile(t *testing.T) {
	n := parse(t, "# only comment\n\n")
	if n.Kind != Map || len(n.Entries) != 0 {
		t.Fatalf("%+v", n)
	}
}

func TestErrTab(t *testing.T) {
	wantErr(t, parseErr(t, "a:\n\tb: 1\n"), 2, 1, "탭")
}

func TestErrMultiDoc(t *testing.T) {
	wantErr(t, parseErr(t, "a: 1\n---\nb: 2\n"), 2, 1, "여러 문서")
}

func TestErrDuplicateKey(t *testing.T) {
	e := parseErr(t, "a: 1\nb: 2\na: 3\n")
	wantErr(t, e, 3, 1, "중복 키")
	if !strings.Contains(e.Msg, "d.yaml:1:1") {
		t.Fatalf("first position missing: %q", e.Msg)
	}
}

func TestErrBadIndent(t *testing.T) {
	wantErr(t, parseErr(t, "a:\n  b: 1\n c: 2\n"), 3, 2, "들여쓰기")
}

func TestErrNoColon(t *testing.T) {
	wantErr(t, parseErr(t, "a: 1\njust text\n"), 2, 1, "키: 값")
}

func TestErrSeqInMap(t *testing.T) {
	wantErr(t, parseErr(t, "a: 1\n- b\n"), 2, 1, "목록 항목")
}

func TestErrMergeKey(t *testing.T) {
	wantErr(t, parseErr(t, "<<: base\n"), 1, 1, "병합 키")
}

func TestErrComplexKey(t *testing.T) {
	wantErr(t, parseErr(t, "? a\n"), 1, 1, "복합 키")
}

func TestQuotedScalar(t *testing.T) {
	n := parse(t, "a: \"x: y, z\"\nb: \"줄\\n탭\\t\\\"따옴표\\\" \\\\\"\n")
	if n.Get("a").Value != "x: y, z" || !n.Get("a").Quoted {
		t.Fatalf("a %+v", n.Get("a"))
	}
	if n.Get("b").Value != "줄\n탭\t\"따옴표\" \\" {
		t.Fatalf("b %q", n.Get("b").Value)
	}
}

func TestQuotedNumberStaysString(t *testing.T) {
	n := parse(t, "a: \"12\"\nb: 12\nc: 2s\n")
	if _, ok := n.Get("a").Int(); ok {
		t.Fatal("quoted number must not be int")
	}
	if v, ok := n.Get("b").Int(); !ok || v != 12 {
		t.Fatalf("b %v %v", v, ok)
	}
	if _, ok := n.Get("c").Int(); ok || n.Get("c").Value != "2s" {
		t.Fatalf("c %+v", n.Get("c"))
	}
}

func TestBoolScalar(t *testing.T) {
	n := parse(t, "a: true\nb: false\nc: yes\n")
	if v, ok := n.Get("a").Bool(); !ok || !v {
		t.Fatal("a")
	}
	if v, ok := n.Get("b").Bool(); !ok || v {
		t.Fatal("b")
	}
	if _, ok := n.Get("c").Bool(); ok {
		t.Fatal("yes is not bool")
	}
}

func TestErrUnterminatedQuote(t *testing.T) {
	wantErr(t, parseErr(t, "a: \"abc\n"), 1, 8, "닫히지 않은 따옴표")
}

func TestErrTrailingAfterQuote(t *testing.T) {
	wantErr(t, parseErr(t, "a: \"abc\" def\n"), 1, 9, "닫는 따옴표 뒤")
}

func TestErrBadEscape(t *testing.T) {
	wantErr(t, parseErr(t, "a: \"\\q\"\n"), 1, 5, "이스케이프")
}

func TestErrUnsupportedScalars(t *testing.T) {
	cases := []struct{ src, want string }{
		{"a: 'x'\n", "홑따옴표"},
		{"a: &anchor\n", "앵커"},
		{"a: *alias\n", "앵커"},
		{"a: !!str x\n", "태그"},
		{"a: |\n  text\n", "여러 줄 문자열"},
		{"a: >\n  text\n", "여러 줄 문자열"},
		{"a: %x\n", "지원하지 않는다"},
	}
	for _, c := range cases {
		wantErr(t, parseErr(t, c.src), 1, 4, c.want)
	}
}

func TestQuotedKey(t *testing.T) {
	n := parse(t, "\"a b\": 1\n")
	if n.Entries[0].Key != "a b" || n.Entries[0].Value.Value != "1" {
		t.Fatalf("%+v", n.Entries[0])
	}
}
