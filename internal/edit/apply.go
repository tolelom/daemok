package edit

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/tolelom/daemok/internal/model"
)

// ErrBreaksLoad는 고친 결과가 편집 전에 없던 설계 읽기 오류를 낸다는 뜻이다. 쓰지 않는다.
var ErrBreaksLoad = errors.New("고치면 설계를 읽을 수 없게 된다")

// ErrIO는 설계 파일을 읽거나 쓰지 못했다는 뜻이다.
var ErrIO = errors.New("설계 파일을 읽거나 쓸 수 없다")

// Do는 설계 파일 하나의 내용에 작업 하나를 적용한다.
func Do(op string, src []byte, file, id, value string) ([]byte, error) {
	switch op {
	case "tier":
		return SetTier(src, file, id, value)
	case "add_dep":
		return AddDep(src, file, id, value)
	case "remove_dep":
		return RemoveDep(src, file, id, value)
	case "add_module":
		return AddModule(src, file, id, value)
	case "remove_module":
		return RemoveModule(src, file, id)
	}
	return nil, errors.New("알 수 없는 작업: " + op)
}

// Ops는 Do가 받는 작업 이름이다.
var Ops = []string{"tier", "add_dep", "remove_dep", "add_module", "remove_module"}

// ErrOutsideDir는 고칠 설계 파일이 심볼릭 링크로 설계 디렉토리 밖을 가리킨다는 뜻이다. 쓰지 않는다.
var ErrOutsideDir = errors.New("설계 파일이 설계 디렉토리 밖을 가리킨다")

// Apply는 설계 디렉토리의 *.yaml 가운데 대상이 있는 파일에 작업을 적용해 쓰고 그 경로를 준다.
// 고친 결과가 편집 전에 없던 읽기 오류를 내면 쓰지 않는다. 설계 검사 오류는 막지 않는다.
// 설계 파일이 심볼릭 링크면 가리키는 파일을 고친다.
func Apply(designDir, op, id, value string) (string, error) {
	return apply(designDir, op, id, value, false)
}

// ApplyConfined는 Apply와 같되, 설계 파일이 심볼릭 링크로 설계 디렉토리 밖을 가리키면 쓰지 않는다.
// 사람이 화면에서 누르지 않는 경로(MCP)가 설계 디렉토리 밖을 고치지 못하게 한다.
func ApplyConfined(designDir, op, id, value string) (string, error) {
	return apply(designDir, op, id, value, true)
}

func apply(designDir, op, id, value string, confined bool) (string, error) {
	paths, _ := filepath.Glob(filepath.Join(designDir, "*.yaml"))
	sort.Strings(paths)
	files := map[string][]byte{}
	for _, p := range paths {
		b, err := os.ReadFile(p)
		if err != nil {
			return "", fmt.Errorf("%w: %v", ErrIO, err)
		}
		files[p] = b
	}
	target, lastErr := "", errors.New("설계 파일이 없다")
	var patched []byte
	for _, p := range paths {
		out, err := Do(op, files[p], p, id, value)
		if err == nil {
			target, patched = p, out
			break
		}
		lastErr = err
		// 다른 파일에 있을 수 있는 경우만 다음 파일을 본다.
		if !errors.Is(err, ErrModuleNotInFile) && !errors.Is(err, ErrNoModuleBlock) {
			break
		}
	}
	if target == "" {
		return "", lastErr
	}
	if confined {
		if err := inside(designDir, target); err != nil {
			return "", err
		}
	}
	before := loadErrors(files)
	files[target] = patched
	// 편집 전부터 있던 읽기 오류는 빼고, 편집이 새로 만든 오류만 본다.
	var added []string
	for msg, n := range loadErrors(files) {
		if n > before[msg] {
			added = append(added, msg)
		}
	}
	sort.Strings(added)
	if len(added) > 0 {
		return "", fmt.Errorf("%w: %s", ErrBreaksLoad, strings.Join(added, "; "))
	}
	if err := writeAtomic(target, patched); err != nil {
		return "", fmt.Errorf("%w: %v", ErrIO, err)
	}
	return target, nil
}

// loadErrors는 설계 파일들을 읽을 때 나는 오류 메시지와 그 수다.
func loadErrors(files map[string][]byte) map[string]int {
	_, diags := model.LoadSources(files)
	out := map[string]int{}
	for _, d := range diags {
		if d.Level == model.Error {
			out[d.Msg]++
		}
	}
	return out
}

// writeAtomic은 같은 디렉토리의 임시 파일에 쓴 뒤 이름을 바꾼다.
// 심볼릭 링크면 링크는 두고 가리키는 파일을 바꾼다.
func writeAtomic(path string, b []byte) error {
	if real, err := filepath.EvalSymlinks(path); err == nil {
		path = real
	}
	mode := os.FileMode(0o644)
	if st, err := os.Stat(path); err == nil {
		mode = st.Mode().Perm()
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".daemok-edit-*")
	if err != nil {
		return err
	}
	if _, err := tmp.Write(b); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	if err := os.Chmod(tmp.Name(), mode); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	if err := renameRetry(tmp.Name(), path); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	return nil
}

// renameRetry는 Windows에서 다른 쪽(감시, 에디터, 백신)이 대상을 잠깐 열고 있어
// 이름 바꾸기가 실패하면 짧게 기다렸다 다시 한다.
func renameRetry(from, to string) error {
	err := os.Rename(from, to)
	if runtime.GOOS != "windows" {
		return err
	}
	for wait := time.Millisecond; err != nil && wait < time.Second; wait *= 2 {
		time.Sleep(wait)
		err = os.Rename(from, to)
	}
	return err
}

// inside는 path의 실제 위치가 dir의 실제 위치 아래인지 본다.
func inside(dir, path string) error {
	realDir, err := filepath.EvalSymlinks(dir)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrIO, err)
	}
	real, err := filepath.EvalSymlinks(path)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrIO, err)
	}
	rel, err := filepath.Rel(realDir, real)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("%w: %s → %s", ErrOutsideDir, path, real)
	}
	return nil
}
