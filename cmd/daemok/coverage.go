package main

import (
	"flag"
	"fmt"
	"io"

	"github.com/tolelom/daemok/internal/check"
	"github.com/tolelom/daemok/internal/coverage"
	"github.com/tolelom/daemok/internal/model"
)

// runCoverage는 패키지마다 최상위 선언 중 설계 노드에 대응하는 것의 비율을 출력한다.
// 설계를 읽지 못하거나 설계 검사 오류가 있으면 2다.
func runCoverage(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("coverage", flag.ContinueOnError)
	fs.SetOutput(stderr)
	design, root := pathFlags(fs)
	if err := fs.Parse(args); err != nil {
		return 2
	}
	m, diags := model.Load(designDir(*design, *root))
	if !diags.HasErrors() {
		diags = append(diags, check.Run(m)...)
	}
	if diags.HasErrors() {
		printDiags(stdout, diags)
		fmt.Fprintln(stderr, "설계 오류가 있어 셀 수 없다")
		return 2
	}
	rows, err := coverage.Count(m, *root)
	if err != nil {
		fmt.Fprintf(stderr, "셀 수 없다: %v\n", err)
		return 2
	}
	var decls, matched int
	fmt.Fprintln(stdout, "| 패키지 | 최상위 선언 | 설계 대응 | 비율 |\n|---|---|---|---|")
	for _, r := range rows {
		decls += r.Decls
		matched += r.Matched
		fmt.Fprintf(stdout, "| `%s` | %d | %d | %s |\n", r.Pkg, r.Decls, r.Matched, percent(r.Matched, r.Decls))
	}
	fmt.Fprintf(stdout, "| 합계 | %d | %d | %s |\n", decls, matched, percent(matched, decls))
	return 0
}

func percent(a, b int) string {
	if b == 0 {
		return "-"
	}
	return fmt.Sprintf("%d%%", a*100/b)
}
