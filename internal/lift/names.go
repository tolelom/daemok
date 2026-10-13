package lift

import (
	"regexp"
	"strings"
	"unicode"

	"github.com/tolelom/daemok/internal/model"
)

// quote는 설계 YAML의 큰따옴표 문자열로 쓴다. 파서가 받는 이스케이프(\" \\ \n \t)만 쓴다.
func quote(s string) (string, bool) {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range s {
		switch {
		case r == '"' || r == '\\':
			b.WriteRune('\\')
			b.WriteRune(r)
		case r == '\n':
			b.WriteString(`\n`)
		case r == '\t':
			b.WriteString(`\t`)
		case !unicode.IsPrint(r):
			return "", false
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return b.String(), true
}

var snakeName = regexp.MustCompile(`^[a-z][a-z0-9]*(_[a-z0-9]+)*$`)

// snakeID는 Go 이름을 snake_case ID로 바꾼다. GoName(ID)가 원래 이름과 다르면 goName에 원래 이름을 준다.
// 대문자 연속(HTTPPort)은 마지막 대문자 앞에서 끊는다.
func snakeID(name string) (id, goName string, ok bool) {
	rs := []rune(name)
	var parts []string
	start := 0
	for i := 1; i < len(rs); i++ {
		// 숫자 뒤의 대문자는 숫자 앞이 소문자일 때만 끊는다(Sha256Size, P2PPort).
		j := i - 1
		for j > 0 && unicode.IsDigit(rs[j]) {
			j--
		}
		lowerToUpper := unicode.IsUpper(rs[i]) && unicode.IsLower(rs[j])
		// 약어 뒤의 복수 s(IDs, URLsByID)는 약어에 붙인다.
		plural := i+1 < len(rs) && rs[i+1] == 's' && (i+2 == len(rs) || unicode.IsUpper(rs[i+2]))
		acronymEnd := unicode.IsUpper(rs[i]) && unicode.IsUpper(rs[i-1]) && i+1 < len(rs) && unicode.IsLower(rs[i+1]) && !plural
		if lowerToUpper || acronymEnd {
			parts = append(parts, string(rs[start:i]))
			start = i
		}
	}
	parts = append(parts, string(rs[start:]))
	id = strings.ToLower(strings.Join(parts, "_"))
	if !snakeName.MatchString(id) {
		return "", "", false
	}
	if model.GoName(id) != name {
		goName = name
	}
	return id, goName, true
}
