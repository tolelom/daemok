package main

import (
	"flag"
	"path/filepath"
)

// pathFlags는 모든 명령이 같은 뜻으로 쓰는 --design과 --root를 등록한다.
func pathFlags(fs *flag.FlagSet) (design, root *string) {
	design = fs.String("design", "", "설계 디렉토리 (기본 <root>/design)")
	root = fs.String("root", ".", "코드 루트")
	return design, root
}

// designDir는 --design이 비어 있으면 코드 루트 아래 design이다.
func designDir(design, root string) string {
	if design == "" {
		return filepath.Join(root, "design")
	}
	return design
}
