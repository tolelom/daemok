package main

import (
	"flag"
	"fmt"
	"io"

	"github.com/tolelom/daemok/internal/model"
	"github.com/tolelom/daemok/internal/workspace"
)

func runCheck(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("check", flag.ContinueOnError)
	fs.SetOutput(stderr)
	design := fs.String("design", "design", "설계 디렉토리")
	root := fs.String("root", ".", "코드 루트")
	strict := fs.Bool("strict", false, "경고도 실패로 본다")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	r := workspace.Run(*design, *root)
	printDiags(stdout, r.Diags)
	switch {
	case r.Fatal:
		return 2
	case r.Diags.HasErrors() || (*strict && r.Diags.Count(model.Warning) > 0):
		return 1
	}
	return 0
}

func printDiags(w io.Writer, diags model.Diagnostics) {
	for _, d := range diags {
		fmt.Fprintln(w, d.String())
	}
	fmt.Fprintf(w, "오류 %d, 경고 %d, 정보 %d\n", diags.Count(model.Error), diags.Count(model.Warning), diags.Count(model.Info))
}
