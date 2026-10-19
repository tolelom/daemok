package lift

import (
	"regexp"
	"strings"
	"unicode"

	"github.com/tolelom/daemok/internal/model"
)

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

func snakeID(name string) (id, goName string, ok bool) {
	rs := []rune(name)
	var parts []string
	start := 0
	for i := 1; i < len(rs); i++ {
		j := i - 1
		for j > 0 && unicode.IsDigit(rs[j]) {
			j--
		}
		lowerToUpper := unicode.IsUpper(rs[i]) && unicode.IsLower(rs[j])
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
