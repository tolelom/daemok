// daemok은 설계 파일을 읽어 검사하고 코드를 생성하는 CLI다.
package main

import (
	"fmt"
	"io"
	"os"
)

const usage = `사용법:
  daemok check [--design DIR] [--root DIR] [--strict] [--format text|json]
  daemok gen   [--design DIR] [--root DIR] [--dry-run]
  daemok coverage [--design DIR] [--root DIR]
  daemok diff     [--base REF] [--design DIR] [--root DIR]
  daemok rename <종류> <옛 ID> <새 ID> [--code] [--design DIR] [--root DIR]
  daemok mcp      [--design DIR] [--root DIR]
  daemok version
  daemok init  [--root DIR] [--design DIR] [--dry-run]
  daemok serve [--design DIR] [--root DIR] [--addr HOST:PORT] [--workspace DIR]
`

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stderr, usage)
		return 2
	}
	switch args[0] {
	case "check":
		return runCheck(args[1:], stdout, stderr)
	case "gen":
		return runGen(args[1:], stdout, stderr)
	case "mcp":
		return runMCP(args[1:], os.Stdin, stdout, stderr)
	case "version":
		return runVersion(stdout)
	case "coverage":
		return runCoverage(args[1:], stdout, stderr)
	case "init":
		return runInit(args[1:], stdout, stderr)
	case "serve":
		return runServe(args[1:], stdout, stderr)
	case "diff":
		return runDiff(args[1:], stdout, stderr)
	case "rename":
		return runRename(args[1:], stdout, stderr)
	}
	fmt.Fprintf(stderr, "알 수 없는 명령 %q\n%s", args[0], usage)
	return 2
}
