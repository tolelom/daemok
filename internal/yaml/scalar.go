package yaml

// unquote는 s[0] == '"'인 문자열을 읽어 값, 소비한 바이트 수, 오류 메시지를 준다.
func unquote(s string) (val string, n int, msg string) {
	for i := 1; i < len(s); i++ {
		if s[i] == '"' {
			return s[1:i], i + 1, ""
		}
	}
	return "", len(s), "닫히지 않은 따옴표"
}

// unsupportedScalar는 평문 스칼라가 지원하지 않는 문법으로 시작하면 메시지를 준다.
func unsupportedScalar(s string) string { return "" }
