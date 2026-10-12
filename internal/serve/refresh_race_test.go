package serve

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// 옛 설계를 읽은 상태 만들기가 늦게 끝나도, 마지막에 나가는 상태는 디스크의 새 설계다.
// 상태 만들기가 겹치면 옛 상태가 더 큰 버전으로 나가고, 서명이 그대로라 Poll도 고치지 않는다.
func TestLateRefreshDoesNotOverwriteNewer(t *testing.T) {
	s, dir := setup(t, design)
	reached := make(chan struct{})
	release := make(chan struct{})
	first := true
	refreshBuilt = func() {
		if first {
			first = false
			close(reached)
			<-release
		}
	}
	defer func() { refreshBuilt = func() {} }()

	aDone := make(chan struct{})
	go func() { defer close(aDone); s.Refresh() }()
	<-reached

	newer := strings.Replace(design, "modules:\n", "modules:\n  extra: { tier: a }\n", 1)
	if err := os.WriteFile(filepath.Join(dir, "d.yaml"), []byte(newer), 0o644); err != nil {
		t.Fatal(err)
	}
	bDone := make(chan struct{})
	go func() { defer close(bDone); s.Refresh() }()
	// 겹칠 수 있으면 B가 먼저 끝난다. 잠금으로 막혀 있으면 기다리다 A를 풀어 준다.
	select {
	case <-bDone:
	case <-time.After(200 * time.Millisecond):
	}
	close(release)
	<-aDone
	<-bDone

	if modTier(t, s, "extra") != "a" {
		t.Fatal("늦게 끝난 옛 상태가 새 설계의 상태를 덮었다")
	}
}
