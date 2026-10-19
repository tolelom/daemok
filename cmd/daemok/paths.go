package main

import (
	"flag"
	"fmt"

	"github.com/tolelom/daemok/internal/workspace"
)

func pathFlags(fs *flag.FlagSet) (design, root *string) {
	design = fs.String("design", "", "설계 디렉토리 (기본 <root>/design)")
	root = fs.String("root", ".", "코드 루트")
	return design, root
}

func parse(fs *flag.FlagSet, args []string) bool {
	if fs.Parse(args) != nil {
		return false
	}
	if fs.NArg() > 0 {
		fmt.Fprintf(fs.Output(), "알 수 없는 인자: %q (경로는 --design, --root로 준다)\n", fs.Args())
		return false
	}
	return true
}

func designDir(design, root string) string {
	if design == "" {
		return workspace.DefaultDesignDir(root)
	}
	return design
}
