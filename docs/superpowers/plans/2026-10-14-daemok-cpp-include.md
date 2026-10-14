# C++ include 검사 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 설계에 `cpp.include_dirs`가 있으면 `daemok check`가 모듈 폴더의 C++ `#include`를 `depends_on`과 대조한다.

**Architecture:** 모델이 `cpp`를 `Model.CPP`로 읽고 설계 검사가 경로 규칙을 본다. `codescan`에 `cpp.go`(include 읽기·해석·판정)를 더하고 `Scan`이 `CPP`가 있을 때 부른다. 의존 판정은 Go import와 같은 함수를 쓴다.

**Tech Stack:** Go 표준 라이브러리.

**Spec:** `docs/superpowers/specs/2026-10-14-daemok-cpp-include-design.md`

## Global Constraints

- Go 표준 라이브러리만 쓴다.
- 오류 메시지: `설계에 없는 의존을 include한다: a → b`, `관리하지 않는 파일을 include한다: a → 경로`, `include 디렉토리가 없다: 경로`.
- 커밋 메시지는 `feat:`/`docs:` + 한국어 요약 + `Claude-Session` 줄. 항목마다 `go vet ./...`, `go test ./...` 통과 후 커밋.

## Review Focus

- Public과 Private에 같은 이름 헤더가 있음: `"x"`는 포함한 파일 폴더, 그다음 include_dirs 순서로 처음 찾은 것.
- 모듈 폴더 안 하위 폴더가 다른 모듈의 path: 가장 깊은 폴더의 모듈.
- 엔진 헤더(`"Components/Foo.h"`)가 프로젝트 폴더 이름과 겹치지만 파일이 없음: 외부로 넘김.
- 블록 주석 안의 `#include`: 세지 않음.
- 같은 관리 밖 파일을 여러 곳에서 include: 정보 한 번.

---

### Task 1: 모델과 설계 검사

**Files:** Modify `internal/model/model.go`, `internal/model/load.go`, `internal/model/decode_layers.go`, `internal/check/names.go`; Test `internal/model/load_test.go`, `internal/check/check_test.go`

**Interfaces:** Produces `type CPP struct { IncludeDirs []string; Pos yaml.Pos }`, `Model.CPP *CPP`.

- [ ] 실패하는 테스트: `cpp: { include_dirs: [a/Public, a/Private] }`를 읽음, 모르는 키 오류, 두 파일에 `cpp` 오류. check: 빈 목록, `/abs`, `../x` 오류
- [ ] 구현: `topKeys`에 `cpp`, `mg.cpp(root.Get("cpp"))`, 검사 `cppDirs`
- [ ] 커밋 `feat: 설계 형식에 cpp.include_dirs`

### Task 2: include 스캐너

**Files:** Create `internal/codescan/cpp.go`; Modify `internal/codescan/scan.go`; Test `internal/codescan/cpp_test.go`

**Interfaces:** Consumes `Model.CPP`. Produces `func includes(src []byte) []include` (`include{line, col int; path string; angle bool}`), `func (s *scanner) cppScan(us []unit)`. Go import 판정을 `func (s *scanner) depend(id, target string, pos yaml.Pos, verb string)`로 묶어 함께 쓴다.

- [ ] 실패하는 테스트: `includes`의 주석·문자열·공백·`<x>`. 임시 코드 루트에서 Scan이 허용 의존 무진단, 비허용 오류와 위치, Public·Private 한 모듈, 하위 폴더, 가장 깊은 모듈, 같은 폴더 우선, 외부 헤더 무시, 관리 밖 정보 한 번, 쓰지 않는 depends_on 정보, 없는 include 디렉토리 오류
- [ ] 구현
- [ ] 커밋 `feat: C++ include를 depends_on과 대조`

### Task 3: diff 설정, 문서, 자기 설계, 실사례

- [ ] `designdiff` settings에 `cpp`. 테스트 한 줄
- [ ] `docs/design/01-model.md` 최상위 키와 검사 표, `00-overview.md` 다른 언어 절, README. `tools/doc-lint.sh`
- [ ] `go run ./cmd/daemok check` 오류 0
- [ ] claw-machine-shop-simulator에 scratch 설계로 check, 결과 기록(커밋하지 않음)
- [ ] 커밋 `docs: C++ include 검사 반영`
