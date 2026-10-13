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
	case "remove_module":
		return edit.RemoveModule(src, file, id)
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
	// 화면이 본 상태 뒤로 설계 파일 내용이 바뀌었을 때만 거부한다. 코드만 바뀐 버전은 받는다.
	if since, now := s.designVersions(); req.Version < since || req.Version > now {
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
	target, lastErr := "", errors.New("설계 파일이 없다")
	var patched []byte
	for _, p := range paths {
		out, err := editFile(req.Op, files[p], p, req.ID, req.Value)
		if err == nil {
			target, patched = p, out
			break
		}
		lastErr = err
		// 다른 파일에 있을 수 있는 경우만 다음 파일을 본다.
		if !errors.Is(err, edit.ErrModuleNotInFile) && !errors.Is(err, edit.ErrNoModuleBlock) {
			break
		}
	}
	if target == "" {
		writeErr(w, http.StatusBadRequest, lastErr.Error())
		return
	}
	before := loadErrors(files)
	files[target] = patched
	// 편집 전부터 있던 읽기 오류는 빼고, 편집이 새로 만든 오류만 본다.
	var added []string
	for msg, n := range loadErrors(files) {
		if n > before[msg] {
			added = append(added, msg)
		}
	}
	sort.Strings(added)
	if len(added) > 0 {
		writeErr(w, http.StatusUnprocessableEntity, "고치면 설계를 읽을 수 없게 된다: "+strings.Join(added, "; "))
		return
	}
	if err := writeAtomic(target, patched); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.Poll()
	writeJSON(w, http.StatusOK, map[string]int{"version": s.Version()})
}

// loadErrors는 설계 파일들을 읽을 때 나는 오류 메시지와 그 수다.
func loadErrors(files map[string][]byte) map[string]int {
	_, diags := model.LoadSources(files)
	out := map[string]int{}
	for _, d := range diags {
		if d.Level == model.Error {
			out[d.Msg]++
		}
	}
	return out
}

// writeAtomic은 같은 디렉토리의 임시 파일에 쓴 뒤 이름을 바꾼다.
// 심볼릭 링크면 링크는 두고 가리키는 파일을 바꾼다.
func writeAtomic(path string, b []byte) error {
	if real, err := filepath.EvalSymlinks(path); err == nil {
		path = real
	}
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
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	if err := os.Chmod(tmp.Name(), mode); err != nil {
		os.Remove(tmp.Name())
		return err
	}
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
