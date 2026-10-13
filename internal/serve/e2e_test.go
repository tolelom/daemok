package serve

import (
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// 화면 편집을 headless 브라우저로 끝까지 해 본다. 브라우저와 node가 필요해 DAEMOK_E2E=1일 때만 돈다.
func TestUIEditE2E(t *testing.T) {
	if os.Getenv("DAEMOK_E2E") != "1" {
		t.Skip("DAEMOK_E2E=1일 때만 돈다")
	}
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node가 없다")
	}
	s, dir := setup(t, `version: 0.5
go_module: example.com/x
tiers: [a, b]
modules:
  core: { tier: a }
  util: { tier: a }
  api:  { tier: b, depends_on: [core] }
`)
	local(t, s)
	srv := httptest.NewServer(s.Handler())
	defer srv.Close()
	out, err := exec.Command(node, filepath.Join("..", "..", "scripts", "e2e", "edit.mjs"), srv.URL+"/").CombinedOutput()
	if err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	b, _ := os.ReadFile(filepath.Join(dir, "d.yaml"))
	got := string(b)
	for _, want := range []string{"api:  { tier: a, depends_on: [util] }", "  web: { tier: b }\n"} {
		if !strings.Contains(got, want) {
			t.Fatalf("%q가 없다\n%s\n%s", want, got, out)
		}
	}
}

// 지도의 관계선 표시, tier 안 배치, 모듈 탭 진단 묶음을 headless 브라우저로 본다. DAEMOK_E2E=1일 때만 돈다.
// 설계 순서대로 놓으면 x→q, y→p가 교차한다. 배치가 교차를 줄이면 x·y의 좌우 순서와 q·p의 좌우 순서가 같다.
func TestUIMapE2E(t *testing.T) {
	if os.Getenv("DAEMOK_E2E") != "1" {
		t.Skip("DAEMOK_E2E=1일 때만 돈다")
	}
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node가 없다")
	}
	s, dir := setup(t, `version: 0.5
go_module: example.com/x
tiers: [a, b]
modules:
  p: { tier: a }
  q: { tier: a }
  x: { tier: b, depends_on: [q] }
  y: { tier: b, depends_on: [p] }
errors:
  ErrBad: { in: p, message: bad }
`)
	root := filepath.Dir(dir)
	os.MkdirAll(filepath.Join(root, "p"), 0o755)
	os.WriteFile(filepath.Join(root, "p", "p.go"), []byte("package p\n\nimport \"errors\"\n\nfunc a() error { return errors.New(\"a\") }\nfunc b() error { return errors.New(\"b\") }\nfunc c() error { return errors.New(\"c\") }\n"), 0o644)
	s.Poll()
	local(t, s)
	srv := httptest.NewServer(s.Handler())
	defer srv.Close()
	out, err := exec.Command(node, filepath.Join("..", "..", "scripts", "e2e", "map.mjs"), srv.URL+"/").CombinedOutput()
	if err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
}
