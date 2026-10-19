package main

import (
	"fmt"
	"io"
	"runtime/debug"
)

func runVersion(stdout io.Writer) int {
	fmt.Fprintln(stdout, versionString(debug.ReadBuildInfo()))
	return 0
}

func versionString(info *debug.BuildInfo, ok bool) string {
	if !ok {
		return "daemok (빌드 정보 없음)"
	}
	v := info.Main.Version
	if v == "" {
		v = "(devel)"
	}
	var rev string
	modified := false
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			rev = s.Value
		case "vcs.modified":
			modified = s.Value == "true"
		}
	}
	if len(rev) > 7 {
		rev = rev[:7]
	}
	switch {
	case rev != "" && modified:
		return fmt.Sprintf("daemok %s %s, 커밋 안 된 변경 있음", v, rev)
	case rev != "":
		return fmt.Sprintf("daemok %s %s", v, rev)
	}
	return "daemok " + v
}
