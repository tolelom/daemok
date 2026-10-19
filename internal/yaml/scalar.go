package yaml

import (
	"fmt"
	"strings"
)

func unquote(s string) (val string, n int, msg string) {
	var b strings.Builder
	for i := 1; i < len(s); i++ {
		switch s[i] {
		case '"':
			return b.String(), i + 1, ""
		case '\\':
			if i+1 >= len(s) {
				return "", i, "닫히지 않은 따옴표"
			}
			i++
			switch s[i] {
			case '"':
				b.WriteByte('"')
			case '\\':
				b.WriteByte('\\')
			case 'n':
				b.WriteByte('\n')
			case 't':
				b.WriteByte('\t')
			default:
				return "", i - 1, fmt.Sprintf("지원하지 않는 이스케이프 \\%c", s[i])
			}
		default:
			b.WriteByte(s[i])
		}
	}
	return "", len(s), "닫히지 않은 따옴표"
}

func unsupportedScalar(s string) string {
	switch s[0] {
	case '\'':
		return "홑따옴표 문자열은 지원하지 않는다. \"...\"를 쓴다"
	case '&', '*':
		return "앵커와 별칭은 지원하지 않는다"
	case '!':
		return "태그는 지원하지 않는다"
	case '|', '>':
		return "여러 줄 문자열은 지원하지 않는다"
	case '%', '@', '`':
		return fmt.Sprintf("지원하지 않는 첫 문자: %q", s[0])
	}
	return ""
}
