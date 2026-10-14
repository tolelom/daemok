package rename

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tolelom/daemok/internal/check"
	"github.com/tolelom/daemok/internal/model"
)

// clone은 도입 사례 클론의 코드 루트다. 저장소 옆 디렉토리나 ~/src에 없으면 건너뛴다.
func clone(t *testing.T, name string) string {
	t.Helper()
	cands := []string{filepath.Join("..", "..", "..", name)}
	if home, err := os.UserHomeDir(); err == nil {
		cands = append(cands, filepath.Join(home, "src", name))
	}
	for _, c := range cands {
		if _, err := os.Stat(filepath.Join(c, "design")); err == nil {
			return c
		}
	}
	t.Skip(name + " 클론이 없다")
	return ""
}

// copyDesign은 클론의 설계를 임시 디렉토리로 복사한다. 클론은 고치지 않는다.
func copyDesign(t *testing.T, root string) string {
	t.Helper()
	dir := t.TempDir()
	paths, _ := filepath.Glob(filepath.Join(root, "design", "*.yaml"))
	for _, p := range paths {
		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		os.WriteFile(filepath.Join(dir, filepath.Base(p)), b, 0o644)
	}
	return dir
}

func renamed(id string) string {
	switch {
	case strings.Contains(id, "-"):
		return id + "-rn"
	case id[0] >= 'A' && id[0] <= 'Z':
		return id + "Rn"
	}
	return id + "_rn"
}

func diagCount(t *testing.T, dir string) int {
	t.Helper()
	m, ds := model.Load(dir)
	return len(append(ds, check.Run(m)...))
}

// 도입 사례 설계에서 종류마다 노드 몇 개씩 바꾼다. 기본 모드는 성공하거나 --code를 안내하며 거부해야 하고,
// --code는 고정 불가 종류에서 성공해야 한다. 어느 쪽이든 진단 수는 그대로다.
func TestRealProjects(t *testing.T) {
	for _, name := range []string{"tolchain", "fiber_api_server"} {
		t.Run(name, func(t *testing.T) {
			root := clone(t, name)
			base := copyDesign(t, root)
			m, _ := model.Load(base)
			want := diagCount(t, base)
			for _, kind := range Kinds() {
				var ids []string
				if kind == "tiers" {
					for _, r := range m.Tiers {
						ids = append(ids, r.Name)
					}
				} else {
					for _, meta := range metas(m, kind) {
						ids = append(ids, meta.ID)
					}
				}
				for i, id := range ids {
					if i == 3 {
						break
					}
					for _, code := range []bool{false, true} {
						design := copyDesign(t, root)
						_, err := Run(design, root, kind, id, renamed(id), Options{Code: code})
						switch {
						case err == nil:
							if got := diagCount(t, design); got != want {
								t.Errorf("%s %s code=%v: 진단 %d → %d", kind, id, code, want, got)
							}
						case !code && strings.Contains(err.Error(), "--code"):
							t.Logf("%s %s: %v", kind, id, err)
						default:
							t.Errorf("%s %s code=%v: %v", kind, id, code, err)
						}
					}
				}
			}
		})
	}
}
