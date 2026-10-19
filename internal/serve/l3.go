package serve

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/tolelom/daemok/internal/lift"
)

func (s *Server) l3Draft(w http.ResponseWriter, r *http.Request) {
	design, root := s.paths()
	res, err := lift.Draft(design, root)
	if err != nil {
		writeErr(w, http.StatusConflict, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"yaml": string(res.YAML), "nodes": res.Nodes, "skipped": res.Skipped, "funcs": res.Funcs, "coverage": res.Coverage,
		"canSave": s.options().Local,
	})
}

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
