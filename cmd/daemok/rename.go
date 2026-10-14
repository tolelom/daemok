package main

import (
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/tolelom/daemok/internal/rename"
)

// runRename은 설계 노드 하나의 ID와 그 참조를 바꾼다. 기본은 코드 이름을 고정 키로 유지한다.
// 인자 오류는 2, 거부는 1이다.
func runRename(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("rename", flag.ContinueOnError)
	fs.SetOutput(stderr)
	design, root := pathFlags(fs)
	code := fs.Bool("code", false, "생성 코드 이름도 새 ID를 따른다(고정 키를 더하지 않는다)")
	// flag는 첫 위치 인자에서 멈추므로 위치 인자를 하나씩 떼며 나머지 플래그를 계속 읽는다.
	var pos []string
	for {
		if fs.Parse(args) != nil {
			return 2
		}
		if fs.NArg() == 0 {
			break
		}
		pos = append(pos, fs.Arg(0))
		args = fs.Args()[1:]
	}
	if len(pos) != 3 {
		fmt.Fprintf(stderr, "사용법: daemok rename <종류> <옛 ID> <새 ID> [--code] [--design DIR] [--root DIR]\n종류: %s\n", strings.Join(rename.Kinds(), ", "))
		return 2
	}
	res, err := rename.Run(designDir(*design, *root), *root, pos[0], pos[1], pos[2], rename.Options{Code: *code})
	if err != nil {
		fmt.Fprintf(stderr, "바꾸지 않았다: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "고친 파일: %s\n", strings.Join(res.Files, ", "))
	if res.Pinned != "" {
		fmt.Fprintf(stdout, "고정: %s\n", res.Pinned)
	}
	for _, mv := range res.Moves {
		fmt.Fprintf(stdout, "옮길 디렉토리: %s\n", mv)
	}
	for _, n := range res.Notes {
		fmt.Fprintf(stdout, "안내: %s\n", n)
	}
	return 0
}
