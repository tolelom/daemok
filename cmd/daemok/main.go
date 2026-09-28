// daemok은 설계 파일을 읽어 검사하고 코드를 생성하는 CLI다.
package main

import (
	"fmt"
	"io"
	"os"
)

const usage = `사용법:
  daemok check [--design DIR] [--root DIR] [--strict]
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
	}
	fmt.Fprintf(stderr, "알 수 없는 명령 %q\n%s", args[0], usage)
	return 2
}
