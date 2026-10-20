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

func Files(designDir, ref string) (files map[string][]byte, ok bool, err error) {
	if ref == "" || strings.HasPrefix(ref, "-") {
		return nil, false, fmt.Errorf("ref가 아니다: %q", ref)
	}
	if _, err := git(designDir, "rev-parse", "--is-inside-work-tree"); err != nil {
		return nil, false, nil
	}
	if _, err := git(designDir, "rev-parse", "--verify", "--quiet", ref+"^{commit}"); err != nil {
		if ref == "HEAD" {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("없는 ref: %s", ref)
	}
	out, err := git(designDir, "ls-tree", "-z", "--full-name", "--name-only", ref, ".")
	if err != nil {
		return nil, false, err
	}
	files = map[string][]byte{}
	for _, p := range strings.Split(string(out), "\x00") {
		if p == "" || !strings.HasSuffix(p, ".yaml") {
			continue
		}
		src, err := git(designDir, "show", ref+":"+p)
		if err != nil {
			return nil, false, err
		}
		files[ref+":"+p] = src
	}
	return files, true, nil
}

type Worktree struct {
	Path   string
	Branch string
}

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

func Branch(dir string) string {
	out, err := git(dir, "symbolic-ref", "--quiet", "--short", "HEAD")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func HasHead(dir string) bool {
	_, err := git(dir, "rev-parse", "--verify", "--quiet", "HEAD")
	return err == nil
}

func Dirty(dir string) int {
	out, err := git(dir, "status", "--porcelain", "-z")
	if err != nil {
		return 0
	}
	n := 0
	fields := strings.Split(string(out), "\x00")
	for i := 0; i < len(fields); i++ {
		f := fields[i]
		if len(f) < 4 {
			continue
		}
		n++
		if f[0] == 'R' || f[0] == 'C' {
			i++
		}
	}
	return n
}

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

func Prefix(dir string) (string, error) {
	out, err := git(dir, "rev-parse", "--show-prefix")
	if err != nil {
		return "", fmt.Errorf("git 저장소가 아니다: %s", dir)
	}
	return filepath.FromSlash(strings.TrimSuffix(strings.TrimSpace(string(out)), "/")), nil
}

func RemoveWorktree(dir, path, branch string) error {
	if out, err := gitOut(dir, "worktree", "remove", "--force", path); err != nil {
		return fmt.Errorf("git worktree remove: %s", strings.TrimSpace(out))
	}
	if out, err := gitOut(dir, "branch", "-D", branch); err != nil {
		return fmt.Errorf("git branch -D: %s", strings.TrimSpace(out))
	}
	return nil
}

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
