package edit

import (
	"errors"
	"fmt"

	"github.com/tolelom/daemok/internal/yaml"
)

// ErrNodeNotInFile은 고정 키를 더할 노드가 이 파일에 없다는 뜻이다.
var ErrNodeNotInFile = errors.New("이 파일에 노드가 없다")

// ErrKeyExists는 더할 키가 노드에 이미 있다는 뜻이다.
var ErrKeyExists = errors.New("이미 있는 키")

// Pin은 path가 가리키는 맵 노드에 key: value(sub가 비면) 또는 key: { sub: value }를 더한다.
// key 맵이 이미 있으면 그 안에 sub: value를 더한다. 같은 키가 이미 있으면 거부한다.
// 흐름 형식이면 닫는 괄호 앞에, 블록 형식이면 노드의 마지막 내용 줄 뒤에 같은 들여쓰기로 넣는다.
func Pin(src []byte, file string, path []string, key, sub, value string) ([]byte, error) {
	t, err := parse(src, file)
	if err != nil {
		return nil, err
	}
	n, _ := yaml.Parse(file, src)
	for _, k := range path {
		n = n.Get(k)
		if n == nil {
			return nil, fmt.Errorf("%w: %v", ErrNodeNotInFile, path)
		}
	}
	if n.Kind != yaml.Map {
		return nil, fmt.Errorf("맵 노드가 아니다: %v", path)
	}
	target, k, v := n, key, value
	if sub != "" {
		if inner := n.Get(key); inner != nil {
			if inner.Kind != yaml.Map {
				return nil, fmt.Errorf("%s가 맵이 아니다: %v", key, path)
			}
			target, k = inner, sub
		} else {
			v = "{ " + sub + ": " + value + " }"
		}
	}
	if target.Get(k) != nil {
		return nil, fmt.Errorf("%w %s: %v", ErrKeyExists, k, path)
	}
	if t.flow(target) {
		end := t.closing(t.off(target.Pos))
		at := end
		for at > 0 && src[at-1] == ' ' {
			at--
		}
		if len(target.Entries) == 0 {
			return splice(src, t.off(target.Pos)+1, end, " "+k+": "+v+" "), nil
		}
		return splice(src, at, at, ", "+k+": "+v), nil
	}
	if len(target.Entries) == 0 {
		return nil, fmt.Errorf("빈 블록 맵에는 더하지 않는다: %v", path)
	}
	// 블록 맵의 위치는 첫 항목 위치라 들여쓰기는 첫 항목 열에서 얻는다.
	indent := target.Entries[0].Pos.Col - 1
	at := t.blockEnd(target.Entries[0].Pos.Line, indent)
	return t.insertLine(at, spaces(indent)+k+": "+v), nil
}
