package serve

import (
	"bufio"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const design = `version: 0.5
go_module: example.com/x
tiers: [a, b]
modules:
  core: { tier: a }
  api:  { tier: b, depends_on: [core] }
enums:
  kind: { in: core, values: [x, y] }
`

func setup(t *testing.T, src string) (*Server, string) {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "design")
	os.MkdirAll(dir, 0o755)
	os.WriteFile(filepath.Join(dir, "d.yaml"), []byte(src), 0o644)
	s := New(dir, root)
	s.Poll()
	return s, dir
}

func getState(t *testing.T, s *Server) map[string]any {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/state", nil)
	req.Host = "127.0.0.1:7412"
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("status %d", rec.Code)
	}
	var st map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &st); err != nil {
		t.Fatal(err)
	}
	return st
}

func TestState(t *testing.T) {
	s, _ := setup(t, design)
	st := getState(t, s)
	if st["mode"] != "check" {
		t.Fatalf("mode %v", st["mode"])
	}
	mods := st["model"].(map[string]any)["mods"].([]any)
	if len(mods) != 2 || mods[0].(map[string]any)["id"] != "core" || mods[1].(map[string]any)["id"] != "api" {
		t.Fatalf("mods 순서: %v", mods)
	}
	files := st["files"].([]any)
	if len(files) != 1 {
		t.Fatalf("files %v", files)
	}
	f := files[0].(map[string]any)
	if f["path"] != "core/kind.gen.go" || f["sources"].([]any)[0] != "enum:kind" {
		t.Fatalf("file %v", f)
	}
	// 생성 파일이 없다는 진단이 enum 노드가 아니라 경로로 나온다
	found := false
	for _, it := range st["issues"].([]any) {
		i := it.(map[string]any)
		if strings.Contains(i["msg"].(string), "생성 파일이 없다") {
			found = true
		}
	}
	if !found {
		t.Fatalf("issues %v", st["issues"])
	}
	if st["hasBase"] != false {
		t.Fatal("git 저장소가 아니면 기준이 없다")
	}
}

func TestIssueKeyForDesignDiag(t *testing.T) {
	s, _ := setup(t, strings.Replace(design, "depends_on: [core]", "depends_on: [nope]", 1))
	for _, it := range getState(t, s)["issues"].([]any) {
		i := it.(map[string]any)
		if strings.Contains(i["msg"].(string), `없는 모듈 "nope"`) {
			if i["key"] != "mod:api" || i["origin"] != "design" {
				t.Fatalf("%v", i)
			}
			return
		}
	}
	t.Fatal("진단이 없다")
}

func TestPollAndBrokenFile(t *testing.T) {
	s, dir := setup(t, design)
	v := s.Version()
	if s.Poll() {
		t.Fatal("바뀐 것이 없는데 갱신했다")
	}
	time.Sleep(10 * time.Millisecond)
	os.WriteFile(filepath.Join(dir, "d.yaml"), []byte("version: 0.5\nmodules: [\n"), 0o644)
	if !s.Poll() || s.Version() != v+1 {
		t.Fatal("수정을 감지하지 못했다")
	}
	st := getState(t, s)
	if st["fatal"] != true || len(st["issues"].([]any)) == 0 {
		t.Fatalf("문법 오류는 진단으로 보여야 한다: %v", st)
	}
	os.WriteFile(filepath.Join(dir, "e.yaml"), []byte("version: 0.5\n"), 0o644)
	if !s.Poll() {
		t.Fatal("파일 추가를 감지하지 못했다")
	}
}

func TestEventsStream(t *testing.T) {
	s, dir := setup(t, design)
	srv := httptest.NewServer(s.Handler())
	defer srv.Close()
	resp, err := http.Get(srv.URL + "/api/events")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	lines := make(chan string, 10)
	go func() {
		sc := bufio.NewScanner(resp.Body)
		for sc.Scan() {
			if strings.HasPrefix(sc.Text(), "data: ") {
				lines <- strings.TrimPrefix(sc.Text(), "data: ")
			}
		}
	}()
	<-lines // 연결 직후의 현재 버전
	os.WriteFile(filepath.Join(dir, "d.yaml"), []byte(design+"\n"), 0o644)
	s.Poll()
	select {
	case got := <-lines:
		if got == "" {
			t.Fatal("빈 이벤트")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("변경 이벤트가 오지 않았다")
	}
}

func TestIndexServed(t *testing.T) {
	s, _ := setup(t, design)
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), "daemok 설계 지도") {
		t.Fatalf("%d", rec.Code)
	}
}

func TestFirstStateWithoutDesignFiles(t *testing.T) {
	root := t.TempDir()
	s := New(filepath.Join(root, "design"), root)
	s.Poll()
	if s.Version() != 1 {
		t.Fatal("설계 파일이 없어도 첫 상태를 만들어야 한다")
	}
	if st := getState(t, s); st["fatal"] != true {
		t.Fatalf("%v", st)
	}
}

func TestPollSeesCodeChanges(t *testing.T) {
	s, dir := setup(t, design)
	root := filepath.Dir(dir)
	os.MkdirAll(filepath.Join(root, "core"), 0o755)
	os.WriteFile(filepath.Join(root, "core", "x.go"), []byte("package core\n"), 0o644)
	if !s.Poll() {
		t.Fatal("코드 파일 추가를 감지하지 못했다")
	}
	os.MkdirAll(filepath.Join(root, "testdata"), 0o755)
	os.WriteFile(filepath.Join(root, "testdata", "y.go"), []byte("package y\n"), 0o644)
	if s.Poll() {
		t.Fatal("범위 밖 디렉토리 변경으로 갱신했다")
	}
}

func TestGroupWithoutMembersHasEmptyList(t *testing.T) {
	s, _ := setup(t, design+"groups:\n  g: { tier: a, kind: k, members: {} }\n")
	for _, g := range getState(t, s)["model"].(map[string]any)["groups"].([]any) {
		if _, ok := g.(map[string]any)["members"].([]any); !ok {
			t.Fatalf("members가 배열이 아니다: %v", g)
		}
	}
}

func TestQueryPackageDiagHasModuleKey(t *testing.T) {
	src := `version: 0.5
go_module: example.com/x
tiers: [a, b]
modules:
  core: { tier: a }
  vm:   { tier: b, depends_on: [core] }
  api:  { tier: b, needs: [reader] }
contracts:
  reader: { in: core, kind: query, methods: { get: { errors: [] } } }
`
	s, dir := setup(t, src)
	root := filepath.Dir(dir)
	os.MkdirAll(filepath.Join(root, "core", "query"), 0o755)
	os.WriteFile(filepath.Join(root, "core", "query", "q.go"), []byte("package query\n\nimport _ \"example.com/x/vm\"\n"), 0o644)
	s.Poll()
	for _, it := range getState(t, s)["issues"].([]any) {
		i := it.(map[string]any)
		if strings.Contains(i["msg"].(string), "query 패키지가") {
			if i["key"] != "mod:core" || i["origin"] != "code" {
				t.Fatalf("%v", i)
			}
			return
		}
	}
	t.Fatal("진단이 없다")
}

func TestCodeScanDiagOnDesignFileIsCodeOrigin(t *testing.T) {
	s, dir := setup(t, design)
	root := filepath.Dir(dir)
	os.MkdirAll(filepath.Join(root, "api"), 0o755)
	os.WriteFile(filepath.Join(root, "api", "api.go"), []byte("package api\n"), 0o644)
	s.Poll()
	for _, it := range getState(t, s)["issues"].([]any) {
		i := it.(map[string]any)
		if strings.Contains(i["msg"].(string), "설계상 의존하지만 import하지 않는다") {
			if i["origin"] != "code" {
				t.Fatalf("%v", i)
			}
			return
		}
	}
	t.Fatal("진단이 없다")
}

const relDesign = `version: 0.5
go_module: example.com/x
tiers: [a, b, c]
modules:
  core:  { tier: a }
  store: { tier: a, role: infra }
  vm:    { tier: b, depends_on: [core] }
  api:   { tier: c, depends_on: [core], needs: [{ contract: kv, uses: [get], why: "캐시" }] }
groups:
  mods:
    tier: b
    kind: h
    depends_on: [core, vm]
    members:
      m1: { handles: [x], exceptions: [{ depends_on: m2, reason: "재사용" }] }
      m2: { handles: [y] }
enums:
  hkey: { in: core, values: [x, y] }
  ev_type: { in: core, values: [done] }
handler_kinds:
  h: { in: vm, key: hkey, params: {}, returns: [error] }
contracts:
  kv: { in: store, kind: port, methods: { get: { errors: [] } } }
event_kinds:
  evk: { in: core, key: ev_type, subscribers: [api] }
events:
  done: { kind: evk, publishers: [m1], subscribers: [vm] }
flows:
  f:
    in: api
    steps:
      - { id: a, undo: ua }
      - { id: b, undo: ub }
      - { id: c, commit: true, fatal: true }
      - { id: d }
`

func TestRelations(t *testing.T) {
	s, _ := setup(t, relDesign)
	var got []string
	for _, it := range getState(t, s)["relations"].([]any) {
		r := it.(map[string]any)
		got = append(got, r["kind"].(string)+" "+r["from"].(string)+">"+r["to"].(string)+" "+r["label"].(string))
	}
	for _, want := range []string{
		"depends vm>core ",
		"depends mods>core ",
		"exception m1>m2 재사용",
		"port api>store kv",
		"event m1>vm done",
		"event m1>api done",
		"register m1>vm h",
		"register m2>vm h",
	} {
		found := false
		for _, g := range got {
			if g == want {
				found = true
			}
		}
		if !found {
			t.Errorf("관계 없음: %q\n%v", want, got)
		}
	}
}

func TestFlowFailurePaths(t *testing.T) {
	s, _ := setup(t, relDesign)
	flows := getState(t, s)["model"].(map[string]any)["flows"].([]any)
	steps := flows[0].(map[string]any)["steps"].([]any)
	want := []struct {
		undo  string
		fatal bool
	}{{"", false}, {"ua", false}, {"", true}, {"", false}}
	for i, st := range steps {
		m := st.(map[string]any)
		var undo []string
		for _, u := range m["onFailUndo"].([]any) {
			undo = append(undo, u.(string))
		}
		if strings.Join(undo, ",") != want[i].undo || m["onFailFatal"].(bool) != want[i].fatal {
			t.Errorf("step %s: undo %v fatal %v", m["id"], undo, m["onFailFatal"])
		}
	}
}

func goModule(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/demo\n"), 0o644)
	os.MkdirAll(filepath.Join(root, "core"), 0o755)
	os.WriteFile(filepath.Join(root, "core", "c.go"), []byte("package core\n"), 0o644)
	os.MkdirAll(filepath.Join(root, "api"), 0o755)
	os.WriteFile(filepath.Join(root, "api", "a.go"), []byte("package api\nimport _ \"example.com/demo/core\"\n"), 0o644)
	return root
}

func TestAnalyzeMode(t *testing.T) {
	root := goModule(t)
	s := New(filepath.Join(root, "design"), root)
	s.Poll()
	st := getState(t, s)
	if st["mode"] != "analyze" || st["fatal"] != false {
		t.Fatalf("%v %v", st["mode"], st["fatal"])
	}
	mods := st["model"].(map[string]any)["mods"].([]any)
	if len(mods) != 2 {
		t.Fatalf("%v", mods)
	}
	d := st["draft"].(map[string]any)
	if !strings.Contains(d["yaml"].(string), "api: { tier: tier1, depends_on: [core] }") {
		t.Fatal(d["yaml"])
	}
	if d["report"].(map[string]any)["modules"].(float64) != 2 || !strings.HasPrefix(d["command"].(string), "daemok init --root ") {
		t.Fatalf("%v", d)
	}
	for _, it := range st["issues"].([]any) {
		if it.(map[string]any)["lv"] == "err" {
			t.Fatalf("초안에 오류: %v", it)
		}
	}
}

func TestAnalyzeModeSeesCodeChange(t *testing.T) {
	root := goModule(t)
	s := New(filepath.Join(root, "design"), root)
	s.Poll()
	os.MkdirAll(filepath.Join(root, "web"), 0o755)
	os.WriteFile(filepath.Join(root, "web", "w.go"), []byte("package web\n"), 0o644)
	if !s.Poll() {
		t.Fatal("코드 변경을 감지하지 못했다")
	}
	if n := len(getState(t, s)["model"].(map[string]any)["mods"].([]any)); n != 3 {
		t.Fatalf("%d", n)
	}
}

func TestNoneMode(t *testing.T) {
	root := t.TempDir()
	s := New(filepath.Join(root, "design"), root)
	s.Poll()
	st := getState(t, s)
	if st["mode"] != "none" || st["fatal"] != true || !strings.Contains(st["problem"].(string), "Go 모듈이 아니다") {
		t.Fatalf("%v", st)
	}
}

func TestIssueGroupIsMessageKind(t *testing.T) {
	s, _ := setup(t, design)
	for _, it := range getState(t, s)["issues"].([]any) {
		i := it.(map[string]any)
		if strings.HasPrefix(i["msg"].(string), "생성 파일이 없다") && i["group"] != "생성 파일이 없다" {
			t.Fatalf("%v", i)
		}
		if i["group"] == "" || i["group"] == nil {
			t.Fatalf("묶음 키가 없다: %v", i)
		}
	}
}

func TestIssueHasAbsolutePath(t *testing.T) {
	s, _ := setup(t, design)
	for _, it := range getState(t, s)["issues"].([]any) {
		i := it.(map[string]any)
		if i["file"] != "" && !filepath.IsAbs(i["abs"].(string)) {
			t.Fatalf("%v", i)
		}
	}
}

func TestMissingRootProblem(t *testing.T) {
	root := filepath.Join(t.TempDir(), "nope")
	s := New(filepath.Join(root, "design"), root)
	s.Poll()
	st := getState(t, s)
	if p, _ := st["problem"].(string); !strings.Contains(p, "코드 루트가 없다") {
		t.Fatalf("%v", st["problem"])
	}
}
