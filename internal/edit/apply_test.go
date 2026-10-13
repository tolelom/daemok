package edit

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func designDir(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, src := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestApplyFindsFile(t *testing.T) {
	dir := designDir(t, map[string]string{
		"a.yaml": "version: 0.5\ntiers: [a, b]\n",
		"b.yaml": "modules:\n  core: { tier: a }\n",
	})
	p, err := Apply(dir, "tier", "core", "b")
	if err != nil || filepath.Base(p) != "b.yaml" {
		t.Fatalf("%q %v", p, err)
	}
	b, _ := os.ReadFile(p)
	if !strings.Contains(string(b), "core: { tier: b }") {
		t.Fatal(string(b))
	}
}

func TestApplyRefusesLoadError(t *testing.T) {
	src := "version: 0.5\ntiers: [a]\nmodules:\n  core: { tier: a }\n"
	dir := designDir(t, map[string]string{"a.yaml": src, "b.yaml": "modules:\n  web: { tier: a }\n"})
	// a.yaml에 web을 더하면 b.yaml의 web과 ID가 겹쳐 읽기 오류다
	if _, err := Apply(dir, "add_module", "web", "a"); !errors.Is(err, ErrBreaksLoad) {
		t.Fatalf("%v", err)
	}
	if b, _ := os.ReadFile(filepath.Join(dir, "a.yaml")); string(b) != src {
		t.Fatal("거부했는데 썼다")
	}
}

func TestApplyUnknown(t *testing.T) {
	dir := designDir(t, map[string]string{"a.yaml": "version: 0.5\ntiers: [a]\nmodules:\n  core: { tier: a }\n"})
	for _, c := range [][3]string{{"explode", "core", ""}, {"tier", "nope", "a"}} {
		if _, err := Apply(dir, c[0], c[1], c[2]); err == nil || errors.Is(err, ErrBreaksLoad) {
			t.Errorf("%v: %v", c, err)
		}
	}
}
