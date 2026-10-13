package serve

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/tolelom/daemok/internal/gitbase"
	"github.com/tolelom/daemok/internal/gosrc"
	"github.com/tolelom/daemok/internal/infer"
)

// Options는 프로젝트 열기에 쓰는 설정이다.
type Options struct {
	Workspace string // 프로젝트를 찾을 작업 폴더. 비면 목록이 없다
	ConfigDir string // recent.json을 둘 디렉토리. 비면 저장하지 않는다
	Local     bool   // 루프백 주소에서 들을 때 참. 참일 때만 열기·저장을 허용하고 Host·Origin을 검사한다
}

const recentMax = 10

// SetOptions는 설정을 바꾸고 상태를 다시 만든다. 상태의 canEdit이 설정을 따른다.
func (s *Server) SetOptions(o Options) {
	s.mu.Lock()
	s.opts = o
	s.mu.Unlock()
	s.Refresh()
}

func (s *Server) options() Options {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.opts
}

// Open은 path를 현재 프로젝트로 연다. 설계 디렉토리는 path/design이다.
// 디렉토리가 아니거나 설계 파일과 go.mod가 모두 없으면 바꾸지 않고 오류를 준다.
func (s *Server) Open(path string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return errors.New("경로가 없다")
	}
	path = expandHome(path)
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	if st, err := os.Stat(abs); err != nil || !st.IsDir() {
		return fmt.Errorf("디렉토리가 아니다: %s", abs)
	}
	design := filepath.Join(abs, "design")
	if !infer.HasDesignFiles(design) {
		if _, err := os.Stat(filepath.Join(abs, "go.mod")); err != nil {
			return fmt.Errorf("설계 파일도 go.mod도 없다: %s", abs)
		}
	}
	s.mu.Lock()
	s.design, s.root, s.sig = design, abs, "\x00"
	s.mu.Unlock()
	s.remember(abs)
	s.Poll()
	return nil
}

// expandHome은 ~와 ~/로 시작하는 경로의 ~를 홈 디렉토리로 바꾼다.
func expandHome(path string) string {
	if path != "~" && !strings.HasPrefix(path, "~/") && !strings.HasPrefix(path, "~"+string(filepath.Separator)) {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	return filepath.Join(home, path[1:])
}

func (s *Server) recentPath() string {
	if d := s.options().ConfigDir; d != "" {
		return filepath.Join(d, "recent.json")
	}
	return ""
}

func (s *Server) recent() []string {
	p := s.recentPath()
	if p == "" {
		return nil
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return nil
	}
	var list []string
	if json.Unmarshal(b, &list) != nil {
		return nil
	}
	return list
}

func (s *Server) remember(path string) {
	p := s.recentPath()
	if p == "" {
		return
	}
	list := []string{path}
	for _, x := range s.recent() {
		if x != path && len(list) < recentMax {
			list = append(list, x)
		}
	}
	b, _ := json.Marshal(list)
	if os.MkdirAll(filepath.Dir(p), 0o755) == nil {
		os.WriteFile(p, b, 0o644)
	}
}

// findModules는 dir 아래 깊이 maxDepth까지에서 go.mod가 있는 디렉토리를 찾는다. 모듈 아래로는 내려가지 않는다.
func findModules(dir string, maxDepth int) []string {
	var out []string
	base := filepath.Clean(dir)
	filepath.WalkDir(base, func(p string, d fs.DirEntry, err error) error {
		if err != nil || !d.IsDir() {
			return nil
		}
		if p != base && (gosrc.SkipDir(d.Name()) || strings.HasPrefix(d.Name(), "_")) {
			return filepath.SkipDir
		}
		if _, err := os.Stat(filepath.Join(p, "go.mod")); err == nil {
			out = append(out, p)
			return filepath.SkipDir
		}
		rel, _ := filepath.Rel(base, p)
		if rel != "." && len(strings.Split(filepath.ToSlash(rel), "/")) >= maxDepth {
			return filepath.SkipDir
		}
		return nil
	})
	return out
}

type vProject struct {
	Path    string `json:"path"`
	Exists  bool   `json:"exists"`
	Branch  string `json:"branch"`
	Mode    string `json:"mode"` // check(설계 있음), analyze(go.mod만), 없거나 열 수 없으면 빈 값
	Current bool   `json:"current"`
}

// project는 경로 하나의 목록 항목이다. 브랜치를 모르면 branch를 git으로 읽는다.
func project(path, branch, current string, branchKnown bool) vProject {
	p := vProject{Path: path, Current: samePath(path, current)}
	st, err := os.Stat(path)
	if err != nil || !st.IsDir() {
		return p
	}
	p.Exists = true
	switch {
	case infer.HasDesignFiles(filepath.Join(path, "design")):
		p.Mode = "check"
	case fileExists(filepath.Join(path, "go.mod")):
		p.Mode = "analyze"
	}
	if branchKnown {
		p.Branch = branch
	} else {
		p.Branch = gitbase.Branch(path)
	}
	return p
}

// repoName은 root가 주 worktree가 아닌 worktree면 주 worktree의 폴더 이름을 준다. worktree 폴더 이름은
// 저장소를 알려 주지 못한다(.worktrees/daemok).
func repoName(root string) string {
	wts, _ := gitbase.Worktrees(root)
	if len(wts) < 2 || samePath(wts[0].Path, root) {
		return ""
	}
	return filepath.Base(wts[0].Path)
}

// samePath는 심볼릭 링크를 푼 두 경로가 같은지 본다. git은 링크를 푼 경로를 준다.
func samePath(a, b string) bool {
	if a == b {
		return true
	}
	ra, err1 := filepath.EvalSymlinks(a)
	rb, err2 := filepath.EvalSymlinks(b)
	return err1 == nil && err2 == nil && ra == rb
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

func (s *Server) projects(w http.ResponseWriter, r *http.Request) {
	o := s.options()
	_, root := s.paths()
	body := struct {
		Current   string     `json:"current"`
		Worktrees []vProject `json:"worktrees"`
		Recent    []vProject `json:"recent"`
		Workspace []string   `json:"workspace"`
		CanOpen   bool       `json:"canOpen"`
	}{Current: root, Worktrees: []vProject{}, Recent: []vProject{}, Workspace: []string{}, CanOpen: o.Local}
	// worktree가 하나뿐이면 지금 연 것뿐이라 보이지 않는다.
	if wts, _ := gitbase.Worktrees(root); len(wts) > 1 {
		for _, wt := range wts {
			body.Worktrees = append(body.Worktrees, project(wt.Path, wt.Branch, root, true))
		}
	}
	for _, p := range s.recent() {
		body.Recent = append(body.Recent, project(p, "", root, false))
	}
	if o.Workspace != "" {
		body.Workspace = append(body.Workspace, findModules(o.Workspace, 3)...)
	}
	writeJSON(w, http.StatusOK, body)
}

func (s *Server) open(w http.ResponseWriter, r *http.Request) {
	if !s.options().Local {
		writeErr(w, http.StatusForbidden, "루프백 주소로 띄웠을 때만 열 수 있다")
		return
	}
	var req struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "요청 본문을 읽을 수 없다")
		return
	}
	if err := s.Open(req.Path); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	_, root := s.paths()
	writeJSON(w, http.StatusOK, map[string]string{"root": root})
}

func (s *Server) initDraft(w http.ResponseWriter, r *http.Request) {
	if !s.options().Local {
		writeErr(w, http.StatusForbidden, "루프백 주소로 띄웠을 때만 저장할 수 있다")
		return
	}
	design, root := s.paths()
	if infer.HasDesignFiles(design) {
		writeErr(w, http.StatusConflict, "design/에 이미 설계 파일이 있다")
		return
	}
	a, err := infer.Analyze(root)
	if err != nil {
		writeErr(w, http.StatusConflict, err.Error())
		return
	}
	p, err := infer.Write(design, a)
	switch {
	case errors.Is(err, infer.ErrDesignExists):
		writeErr(w, http.StatusConflict, "design/에 이미 설계 파일이 있다")
		return
	case err != nil:
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.Poll()
	// 화면 알림이 짧도록 코드 루트 기준 슬래시 경로로 준다.
	if rel, err := filepath.Rel(root, p); err == nil {
		p = filepath.ToSlash(rel)
	}
	writeJSON(w, http.StatusOK, map[string]string{"path": p})
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}
