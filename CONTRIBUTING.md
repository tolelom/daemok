# 기여

## 검사

CI(`.github/workflows/ci.yml`)와 같은 검사입니다.

```bash
go vet ./...
go test ./...
gofmt -l $(git ls-files '*.go' | grep -v /testdata/)   # 출력이 없어야 통과
```

`.github/workflows/vuln.yml`은 최신 stable Go로 govulncheck를 돌립니다. push·PR 때와 주 1회 돕니다.

```bash
go run golang.org/x/vuln/cmd/govulncheck@latest ./...
```

## 화면 e2e

headless 브라우저와 node가 필요해 CI에서는 돌지 않습니다. 브라우저는 `DAEMOK_BROWSER`, 없으면 흔한 설치 경로에서 찾습니다.

```bash
DAEMOK_E2E=1 go test ./internal/serve -run TestUI
```

## 코드 규칙

- 운영 코드(`_test.go`가 아닌 `.go`)에는 주석을 두지 않고, 설명은 `docs/code-notes.md`에 둡니다. 예외는 패키지 주석과 `//go:embed` 같은 지시문입니다.
- 패키지 주석은 `daemok init`이 모듈 summary로 읽으므로 코드에 둡니다.
