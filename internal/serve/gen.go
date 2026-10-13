package serve

import (
	"net/http"
	"os"
	"path/filepath"
	"sort"

	"github.com/tolelom/daemok/internal/workspace"
)

// genFile은 생성 미리보기의 파일 하나다. Status는 same(그대로), new(새로 씀), changed(내용이 바뀜),
// delete(설계에 없어 지움), refuse(생성 경로에 생성 헤더 없는 파일 등이 있어 쓰지 않음)다.
type genFile struct {
	Path   string `json:"path"`
	Status string `json:"status"`
	Reason string `json:"reason,omitempty"`
}

// genPreview는 daemok gen이 할 일을 파일마다 준다. 쓰지 않는다. 설계 대조를 다시 돌리므로 요청할 때만 만든다.
func (s *Server) genPreview(w http.ResponseWriter, r *http.Request) {
	res, ok := s.genRun(w)
	if !ok {
		return
	}
	p := res.Plan
	status := map[string]genFile{}
	for _, f := range res.Files {
		status[f.Path] = genFile{Path: f.Path, Status: "same"}
	}
	_, root := s.paths()
	for _, f := range p.Write {
		st := "changed"
		if _, err := os.Lstat(filepath.Join(root, filepath.FromSlash(f.Path))); err != nil {
			st = "new"
		}
		status[f.Path] = genFile{Path: f.Path, Status: st}
	}
	for _, d := range p.Delete {
		status[d] = genFile{Path: d, Status: "delete"}
	}
	for _, f := range p.Refuse {
		status[f.Path] = genFile{Path: f.Path, Status: "refuse", Reason: f.Reason}
	}
	files := []genFile{}
	for _, f := range status {
		files = append(files, f)
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	writeJSON(w, http.StatusOK, map[string]any{"files": files})
}

// genContent는 생성 파일 하나의 내용을 준다. 이번 생성 결과에 있는 경로만 받는다.
func (s *Server) genContent(w http.ResponseWriter, r *http.Request) {
	res, ok := s.genRun(w)
	if !ok {
		return
	}
	want := r.URL.Query().Get("path")
	for _, f := range res.Files {
		if f.Path == want {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.Write(f.Content)
			return
		}
	}
	writeErr(w, http.StatusNotFound, "생성 파일이 아니다: "+want)
}

func (s *Server) genRun(w http.ResponseWriter) (workspace.Result, bool) {
	design, root := s.paths()
	res := workspace.Run(design, root)
	if res.Plan == nil {
		writeErr(w, http.StatusConflict, "설계 검사 오류가 있어 생성할 수 없다")
		return res, false
	}
	return res, true
}
