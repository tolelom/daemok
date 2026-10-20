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

type state struct {
	designSum string

	Version   int         `json:"version"`
	Design    string      `json:"design"`
	Root      string      `json:"root"`
	Branch    string      `json:"branch"`
	Repo      string      `json:"repo"`
	Model     *vModel     `json:"model"`
	Base      *vModel     `json:"base"`
	HasBase   bool        `json:"hasBase"`
	Issues    []vIssue    `json:"issues"`
	Relations []vRelation `json:"relations"`
	Files     []vFile     `json:"files"`
	Fatal     bool        `json:"fatal"`
	Updated   time.Time   `json:"updated"`
	Mode      string      `json:"mode"`
	Draft     *vDraft     `json:"draft,omitempty"`
	Problem   string      `json:"problem,omitempty"`
	CanEdit   bool        `json:"canEdit"`
}

type vDraft struct {
	Path    string       `json:"path"`
	YAML    string       `json:"yaml"`
	Report  infer.Report `json:"report"`
	Command string       `json:"command"`
	Git     *vDraftGit   `json:"git,omitempty"`
}

type vDraftGit struct {
	Branch string `json:"branch"`
	Dirty  int    `json:"dirty"`
}

type Server struct {
	design, root string

	mu      sync.Mutex
	body    []byte
	version int
	sig     string
	subs    map[chan int]bool
	opts    Options
	mode    string
	editMu  sync.Mutex

	designSum   string
	designSince int

	refreshMu sync.Mutex
}

func New(design, root string) *Server {
	if abs, err := filepath.Abs(root); err == nil {
		root = abs
	}
	return &Server{design: design, root: root, subs: map[chan int]bool{}, sig: "\x00"}
}

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
		if files, ok, err := gitbase.Files(design, "HEAD"); err == nil && ok {
			base, _ := model.LoadSources(files)
			st.Base, st.HasBase = toView(base), true
		}
	} else if a, err := infer.Analyze(root); err == nil {
		st.Mode = "analyze"
		name := filepath.Join(design, a.Draft.System+".yaml")
		fill(&st, workspace.RunSources(map[string][]byte{name: a.YAML}, design, root), root)
		st.Draft = &vDraft{Path: name, YAML: string(a.YAML), Report: a.Report, Command: "daemok init --root " + shellQuote(root)}
		if s.options().Local && gitbase.HasHead(root) {
			st.Draft.Git = &vDraftGit{Branch: draftBranch, Dirty: gitbase.Dirty(root)}
		}
	} else {
		st.Mode, st.Fatal, st.Problem = "none", true, err.Error()
	}
	st.CanEdit = st.Mode == "check" && s.options().Local
	refreshBuilt()
	s.publish(st)
}

var refreshBuilt = func() {}

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
	st.Issues = toIssues(r.Model, root, r.Diagnostics, r.Code)
	st.Files = toFiles(r.Model, r.Files)
	st.Relations = toRelations(r.Model)
}

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

func (s *Server) Version() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.version
}

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
	mux.HandleFunc("POST /api/init-worktree", s.initWorktree)
	mux.HandleFunc("POST /api/edit", s.editDesign)
	mux.HandleFunc("GET /api/l3", s.l3Draft)
	mux.HandleFunc("POST /api/l3", s.l3Save)
	mux.HandleFunc("GET /api/gen", s.genPreview)
	mux.HandleFunc("GET /api/gen/file", s.genContent)
	return s.guard(mux)
}

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
