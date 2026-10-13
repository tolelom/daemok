package main

import (
	"flag"
	"fmt"
	"path/filepath"
)

// pathFlags는 모든 명령이 같은 뜻으로 쓰는 --design과 --root를 등록한다.
func pathFlags(fs *flag.FlagSet) (design, root *string) {
	design = fs.String("design", "", "설계 디렉토리 (기본 <root>/design)")
	root = fs.String("root", ".", "코드 루트")
	return design, root
}

// parse는 플래그를 읽고, 플래그가 아닌 인자가 남으면 거부한다.
// flag는 첫 위치 인자에서 멈추므로 남은 인자를 두면 그 뒤의 플래그까지 조용히 버려진다.
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

// designDir는 --design이 비어 있으면 코드 루트 아래 design이다.
func designDir(design, root string) string {
	if design == "" {
		return filepath.Join(root, "design")
	}
	return design
}
