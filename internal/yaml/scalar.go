package yaml

import (
	"fmt"
	"strings"
)

// unquote는 s[0] == '"'인 문자열을 읽어 값, 소비한 바이트 수, 오류 메시지를 준다.
// 오류일 때 n은 오류 위치 오프셋이다.
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

// unsupportedScalar는 평문 스칼라가 지원하지 않는 문법으로 시작하면 메시지를 준다.
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
		return fmt.Sprintf("%q로 시작하는 값은 지원하지 않는다", s[0])
	}
	return ""
}
