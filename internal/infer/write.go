package infer

import (
	"errors"
	"os"
	"path/filepath"
)

var ErrDesignExists = errors.New("설계 디렉토리에 이미 설계 파일이 있다")

// HasDesignFiles는 designDir에 *.yaml이 있는지다.
func HasDesignFiles(designDir string) bool {
	paths, _ := filepath.Glob(filepath.Join(designDir, "*.yaml"))
	return len(paths) > 0
}

// Write는 designDir에 *.yaml이 없을 때만 초안을 <system>.yaml로 새로 만든다. 만든 파일 경로를 준다.
func Write(designDir string, a *Analysis) (string, error) {
	if HasDesignFiles(designDir) {
		return "", ErrDesignExists
	}
	if err := os.MkdirAll(designDir, 0o755); err != nil {
		return "", err
	}
	p := filepath.Join(designDir, a.Draft.System+".yaml")
	f, err := os.OpenFile(p, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return "", err
	}
	if _, err := f.Write(a.YAML); err != nil {
		f.Close()
		return "", err
	}
	return p, f.Close()
}
