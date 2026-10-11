package main

import (
	"flag"
	"fmt"
	"io"
	"runtime/debug"

	"github.com/tolelom/daemok/internal/mcp"
)

// runMCP는 표준 입출력으로 MCP 서버를 돌린다. 입력이 끝나면 0이다.
func runMCP(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("mcp", flag.ContinueOnError)
	fs.SetOutput(stderr)
	design := fs.String("design", "design", "설계 디렉토리")
	root := fs.String("root", ".", "코드 루트")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	s := mcp.New(*design, *root)
	s.Version = versionString(debug.ReadBuildInfo())
	if err := s.Serve(stdin, stdout); err != nil {
		fmt.Fprintf(stderr, "MCP 서버 오류: %v\n", err)
		return 2
	}
	return 0
}
