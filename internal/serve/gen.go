package serve

import (
	"net/http"

	"github.com/tolelom/daemok/internal/workspace"
)

// genPreview는 daemok gen이 할 일을 파일마다 준다. 쓰지 않는다. 설계 대조를 다시 돌리므로 요청할 때만 만든다.
func (s *Server) genPreview(w http.ResponseWriter, r *http.Request) {
	res, ok := s.genRun(w)
	if !ok {
		return
	}
	_, root := s.paths()
	writeJSON(w, http.StatusOK, map[string]any{"files": workspace.Preview(res, root)})
}

// genContent는 생성 파일 하나의 내용을 준다. 이번 생성 결과에 있는 경로만 받는다.
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
