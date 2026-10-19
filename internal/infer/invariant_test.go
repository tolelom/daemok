package infer_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tolelom/daemok/internal/infer"
	"github.com/tolelom/daemok/internal/model"
	"github.com/tolelom/daemok/internal/workspace"
)

// draftErrors는 초안을 코드 루트에 대조한 오류 중 코드의 사실을 알리는 것(파싱 실패, 순환 import)이 아닌 것이다.
// 순환 import 오류가 있었는지도 준다.
func draftErrors(t *testing.T, root string) (errs []string, cycle bool) {
	t.Helper()
	a, err := infer.Analyze(root)
	if err != nil {
		t.Fatal(err)
	}
	design := filepath.Join(root, "design")
	r := workspace.RunSources(map[string][]byte{filepath.Join(design, a.Draft.System+".yaml"): a.YAML}, design, root)
	for _, d := range r.Diags {
		switch {
		case d.Level != model.Error, strings.Contains(d.Message, "Go 파일을 읽을 수 없다"):
		case strings.Contains(d.Message, "의존 순환"):
			cycle = true
		default:
			errs = append(errs, d.String())
		}
	}
	if r.Fatal {
		errs = append(errs, "fatal")
	}
	return errs, cycle
}

func TestDraftPassesCheck(t *testing.T) {
	for _, name := range []string{"layers", "rootmain"} {
		if errs, cycle := draftErrors(t, filepath.Join("testdata", name)); len(errs) > 0 || cycle {
			t.Errorf("%s: %v %v", name, errs, cycle)
		}
	}
}

// codescan fixture에서 설계를 빼고 go.mod를 더한 복사본에도 같은 불변식이 선다.
// 이 fixture는 plugins/a와 plugins/b가 서로 import한다.
func TestDraftPassesCheckOnCodescanFixture(t *testing.T) {
	src := filepath.Join("..", "codescan", "testdata", "scan")
	dst := t.TempDir()
	err := filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, p)
		if d.IsDir() {
			if d.Name() == "design" {
				return filepath.SkipDir
			}
			return os.MkdirAll(filepath.Join(dst, rel), 0o755)
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(dst, rel), b, 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(dst, "go.mod"), []byte("module example.com/scan\n"), 0o644)
	errs, cycle := draftErrors(t, dst)
	if len(errs) > 0 {
		t.Fatal(errs)
	}
	if !cycle {
		t.Fatal("plugins/a와 plugins/b의 순환 import를 알려야 한다")
	}
}
