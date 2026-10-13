package serve

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"os/exec"
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

func gitRun(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

// worktreeRepo는 설계가 없는 주 worktree와 설계가 있는 feat/daemok worktree를 만든다.
func worktreeRepo(t *testing.T) (main, wt string) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git이 없다")
	}
	main = goModule(t)
	gitRun(t, main, "init", "-q", "-b", "main")
	gitRun(t, main, "add", ".")
	gitRun(t, main, "commit", "-q", "-m", "init")
	wt = filepath.Join(main, ".worktrees", "daemok")
	gitRun(t, main, "worktree", "add", "-q", "-b", "feat/daemok", wt)
	os.MkdirAll(filepath.Join(wt, "design"), 0o755)
	os.WriteFile(filepath.Join(wt, "design", "d.yaml"), []byte(design), 0o644)
	return main, wt
}

type projectItem struct {
	Path    string `json:"path"`
	Exists  bool   `json:"exists"`
	Branch  string `json:"branch"`
	Mode    string `json:"mode"`
	Current bool   `json:"current"`
}

func getProjects(t *testing.T, s *Server) (worktrees, recent []projectItem) {
	t.Helper()
	req := httptest.NewRequest("GET", "/api/projects", nil)
	req.Host = "localhost:7412"
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	var out struct {
		Worktrees []projectItem `json:"worktrees"`
		Recent    []projectItem `json:"recent"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	return out.Worktrees, out.Recent
}

// 같은 저장소의 worktree를 브랜치·모드와 함께 보이고, 최근 목록에도 브랜치·모드를 붙인다.
func TestProjectsWorktrees(t *testing.T) {
	main, wt := worktreeRepo(t)
	s := New(filepath.Join(main, "design"), main)
	s.Poll()
	local(t, s)
	for _, p := range []string{wt, main} {
		body, _ := json.Marshal(map[string]string{"path": p})
		if code, out := post(t, s, "/api/open", string(body)); code != 200 {
			t.Fatalf("%d %v", code, out)
		}
	}
	if st := getState(t, s); st["branch"] != "main" || st["repo"] != "" {
		t.Fatalf("주 worktree: %v %v", st["branch"], st["repo"])
	}
	body, _ := json.Marshal(map[string]string{"path": wt})
	post(t, s, "/api/open", string(body))
	if st := getState(t, s); st["branch"] != "feat/daemok" || st["repo"] != filepath.Base(main) {
		t.Fatalf("다른 worktree는 저장소 이름을 준다: %v %v", st["branch"], st["repo"])
	}
	body, _ = json.Marshal(map[string]string{"path": main})
	post(t, s, "/api/open", string(body))
	wts, recent := getProjects(t, s)
	want := []projectItem{{main, true, "main", "analyze", true}, {wt, true, "feat/daemok", "check", false}}
	if len(wts) != 2 || wts[0] != want[0] || wts[1] != want[1] {
		t.Fatalf("worktrees %+v", wts)
	}
	if len(recent) != 2 || recent[0] != want[0] || recent[1] != want[1] {
		t.Fatalf("recent %+v", recent)
	}
}

// 저장소가 아니거나 worktree가 하나뿐이면 worktree 목록은 비어 있다.
func TestProjectsNoWorktrees(t *testing.T) {
	s, _ := setup(t, design)
	local(t, s)
	if wts, _ := getProjects(t, s); len(wts) != 0 {
		t.Fatalf("%+v", wts)
	}
}

// 경로의 ~는 홈 디렉토리다.
func TestOpenTilde(t *testing.T) {
	s, _ := setup(t, design)
	local(t, s)
	root := goModule(t)
	t.Setenv("HOME", filepath.Dir(root))
	body, _ := json.Marshal(map[string]string{"path": "~/" + filepath.Base(root)})
	if code, out := post(t, s, "/api/open", string(body)); code != 200 {
		t.Fatalf("%d %v", code, out)
	}
	if st := getState(t, s); st["root"] != root {
		t.Fatalf("%v", st["root"])
	}
	if code, out := post(t, s, "/api/open", `{"path":"~"}`); code != 400 || !strings.Contains(out["error"].(string), filepath.Dir(root)) {
		t.Fatalf("%d %v", code, out)
	}
}

// 분석 모드의 초안을 새 worktree에 저장하고 그 worktree를 연다. 원래 작업 폴더는 그대로다.
func TestInitWorktree(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git이 없다")
	}
	root := goModule(t)
	gitRun(t, root, "init", "-q", "-b", "main")
	gitRun(t, root, "add", ".")
	gitRun(t, root, "commit", "-q", "-m", "init")
	os.WriteFile(filepath.Join(root, "core", "extra.go"), []byte("package core\n"), 0o644)
	s := New(filepath.Join(root, "design"), root)
	s.Poll()
	local(t, s)
	st := getState(t, s)
	g, _ := st["draft"].(map[string]any)["git"].(map[string]any)
	if g == nil || g["dirty"] != float64(1) || g["branch"] != "daemok/design" {
		t.Fatalf("초안의 git 정보: %v", st["draft"].(map[string]any)["git"])
	}
	code, out := post(t, s, "/api/init-worktree", `{"branch":"daemok/design"}`)
	wt := filepath.Join(root, ".worktrees", "daemok-design")
	if code != 200 || out["root"] != wt || out["path"] != "design/demo.yaml" {
		t.Fatalf("%d %v", code, out)
	}
	if st := getState(t, s); st["mode"] != "check" || st["root"] != wt || st["branch"] != "daemok/design" {
		t.Fatalf("%v %v %v", st["mode"], st["root"], st["branch"])
	}
	if _, err := os.Stat(filepath.Join(root, "design")); !os.IsNotExist(err) {
		t.Fatal("원래 작업 폴더에 design/이 생겼다")
	}
	if code, out := post(t, s, "/api/init-worktree", `{"branch":"daemok/design"}`); code != 409 {
		t.Fatalf("대조 모드에서 다시: %d %v", code, out)
	}
	body, _ := json.Marshal(map[string]string{"path": root})
	post(t, s, "/api/open", string(body))
	if code, out := post(t, s, "/api/init-worktree", `{"branch":"daemok/design"}`); code != 400 || !strings.Contains(out["error"].(string), "이미 있는 브랜치") {
		t.Fatalf("이미 있는 브랜치: %d %v", code, out)
	}
}

// git 저장소가 아니면 초안에 git 정보가 없고 worktree 저장은 거부한다.
func TestInitWorktreeNeedsGit(t *testing.T) {
	root := goModule(t)
	s := New(filepath.Join(root, "design"), root)
	s.Poll()
	local(t, s)
	if g := getState(t, s)["draft"].(map[string]any)["git"]; g != nil {
		t.Fatalf("%v", g)
	}
	if code, _ := post(t, s, "/api/init-worktree", `{"branch":"x"}`); code != 400 {
		t.Fatalf("%d", code)
	}
	s.SetOptions(Options{Local: false})
	if code, _ := post(t, s, "/api/init-worktree", `{"branch":"x"}`); code != 403 {
		t.Fatalf("%d", code)
	}
}
