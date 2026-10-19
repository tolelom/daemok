package main

import (
	"flag"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/tolelom/daemok/internal/rename"
)

func runRename(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("rename", flag.ContinueOnError)
	fs.SetOutput(stderr)
	design, root := pathFlags(fs)
	code := fs.Bool("code", false, "생성 코드 이름도 새 ID를 따른다(고정 키를 더하지 않는다)")
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
	if !slices.Contains(rename.Kinds(), pos[0]) {
		fmt.Fprintf(stderr, "알 수 없는 종류 %q\n종류: %s\n", pos[0], strings.Join(rename.Kinds(), ", "))
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
