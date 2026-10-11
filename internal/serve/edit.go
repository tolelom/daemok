package serve

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/tolelom/daemok/internal/edit"
	"github.com/tolelom/daemok/internal/model"
)

var errNoFile = errors.New("이 파일에 모듈이 없다")

// editFile은 설계 파일 하나에 작업을 적용한다.
func editFile(op string, src []byte, file, id, value string) ([]byte, error) {
	switch op {
	case "tier":
		return edit.SetTier(src, file, id, value)
	case "add_dep":
		return edit.AddDep(src, file, id, value)
	case "remove_dep":
		return edit.RemoveDep(src, file, id, value)
	case "add_module":
		return edit.AddModule(src, file, id, value)
	}
	return nil, errors.New("알 수 없는 작업: " + op)
}

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
	if req.Version != s.Version() {
		writeErr(w, http.StatusConflict, "그사이 설계가 바뀌었다. 화면을 새로 고친 뒤 다시 한다")
		return
	}
	design, _ := s.paths()
	paths, _ := filepath.Glob(filepath.Join(design, "*.yaml"))
	sort.Strings(paths)
	files := map[string][]byte{}
	for _, p := range paths {
		b, err := os.ReadFile(p)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		files[p] = b
	}
	target, lastErr := "", error(errNoFile)
	var patched []byte
	for _, p := range paths {
		out, err := editFile(req.Op, files[p], p, req.ID, req.Value)
		if err == nil {
			target, patched = p, out
			break
		}
		// 다른 파일에 있을 수 있는 경우만 다음 파일을 본다.
		if !strings.Contains(err.Error(), "이 파일에 모듈이 없다") && !strings.Contains(err.Error(), "modules가") {
			lastErr = err
			break
		}
		lastErr = err
	}
	if target == "" {
		writeErr(w, http.StatusBadRequest, lastErr.Error())
		return
	}
	files[target] = patched
	if _, diags := model.LoadSources(files); diags.HasErrors() {
		var msgs []string
		for _, d := range diags {
			if d.Level == model.Error {
				msgs = append(msgs, d.Msg)
			}
		}
		writeErr(w, http.StatusUnprocessableEntity, "고치면 설계를 읽을 수 없게 된다: "+strings.Join(msgs, "; "))
		return
	}
	if err := writeAtomic(target, patched); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.Poll()
	writeJSON(w, http.StatusOK, map[string]int{"version": s.Version()})
}

// writeAtomic은 같은 디렉토리의 임시 파일에 쓴 뒤 이름을 바꾼다.
func writeAtomic(path string, b []byte) error {
	mode := os.FileMode(0o644)
	if st, err := os.Stat(path); err == nil {
		mode = st.Mode().Perm()
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".daemok-edit-*")
	if err != nil {
		return err
	}
	if _, err := tmp.Write(b); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	os.Chmod(tmp.Name(), mode)
	if err := renameRetry(tmp.Name(), path); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	return nil
}

// renameRetry는 Windows에서 다른 쪽(감시, 에디터, 백신)이 대상을 잠깐 열고 있어
// 이름 바꾸기가 실패하면 짧게 기다렸다 다시 한다.
func renameRetry(from, to string) error {
	err := os.Rename(from, to)
	if runtime.GOOS != "windows" {
		return err
	}
	for wait := time.Millisecond; err != nil && wait < time.Second; wait *= 2 {
		time.Sleep(wait)
		err = os.Rename(from, to)
	}
	return err
}
