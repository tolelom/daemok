package yaml

import (
	"fmt"
	"strings"
)

type fc struct {
	r   byte
	pos Pos
}

func (p *parser) parseFlow(ln line, rest string, col int) (*Node, error) {
	var cs []fc
	push := func(s string, num, col int) {
		for i := 0; i < len(s); i++ {
			cs = append(cs, fc{s[i], Pos{p.file, num, col + i}})
		}
	}
	push(rest, ln.num, col)
	depth, err := flowDepth(rest, 0)
	if err != "" {
		return nil, p.errAt(ln.num, col, err)
	}
	if depth > 0 && rest[0] == '{' {
		return nil, p.errAt(ln.num, col, "{ }는 한 줄 안에서 닫혀야 한다")
	}
	for depth > 0 {
		if p.i >= len(p.lines) {
			return nil, p.errAt(ln.num, col, "닫히지 않은 [")
		}
		nx := p.lines[p.i]
		p.i++
		cs = append(cs, fc{' ', Pos{p.file, nx.num, nx.indent}})
		push(nx.text, nx.num, nx.indent+1)
		if depth, err = flowDepth(nx.text, depth); err != "" {
			return nil, p.errAt(nx.num, nx.indent+1, err)
		}
	}
	f := &flow{cs: cs}
	n, e := f.value()
	if e != nil {
		return nil, e
	}
	f.skipSpace()
	if !f.eof() {
		return nil, f.err("닫는 괄호 뒤에 문자가 있다")
	}
	return n, nil
}

func flowDepth(s string, depth int) (int, string) {
	inQ := false
	for i := 0; i < len(s); i++ {
		switch {
		case inQ && s[i] == '\\':
			i++
		case s[i] == '"':
			inQ = !inQ
		case inQ:
		case s[i] == '{' || s[i] == '[':
			depth++
		case s[i] == '}' || s[i] == ']':
			depth--
		}
	}
	if inQ {
		return depth, "닫히지 않은 따옴표"
	}
	return depth, ""
}

type flow struct {
	cs []fc
	i  int
}

func (f *flow) eof() bool  { return f.i >= len(f.cs) }
func (f *flow) peek() byte { return f.cs[f.i].r }

func (f *flow) pos() Pos {
	if f.eof() {
		last := f.cs[len(f.cs)-1].pos
		last.Col++
		return last
	}
	return f.cs[f.i].pos
}

func (f *flow) err(msg string) error { return &Error{f.pos(), msg} }

func (f *flow) skipSpace() {
	for !f.eof() && f.peek() == ' ' {
		f.i++
	}
}

func (f *flow) rest() string {
	b := make([]byte, 0, len(f.cs)-f.i)
	for _, c := range f.cs[f.i:] {
		b = append(b, c.r)
	}
	return string(b)
}

func (f *flow) value() (*Node, error) {
	f.skipSpace()
	if f.eof() {
		return nil, f.err("값이 없다")
	}
	switch f.peek() {
	case '{':
		return f.mapping()
	case '[':
		return f.sequence()
	case '"':
		return f.quoted()
	}
	return f.plain()
}

func (f *flow) quoted() (*Node, error) {
	pos := f.pos()
	s, n, msg := unquote(f.rest())
	if msg != "" {
		f.i += n
		return nil, f.err(msg)
	}
	f.i += n
	return &Node{Kind: Scalar, Pos: pos, Value: s, Quoted: true}, nil
}

func (f *flow) plainText(stops string) string {
	start := f.i
	for !f.eof() && strings.IndexByte(stops, f.peek()) < 0 {
		f.i++
	}
	b := make([]byte, 0, f.i-start)
	for _, c := range f.cs[start:f.i] {
		b = append(b, c.r)
	}
	return strings.TrimRight(string(b), " ")
}

func (f *flow) plain() (*Node, error) {
	pos := f.pos()
	start := f.i
	s := f.plainText(",}]")
	if s == "" {
		return nil, &Error{pos, "값이 없다"}
	}
	if i := strings.Index(s, ": "); i >= 0 || strings.HasSuffix(s, ":") {
		if i < 0 {
			i = len(s) - 1
		}
		return nil, &Error{f.cs[start+i].pos, "쉼표가 빠졌거나 값 안에 \": \"가 있다"}
	}
	if msg := unsupportedScalar(s); msg != "" {
		return nil, &Error{pos, msg}
	}
	return &Node{Kind: Scalar, Pos: pos, Value: s}, nil
}

func (f *flow) afterItem(close byte) (bool, error) {
	f.skipSpace()
	if f.eof() {
		return false, f.err(fmt.Sprintf("닫히지 않은 %c", open(close)))
	}
	switch f.peek() {
	case ',':
		f.i++
		f.skipSpace()
		if !f.eof() && f.peek() == close {
			return false, f.err("쉼표 뒤에 항목이 없다")
		}
		return false, nil
	case close:
		f.i++
		return true, nil
	}
	return false, f.err(fmt.Sprintf("쉼표나 %c가 와야 한다", close))
}

func open(close byte) byte {
	if close == '}' {
		return '{'
	}
	return '['
}

func (f *flow) mapping() (*Node, error) {
	n := &Node{Kind: Map, Pos: f.pos()}
	f.i++
	seen := map[string]Pos{}
	f.skipSpace()
	if !f.eof() && f.peek() == '}' {
		f.i++
		return n, nil
	}
	for {
		f.skipSpace()
		if f.eof() {
			return nil, f.err("닫히지 않은 {")
		}
		keyPos := f.pos()
		var key string
		if f.peek() == '"' {
			q, err := f.quoted()
			if err != nil {
				return nil, err
			}
			key = q.Value
		} else {
			key = f.plainText(":,}]")
			if key == "" {
				return nil, &Error{keyPos, "키가 없다"}
			}
		}
		f.skipSpace()
		if f.eof() || f.peek() != ':' {
			return nil, f.err("키 뒤에 콜론이 없다")
		}
		f.i++
		if !f.eof() && f.peek() != ' ' && f.peek() != ',' && f.peek() != '}' {
			return nil, f.err("콜론 뒤에 공백이 필요하다")
		}
		if prev, dup := seen[key]; dup {
			return nil, &Error{keyPos, fmt.Sprintf("중복 키 %q (처음: %s)", key, prev)}
		}
		seen[key] = keyPos
		f.skipSpace()
		var val *Node
		if !f.eof() && (f.peek() == ',' || f.peek() == '}') {
			val = &Node{Kind: Scalar, Pos: f.pos()}
		} else {
			v, err := f.value()
			if err != nil {
				return nil, err
			}
			val = v
		}
		n.Entries = append(n.Entries, Entry{key, keyPos, val})
		closed, err := f.afterItem('}')
		if err != nil {
			return nil, err
		}
		if closed {
			return n, nil
		}
	}
}

func (f *flow) sequence() (*Node, error) {
	n := &Node{Kind: Seq, Pos: f.pos()}
	f.i++
	f.skipSpace()
	if !f.eof() && f.peek() == ']' {
		f.i++
		return n, nil
	}
	for {
		item, err := f.value()
		if err != nil {
			return nil, err
		}
		n.Items = append(n.Items, item)
		closed, err := f.afterItem(']')
		if err != nil {
			return nil, err
		}
		if closed {
			return n, nil
		}
	}
}
