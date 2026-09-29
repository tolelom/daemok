package workspace

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tolelom/daemok/internal/gen/golang"
	"github.com/tolelom/daemok/internal/model"
)

const design = `version: 0.4
go_module: example.com/x
tiers: [a]
modules:
  core: { tier: a }
enums:
  kind: { in: core, values: [x, y] }
`

// setup은 코드 루트와 그 아래 design/을 만든다.
func setup(t *testing.T, src string) (root, designDir string) {
	t.Helper()
	root = t.TempDir()
	designDir = filepath.Join(root, "design")
	write(t, filepath.Join(designDir, "d.yaml"), src)
	return root, designDir
}

func write(t *testing.T, p, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func wantDiag(t *testing.T, r Result, substr string) {
	t.Helper()
	for _, d := range r.Diags {
		if d.Level == model.Error && strings.Contains(d.Msg, substr) {
			return
		}
	}
	t.Fatalf("no error containing %q in %v", substr, r.Diags)
}

func TestLoadErrorIsFatal(t *testing.T) {
	root, dir := setup(t, "version: 0.4\nmodulez: {}\n")
	if r := Run(dir, root); !r.Fatal || r.Plan != nil {
		t.Fatalf("%+v", r)
	}
}

func TestCheckErrorHasNoPlan(t *testing.T) {
	root, dir := setup(t, "version: 0.4\ntiers: [a, b]\nmodules:\n  low: { tier: a, depends_on: [high] }\n  high: { tier: b }\n")
	r := Run(dir, root)
	if r.Fatal || r.Plan != nil || !r.Diags.HasErrors() {
		t.Fatalf("%+v", r)
	}
}

func TestMissingThenFreshAfterApply(t *testing.T) {
	root, dir := setup(t, design)
	r := Run(dir, root)
	wantDiag(t, r, "생성 파일이 없다: core/kind.gen.go")
	if len(r.Plan.Write) != 1 {
		t.Fatalf("%+v", r.Plan)
	}
	if err := Apply(root, r.Plan); err != nil {
		t.Fatal(err)
	}
	r = Run(dir, root)
	if len(r.Diags) != 0 || len(r.Plan.Write)+len(r.Plan.Delete)+len(r.Plan.Refuse) != 0 {
		t.Fatalf("최신이어야 한다: %v %+v", r.Diags, r.Plan)
	}
}

func TestChangedAndCRLF(t *testing.T) {
	root, dir := setup(t, design)
	Apply(root, Run(dir, root).Plan)
	p := filepath.Join(root, "core", "kind.gen.go")
	src, _ := os.ReadFile(p)
	// CRLF로 바꿔도 최신이다
	write(t, p, strings.ReplaceAll(string(src), "\n", "\r\n"))
	if r := Run(dir, root); len(r.Diags) != 0 {
		t.Fatalf("CRLF: %v", r.Diags)
	}
	write(t, p, string(src)+"// 손댐\n")
	wantDiag(t, Run(dir, root), "생성 파일이 설계와 다르다: core/kind.gen.go")
}

func TestRefuseHandWrittenFile(t *testing.T) {
	root, dir := setup(t, design)
	write(t, filepath.Join(root, "core", "kind.gen.go"), "package core\n")
	r := Run(dir, root)
	wantDiag(t, r, "생성 헤더가 없는 파일이 생성 경로에 있다: core/kind.gen.go")
	if len(r.Plan.Refuse) != 1 || len(r.Plan.Write) != 0 {
		t.Fatalf("%+v", r.Plan)
	}
}

func TestStaleGeneratedFileAnywhere(t *testing.T) {
	root, dir := setup(t, design)
	Apply(root, Run(dir, root).Plan)
	// 설계에서 지운 모듈의 디렉토리에 남은 생성 파일
	write(t, filepath.Join(root, "old", "gone.gen.go"), golang.Header+"\n\npackage old\n")
	// 범위 밖 디렉토리는 보지 않는다
	for _, d := range []string{"testdata", "vendor", ".git", "node_modules"} {
		write(t, filepath.Join(root, d, "x.go"), golang.Header+"\n\npackage x\n")
	}
	r := Run(dir, root)
	wantDiag(t, r, "설계에 없는 생성 파일: old/gone.gen.go")
	if len(r.Plan.Delete) != 1 || r.Plan.Delete[0] != "old/gone.gen.go" {
		t.Fatalf("%+v", r.Plan)
	}
	Apply(root, r.Plan)
	if _, err := os.Stat(filepath.Join(root, "old", "gone.gen.go")); !os.IsNotExist(err) {
		t.Fatal("지워지지 않았다")
	}
}

func TestCodeScanIncluded(t *testing.T) {
	root, dir := setup(t, "version: 0.4\ngo_module: example.com/x\ntiers: [a]\nmodules:\n  core: { tier: a }\n  api: { tier: a }\n")
	write(t, filepath.Join(root, "core", "core.go"), "package core\n\nimport _ \"example.com/x/api\"\n")
	wantDiag(t, Run(dir, root), "설계에 없는 의존을 import한다: core → api")
}

func TestSymlinkAndDirectoryRefused(t *testing.T) {
	root, dir := setup(t, design)
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, "core")); err != nil {
		t.Skip("심볼릭 링크를 만들 수 없다")
	}
	r := Run(dir, root)
	wantDiag(t, r, "생성 경로에 심볼릭 링크가 있다: core/kind.gen.go")
	if len(r.Plan.Refuse) != 1 || r.Plan.Refuse[0].Reason != "심볼릭 링크" || len(r.Plan.Write) != 0 {
		t.Fatalf("%+v", r.Plan)
	}

	root2, dir2 := setup(t, design)
	os.MkdirAll(filepath.Join(root2, "core", "kind.gen.go"), 0o755)
	r = Run(dir2, root2)
	wantDiag(t, r, "생성 경로에 디렉토리가 있다: core/kind.gen.go")
}

func TestDeleteRemovesEmptyDirs(t *testing.T) {
	root, dir := setup(t, design)
	Apply(root, Run(dir, root).Plan)
	write(t, filepath.Join(root, "gone", "deep", "x.gen.go"), golang.Header+"\n\npackage deep\n")
	Apply(root, Run(dir, root).Plan)
	if _, err := os.Stat(filepath.Join(root, "gone")); !os.IsNotExist(err) {
		t.Fatal("빈 디렉토리가 남았다")
	}
	if _, err := os.Stat(root); err != nil {
		t.Fatal("코드 루트까지 지웠다")
	}
}

func TestMissingRootIsEmpty(t *testing.T) {
	_, dir := setup(t, design)
	root := filepath.Join(t.TempDir(), "new")
	r := Run(dir, root)
	if r.Fatal || len(r.Plan.Write) != 1 {
		t.Fatalf("%+v", r)
	}
	if err := Apply(root, r.Plan); err != nil {
		t.Fatal(err)
	}
}
