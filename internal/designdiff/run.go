package designdiff

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/tolelom/daemok/internal/gitbase"
)

// Run은 designDir의 base 시점 설계와 작업 폴더 설계를 비교해 마크다운을 준다.
// 기준을 읽지 못하거나 설계 파일을 YAML로 읽지 못하면 오류다.
func Run(designDir, base string) (string, error) {
	before, ok, err := gitbase.Files(designDir, base)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", fmt.Errorf("기준 %s를 읽을 수 없다: git 저장소가 아니거나 커밋이 없다", base)
	}
	paths, _ := filepath.Glob(filepath.Join(designDir, "*.yaml"))
	work := map[string][]byte{}
	for _, p := range paths {
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
