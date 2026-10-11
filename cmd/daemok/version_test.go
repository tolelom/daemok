package main

import (
	"runtime/debug"
	"strings"
	"testing"
)

func TestVersionString(t *testing.T) {
	info := &debug.BuildInfo{Main: debug.Module{Version: "v0.5.0"}, Settings: []debug.BuildSetting{{Key: "vcs.revision", Value: "abcdef0123"}, {Key: "vcs.modified", Value: "true"}}}
	cases := []struct {
		info *debug.BuildInfo
		ok   bool
		want string
	}{
		{info, true, "daemok v0.5.0 abcdef0, 커밋 안 된 변경 있음"},
		{&debug.BuildInfo{Main: debug.Module{Version: "v0.5.0"}}, true, "daemok v0.5.0"},
		{&debug.BuildInfo{}, true, "daemok (devel)"},
		{nil, false, "daemok (빌드 정보 없음)"},
	}
	for _, c := range cases {
		if got := versionString(c.info, c.ok); got != c.want {
			t.Errorf("got %q, want %q", got, c.want)
		}
	}
}

func TestVersionCommand(t *testing.T) {
	code, out, _ := runCLI(t, "version")
	if code != 0 || !strings.HasPrefix(out, "daemok ") {
		t.Fatalf("%d %q", code, out)
	}
}
