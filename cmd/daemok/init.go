package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/tolelom/daemok/internal/infer"
	"github.com/tolelom/daemok/internal/lift"
)

// runInit은 설계 파일이 없는 Go 모듈의 코드에서 초안을 만들어 설계 디렉토리에 쓴다.
// 이미 설계 파일이 있으면 쓰지 않고 1, Go 모듈이 아니거나 쓰지 못하면 2다. --dry-run은 쓰지 않고 출력만 한다.
// --l3는 설계가 있는 저장소에서 L3 초안을 stdout에, 옮기지 않은 선언을 stderr에 낸다. 쓰지 않는다.
func runInit(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	fs.SetOutput(stderr)
	design, root := pathFlags(fs)
	dryRun := fs.Bool("dry-run", false, "쓰지 않고 초안과 분석 메모만 출력한다")
	l3 := fs.Bool("l3", false, "설계에 없는 손으로 쓴 선언을 gen: false L3 노드 초안으로 출력한다")
	if !parse(fs, args) {
		return 2
	}
	dir := designDir(*design, *root)
	if *l3 {
		return initL3(dir, *root, stdout, stderr)
	}
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

func initL3(dir, root string, stdout, stderr io.Writer) int {
	r, err := lift.Draft(dir, root)
	if err != nil {
		fmt.Fprintf(stderr, "초안을 만들 수 없다: %v\n", err)
		if errors.Is(err, lift.ErrDesign) {
			fmt.Fprintln(stderr, "daemok check로 설계 오류를 먼저 고친다")
		}
		return 2
	}
	stdout.Write(r.YAML)
	for _, s := range r.Skipped {
		fmt.Fprintf(stderr, "옮기지 않음: %s %s (%s)\n", s.Where, s.Name, s.Reason)
	}
	fmt.Fprintf(stderr, "노드 %d개. 설계에 없는 함수 %d개는 옮기지 않는다\n", len(r.Nodes), r.Funcs)
	c := r.Coverage
	fmt.Fprintf(stderr, "coverage %s → %s (선언 %d개 중 %d → %d)\n", percent(c.Before, c.Decls), percent(c.After, c.Decls), c.Decls, c.Before, c.After)
	return 0
}
