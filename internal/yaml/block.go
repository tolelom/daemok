package yaml

import (
	"fmt"
	"strings"
)

// Parse는 src를 파싱한다. 빈 파일은 빈 맵이다. 실패하면 *Error를 돌려준다.
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

// splitKey는 "key: rest"를 나눈다. rest가 비면 값이 다음 줄에 있거나 null이다.
// restCol은 rest가 시작하는 열이다.
func (p *parser) splitKey(ln line) (key string, keyPos Pos, rest string, restCol int, err error) {
	keyPos = Pos{p.file, ln.num, ln.indent + 1}
	text := ln.text
	switch {
	case strings.HasPrefix(text, "? "):
		return "", keyPos, "", 0, &Error{keyPos, "복합 키는 지원하지 않는다"}
	case strings.HasPrefix(text, "<<"):
		return "", keyPos, "", 0, &Error{keyPos, "병합 키는 지원하지 않는다"}
	}
	end := 0 // ':'의 인덱스
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

// colonAfterKey는 뒤에 공백이 오거나 줄 끝인 첫 ':'의 인덱스를 준다. 없으면 -1.
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
			break // 같은 열의 다음 키. 부모 맵이 이어 읽는다
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
			// "- key: v"는 줄을 key부터 시작하는 것으로 바꿔 맵으로 읽는다.
			// 이어지는 줄은 같은 열에 맞춰야 한다.
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

// parseInline은 "key: " 뒤나 "- " 뒤의 값을 읽는다. 플로우 컬렉션, 따옴표 문자열, 평문 스칼라.
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

// continuePlain은 평문 스칼라 n 뒤에서 indent보다 깊게 들여 쓴 줄을 이어 붙인다. 줄바꿈은 공백 하나가 된다.
// 키나 목록 항목처럼 보이는 줄은 이어 붙이지 않아 들여쓰기 오류로 남는다.
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
