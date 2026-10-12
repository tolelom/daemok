package edit

import (
	"errors"
	"strings"
	"testing"
)

// mustEdit(t)(SetTier(...))처럼 쓴다.
func mustEdit(t *testing.T) func([]byte, error) string {
	return func(out []byte, err error) string {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
		return string(out)
	}
}

func TestRemoveLastBlockDepKeepsComments(t *testing.T) {
	src := `modules:
  core: { tier: a }
  api:
    tier: b
    depends_on:   # 이유
      # 중간 주석
      - core
  web: { tier: b }
`
	got := mustEdit(t)(RemoveDep([]byte(src), "f", "api", "core"))
	want := strings.Replace(src, "    depends_on:   # 이유\n      # 중간 주석\n      - core\n", "    depends_on: []   # 이유\n      # 중간 주석\n", 1)
	if got != want {
		t.Fatalf("%q", got)
	}
}

func TestAddDepEmptyFlowMap(t *testing.T) {
	src := "modules:\n  core: { tier: a }\n  x: {}\n"
	got := mustEdit(t)(AddDep([]byte(src), "f", "x", "core"))
	if got != "modules:\n  core: { tier: a }\n  x: {depends_on: [core]}\n" {
		t.Fatalf("%q", got)
	}
}

func TestSetTierEmptyValue(t *testing.T) {
	src := "modules:\n  api:\n    tier:\n    summary: s\n"
	got := mustEdit(t)(SetTier([]byte(src), "f", "api", "b"))
	if got != "modules:\n  api:\n    tier: b\n    summary: s\n" {
		t.Fatalf("%q", got)
	}
}

func TestAddDepNullDependsOn(t *testing.T) {
	src := "modules:\n  api:\n    tier: b\n    depends_on:   # 나중에\n  core: { tier: a }\n"
	got := mustEdit(t)(AddDep([]byte(src), "f", "api", "core"))
	if got != "modules:\n  api:\n    tier: b\n    depends_on: [core]   # 나중에\n  core: { tier: a }\n" {
		t.Fatalf("%q", got)
	}
}

// 모델은 모듈 ID를 snake_case로 강제하지 않으므로 그런 ID도 의존으로 더할 수 있다.
// YAML 구문이 섞인 값은 막는다.
func TestAddDepIDShapes(t *testing.T) {
	src := "modules:\n  apiV2: { tier: a }\n  web: { tier: b }\n"
	got := mustEdit(t)(AddDep([]byte(src), "f", "web", "apiV2"))
	if !strings.Contains(got, "web: { tier: b, depends_on: [apiV2] }") {
		t.Fatalf("%q", got)
	}
	for _, bad := range []string{"a, b", "a]", "", "a #b", "a: b"} {
		if _, err := AddDep([]byte(src), "f", "web", bad); err == nil {
			t.Fatalf("%q를 받았다", bad)
		}
	}
}

// 줄을 넣을 때는 파일 어딘가가 아니라 넣는 자리 앞줄의 줄바꿈을 따른다.
func TestInsertFollowsLocalLineEnding(t *testing.T) {
	src := "version: 0.5\r\nmodules:\n  core: { tier: a }\n"
	got := mustEdit(t)(AddModule([]byte(src), "f", "web", "a"))
	if got != "version: 0.5\r\nmodules:\n  core: { tier: a }\n  web: { tier: a }\n" {
		t.Fatalf("%q", got)
	}
}

func TestEditAfterBOM(t *testing.T) {
	src := "\ufeffmodules: {api: {tier: a}}\n"
	got := mustEdit(t)(SetTier([]byte(src), "f", "api", "b"))
	if got != "\ufeffmodules: {api: {tier: b}}\n" {
		t.Fatalf("%q", got)
	}
}

func TestSentinelErrors(t *testing.T) {
	if _, err := SetTier([]byte("modules:\n  a: { tier: a }\n"), "f", "zz", "b"); !errors.Is(err, ErrModuleNotInFile) {
		t.Fatalf("%v", err)
	}
	if _, err := SetTier([]byte("tiers: [a]\n"), "f", "zz", "b"); !errors.Is(err, ErrModuleNotInFile) {
		t.Fatalf("%v", err)
	}
	for _, src := range []string{"tiers: [a]\n", "modules: {}\n"} {
		if _, err := AddModule([]byte(src), "f", "web", "a"); !errors.Is(err, ErrNoModuleBlock) {
			t.Fatalf("%q: %v", src, err)
		}
	}
}
