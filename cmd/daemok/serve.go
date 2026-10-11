package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/tolelom/daemok/internal/serve"
)

// runServe는 설계 지도를 로컬 웹 페이지로 띄운다. 인터럽트를 받으면 0으로 끝난다.
func runServe(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	fs.SetOutput(stderr)
	design := fs.String("design", "design", "설계 디렉토리")
	root := fs.String("root", ".", "코드 루트")
	addr := fs.String("addr", "127.0.0.1:7412", "들을 주소")
	workspaceDir := fs.String("workspace", "", "프로젝트를 찾을 작업 폴더")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	s := serve.New(*design, *root)
	s.Poll()
	ln, err := net.Listen("tcp", *addr)
	if err != nil {
		fmt.Fprintf(stderr, "주소를 열 수 없다: %v\n", err)
		return 2
	}
	local := isLoopback(ln.Addr())
	cfg := ""
	if d, err := os.UserConfigDir(); err == nil {
		cfg = filepath.Join(d, "daemok")
	}
	s.SetOptions(serve.Options{Workspace: *workspaceDir, ConfigDir: cfg, Local: local})
	if !local {
		fmt.Fprintln(stderr, "루프백이 아닌 주소라 프로젝트 열기와 초안 저장을 끈다")
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go s.Watch(ctx, time.Second)
	srv := &http.Server{Handler: s.Handler()}
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		srv.Shutdown(shutdown)
	}()
	fmt.Fprintf(stdout, "http://%s\n", ln.Addr())
	if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
		fmt.Fprintf(stderr, "서버 오류: %v\n", err)
		return 2
	}
	return 0
}

func isLoopback(a net.Addr) bool {
	t, ok := a.(*net.TCPAddr)
	return ok && t.IP.IsLoopback()
}
