package gitbase

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func run(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func TestFiles(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git이 없다")
	}
	repo := t.TempDir()
	design := filepath.Join(repo, "design")
	os.MkdirAll(design, 0o755)
	// 저장소 밖
	if _, ok, err := Files(design); ok || err != nil {
		t.Fatalf("저장소 밖: ok=%v err=%v", ok, err)
	}
	run(t, repo, "init", "-q")
	os.WriteFile(filepath.Join(design, "a.yaml"), []byte("version: 0.5\n"), 0o644)
	os.WriteFile(filepath.Join(design, "note.txt"), []byte("x"), 0o644)
	// 커밋 전
	if _, ok, _ := Files(design); ok {
		t.Fatal("커밋 전에는 기준이 없다")
	}
	run(t, repo, "add", ".")
	run(t, repo, "commit", "-q", "-m", "init")
	// 작업 트리에서 지워도 HEAD에는 있다
	os.Remove(filepath.Join(design, "a.yaml"))
	os.WriteFile(filepath.Join(design, "b.yaml"), []byte("version: 0.5\n"), 0o644)
	files, ok, err := Files(design)
	if !ok || err != nil || len(files) != 1 || string(files["HEAD:design/a.yaml"]) != "version: 0.5\n" {
		t.Fatalf("ok=%v err=%v files=%v", ok, err, files)
	}
	// HEAD에 설계 디렉토리가 없으면 빈 목록
	other := filepath.Join(repo, "other")
	os.MkdirAll(other, 0o755)
	if files, ok, err := Files(other); !ok || err != nil || len(files) != 0 {
		t.Fatalf("빈 기준: ok=%v err=%v files=%v", ok, err, files)
	}
}

func TestNonASCIIPath(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git이 없다")
	}
	repo := t.TempDir()
	design := filepath.Join(repo, "d 설계")
	os.MkdirAll(design, 0o755)
	os.WriteFile(filepath.Join(design, "모듈.yaml"), []byte("version: 0.5\n"), 0o644)
	run(t, repo, "init", "-q")
	run(t, repo, "add", ".")
	run(t, repo, "commit", "-q", "-m", "init")
	files, ok, err := Files(design)
	if !ok || err != nil || string(files["HEAD:d 설계/모듈.yaml"]) != "version: 0.5\n" {
		t.Fatalf("ok=%v err=%v files=%v", ok, err, files)
	}
}

func TestWorktrees(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git이 없다")
	}
	if wts, err := Worktrees(t.TempDir()); wts != nil || err != nil {
		t.Fatalf("저장소 밖: %v %v", wts, err)
	}
	repo := t.TempDir()
	run(t, repo, "init", "-q", "-b", "main")
	os.WriteFile(filepath.Join(repo, "a.txt"), []byte("a"), 0o644)
	run(t, repo, "add", ".")
	run(t, repo, "commit", "-q", "-m", "init")
	wt := filepath.Join(repo, ".worktrees", "feat")
	run(t, repo, "worktree", "add", "-q", "-b", "feat/x", wt)
	det := filepath.Join(t.TempDir(), "det")
	run(t, repo, "worktree", "add", "-q", "--detach", det)
	wts, err := Worktrees(wt)
	if err != nil {
		t.Fatal(err)
	}
	want := []Worktree{{repo, "main"}, {wt, "feat/x"}, {det, ""}}
	if len(wts) != len(want) {
		t.Fatalf("%+v", wts)
	}
	for i := range want {
		if wts[i] != want[i] {
			t.Fatalf("%d: %+v, want %+v", i, wts[i], want[i])
		}
	}
	if b := Branch(wt); b != "feat/x" {
		t.Fatalf("Branch %q", b)
	}
	if b := Branch(t.TempDir()); b != "" {
		t.Fatalf("저장소 밖 Branch %q", b)
	}
}
