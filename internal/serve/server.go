// Package serve는 설계 지도를 로컬 웹 페이지로 보여준다. 계산은 서버가 하고 화면은 /api/state를 그린다.
package serve

import (
	"context"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/tolelom/daemok/internal/gitbase"
	"github.com/tolelom/daemok/internal/infer"
	"github.com/tolelom/daemok/internal/model"
	"github.com/tolelom/daemok/internal/workspace"
)

//go:embed web
var webFS embed.FS

// state는 /api/state의 본문이다.
type state struct {
	designSum string // 설계 파일 내용 요약. 내보내지 않는다.

	Version   int         `json:"version"`
	Design    string      `json:"design"`
	Root      string      `json:"root"`
	Branch    string      `json:"branch"` // git 브랜치. 저장소가 아니거나 detached면 비어 있다
	Repo      string      `json:"repo"`   // 주 worktree가 아닌 worktree면 주 worktree의 폴더 이름
	Model     *vModel     `json:"model"`
	Base      *vModel     `json:"base"`
	HasBase   bool        `json:"hasBase"`
	Issues    []vIssue    `json:"issues"`
	Relations []vRelation `json:"relations"`
	Files     []vFile     `json:"files"`
	Fatal     bool        `json:"fatal"`
	Updated   time.Time   `json:"updated"`
	Mode      string      `json:"mode"` // check, analyze, none
	Draft     *vDraft     `json:"draft,omitempty"`
	Problem   string      `json:"problem,omitempty"`
	CanEdit   bool        `json:"canEdit"` // 루프백이고 대조 모드면 화면에서 L2를 고칠 수 있다
}

// vDraft는 분석 모드의 초안이다.
type vDraft struct {
	Path    string       `json:"path"`
	YAML    string       `json:"yaml"`
	Report  infer.Report `json:"report"`
	Command string       `json:"command"`
}

// Server는 설계 디렉토리 하나의 지도를 제공한다.
type Server struct {
	design, root string

	mu      sync.Mutex
	body    []byte
	version int
	sig     string
	subs    map[chan int]bool
	opts    Options
	mode    string // 마지막으로 낸 상태의 모드
	editMu  sync.Mutex

	// 설계 파일 내용의 요약과, 그 내용이 처음 나간 상태 버전. 편집의 409 판정에 쓴다.
	designSum   string
	designSince int

	// refreshMu는 상태 만들기를 한 번에 하나로 한다. 늦게 끝난 옛 상태가 새 버전 번호로 나가지 않게 한다.
	refreshMu sync.Mutex
}

func New(design, root string) *Server {
	// 화면이 프로젝트 이름을 보이도록 절대경로로 둔다. 실패하면 받은 그대로 쓴다.
	if abs, err := filepath.Abs(root); err == nil {
		root = abs
	}
	// 첫 Poll이 반드시 상태를 만들도록 서명을 어떤 실제 서명과도 다른 값으로 둔다.
	return &Server{design: design, root: root, subs: map[chan int]bool{}, sig: "\x00"}
}

// Refresh는 설계나 코드를 다시 읽고 상태를 새로 만든 뒤 구독자에게 알린다.
// 설계 파일이 있으면 대조 모드, 없고 코드 루트가 Go 모듈이면 분석 모드다.
func (s *Server) Refresh() {
	s.refreshMu.Lock()
	defer s.refreshMu.Unlock()
	s.refresh()
}

func (s *Server) refresh() {
	design, root := s.paths()
	st := state{Design: design, Root: root, Updated: time.Now(), designSum: designSum(design)}
	st.Branch, st.Repo = gitbase.Branch(root), repoName(root)
	if fi, err := os.Stat(root); err != nil || !fi.IsDir() {
		st.Mode, st.Fatal, st.Problem = "none", true, "코드 루트가 없다: "+root
	} else if infer.HasDesignFiles(design) {
		st.Mode = "check"
		fill(&st, workspace.Run(design, root), root)
		if files, ok, err := gitbase.Files(design); err == nil && ok {
			base, _ := model.LoadSources(files)
			st.Base, st.HasBase = toView(base), true
		}
	} else if a, err := infer.Analyze(root); err == nil {
		st.Mode = "analyze"
		name := filepath.Join(design, a.Draft.System+".yaml")
		fill(&st, workspace.RunSources(map[string][]byte{name: a.YAML}, design, root), root)
		st.Draft = &vDraft{Path: name, YAML: string(a.YAML), Report: a.Report, Command: "daemok init --root " + shellQuote(root)}
	} else {
		st.Mode, st.Fatal, st.Problem = "none", true, err.Error()
	}
	st.CanEdit = st.Mode == "check" && s.options().Local
	refreshBuilt()
	s.publish(st)
}

// refreshBuilt는 상태를 다 만들고 내기 직전에 불린다. 테스트가 상태 만들기를 겹치게 하려고 바꾼다.
var refreshBuilt = func() {}

// designSum은 설계 디렉토리 *.yaml의 이름과 내용 요약이다.
func designSum(design string) string {
	h := sha256.New()
	paths, _ := filepath.Glob(filepath.Join(design, "*.yaml"))
	sort.Strings(paths)
	for _, p := range paths {
		b, _ := os.ReadFile(p)
		fmt.Fprintf(h, "%s\x00%d\x00", filepath.Base(p), len(b))
		h.Write(b)
	}
	return hex.EncodeToString(h.Sum(nil))
}

// designVersions는 지금 설계 내용이 처음 나간 버전과 마지막 버전이다.
func (s *Server) designVersions() (since, now int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.designSince, s.version
}

func (s *Server) currentMode() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.mode
}

func fill(st *state, r workspace.Result, root string) {
	st.Model, st.Fatal = toView(r.Model), r.Fatal
	st.Issues = toIssues(r.Model, root, r.Diags, r.Code)
	st.Files = toFiles(r.Model, r.Files)
	st.Relations = toRelations(r.Model)
}

// shellQuote는 공백이 있는 경로를 큰따옴표로 감싼다.
func shellQuote(p string) string {
	if strings.ContainsAny(p, " \t") {
		return `"` + p + `"`
	}
	return p
}

func (s *Server) paths() (design, root string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.design, s.root
}

// publish는 상태에 버전을 붙여 저장하고 구독자에게 알린다.
// 그사이 다른 프로젝트를 열었으면 버린다. 늦게 끝난 옛 프로젝트의 상태가 새 상태를 덮지 않게 한다.
func (s *Server) publish(st state) {
	s.mu.Lock()
	if st.Design != s.design || st.Root != s.root {
		s.mu.Unlock()
		return
	}
	s.version++
	s.mode = st.Mode
	if st.designSum != s.designSum {
		s.designSum, s.designSince = st.designSum, s.version
	}
	st.Version = s.version
	body, err := json.Marshal(st)
	if err != nil {
		body = []byte(fmt.Sprintf(`{"version":%d,"error":%q}`, s.version, err.Error()))
	}
	s.body = body
	v := s.version
	subs := make([]chan int, 0, len(s.subs))
	for ch := range s.subs {
		subs = append(subs, ch)
	}
	s.mu.Unlock()
	for _, ch := range subs {
		select {
		case ch <- v:
		default:
		}
	}
}

// Version은 마지막 상태의 번호다.
func (s *Server) Version() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.version
}

// signature는 설계 디렉토리의 *.yaml과 코드 루트의 .go 파일의 이름·크기·수정 시각이다.
// 파일 추가·삭제·수정을 잡는다. 코드 쪽은 생성 파일 범위와 같은 디렉토리만 본다.
func signature(design, root string) string {
	var b strings.Builder
	paths, _ := filepath.Glob(filepath.Join(design, "*.yaml"))
	sort.Strings(paths)
	for _, p := range paths {
		if st, err := os.Stat(p); err == nil {
			fmt.Fprintf(&b, "d|%s|%d|%d\n", filepath.Base(p), st.Size(), st.ModTime().UnixNano())
		}
	}
	filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if p != root && workspace.SkipDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(p, ".go") {
			return nil
		}
		if info, err := d.Info(); err == nil {
			fmt.Fprintf(&b, "c|%s|%d|%d\n", p, info.Size(), info.ModTime().UnixNano())
		}
		return nil
	})
	return b.String()
}

// Poll은 설계나 코드가 바뀌었으면 Refresh하고 true를 준다. 도중에 프로젝트가 바뀌면 아무것도 하지 않는다.
func (s *Server) Poll() bool {
	s.refreshMu.Lock()
	defer s.refreshMu.Unlock()
	design, root := s.paths()
	sig := signature(design, root)
	s.mu.Lock()
	if s.design != design || s.root != root {
		s.mu.Unlock()
		return false
	}
	changed := sig != s.sig
	s.sig = sig
	s.mu.Unlock()
	if changed {
		s.refresh()
	}
	return changed
}

// Watch는 ctx가 끝날 때까지 interval마다 Poll한다.
func (s *Server) Watch(ctx context.Context, interval time.Duration) {
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			s.Poll()
		}
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	web, _ := fs.Sub(webFS, "web")
	mux.Handle("/", http.FileServer(http.FS(web)))
	mux.HandleFunc("/api/state", func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		body := s.body
		s.mu.Unlock()
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		w.Write(body)
	})
	mux.HandleFunc("/api/events", s.events)
	mux.HandleFunc("GET /api/projects", s.projects)
	mux.HandleFunc("POST /api/open", s.open)
	mux.HandleFunc("POST /api/init", s.initDraft)
	mux.HandleFunc("POST /api/edit", s.editDesign)
	mux.HandleFunc("GET /api/l3", s.l3Draft)
	mux.HandleFunc("POST /api/l3", s.l3Save)
	mux.HandleFunc("GET /api/gen", s.genPreview)
	mux.HandleFunc("GET /api/gen/file", s.genContent)
	return s.guard(mux)
}

// events는 상태가 바뀔 때마다 버전 번호를 SSE로 보낸다.
func (s *Server) events(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "스트리밍을 지원하지 않는다", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-store")
	ch := make(chan int, 1)
	s.mu.Lock()
	s.subs[ch] = true
	v := s.version
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		delete(s.subs, ch)
		s.mu.Unlock()
	}()
	fmt.Fprintf(w, "data: %d\n\n", v)
	flusher.Flush()
	for {
		select {
		case <-r.Context().Done():
			return
		case v := <-ch:
			fmt.Fprintf(w, "data: %d\n\n", v)
			flusher.Flush()
		}
	}
}
