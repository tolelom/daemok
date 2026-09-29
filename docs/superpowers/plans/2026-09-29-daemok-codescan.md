# daemok codescan 구현 계획

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 코드의 import와 이벤트 상수 참조를 설계와 대조해 `daemok check`의 진단에 더한다.

**Architecture:** `internal/codescan.Scan(m, root) model.Diagnostics`가 설계 모듈·멤버 디렉토리와 `<모듈>/query` 디렉토리의 Go 파일(`_test.go` 제외)을 `go/parser`로 읽는다. `internal/workspace.Run`이 설계 검사를 통과한 모델에 대해 부른다. enum 상수 이름 규칙은 생성기와 같아야 하므로 `model`로 옮겨 둘이 함께 쓴다.

**Tech Stack:** `go/parser`, `go/ast`, `go/token`

**Spec:** `docs/superpowers/specs/2026-09-27-daemok-mvp-design.md`(import 대조), `docs/superpowers/specs/2026-09-29-daemok-format-v04-design.md`(관계·이벤트의 import·코드 검사)

## Global Constraints

- import 경로가 `go_module/<모듈 path>`와 정확히 같을 때만 그 모듈이다. 하위 패키지를 포함해 설계에 없는 `go_module` 아래 패키지는 관리하지 않는 패키지로 정보만 낸다.
- 진단 위치는 Go 파일의 줄·열이다. 메시지는 "설명: 대상" 형식.
- 모듈 디렉토리가 없으면 조용히 넘긴다(코드가 아직 없는 설계).

## Review Focus

- 그룹 멤버가 형제를 import: 예외가 있으면 통과, 없으면 오류. Task 2에서 테스트한다.
- `needs`로만 이어진 모듈을 직접 import: 일반 위반과 다른 메시지로 오류. Task 2에서 테스트한다.
- 이벤트 종류 모듈 안에서 상수를 패키지 이름 없이 쓰는 경우: 참조로 센다. Task 3에서 테스트한다.
- import 별칭(`ev "x/events"`)으로 상수를 쓰는 경우: 참조로 센다. Task 3에서 테스트한다.
- 문법 오류가 있는 Go 파일: 조용히 넘기지 않고 오류 진단. Task 2에서 테스트한다.

---

## Task 1: 이름 규칙을 model로

- [ ] `model`에 `(*Enum).GoType()`, `(*Enum).GoPrefix()`, `(*Enum).ConstName(value string)`, `(*Model).ImportPath(ownerID string) string`을 두고 생성기의 `enumName`·`enumPrefix`를 이것으로 바꾼다. 골든이 바뀌지 않는지 확인한다.
- [ ] 커밋 `refactor: enum Go 이름 규칙을 model로`

## Task 2: import 대조

**Files:** Create `internal/codescan/{scan.go,scan_test.go}`, `internal/codescan/testdata/<사례>/`

규칙:

| 경우 | 진단 |
|---|---|
| 모듈 A가 설계 모듈 B를 import하고 `CanDepend(A, B)`가 거짓, A가 B의 계약을 needs로 받음 | 오류 `needs로만 이어진 모듈을 직접 import한다: A → B` |
| 위와 같고 needs도 없음 | 오류 `설계에 없는 의존을 import한다: A → B` |
| `go_module` 아래지만 설계 모듈이 아닌 경로 | 정보 `관리하지 않는 패키지를 import한다: A → <경로>` |
| 모듈의 `depends_on` 대상(그룹이면 멤버 중 하나)을 import하지 않음 | 정보 `설계상 의존하지만 import하지 않는다: A → B` |
| `<모듈>/query` 패키지가 `role: data`가 아닌 설계 모듈을 import | 오류 `query 패키지가 data가 아닌 모듈을 import한다: <모듈>/query → B` |
| Go 파일을 읽을 수 없음 | 오류 `Go 파일을 읽을 수 없다: <오류>` |

- [ ] 테스트 사례: 허용, 위반, 멤버 예외, needs만 있는 import, 관리하지 않는 패키지, 쓰지 않는 의존, query 격리 위반, 문법 오류, `_test.go` 제외.
- [ ] 구현, 커밋 `feat: codescan: import 대조`

## Task 3: 이벤트 상수 참조

규칙:

| 경우 | 진단 |
|---|---|
| 발행자·구독자로 적힌 모듈의 코드가 이벤트 상수(`<prefix><값>`)를 쓰지 않음 | 오류 `이벤트 상수를 쓰지 않는다: 모듈 A, 이벤트 e (<상수>)` |
| 이벤트 종류의 `subscribers` 모듈이 `<타입>Values`를 쓰지 않음 | 오류 `이벤트 목록 함수를 쓰지 않는다: 모듈 A, 이벤트 종류 k (<타입>Values)` |
| 발행자·구독자로 적히지 않은 모듈이 이벤트 상수를 씀(이벤트 종류의 모듈 제외) | 경고 `발행자나 구독자로 적히지 않은 모듈이 이벤트를 쓴다: 모듈 A, 이벤트 e` |

- [ ] 참조는 이벤트 종류 모듈을 import한 이름(별칭 포함)의 선택자 `pkg.Const`, 또는 이벤트 종류 모듈 안의 식별자 `Const`다.
- [ ] 테스트, 구현, 커밋 `feat: codescan: 이벤트 상수 참조`

## Task 4: workspace와 문서

- [ ] `workspace.Run`에서 생성 파일 대조 뒤 `codescan.Scan`을 부른다. workspace 테스트에 import 위반 사례 하나.
- [ ] `docs/design/01-model.md` 설계 검사 표에 코드 대조 행. doc-lint.
- [ ] 커밋 `feat: check에 코드 대조`
