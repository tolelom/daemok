// Package gitbase는 git HEAD에 있는 설계 파일을 읽는다. 설계의 기준은 HEAD다.
package gitbase

import (
	"bytes"
	"os/exec"
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
	out, err := git(designDir, "ls-tree", "--full-name", "--name-only", "HEAD", ".")
	if err != nil {
		return nil, false, err
	}
	files = map[string][]byte{}
	for _, p := range strings.Split(strings.TrimSpace(string(out)), "\n") {
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

func git(dir string, args ...string) ([]byte, error) {
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	return out.Bytes(), err
}
