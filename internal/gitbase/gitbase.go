// Package gitbase는 git HEAD에 있는 설계 파일과 저장소의 worktree를 읽는다. 설계의 기준은 HEAD다.
package gitbase

import (
	"bytes"
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

func git(dir string, args ...string) ([]byte, error) {
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	return out.Bytes(), err
}
