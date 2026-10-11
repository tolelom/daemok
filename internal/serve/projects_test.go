package serve

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func local(t *testing.T, s *Server) string {
	t.Helper()
	cfg := t.TempDir()
	s.SetOptions(Options{ConfigDir: cfg, Local: true})
	return cfg
}

func post(t *testing.T, s *Server, path, body string) (int, map[string]any) {
	t.Helper()
	req := httptest.NewRequest("POST", path, strings.NewReader(body))
	req.Host = "127.0.0.1:7412"
	req.Header.Set("Origin", "http://127.0.0.1:7412")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	var out map[string]any
	json.Unmarshal(rec.Body.Bytes(), &out)
	return rec.Code, out
}

func TestOpenSwitchesProject(t *testing.T) {
	s, _ := setup(t, design)
	cfg := local(t, s)
	root := goModule(t)
	body, _ := json.Marshal(map[string]string{"path": "  " + root + string(filepath.Separator) + "  "})
	if code, out := post(t, s, "/api/open", string(body)); code != 200 {
		t.Fatalf("%d %v", code, out)
	}
	st := getState(t, s)
	if st["mode"] != "analyze" || st["root"] != root {
		t.Fatalf("%v %v", st["mode"], st["root"])
	}
	b, _ := os.ReadFile(filepath.Join(cfg, "recent.json"))
	if !strings.Contains(string(b), filepath.Base(root)) {
		t.Fatal(string(b))
	}
}

func TestOpenRejectsAndKeepsProject(t *testing.T) {
	s, dir := setup(t, design)
	local(t, s)
	before := filepath.Dir(dir)
	file := filepath.Join(t.TempDir(), "f.txt")
	os.WriteFile(file, nil, 0o644)
	for _, p := range []string{filepath.Join(before, "none"), file, t.TempDir(), ""} {
		body, _ := json.Marshal(map[string]string{"path": p})
		if code, _ := post(t, s, "/api/open", string(body)); code != 400 {
			t.Errorf("%q: %d", p, code)
		}
	}
	if st := getState(t, s); st["root"] != before || st["mode"] != "check" {
		t.Fatalf("현재 프로젝트가 바뀌었다: %v", st["root"])
	}
}

func TestInitWritesDraftAndSwitchesToCheck(t *testing.T) {
	root := goModule(t)
	s := New(filepath.Join(root, "design"), root)
	s.Poll()
	local(t, s)
	code, out := post(t, s, "/api/init", "{}")
	if code != 200 || out["path"] != "design/demo.yaml" {
		t.Fatalf("%d %v", code, out)
	}
	if st := getState(t, s); st["mode"] != "check" {
		t.Fatalf("저장 뒤 대조 모드여야 한다: %v", st["mode"])
	}
	if code, _ := post(t, s, "/api/init", "{}"); code != 409 {
		t.Fatalf("%d", code)
	}
}

func TestOpenAndInitNeedLocal(t *testing.T) {
	s, _ := setup(t, design)
	s.SetOptions(Options{Local: false})
	for _, p := range []string{"/api/open", "/api/init"} {
		if code, _ := post(t, s, p, `{"path":"x"}`); code != 403 {
			t.Errorf("%s: %d", p, code)
		}
	}
}

func TestProjectsList(t *testing.T) {
	ws := t.TempDir()
	for _, p := range []string{"a", "g/b", "g/h/c", "g/h/i/d", ".hidden/e", "a/inner"} {
		os.MkdirAll(filepath.Join(ws, p), 0o755)
		os.WriteFile(filepath.Join(ws, p, "go.mod"), []byte("module x\n"), 0o644)
	}
	s, _ := setup(t, design)
	cfg := t.TempDir()
	s.SetOptions(Options{Workspace: ws, ConfigDir: cfg, Local: true})
	gone, _ := json.Marshal([]string{filepath.Join(ws, "gone")})
	os.WriteFile(filepath.Join(cfg, "recent.json"), gone, 0o644)
	req := httptest.NewRequest("GET", "/api/projects", nil)
	req.Host = "localhost:7412"
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	var out struct {
		Recent []struct {
			Path   string `json:"path"`
			Exists bool   `json:"exists"`
		} `json:"recent"`
		Workspace []string `json:"workspace"`
		CanOpen   bool     `json:"canOpen"`
	}
	json.Unmarshal(rec.Body.Bytes(), &out)
	want := []string{filepath.Join(ws, "a"), filepath.Join(ws, "g", "b"), filepath.Join(ws, "g", "h", "c")}
	if strings.Join(out.Workspace, "|") != strings.Join(want, "|") {
		t.Fatalf("깊이 3까지, 숨김 제외, 모듈 아래로 내려가지 않음: %v", out.Workspace)
	}
	if len(out.Recent) != 1 || out.Recent[0].Exists || !out.CanOpen {
		t.Fatalf("%+v", out)
	}
}

// 감시 루프가 옛 프로젝트로 늦게 끝낸 상태는 새 프로젝트의 상태를 덮지 않는다.
func TestStalePublishIsDropped(t *testing.T) {
	s, _ := setup(t, design)
	local(t, s)
	root := goModule(t)
	if err := s.Open(root); err != nil {
		t.Fatal(err)
	}
	v := s.Version()
	s.publish(state{Design: "old/design", Root: "old"})
	if s.Version() != v || getState(t, s)["root"] != root {
		t.Fatal("옛 프로젝트의 상태가 새 상태를 덮었다")
	}
}

func TestRelativeRootBecomesAbsolute(t *testing.T) {
	root := goModule(t)
	t.Chdir(root)
	s := New("design", ".")
	s.Poll()
	if st := getState(t, s); st["root"] != root || st["mode"] != "analyze" {
		t.Fatalf("%v %v", st["root"], st["mode"])
	}
}
