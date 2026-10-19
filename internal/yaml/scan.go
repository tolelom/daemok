package yaml

import "strings"

type line struct {
	num    int
	indent int
	text   string
}

type parser struct {
	file  string
	lines []line
	i     int
}

func (p *parser) errAt(num, col int, msg string) error {
	return &Error{Pos: Pos{p.file, num, col}, Msg: msg}
}

func (p *parser) scan(src []byte) error {
	text := strings.TrimPrefix(string(src), "\ufeff")
	raw := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	for idx, r := range raw {
		num := idx + 1
		indent := 0
		for indent < len(r) && r[indent] == ' ' {
			indent++
		}
		if indent < len(r) && r[indent] == '\t' {
			return p.errAt(num, indent+1, "탭 들여쓰기는 지원하지 않는다")
		}
		text := strings.TrimRight(stripComment(r[indent:]), " ")
		if text == "" {
			continue
		}
		if text == "---" || text == "..." {
			return p.errAt(num, indent+1, "한 파일에 여러 문서는 지원하지 않는다")
		}
		p.lines = append(p.lines, line{num, indent, text})
	}
	return nil
}

func stripComment(s string) string {
	inQuote := false
	for i := 0; i < len(s); i++ {
		switch {
		case inQuote && s[i] == '\\':
			i++
		case s[i] == '"':
			inQuote = !inQuote
		case s[i] == '#' && !inQuote && (i == 0 || s[i-1] == ' '):
			return s[:i]
		}
	}
	return s
}
