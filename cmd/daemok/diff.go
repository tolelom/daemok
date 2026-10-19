package main

import (
	"flag"
	"fmt"
	"io"

	"github.com/tolelom/daemok/internal/designdiff"
)

func runDiff(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("diff", flag.ContinueOnError)
	fs.SetOutput(stderr)
	design, root := pathFlags(fs)
	base := fs.String("base", "HEAD", "비교할 기준 ref")
	if !parse(fs, args) {
		return 2
	}
	out, err := designdiff.Run(designDir(*design, *root), *base)
	if err != nil {
		fmt.Fprintf(stderr, "비교할 수 없다: %v\n", err)
		return 2
	}
	fmt.Fprint(stdout, out)
	return 0
}
