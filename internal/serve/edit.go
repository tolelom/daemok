package serve

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/tolelom/daemok/internal/edit"
)

// editDesign은 화면에서 온 L2 편집을 설계 파일에 줄 단위로 적용한다.
// 화면이 본 버전과 다르면 쓰지 않고, 고친 결과가 읽기 오류를 내면 쓰지 않는다.
func (s *Server) editDesign(w http.ResponseWriter, r *http.Request) {
	if !s.options().Local || s.currentMode() != "check" {
		writeErr(w, http.StatusForbidden, "루프백 주소로 띄운 대조 모드에서만 고칠 수 있다")
		return
	}
	// 버전 확인부터 쓰기까지 한 번에 하나만 한다. 같은 버전의 두 번째 요청은 409가 된다.
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
	// 화면이 본 상태 뒤로 설계 파일 내용이 바뀌었을 때만 거부한다. 코드만 바뀐 버전은 받는다.
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
