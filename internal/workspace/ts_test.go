package workspace

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tolelom/daemok/internal/gen/golang"
	"github.com/tolelom/daemok/internal/gen/ts"
	"github.com/tolelom/daemok/internal/model"
)

const tsDesign = design + `ts_out: ../web/kind.gen.ts
consts:
  limit: { in: core, type: int, value: 3, export: [ts] }
`

// tsSetup은 base/back(코드 루트)과 base/web(프론트 저장소 자리)을 만든다. TS 파일은 코드 루트 밖에 생긴다.
func tsSetup(t *testing.T, web bool) (root, dir, tsPath string) {
	t.Helper()
	base := t.TempDir()
	root = filepath.Join(base, "back")
	dir = filepath.Join(root, "design")
	write(t, filepath.Join(dir, "d.yaml"), tsDesign)
	if web {
		os.MkdirAll(filepath.Join(base, "web"), 0o755)
	}
	return root, dir, filepath.Join(base, "web", "kind.gen.ts")
}

func TestHeadersMatch(t *testing.T) {
	if ts.Header != golang.Header {
		t.Fatalf("%q != %q", ts.Header, golang.Header)
	}
}

func TestTSOutsideRoot(t *testing.T) {
	root, dir, tsPath := tsSetup(t, true)
	r := Run(dir, root)
	wantDiag(t, r, "생성 파일이 없다: ../web/kind.gen.ts")
	if err := Apply(root, r.Plan); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(tsPath)
	if err != nil || !strings.Contains(string(b), "export const LIMIT = 3;") {
		t.Fatalf("%v %s", err, b)
	}
	if r := Run(dir, root); r.Diags.HasErrors() {
		t.Fatalf("%v", r.Diags)
	}
	write(t, tsPath, ts.Header+"\nexport const LIMIT = 4;\n")
	wantDiag(t, Run(dir, root), "생성 파일이 설계와 다르다: ../web/kind.gen.ts")
	write(t, tsPath, "export const LIMIT = 4;\n")
	r = Run(dir, root)
	wantDiag(t, r, "생성 헤더가 없는 파일이 생성 경로에 있다: ../web/kind.gen.ts")
	if len(r.Plan.Refuse) != 1 {
		t.Fatalf("%+v", r.Plan)
	}
}

func TestTSMissingDirWarns(t *testing.T) {
	root, dir, tsPath := tsSetup(t, false)
	r := Run(dir, root)
	found := false
	for _, d := range r.Diags {
		if d.Level == model.Warning && strings.Contains(d.Message, "ts_out 디렉토리가 없어 TS 파일을 쓰지 않는다") {
			found = true
		}
	}
	if !found {
		t.Fatalf("%v", r.Diags)
	}
	Apply(root, r.Plan)
	if _, err := os.Stat(filepath.Dir(tsPath)); !os.IsNotExist(err) {
		t.Fatal("ts_out 디렉토리를 만들었다")
	}
}
