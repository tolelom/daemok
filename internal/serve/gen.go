package serve

import (
	"net/http"

	"github.com/tolelom/daemok/internal/workspace"
)

func (s *Server) genPreview(w http.ResponseWriter, r *http.Request) {
	res, ok := s.genRun(w)
	if !ok {
		return
	}
	_, root := s.paths()
	writeJSON(w, http.StatusOK, map[string]any{"files": workspace.Preview(res, root)})
}

func (s *Server) genContent(w http.ResponseWriter, r *http.Request) {
	res, ok := s.genRun(w)
	if !ok {
		return
	}
	want := r.URL.Query().Get("path")
	content, ok := workspace.Content(res, want)
	if !ok {
		writeErr(w, http.StatusNotFound, "생성 파일이 아니다: "+want)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write(content)
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
