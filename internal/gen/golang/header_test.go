package golang

import (
	"go/parser"
	"go/token"
	"testing"
)

// daemok이 만든 파일은 첫 줄이 정확히 생성 헤더인 파일뿐이다. 줄 끝 \r은 무시한다.
func TestIsDaemokGenerated(t *testing.T) {
	bom := string([]byte{0xEF, 0xBB, 0xBF})
	cases := []struct {
		name string
		src  string
		want bool
	}{
		{"LF 헤더", Header + "\n\npackage p\n", true},
		{"CRLF 헤더", Header + "\r\n\r\npackage p\r\n", true},
		{"헤더와 \r로 끝나는 파일", Header + "\r", true},
		{"빈 줄로 시작", "\n" + Header + "\n\npackage p\n", false},
		{"BOM으로 시작", bom + Header + "\n\npackage p\n", false},
		{"주석 블록 둘째 줄", "// 설명\n" + Header + "\n\npackage p\n", false},
		{"헤더 뒤에 글자", Header + " x\n\npackage p\n", false},
	}
	for _, c := range cases {
		if got := IsDaemokGenerated([]byte(c.src)); got != c.want {
			t.Errorf("%s: IsDaemokGenerated = %v, want %v", c.name, got, c.want)
		}
		f, err := parser.ParseFile(token.NewFileSet(), "x.go", c.src, parser.ParseComments)
		if err != nil {
			continue // package 절이 없는 파일은 파싱한 파일로 판정할 일이 없다
		}
		if got := IsDaemokGeneratedFile(f); got != c.want {
			t.Errorf("%s: IsDaemokGeneratedFile = %v, want %v", c.name, got, c.want)
		}
	}
}
