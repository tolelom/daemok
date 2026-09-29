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
