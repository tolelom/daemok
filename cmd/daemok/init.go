package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/tolelom/daemok/internal/infer"
)

// runInit은 설계 파일이 없는 Go 모듈의 코드에서 초안을 만들어 설계 디렉토리에 쓴다.
// 이미 설계 파일이 있으면 쓰지 않고 1, Go 모듈이 아니거나 쓰지 못하면 2다. --dry-run은 쓰지 않고 출력만 한다.
func runInit(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	fs.SetOutput(stderr)
	design, root := pathFlags(fs)
	dryRun := fs.Bool("dry-run", false, "쓰지 않고 초안과 분석 메모만 출력한다")
	if !parse(fs, args) {
		return 2
	}
	dir := designDir(*design, *root)
	a, err := infer.Analyze(*root)
	if err != nil {
		fmt.Fprintf(stderr, "분석할 수 없다: %v\n", err)
		return 2
	}
	if *dryRun {
		stdout.Write(a.YAML)
		fmt.Fprintln(stdout)
		printReport(stdout, a.Report)
		return 0
	}
	p, err := infer.Write(dir, a)
	switch {
	case errors.Is(err, infer.ErrDesignExists):
		fmt.Fprintf(stderr, "%v: %s\n", err, dir)
		return 1
	case err != nil:
		fmt.Fprintf(stderr, "쓸 수 없다: %v\n", err)
		return 2
	}
	printReport(stdout, a.Report)
	fmt.Fprintf(stdout, "쓴 파일: %s\n", p)
	return 0
}

func printReport(w io.Writer, r infer.Report) {
	fmt.Fprintf(w, "패키지 %d, 모듈 %d, 실행 단위 %d\n", r.Packages, r.Modules, r.Executables)
	for _, s := range r.Skipped {
		fmt.Fprintf(w, "건너뜀: %s (%s)\n", s.Path, s.Reason)
	}
	for _, c := range r.Cycles {
		fmt.Fprintf(w, "순환 import: %s\n", strings.Join(c, ", "))
	}
	for _, g := range r.GroupCandidates {
		fmt.Fprintf(w, "그룹 후보: %s (%s), 공통 의존 %s", g.Path, strings.Join(g.Members, ", "), strings.Join(g.Common, ", "))
		if len(g.Exceptions) > 0 {
			fmt.Fprintf(w, ", 예외 후보 %s", strings.Join(g.Exceptions, ", "))
		}
		fmt.Fprintln(w)
	}
	for _, e := range r.External {
		users := strings.Join(e.Users, ", ")
		if users == "" {
			users = "쓰는 모듈 없음"
		}
		fmt.Fprintf(w, "외부 의존: %s (%s)\n", e.Module, users)
	}
	for _, h := range r.Hints {
		kind := h.Kind
		if h.Type != "" {
			kind += " " + h.Type
		}
		fmt.Fprintf(w, "L3 후보(%s): %s %s (%s)\n", kind, h.Module, strings.Join(h.Names, ", "), h.Where)
	}
	if len(r.FanIn) > 0 {
		var xs []string
		for _, f := range r.FanIn {
			xs = append(xs, fmt.Sprintf("%s %d", f.ID, f.Count))
		}
		fmt.Fprintf(w, "많이 쓰이는 패키지: %s\n", strings.Join(xs, ", "))
	}
}
