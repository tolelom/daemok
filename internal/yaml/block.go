package yaml

import (
	"fmt"
	"strings"
)

func Parse(file string, src []byte) (*Node, error) {
	p := &parser{file: file}
	if err := p.scan(src); err != nil {
		return nil, err
	}
	if len(p.lines) == 0 {
		return &Node{Kind: Map, Pos: Pos{file, 1, 1}}, nil
	}
	n, err := p.parseBlock(p.lines[0].indent)
	if err != nil {
		return nil, err
	}
	if p.i < len(p.lines) {
		ln := p.lines[p.i]
		return nil, p.errAt(ln.num, ln.indent+1, "예상하지 못한 들여쓰기")
	}
	return n, nil
}

func isSeqItem(text string) bool { return text == "-" || strings.HasPrefix(text, "- ") }

func (p *parser) parseBlock(indent int) (*Node, error) {
	if isSeqItem(p.lines[p.i].text) {
		return p.parseSeq(indent)
	}
	return p.parseMap(indent)
}

func (p *parser) parseMap(indent int) (*Node, error) {
	first := p.lines[p.i]
	n := &Node{Kind: Map, Pos: Pos{p.file, first.num, indent + 1}}
	seen := map[string]Pos{}
	for p.i < len(p.lines) {
		ln := p.lines[p.i]
		if ln.indent < indent {
			break
		}
		if ln.indent > indent {
			return nil, p.errAt(ln.num, ln.indent+1, "예상하지 못한 들여쓰기")
		}
		if isSeqItem(ln.text) {
			return nil, p.errAt(ln.num, ln.indent+1, "맵 안에 목록 항목이 있다")
		}
		key, keyPos, rest, restCol, err := p.splitKey(ln)
		if err != nil {
			return nil, err
		}
		if prev, dup := seen[key]; dup {
			return nil, &Error{keyPos, fmt.Sprintf("중복 키 %q (처음: %s)", key, prev)}
		}
		seen[key] = keyPos
		p.i++
		var val *Node
		if rest == "" {
			switch {
			case p.i < len(p.lines) && p.lines[p.i].indent > indent:
				val, err = p.parseBlock(p.lines[p.i].indent)
			case p.i < len(p.lines) && p.lines[p.i].indent == indent && isSeqItem(p.lines[p.i].text):
				val, err = p.parseSeq(indent)
			default:
				val = &Node{Kind: Scalar, Pos: Pos{p.file, ln.num, restCol}}
			}
		} else {
			val, err = p.parseInline(ln, rest, restCol)
			if err == nil {
				p.continuePlain(val, indent)
			}
		}
		if err != nil {
			return nil, err
		}
		n.Entries = append(n.Entries, Entry{key, keyPos, val})
	}
	return n, nil
}

func (p *parser) splitKey(ln line) (key string, keyPos Pos, rest string, restCol int, err error) {
	keyPos = Pos{p.file, ln.num, ln.indent + 1}
	text := ln.text
	switch {
	case strings.HasPrefix(text, "? "):
		return "", keyPos, "", 0, &Error{keyPos, "복합 키는 지원하지 않는다"}
	case strings.HasPrefix(text, "<<"):
		return "", keyPos, "", 0, &Error{keyPos, "병합 키는 지원하지 않는다"}
	}
	end := 0
	if strings.HasPrefix(text, "\"") {
		s, n, msg := unquote(text)
		if msg != "" {
			return "", keyPos, "", 0, &Error{Pos{p.file, ln.num, ln.indent + 1 + n}, msg}
		}
		if n >= len(text) || text[n] != ':' {
			return "", keyPos, "", 0, &Error{Pos{p.file, ln.num, ln.indent + 1 + n}, "키 뒤에 콜론이 없다"}
		}
		key, end = s, n
	} else {
		end = colonAfterKey(text)
		if end <= 0 {
			return "", keyPos, "", 0, &Error{keyPos, "키: 값 형식이 아니다"}
		}
		key = text[:end]
	}
	rest = text[end+1:]
	if rest != "" && rest[0] != ' ' {
		return "", keyPos, "", 0, &Error{Pos{p.file, ln.num, ln.indent + 1 + end}, "콜론 뒤에 공백이 필요하다"}
	}
	trimmed := strings.TrimLeft(rest, " ")
	restCol = ln.indent + 1 + len(text) - len(trimmed)
	return key, keyPos, trimmed, restCol, nil
}

func colonAfterKey(text string) int {
	for i := 0; i < len(text); i++ {
		if text[i] == ':' && (i+1 == len(text) || text[i+1] == ' ') {
			return i
		}
	}
	return -1
}

func (p *parser) parseSeq(indent int) (*Node, error) {
	first := p.lines[p.i]
	n := &Node{Kind: Seq, Pos: Pos{p.file, first.num, indent + 1}}
	for p.i < len(p.lines) {
		ln := p.lines[p.i]
		if ln.indent < indent {
			break
		}
		if ln.indent > indent {
			return nil, p.errAt(ln.num, ln.indent+1, "예상하지 못한 들여쓰기")
		}
		if !isSeqItem(ln.text) {
			break
		}
		body := strings.TrimLeft(ln.text[1:], " ")
		bodyIndent := ln.indent + len(ln.text) - len(body)
		if body == "" {
			p.i++
			if p.i < len(p.lines) && p.lines[p.i].indent > indent {
				item, err := p.parseBlock(p.lines[p.i].indent)
				if err != nil {
					return nil, err
				}
				n.Items = append(n.Items, item)
			} else {
				n.Items = append(n.Items, &Node{Kind: Scalar, Pos: Pos{p.file, ln.num, bodyIndent + 1}})
			}
			continue
		}
		if isSeqItem(body) {
			return nil, p.errAt(ln.num, bodyIndent+1, "한 줄에 목록 항목을 겹쳐 쓸 수 없다")
		}
		if looksLikeKey(body) {
			p.lines[p.i] = line{ln.num, bodyIndent, body}
			item, err := p.parseMap(bodyIndent)
			if err != nil {
				return nil, err
			}
			n.Items = append(n.Items, item)
			continue
		}
		p.i++
		item, err := p.parseInline(ln, body, bodyIndent+1)
		if err != nil {
			return nil, err
		}
		p.continuePlain(item, indent)
		n.Items = append(n.Items, item)
	}
	return n, nil
}

func looksLikeKey(body string) bool {
	switch body[0] {
	case '{', '[', '"':
		return false
	}
	return colonAfterKey(body) > 0
}

func (p *parser) parseInline(ln line, rest string, col int) (*Node, error) {
	switch rest[0] {
	case '{', '[':
		return p.parseFlow(ln, rest, col)
	case '"':
		s, n, msg := unquote(rest)
		if msg != "" {
			return nil, p.errAt(ln.num, col+n, msg)
		}
		if strings.TrimSpace(rest[n:]) != "" {
			return nil, p.errAt(ln.num, col+n, "닫는 따옴표 뒤에 문자가 있다")
		}
		return &Node{Kind: Scalar, Pos: Pos{p.file, ln.num, col}, Value: s, Quoted: true}, nil
	}
	if msg := unsupportedScalar(rest); msg != "" {
		return nil, p.errAt(ln.num, col, msg)
	}
	return &Node{Kind: Scalar, Pos: Pos{p.file, ln.num, col}, Value: rest}, nil
}

func (p *parser) continuePlain(n *Node, indent int) {
	if n.Kind != Scalar || n.Quoted || n.Value == "" {
		return
	}
	for p.i < len(p.lines) {
		ln := p.lines[p.i]
		if ln.indent <= indent || isSeqItem(ln.text) || colonAfterKey(ln.text) >= 0 {
			return
		}
		n.Value += " " + ln.text
		p.i++
	}
}
