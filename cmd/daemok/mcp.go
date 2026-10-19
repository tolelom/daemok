package main

import (
	"flag"
	"fmt"
	"io"
	"runtime/debug"

	"github.com/tolelom/daemok/internal/mcp"
)

func runMCP(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("mcp", flag.ContinueOnError)
	fs.SetOutput(stderr)
	design, root := pathFlags(fs)
	allowWrite := fs.Bool("allow-write", false, "설계 파일을 고치는 edit 도구를 낸다")
	if !parse(fs, args) {
		return 2
	}
	s := mcp.New(designDir(*design, *root), *root)
	s.Version = versionString(debug.ReadBuildInfo())
	s.AllowWrite = *allowWrite
	if err := s.Serve(stdin, stdout); err != nil {
		fmt.Fprintf(stderr, "MCP 서버 오류: %v\n", err)
		return 2
	}
	return 0
}
