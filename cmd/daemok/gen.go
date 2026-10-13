package main

import (
	"flag"
	"fmt"
	"io"

	"github.com/tolelom/daemok/internal/workspace"
)

// runGen은 생성 파일을 쓰고 지운다. 설계를 읽지 못하거나 검사 오류가 있으면 2,
// 생성 경로에 헤더 없는 파일이 있으면 아무것도 쓰지 않고 1이다.
func runGen(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("gen", flag.ContinueOnError)
	fs.SetOutput(stderr)
	design, root := pathFlags(fs)
	dryRun := fs.Bool("dry-run", false, "쓸 파일, 지울 파일, 거부할 파일 목록만 출력한다")
	if !parse(fs, args) {
		return 2
	}
	r := workspace.Run(designDir(*design, *root), *root)
	if r.Fatal || r.Plan == nil {
		printDiags(stdout, r.Diags)
		return 2
	}
	p := r.Plan
	for _, d := range r.Others {
		fmt.Fprintln(stdout, d.String())
	}
	for _, f := range p.Refuse {
		fmt.Fprintf(stdout, "거부: %s (%s)\n", f.Path, f.Reason)
	}
	if len(p.Refuse) > 0 {
		return 1
	}
	for _, f := range p.Write {
		fmt.Fprintf(stdout, "쓰기: %s\n", f.Path)
	}
	for _, d := range p.Delete {
		fmt.Fprintf(stdout, "지우기: %s\n", d)
	}
	if *dryRun {
		return 0
	}
	if err := workspace.Apply(*root, p); err != nil {
		fmt.Fprintf(stderr, "쓸 수 없다: %v\n", err)
		return 2
	}
	fmt.Fprintf(stdout, "쓴 파일 %d, 지운 파일 %d\n", len(p.Write), len(p.Delete))
	return 0
}
