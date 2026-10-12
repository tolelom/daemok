package edit

import (
	"strings"
	"testing"
)

const flow = `version: 0.5
tiers: [a, b, c]

# ── L2 ──
modules:
  core:    { tier: a, summary: "도메인" }   # 핵심
  api:     { tier: b, depends_on: [core] }
  web:     { tier: c, depends_on: [api, core] }

# ── L3 ──
enums:
  kind: { in: core, values: [x] }
`

const block = `version: 0.5
tiers: [a, b]
modules:
  core:
    tier: a
  api:
    tier: b
    depends_on:
      - core
  solo:
    tier: b
    summary: 혼자

consts:
  n: { in: core, type: int, value: 1 }
`

func TestSetTier(t *testing.T) {
	got, err := SetTier([]byte(flow), "f", "api", "c")
	must(t, err)
	want := strings.Replace(flow, "api:     { tier: b,", "api:     { tier: c,", 1)
	same(t, got, want)
	got, err = SetTier([]byte(block), "f", "core", "b")
	must(t, err)
	same(t, got, strings.Replace(block, "  core:\n    tier: a", "  core:\n    tier: b", 1))
}

func TestAddDep(t *testing.T) {
	cases := []struct{ src, module, dep, from, to string }{
		{flow, "api", "web", "depends_on: [core] }", "depends_on: [core, web] }"},
		{flow, "core", "api", `summary: "도메인" }   # 핵심`, `summary: "도메인", depends_on: [api] }   # 핵심`},
		{block, "api", "solo", "      - core\n", "      - core\n      - solo\n"},
		{block, "solo", "core", "    summary: 혼자\n", "    summary: 혼자\n    depends_on: [core]\n"},
	}
	for _, c := range cases {
		got, err := AddDep([]byte(c.src), "f", c.module, c.dep)
		must(t, err)
		same(t, got, strings.Replace(c.src, c.from, c.to, 1))
	}
	if _, err := AddDep([]byte(flow), "f", "web", "api"); err == nil || !strings.Contains(err.Error(), "이미") {
		t.Fatalf("중복: %v", err)
	}
}

func TestRemoveDep(t *testing.T) {
	cases := []struct{ src, module, dep, from, to string }{
		{flow, "web", "api", "[api, core]", "[core]"},
		{flow, "web", "core", "[api, core]", "[api]"},
		{flow, "api", "core", "depends_on: [core]", "depends_on: []"},
		{block, "api", "core", "    depends_on:\n      - core\n", "    depends_on: []\n"},
	}
	for _, c := range cases {
		got, err := RemoveDep([]byte(c.src), "f", c.module, c.dep)
		must(t, err)
		same(t, got, strings.Replace(c.src, c.from, c.to, 1))
	}
	if _, err := RemoveDep([]byte(flow), "f", "core", "api"); err == nil {
		t.Fatal("없는 의존을 지웠다")
	}
}

func TestAddModule(t *testing.T) {
	got, err := AddModule([]byte(flow), "f", "admin", "c")
	must(t, err)
	same(t, got, strings.Replace(flow, "  web:     { tier: c, depends_on: [api, core] }\n", "  web:     { tier: c, depends_on: [api, core] }\n  admin: { tier: c }\n", 1))
	got, err = AddModule([]byte(block), "f", "admin", "a")
	must(t, err)
	same(t, got, strings.Replace(block, "    summary: 혼자\n", "    summary: 혼자\n  admin: { tier: a }\n", 1))
	if _, err := AddModule([]byte(flow), "f", "core", "a"); err == nil {
		t.Fatal("있는 모듈을 더했다")
	}
	if _, err := AddModule([]byte(flow), "f", "Bad-ID", "a"); err == nil {
		t.Fatal("snake_case가 아닌 ID")
	}
}

func TestCRLFAndMissing(t *testing.T) {
	crlf := strings.ReplaceAll(block, "\n", "\r\n")
	got, err := AddDep([]byte(crlf), "f", "api", "solo")
	must(t, err)
	same(t, got, strings.Replace(crlf, "      - core\r\n", "      - core\r\n      - solo\r\n", 1))
	if _, err := SetTier([]byte(flow), "f", "nope", "a"); err == nil || !strings.Contains(err.Error(), "nope") {
		t.Fatalf("%v", err)
	}
}

func TestNoTrailingNewline(t *testing.T) {
	src := "version: 0.5\ntiers: [a]\nmodules:\n  core: { tier: a }"
	got, err := AddModule([]byte(src), "f", "api", "a")
	must(t, err)
	same(t, got, src+"\n  api: { tier: a }\n")
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func same(t *testing.T, got []byte, want string) {
	t.Helper()
	if string(got) != want {
		t.Fatalf("got\n%s\nwant\n%s", got, want)
	}
}
