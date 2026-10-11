package serve

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func editReq(t *testing.T, s *Server, op, id, value string, version int) (int, map[string]any) {
	t.Helper()
	body, _ := json.Marshal(map[string]any{"op": op, "id": id, "value": value, "version": version})
	return post(t, s, "/api/edit", string(body))
}

func modTier(t *testing.T, s *Server, id string) string {
	t.Helper()
	for _, m := range getState(t, s)["model"].(map[string]any)["mods"].([]any) {
		if m.(map[string]any)["id"] == id {
			return m.(map[string]any)["tier"].(string)
		}
	}
	return ""
}

func TestEditTier(t *testing.T) {
	s, dir := setup(t, design)
	local(t, s)
	if getState(t, s)["canEdit"] != true {
		t.Fatal("루프백 대조 모드면 canEdit")
	}
	code, out := editReq(t, s, "tier", "api", "a", s.Version())
	if code != 200 {
		t.Fatalf("%d %v", code, out)
	}
	if modTier(t, s, "api") != "a" || int(out["version"].(float64)) != s.Version() {
		t.Fatalf("%v", out)
	}
	b, _ := os.ReadFile(filepath.Join(dir, "d.yaml"))
	if !strings.Contains(string(b), "api:  { tier: a, depends_on: [core] }") || !strings.Contains(string(b), "kind: { in: core, values: [x, y] }") {
		t.Fatal(string(b))
	}
}

func TestEditDepsAndAddModule(t *testing.T) {
	s, dir := setup(t, design)
	local(t, s)
	for _, c := range []struct{ op, id, value string }{{"add_module", "web", "b"}, {"add_dep", "web", "api"}, {"remove_dep", "api", "core"}} {
		if code, out := editReq(t, s, c.op, c.id, c.value, s.Version()); code != 200 {
			t.Fatalf("%s: %d %v", c.op, code, out)
		}
	}
	b, _ := os.ReadFile(filepath.Join(dir, "d.yaml"))
	if !strings.Contains(string(b), "web: { tier: b, depends_on: [api] }") || !strings.Contains(string(b), "api:  { tier: b, depends_on: [] }") {
		t.Fatal(string(b))
	}
}

func TestEditRejects(t *testing.T) {
	s, dir := setup(t, design)
	local(t, s)
	if code, _ := editReq(t, s, "tier", "api", "a", s.Version()-1); code != 409 {
		t.Fatalf("버전이 다르면 409: %d", code)
	}
	if code, _ := editReq(t, s, "tier", "nope", "a", s.Version()); code != 400 {
		t.Fatalf("없는 모듈은 400: %d", code)
	}
	if code, _ := editReq(t, s, "explode", "api", "a", s.Version()); code != 400 {
		t.Fatalf("없는 작업은 400: %d", code)
	}
	// 다른 파일에 있는 ID를 더하면 합칠 때 중복 ID 오류라 쓰지 않는다
	os.WriteFile(filepath.Join(dir, "e.yaml"), []byte("modules:\n  web: { tier: a }\n"), 0o644)
	s.Poll()
	before, _ := os.ReadFile(filepath.Join(dir, "d.yaml"))
	if code, out := editReq(t, s, "add_module", "web", "a", s.Version()); code != 422 {
		t.Fatalf("읽기 오류가 생기면 422: %d %v", code, out)
	}
	after, _ := os.ReadFile(filepath.Join(dir, "d.yaml"))
	if string(before) != string(after) {
		t.Fatal("422인데 파일을 썼다")
	}
	s.SetOptions(Options{Local: false})
	if code, _ := editReq(t, s, "tier", "api", "a", s.Version()); code != 403 {
		t.Fatalf("%d", code)
	}
}
