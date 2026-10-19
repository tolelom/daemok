package designdiff

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/tolelom/daemok/internal/gitbase"
)

func Run(designDir, base string) (string, error) {
	ents, err := os.ReadDir(designDir)
	if err != nil {
		return "", fmt.Errorf("설계 디렉토리를 읽을 수 없다: %v", err)
	}
	before, ok, err := gitbase.Files(designDir, base)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", fmt.Errorf("기준 %s를 읽을 수 없다: git 저장소가 아니거나 커밋이 없다", base)
	}
	work := map[string][]byte{}
	for _, e := range ents {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		p := filepath.Join(designDir, e.Name())
		b, err := os.ReadFile(p)
		if err != nil {
			return "", err
		}
		work[p] = b
	}
	r, err := Compare(before, work)
	if err != nil {
		return "", err
	}
	return r.Markdown(base), nil
}
