package codescan

import (
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/tolelom/daemok/internal/yaml"
)

var cppExts = map[string]bool{
	".h": true, ".hh": true, ".hpp": true, ".hxx": true, ".inl": true,
	".c": true, ".cc": true, ".cpp": true, ".cxx": true,
}

// include는 소스의 #include 한 줄이다. 열은 #의 바이트 위치다.
type include struct {
	line, col int
	path      string
	angle     bool
}

// includes는 주석을 걷어 낸 소스에서 줄 머리의 #include를 찾는다.
// 문자열·문자 리터럴 안의 주석 표시는 주석으로 보지 않는다. 리터럴은 줄 끝에서 끝난다고 본다.
func includes(src []byte) []include {
	var out []include
	for i, line := range strings.Split(string(stripComments(src)), "\n") {
		rest := strings.TrimLeft(line, " \t")
		col := len(line) - len(rest) + 1
		if !strings.HasPrefix(rest, "#") {
			continue
		}
		rest = strings.TrimLeft(rest[1:], " \t")
		if !strings.HasPrefix(rest, "include") {
			continue
		}
		rest = strings.TrimLeft(rest[len("include"):], " \t")
		if rest == "" {
			continue
		}
		closer := map[byte]byte{'"': '"', '<': '>'}[rest[0]]
		if closer == 0 {
			continue
		}
		end := strings.IndexByte(rest[1:], closer)
		if end <= 0 {
			continue
		}
		out = append(out, include{line: i + 1, col: col, path: rest[1 : 1+end], angle: rest[0] == '<'})
	}
	return out
}

// stripComments는 주석 글자를 공백으로 바꾼다. 줄바꿈은 남겨 줄·열이 그대로다.
func stripComments(src []byte) []byte {
	out := make([]byte, len(src))
	copy(out, src)
	const (
		code = iota
		lineComment
		blockComment
		literal
	)
	state := code
	var quote byte
	for i := 0; i < len(out); i++ {
		c := out[i]
		switch state {
		case code:
			switch {
			case c == '/' && i+1 < len(out) && out[i+1] == '/':
				state = lineComment
				out[i] = ' '
			case c == '/' && i+1 < len(out) && out[i+1] == '*':
				state = blockComment
				out[i], out[i+1] = ' ', ' '
				i++
			case c == '"' || c == '\'':
				state, quote = literal, c
			}
		case lineComment:
			if c == '\n' {
				state = code
			} else {
				out[i] = ' '
			}
		case blockComment:
			if c == '*' && i+1 < len(out) && out[i+1] == '/' {
				out[i], out[i+1] = ' ', ' '
				i++
				state = code
			} else if c != '\n' {
				out[i] = ' '
			}
		case literal:
			switch {
			case c == '\\' && i+1 < len(out) && out[i+1] != '\n':
				i++
			case c == quote || c == '\n':
				state = code
			}
		}
	}
	return out
}

// cppScan은 모듈·멤버 폴더의 C++ 파일에서 include를 읽어 depends_on과 대조한다.
// 모듈 path는 각 include 디렉토리 아래 폴더이고, 하위 폴더도 그 모듈이다. 겹치면 가장 깊은 폴더가 이긴다.
func (s *scanner) cppScan(us []unit) {
	var incDirs []string
	for _, d := range s.m.CPP.IncludeDirs {
		dir := filepath.Join(s.root, filepath.FromSlash(d.Name))
		if st, err := os.Stat(dir); err != nil || !st.IsDir() {
			s.diags.Errorf(d.Pos, "include 디렉토리가 없다: %s", d.Name)
			continue
		}
		incDirs = append(incDirs, dir)
	}
	owners := map[string]string{} // 폴더 → 모듈·멤버 ID
	for _, u := range us {
		for _, inc := range incDirs {
			owners[filepath.Join(inc, filepath.FromSlash(u.dir))] = u.id
		}
	}
	owner := func(file string) (string, bool) {
		for dir := filepath.Dir(file); ; dir = filepath.Dir(dir) {
			if id, ok := owners[dir]; ok {
				return id, true
			}
			if parent := filepath.Dir(dir); parent == dir {
				return "", false
			}
		}
	}
	resolve := func(from string, inc include) (string, bool) {
		var cands []string
		if !inc.angle {
			cands = append(cands, filepath.Join(filepath.Dir(from), filepath.FromSlash(inc.path)))
		}
		for _, d := range incDirs {
			cands = append(cands, filepath.Join(d, filepath.FromSlash(inc.path)))
		}
		for _, c := range cands {
			if st, err := os.Stat(c); err == nil && st.Mode().IsRegular() {
				rel, err := filepath.Rel(s.root, c)
				return c, err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
			}
		}
		return "", false
	}
	seen := map[string]bool{}
	unmanaged := map[string]bool{}
	for _, dir := range slices.Sorted(maps.Keys(owners)) {
		filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				if !os.IsNotExist(err) {
					s.diags.Errorf(yaml.Pos{File: p}, "C++ 폴더를 읽을 수 없다: %v", err)
				}
				return nil
			}
			if !d.Type().IsRegular() || !cppExts[filepath.Ext(p)] || seen[p] {
				return nil
			}
			seen[p] = true
			id, _ := owner(p)
			src, err := os.ReadFile(p)
			if err != nil {
				s.diags.Errorf(yaml.Pos{File: p}, "C++ 파일을 읽을 수 없다: %v", err)
				return nil
			}
			for _, inc := range includes(src) {
				target, ok := resolve(p, inc)
				if !ok {
					continue
				}
				pos := yaml.Pos{File: p, Line: inc.line, Col: inc.col}
				tid, managed := owner(target)
				if !managed {
					rel, _ := filepath.Rel(s.root, target)
					rel = filepath.ToSlash(rel)
					if !unmanaged[rel] {
						unmanaged[rel] = true
						s.diags.Infof(pos, "관리하지 않는 파일을 include한다: %s → %s", id, rel)
					}
					continue
				}
				if tid != id {
					s.depend(id, tid, pos, "include")
				}
			}
			return nil
		})
	}
}
