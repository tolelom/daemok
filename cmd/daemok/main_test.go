package main

import (
	"bytes"
	"strings"
	"testing"
)

func runCLI(t *testing.T, args ...string) (code int, out, errOut string) {
	t.Helper()
	var o, e bytes.Buffer
	code = run(args, &o, &e)
	return code, o.String(), e.String()
}

func TestNoArgsPrintsUsage(t *testing.T) {
	code, _, errOut := runCLI(t)
	if code != 2 {
		t.Fatalf("exit %d, want 2", code)
	}
	if !strings.Contains(errOut, "daemok check") {
		t.Fatalf("usage missing: %q", errOut)
	}
}

func TestUnknownCommand(t *testing.T) {
	code, _, errOut := runCLI(t, "frobnicate")
	if code != 2 || !strings.Contains(errOut, "frobnicate") {
		t.Fatalf("exit %d, err %q", code, errOut)
	}
}
