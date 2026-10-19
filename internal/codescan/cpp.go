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

type include struct {
	line, col int
	path      string
	angle     bool
}

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
			case c == '\'' && i > 0 && isAlnum(out[i-1]):
			case c == '"' || c == '\'':
				state, quote = literal, c
			}
		case lineComment:
			if c == '\n' && !continued(src, i) {
				state = code
			} else if c != '\n' {
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
	owners := map[string]string{}
	for _, u := range us {
		for _, inc := range incDirs {
			dir := filepath.Join(inc, filepath.FromSlash(u.dir))
			if prev, ok := owners[dir]; ok && prev != u.id {
				s.diags.Errorf(s.m.CPP.Pos, "두 모듈이 같은 C++ 폴더를 가진다: %s, %s (%s)", prev, u.id, dir)
				continue
			}
			owners[dir] = u.id
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
			if st, err := os.Lstat(c); err == nil && st.Mode()&fs.ModeSymlink != 0 {
				return "", false
			} else if err == nil && st.Mode().IsRegular() {
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
			if !d.Type().IsRegular() || !cppExts[strings.ToLower(filepath.Ext(p))] || seen[p] {
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

func isAlnum(c byte) bool {
	return c >= '0' && c <= '9' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c == '_'
}

func continued(src []byte, i int) bool {
	j := i - 1
	if j >= 0 && src[j] == '\r' {
		j--
	}
	return j >= 0 && src[j] == '\\'
}
