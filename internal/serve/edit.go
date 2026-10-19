package serve

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/tolelom/daemok/internal/edit"
)

func (s *Server) editDesign(w http.ResponseWriter, r *http.Request) {
	if !s.options().Local || s.currentMode() != "check" {
		writeErr(w, http.StatusForbidden, "루프백 주소로 띄운 대조 모드에서만 고칠 수 있다")
		return
	}
	s.editMu.Lock()
	defer s.editMu.Unlock()
	var req struct {
		Op      string `json:"op"`
		ID      string `json:"id"`
		Value   string `json:"value"`
		Version int    `json:"version"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "요청 본문을 읽을 수 없다")
		return
	}
	s.Poll()
	if since, now := s.designVersions(); req.Version < since || req.Version > now {
		writeErr(w, http.StatusConflict, "그사이 설계가 바뀌었다. 화면을 새로 고친 뒤 다시 한다")
		return
	}
	design, _ := s.paths()
	if _, err := edit.Apply(design, req.Op, req.ID, req.Value); err != nil {
		switch {
		case errors.Is(err, edit.ErrBreaksLoad):
			writeErr(w, http.StatusUnprocessableEntity, err.Error())
		case errors.Is(err, edit.ErrIO):
			writeErr(w, http.StatusInternalServerError, err.Error())
		default:
			writeErr(w, http.StatusBadRequest, err.Error())
		}
		return
	}
	s.Poll()
	writeJSON(w, http.StatusOK, map[string]int{"version": s.Version()})
}
