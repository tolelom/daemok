package serve

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/tolelom/daemok/internal/model"
	"github.com/tolelom/daemok/internal/workspace"
)

const l3Design = `version: 0.5
go_module: example.com/x
tiers: [a]
modules:
  core: { tier: a }
`

func getL3(t *testing.T, s *Server) (int, map[string]any) {
	t.Helper()
	req := httptest.NewRequest("GET", "/api/l3", nil)
	req.Host = "127.0.0.1:7412"
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	var out map[string]any
	json.Unmarshal(rec.Body.Bytes(), &out)
	return rec.Code, out
}

// 초안을 보고, 새 설계 파일로 저장한다. 이미 있는 파일은 덮어쓰지 않고 다음 이름을 쓴다.
func TestL3DraftAndSave(t *testing.T) {
	s, dir := setup(t, l3Design)
	root := filepath.Dir(dir)
	write := func(name, src string) {
		os.MkdirAll(filepath.Join(root, "core"), 0o755)
		os.WriteFile(filepath.Join(root, "core", name), []byte(src), 0o644)
	}
	write("a.go", "package core\n\nconst Limit = 3\n")
	local(t, s)
	code, out := getL3(t, s)
	if code != 200 || len(out["nodes"].([]any)) != 1 || out["yaml"] == "" {
		t.Fatalf("%d %v", code, out)
	}
	if code, out := post(t, s, "/api/l3", "{}"); code != 200 || out["path"] != "design/l3.yaml" {
		t.Fatalf("%d %v", code, out)
	}
	if code, _ := post(t, s, "/api/l3", "{}"); code != 409 {
		t.Fatalf("옮길 것이 없으면 409: %d", code)
	}
	write("b.go", "package core\n\nconst Size = 4\n")
	if code, out := post(t, s, "/api/l3", "{}"); code != 200 || out["path"] != "design/l3-2.yaml" {
		t.Fatalf("%d %v", code, out)
	}
	if r := workspace.Check(dir, root); r.Diags.Count(model.Error)+r.Diags.Count(model.Warning) != 0 {
		t.Fatalf("저장 뒤 진단: %v", r.Diags)
	}
}

func TestL3SaveNeedsLocal(t *testing.T) {
	s, _ := setup(t, l3Design)
	s.SetOptions(Options{Local: false})
	if code, _ := post(t, s, "/api/l3", "{}"); code != 403 {
		t.Fatalf("%d", code)
	}
}
