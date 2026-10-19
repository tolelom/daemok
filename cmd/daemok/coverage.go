package main

import (
	"flag"
	"fmt"
	"io"

	"github.com/tolelom/daemok/internal/check"
	"github.com/tolelom/daemok/internal/coverage"
	"github.com/tolelom/daemok/internal/model"
)

func runCoverage(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("coverage", flag.ContinueOnError)
	fs.SetOutput(stderr)
	design, root := pathFlags(fs)
	if !parse(fs, args) {
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
	rows, bad, err := coverage.Count(m, *root)
	if err != nil {
		fmt.Fprintf(stderr, "셀 수 없다: %v\n", err)
		return 2
	}
	var sum coverage.Row
	fmt.Fprintln(stdout, "| 패키지 | 최상위 선언 | 설계 대응 | 비율 | 함수 제외 비율 |\n|---|---|---|---|---|")
	for _, r := range rows {
		sum.Decls += r.Decls
		sum.Matched += r.Matched
		sum.Funcs += r.Funcs
		sum.FuncsMatched += r.FuncsMatched
		fmt.Fprintf(stdout, "| `%s` | %d | %d | %s | %s |\n", r.Pkg, r.Decls, r.Matched, percent(r.Matched, r.Decls), percent(r.Matched-r.FuncsMatched, r.Decls-r.Funcs))
	}
	fmt.Fprintf(stdout, "| 합계 | %d | %d | %s | %s |\n", sum.Decls, sum.Matched, percent(sum.Matched, sum.Decls), percent(sum.Matched-sum.FuncsMatched, sum.Decls-sum.Funcs))
	fmt.Fprintf(stdout, "\n함수 제외: 선언 %d개 중 %d개. 함수 %d개 중 설계 대응 %d개\n", sum.Decls-sum.Funcs, sum.Matched-sum.FuncsMatched, sum.Funcs, sum.FuncsMatched)
	for _, b := range bad {
		fmt.Fprintf(stderr, "세지 못한 Go 파일: %v\n", b.Err)
	}
	if len(bad) > 0 {
		return 1
	}
	return 0
}

func percent(a, b int) string {
	if b == 0 {
		return "-"
	}
	return fmt.Sprintf("%d%%", a*100/b)
}
