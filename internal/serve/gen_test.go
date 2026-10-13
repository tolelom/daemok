package serve

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tolelom/daemok/internal/gen/golang"
	"github.com/tolelom/daemok/internal/workspace"
)

func get(t *testing.T, s *Server, path string) (int, []byte) {
	t.Helper()
	req := httptest.NewRequest("GET", path, nil)
	req.Host = "127.0.0.1:7412"
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	return rec.Code, rec.Body.Bytes()
}

// 생성 미리보기는 gen이 할 일을 파일마다 상태로 보이고, 생성할 내용을 준다. 디스크에는 쓰지 않는다.
func TestGenPreview(t *testing.T) {
	s, dir := setup(t, `version: 0.5
go_module: example.com/x
tiers: [a]
modules:
  core: { tier: a }
  util: { tier: a }
  hand: { tier: a }
consts:
  limit: { in: core, type: int, value: 3 }
  size: { in: util, type: int, value: 4 }
  cap: { in: hand, type: int, value: 5 }
`)
	root := filepath.Dir(dir)
	put := func(rel, src string) {
		os.MkdirAll(filepath.Dir(filepath.Join(root, rel)), 0o755)
		os.WriteFile(filepath.Join(root, rel), []byte(src), 0o644)
	}
	put("util/consts.gen.go", golang.Header+"\n\npackage util\n")
	put("core/stale.gen.go", golang.Header+"\n\npackage core\n")
	put("hand/consts.gen.go", "package hand\n")
	s.Poll()
	status := func() map[string]string {
		code, body := get(t, s, "/api/gen")
		var out struct {
			Files []struct{ Path, Status, Reason string }
		}
		if code != 200 || json.Unmarshal(body, &out) != nil {
			t.Fatalf("%d %s", code, body)
		}
		m := map[string]string{}
		for _, f := range out.Files {
			m[f.Path] = f.Status + " " + f.Reason
		}
		return m
	}
	got := status()
	want := map[string]string{
		"core/consts.gen.go": "new ", "util/consts.gen.go": "changed ", "core/stale.gen.go": "delete ",
		"hand/consts.gen.go": "refuse 생성 헤더가 없는 파일",
	}
	for k, v := range want {
		if got[k] != v {
			t.Fatalf("%s: %q\n%v", k, got[k], got)
		}
	}
	if _, err := os.Stat(filepath.Join(root, "core", "consts.gen.go")); err == nil {
		t.Fatal("미리보기가 파일을 썼다")
	}
	code, body := get(t, s, "/api/gen/file?path=core/consts.gen.go")
	if code != 200 || !strings.Contains(string(body), "Limit = 3") {
		t.Fatalf("%d %s", code, body)
	}
	if code, _ := get(t, s, "/api/gen/file?path=design/d.yaml"); code != 404 {
		t.Fatalf("생성 파일이 아닌 경로: %d", code)
	}

	os.Remove(filepath.Join(root, "hand", "consts.gen.go"))
	r := workspace.Run(dir, root)
	if err := workspace.Apply(root, r.Plan); err != nil {
		t.Fatal(err)
	}
	for p, v := range status() {
		if v != "same " {
			t.Fatalf("gen 뒤 %s: %q", p, v)
		}
	}
}

func TestGenPreviewDesignError(t *testing.T) {
	s, _ := setup(t, "version: 0.5\ntiers: [a]\nmodules:\n  core: { tier: a, depends_on: [nope] }\n")
	if code, body := get(t, s, "/api/gen"); code != 409 || !strings.Contains(string(body), "설계 검사 오류") {
		t.Fatalf("%d %s", code, body)
	}
}
