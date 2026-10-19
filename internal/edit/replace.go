package edit

import (
	"bytes"
	"fmt"
	"sort"

	"github.com/tolelom/daemok/internal/yaml"
)

type Replacement struct {
	Pos      yaml.Pos
	Old, New string
}

const plainStop = ",]}#:\r\n \t"

func Replace(src []byte, file string, reps []Replacement) ([]byte, error) {
	t, err := parse(src, file)
	if err != nil {
		return nil, err
	}
	sorted := append([]Replacement(nil), reps...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[j].Pos.Less(sorted[i].Pos) })
	out := src
	for i, r := range sorted {
		if i > 0 && r.Pos == sorted[i-1].Pos {
			if r != sorted[i-1] {
				return nil, fmt.Errorf("%s: 같은 위치에 다른 교체가 있다: %q, %q", r.Pos, sorted[i-1].New, r.New)
			}
			continue
		}
		s := t.off(r.Pos)
		var inner []byte
		if s < len(src) && src[s] == '"' {
			e := t.tokenEnd(s)
			if e-s < 2 || src[e-1] != '"' {
				return nil, fmt.Errorf("%s: 닫히지 않은 따옴표", r.Pos)
			}
			inner, s = src[s+1:e-1], s+1
			if bytes.IndexByte(inner, '\\') >= 0 {
				return nil, fmt.Errorf("%s: 이스케이프가 든 토큰은 바꾸지 않는다: %s", r.Pos, src[s-1:e])
			}
		} else {
			e := min(s+len(r.Old), len(src))
			for e < len(src) && bytes.IndexByte([]byte(plainStop), src[e]) < 0 {
				e++
			}
			inner = src[s:e]
		}
		if string(inner) != r.Old {
			return nil, fmt.Errorf("%s: 원문 토큰이 %q가 아니라 %q다", r.Pos, r.Old, inner)
		}
		out = splice(out, s, s+len(inner), r.New)
	}
	return out, nil
}
