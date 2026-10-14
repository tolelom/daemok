package gitbase

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

func TestNewWorktree(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git이 없다")
	}
	if _, err := NewWorktree(t.TempDir(), "daemok/design"); err == nil {
		t.Fatal("저장소 밖에서 만들었다")
	}
	repo := t.TempDir()
	run(t, repo, "init", "-q", "-b", "main")
	if _, err := NewWorktree(repo, "daemok/design"); err == nil {
		t.Fatal("커밋 없는 저장소에서 만들었다")
	}
	os.WriteFile(filepath.Join(repo, "a.txt"), []byte("a"), 0o644)
	run(t, repo, "add", ".")
	run(t, repo, "commit", "-q", "-m", "init")
	if n := Dirty(repo); n != 0 {
		t.Fatalf("Dirty %d", n)
	}
	os.WriteFile(filepath.Join(repo, "a.txt"), []byte("b"), 0o644)
	os.WriteFile(filepath.Join(repo, "new.txt"), []byte("n"), 0o644)
	if n := Dirty(repo); n != 2 {
		t.Fatalf("Dirty %d", n)
	}
	for _, bad := range []string{"", "a b", "../x", "-x"} {
		if _, err := NewWorktree(repo, bad); err == nil {
			t.Fatalf("브랜치 이름 %q를 받았다", bad)
		}
	}
	p, err := NewWorktree(repo, "daemok/design")
	if err != nil {
		t.Fatal(err)
	}
	if p != filepath.Join(repo, ".worktrees", "daemok-design") || Branch(p) != "daemok/design" {
		t.Fatalf("%s %s", p, Branch(p))
	}
	if b, _ := os.ReadFile(filepath.Join(p, "a.txt")); string(b) != "a" {
		t.Fatalf("HEAD에서 시작해야 한다: %q", b)
	}
	// 다른 worktree 안에서 불러도 주 worktree 아래에 만든다.
	p2, err := NewWorktree(p, "other")
	if err != nil || p2 != filepath.Join(repo, ".worktrees", "other") {
		t.Fatalf("%s %v", p2, err)
	}
	ex, _ := os.ReadFile(filepath.Join(repo, ".git", "info", "exclude"))
	if strings.Count(string(ex), "/.worktrees/\n") != 1 {
		t.Fatalf("exclude:\n%s", ex)
	}
	if n := Dirty(repo); n != 2 {
		t.Fatalf("원래 작업 폴더가 바뀌었다: Dirty %d", n)
	}
	if _, err := NewWorktree(repo, "daemok/design"); err == nil || !strings.Contains(err.Error(), "이미 있는 브랜치") {
		t.Fatalf("이미 있는 브랜치: %v", err)
	}
	os.MkdirAll(filepath.Join(repo, ".worktrees", "taken"), 0o755)
	if _, err := NewWorktree(repo, "taken"); err == nil || !strings.Contains(err.Error(), "이미 있는 경로") {
		t.Fatalf("이미 있는 경로: %v", err)
	}
}

// 이름 바꾸기는 원래 이름이 어떤 글자로 시작하든 한 번만 센다.
func TestDirtyRename(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git이 없다")
	}
	repo := t.TempDir()
	run(t, repo, "init", "-q", "-b", "main")
	os.WriteFile(filepath.Join(repo, "ab c.go"), []byte("package x\n"), 0o644)
	run(t, repo, "add", ".")
	run(t, repo, "commit", "-q", "-m", "init")
	run(t, repo, "mv", "ab c.go", "new.go")
	if n := Dirty(repo); n != 1 {
		t.Fatalf("Dirty %d", n)
	}
}
