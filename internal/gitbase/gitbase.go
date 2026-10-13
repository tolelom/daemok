// Package gitbase는 git HEAD에 있는 설계 파일과 저장소의 worktree를 읽는다. 설계의 기준은 HEAD다.
package gitbase

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Files는 designDir의 HEAD 시점 *.yaml 파일을 준다. 키는 "HEAD:<저장소 기준 경로>"다.
// git 저장소가 아니거나 커밋이 없으면 ok가 거짓이다.
func Files(designDir string) (files map[string][]byte, ok bool, err error) {
	if _, err := git(designDir, "rev-parse", "--is-inside-work-tree"); err != nil {
		return nil, false, nil
	}
	if _, err := git(designDir, "rev-parse", "--verify", "--quiet", "HEAD"); err != nil {
		return nil, false, nil
	}
	// -z는 경로를 따옴표로 감싸지 않고 NUL로 구분한다. 비ASCII 경로도 그대로 받는다.
	out, err := git(designDir, "ls-tree", "-z", "--full-name", "--name-only", "HEAD", ".")
	if err != nil {
		return nil, false, err
	}
	files = map[string][]byte{}
	for _, p := range strings.Split(string(out), "\x00") {
		if p == "" || !strings.HasSuffix(p, ".yaml") {
			continue
		}
		src, err := git(designDir, "show", "HEAD:"+p)
		if err != nil {
			return nil, false, err
		}
		files["HEAD:"+p] = src
	}
	return files, true, nil
}

// Worktree는 저장소의 작업 폴더 하나다. Branch는 짧은 브랜치 이름이고 detached면 비어 있다.
type Worktree struct {
	Path   string
	Branch string
}

// Worktrees는 dir이 속한 저장소의 worktree를 git이 알려 주는 순서(주 worktree 먼저)로 준다.
// git 저장소가 아니면 nil이다.
func Worktrees(dir string) ([]Worktree, error) {
	if _, err := git(dir, "rev-parse", "--is-inside-work-tree"); err != nil {
		return nil, nil
	}
	out, err := git(dir, "worktree", "list", "--porcelain", "-z")
	if err != nil {
		return nil, err
	}
	var wts []Worktree
	for _, f := range strings.Split(string(out), "\x00") {
		switch {
		case strings.HasPrefix(f, "worktree "):
			wts = append(wts, Worktree{Path: filepath.FromSlash(strings.TrimPrefix(f, "worktree "))})
		case strings.HasPrefix(f, "branch ") && len(wts) > 0:
			wts[len(wts)-1].Branch = strings.TrimPrefix(f, "branch refs/heads/")
		}
	}
	return wts, nil
}

// Branch는 dir의 현재 브랜치 이름이다. git 저장소가 아니거나 detached면 비어 있다.
func Branch(dir string) string {
	out, err := git(dir, "symbolic-ref", "--quiet", "--short", "HEAD")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// HasHead는 dir이 커밋이 하나 이상 있는 git 저장소 안인지 본다.
func HasHead(dir string) bool {
	_, err := git(dir, "rev-parse", "--verify", "--quiet", "HEAD")
	return err == nil
}

// Dirty는 dir 작업 폴더의 커밋 안 된 변경(추적 안 하는 파일 포함) 수다. 저장소가 아니면 0이다.
func Dirty(dir string) int {
	out, err := git(dir, "status", "--porcelain", "-z")
	if err != nil {
		return 0
	}
	n := 0
	for _, f := range strings.Split(string(out), "\x00") {
		// 이름 바꾸기는 원래 이름을 다음 칸에 하나 더 준다.
		if len(f) > 3 && f[2] == ' ' {
			n++
		}
	}
	return n
}

// NewWorktree는 dir이 속한 저장소의 HEAD에서 branch를 새로 만들어 <주 worktree>/.worktrees/<branch의 /를 -로>에
// worktree로 꺼내고 그 경로를 준다. .git/info/exclude에 /.worktrees/를 한 번 더해 주 worktree의 status에 보이지 않게 한다.
// 브랜치나 경로가 이미 있으면 아무것도 하지 않고 오류를 준다.
func NewWorktree(dir, branch string) (string, error) {
	if branch == "" || strings.HasPrefix(branch, "-") {
		return "", fmt.Errorf("브랜치 이름이 아니다: %q", branch)
	}
	if _, err := git(dir, "check-ref-format", "--branch", branch); err != nil {
		return "", fmt.Errorf("브랜치 이름이 아니다: %q", branch)
	}
	wts, err := Worktrees(dir)
	if err != nil || len(wts) == 0 {
		return "", fmt.Errorf("git 저장소가 아니다: %s", dir)
	}
	if _, err := git(dir, "rev-parse", "--verify", "--quiet", "HEAD"); err != nil {
		return "", errors.New("커밋이 없어 HEAD에서 시작할 수 없다")
	}
	if _, err := git(dir, "rev-parse", "--verify", "--quiet", "refs/heads/"+branch); err == nil {
		return "", fmt.Errorf("이미 있는 브랜치다: %s", branch)
	}
	path := filepath.Join(wts[0].Path, ".worktrees", strings.ReplaceAll(branch, "/", "-"))
	if _, err := os.Lstat(path); err == nil {
		return "", fmt.Errorf("이미 있는 경로다: %s", path)
	}
	if err := excludeWorktrees(dir); err != nil {
		return "", err
	}
	if out, err := gitOut(dir, "worktree", "add", "-q", "-b", branch, path, "HEAD"); err != nil {
		return "", fmt.Errorf("git worktree add: %s", strings.TrimSpace(out))
	}
	return path, nil
}

// excludeWorktrees는 저장소의 info/exclude에 /.worktrees/ 줄이 없으면 더한다.
func excludeWorktrees(dir string) error {
	out, err := git(dir, "rev-parse", "--path-format=absolute", "--git-common-dir")
	if err != nil {
		return errors.New("git 디렉토리를 찾을 수 없다")
	}
	p := filepath.Join(strings.TrimSpace(string(out)), "info", "exclude")
	b, err := os.ReadFile(p)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	for _, ln := range strings.Split(string(b), "\n") {
		if strings.TrimSpace(ln) == "/.worktrees/" {
			return nil
		}
	}
	if len(b) > 0 && b[len(b)-1] != '\n' {
		b = append(b, '\n')
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	return os.WriteFile(p, append(b, "/.worktrees/\n"...), 0o644)
}

// gitOut은 git을 돌려 표준 출력과 표준 오류를 함께 준다. 오류 메시지를 보이려고 쓴다.
func gitOut(dir string, args ...string) (string, error) {
	out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput()
	return string(out), err
}

func git(dir string, args ...string) ([]byte, error) {
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	return out.Bytes(), err
}
