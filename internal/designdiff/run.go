package designdiff

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/tolelom/daemok/internal/gitbase"
)

// Run은 designDir의 base 시점 설계와 작업 폴더 설계를 비교해 마크다운을 준다.
// 기준을 읽지 못하거나 설계 파일을 YAML로 읽지 못하면 오류다.
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
	// Glob은 디렉토리 이름의 [ ] 를 패턴으로 읽으므로 목록을 직접 거른다.
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
