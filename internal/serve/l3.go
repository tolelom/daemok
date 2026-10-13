package serve

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/tolelom/daemok/internal/lift"
)

// l3Draft는 설계에 없는 손으로 쓴 선언의 L3 초안을 준다. 검증에 설계 대조를 다시 돌리므로 요청할 때만 만든다.
func (s *Server) l3Draft(w http.ResponseWriter, r *http.Request) {
	design, root := s.paths()
	res, err := lift.Draft(design, root)
	if err != nil {
		writeErr(w, http.StatusConflict, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"yaml": string(res.YAML), "nodes": res.Nodes, "skipped": res.Skipped, "funcs": res.Funcs,
		"canSave": s.options().Local,
	})
}

// l3Save는 초안을 다시 만들어 설계 디렉토리의 새 파일(l3.yaml, l3-2.yaml, ...)로 쓴다. 있는 파일은 덮어쓰지 않는다.
func (s *Server) l3Save(w http.ResponseWriter, r *http.Request) {
	if !s.options().Local {
		writeErr(w, http.StatusForbidden, "루프백 주소로 띄웠을 때만 저장할 수 있다")
		return
	}
	design, root := s.paths()
	res, err := lift.Draft(design, root)
	if err != nil {
		writeErr(w, http.StatusConflict, err.Error())
		return
	}
	if len(res.Nodes) == 0 {
		writeErr(w, http.StatusConflict, "설계에 올릴 선언이 없다")
		return
	}
	p, err := createFree(design, res.YAML)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.Poll()
	if rel, err := filepath.Rel(root, p); err == nil {
		p = filepath.ToSlash(rel)
	}
	writeJSON(w, http.StatusOK, map[string]any{"path": p, "nodes": len(res.Nodes)})
}

// createFree는 l3.yaml부터 이름이 비어 있는 첫 파일을 만든다. O_EXCL이라 동시에 저장해도 덮어쓰지 않는다.
func createFree(dir string, content []byte) (string, error) {
	for i := 1; i < 100; i++ {
		name := "l3.yaml"
		if i > 1 {
			name = fmt.Sprintf("l3-%d.yaml", i)
		}
		p := filepath.Join(dir, name)
		f, err := os.OpenFile(p, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if errors.Is(err, os.ErrExist) {
			continue
		}
		if err != nil {
			return "", err
		}
		_, werr := f.Write(content)
		if cerr := f.Close(); werr == nil {
			werr = cerr
		}
		if werr != nil {
			os.Remove(p)
			return "", werr
		}
		return p, nil
	}
	return "", errors.New("l3.yaml부터 l3-99.yaml까지 모두 있다")
}
