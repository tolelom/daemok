// Package serve는 설계 지도를 로컬 웹 페이지로 보여준다. 계산은 서버가 하고 화면은 /api/state를 그린다.
package serve

import (
	"context"
	"embed"
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
	"github.com/tolelom/daemok/internal/model"
	"github.com/tolelom/daemok/internal/workspace"
)

//go:embed web
var webFS embed.FS

// state는 /api/state의 본문이다.
type state struct {
	Version int       `json:"version"`
	Design  string    `json:"design"`
	Root    string    `json:"root"`
	Model   *vModel   `json:"model"`
	Base    *vModel   `json:"base"`
	HasBase bool      `json:"hasBase"`
	Issues  []vIssue  `json:"issues"`
	Files   []vFile   `json:"files"`
	Fatal   bool      `json:"fatal"`
	Updated time.Time `json:"updated"`
}

// Server는 설계 디렉토리 하나의 지도를 제공한다.
type Server struct {
	design, root string

	mu      sync.Mutex
	body    []byte
	version int
	sig     string
	subs    map[chan int]bool
}

func New(design, root string) *Server {
	return &Server{design: design, root: root, subs: map[chan int]bool{}}
}

// Refresh는 설계를 다시 읽고 상태를 새로 만든 뒤 구독자에게 알린다.
func (s *Server) Refresh() {
	r := workspace.Run(s.design, s.root)
	st := state{Design: s.design, Root: s.root, Model: toView(r.Model), Fatal: r.Fatal, Updated: time.Now()}
	st.Issues = toIssues(r.Model, s.root, r.Diags)
	st.Files = toFiles(r.Model, r.Files)
	if files, ok, err := gitbase.Files(s.design); err == nil && ok {
		base, _ := model.LoadSources(files)
		st.Base, st.HasBase = toView(base), true
	}
	s.mu.Lock()
	s.version++
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

// signature는 설계 디렉토리의 *.yaml 이름·크기·수정 시각이다. 파일 추가·삭제·수정을 잡는다.
func (s *Server) signature() string {
	paths, _ := filepath.Glob(filepath.Join(s.design, "*.yaml"))
	sort.Strings(paths)
	var b strings.Builder
	for _, p := range paths {
		st, err := os.Stat(p)
		if err != nil {
			continue
		}
		fmt.Fprintf(&b, "%s|%d|%d\n", filepath.Base(p), st.Size(), st.ModTime().UnixNano())
	}
	return b.String()
}

// Poll은 설계 디렉토리가 바뀌었으면 Refresh하고 true를 준다.
func (s *Server) Poll() bool {
	sig := s.signature()
	s.mu.Lock()
	changed := sig != s.sig
	s.sig = sig
	s.mu.Unlock()
	if changed {
		s.Refresh()
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
	return mux
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
