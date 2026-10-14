# 설계 변경 요약 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `daemok diff [--base REF]`와 MCP `diff` 도구로 기준 ref 대비 설계 변경을 노드·키 단위 마크다운으로 출력한다.

**Architecture:** `gitbase.Files`가 ref를 받는다. 새 패키지 `internal/designdiff`(verify 층)가 두 설계 파일 묶음을 YAML 트리로 읽어 (종류, ID)로 노드를 모으고, 키 경로별로 비교해 `Report`를 만들고 마크다운으로 쓴다. CLI와 MCP는 `designdiff.Run(designDir, base)`를 부른다.

**Tech Stack:** Go 표준 라이브러리, `internal/yaml`, `internal/gitbase`.

**Spec:** `docs/superpowers/specs/2026-10-14-daemok-diff-design.md`

## Global Constraints

- Go 표준 라이브러리만 쓴다.
- 비교는 YAML 값으로 한다. 주석, 정렬, 흐름·블록 형식, 따옴표, 노드가 든 파일은 변경이 아니다.
- 종료 코드는 0 출력함, 2 진행할 수 없음.
- 커밋 메시지는 `feat:`/`fix:`/`docs:` + 한국어 요약 + `Claude-Session` 줄. 항목마다 `go vet ./...`, `go test ./...` 통과 후 커밋.

## Review Focus

- 같은 종류 키(`modules:`)가 여러 파일에 나뉜 설계: 노드를 모두 모은다.
- 흐름 steps 중간에 단계 하나를 끼워 넣음: 그 단계만 추가로 나오고 뒤 단계는 바뀌지 않는다.
- 멤버를 다른 그룹으로 옮김: 멤버 수정에 그룹 변경이 나온다.
- `daemok rename` 기본 모드 결과: 이름 바뀜 한 건으로 묶인다.
- 기준 ref에 설계 디렉토리가 없음: 모든 노드가 추가로 나온다.

---

### Task 1: gitbase.Files에 ref 인자

**Files:** Modify `internal/gitbase/gitbase.go`, `internal/serve/server.go`; Test `internal/gitbase/gitbase_test.go`

**Interfaces:** Produces `func Files(designDir, ref string) (map[string][]byte, bool, error)`. 키는 `"<ref>:<저장소 기준 경로>"`. git 저장소가 아니면 ok 거짓. ref가 없으면(저장소에 커밋이 없는 HEAD 포함) `HEAD`는 ok 거짓, 그 밖의 ref는 오류. 기준 ref에 설계 디렉토리가 없으면 빈 맵과 ok 참.

- [ ] 실패하는 테스트: 임시 저장소에 커밋 두 개(두 번째에서 설계 파일 수정), `Files(dir, "HEAD~1")`이 첫 내용, 없는 ref는 오류
- [ ] 구현: `rev-parse --verify --quiet <ref>^{commit}`, `ls-tree -z --full-name --name-only <ref> .`, `show <ref>:<p>`. serve는 `"HEAD"`를 넘긴다
- [ ] `go vet ./... && go test ./...`, 커밋 `feat: gitbase.Files가 기준 ref를 받음`

### Task 2: designdiff 노드 모으기와 값 비교

**Files:** Create `internal/designdiff/collect.go`, `internal/designdiff/compare.go`; Test `internal/designdiff/compare_test.go`

**Interfaces:**
- `func Compare(base, work map[string][]byte) (*Report, error)`: 파일마다 `yaml.Parse`, 실패하면 오류.
- `type Report struct { Settings []string; Added, Removed []NodeRef; Renamed []Rename; Modified []Modified }`
- `type NodeRef struct { Kind, ID string }`, `type Rename struct { Kind, From, To string }`, `type Modified struct { NodeRef; Lines []string }`
- 종류 순서 `kinds`: requirements, system, actors, interfaces, executables, modules, groups, members, enums, consts, errors, types, contracts, handler_kinds, event_kinds, events, funcs, flows.

규칙:
- 최상위 맵 키(`requirements` 등)의 항목을 노드로 모은다. `groups`는 `members` 키를 빼고 그룹 노드로, 멤버는 `members` 종류로 모으며 소속 그룹을 기억해 바뀌면 `그룹: a → b` 줄을 더한다. `system`은 `actors`, `interfaces`를 빼고 ID 없는 노드 하나로, 두 맵의 항목은 각 종류로 모은다.
- 설정: `version`, `go_module`, `ts_out`, `tiers`를 경로 이름으로 비교한다.
- 값 비교(`diffValue(path, a, b)`):
  - 스칼라는 `Value`로 비교해 `path: a → b`.
  - 한쪽만 있으면 `path 추가: v`, `path 삭제: v`. 값은 `render`(스칼라 그대로, 맵 `{ k: v }`, 목록 `[a, b]`).
  - 맵은 키 합집합(기준 순서 뒤 새 키)으로 재귀.
  - 목록 원소가 모두 스칼라면 다중집합 차이 `path: +x -y`, 같으면 순서만 비교해 `path: 순서 바뀜`.
  - 목록 원소가 모두 맵이고 짝 키(`steps`는 `id`, `cases`는 `name`)가 모든 원소에 있으면 짝지어 `path[키값]`으로 재귀, 짝 없는 원소는 추가·삭제, 공통 원소 순서가 다르면 `path: 순서 바뀜`.
  - 그 밖에 다르면 `path: 값 바뀜`.
  - 노드 최상위 경로는 키 이름만(`tier`), 하위는 점으로 잇는다(`fields.title`).
- 이름 바뀜: 같은 종류의 삭제·추가 노드에서 `path`와 `go.name`·`go.type`(남은 `go`가 비면 `go`도)을 뺀 값이 같은 짝이 서로 하나뿐이면 묶는다.

- [ ] 실패하는 테스트(표 형식): 파일 옮김·형식·주석·따옴표만 바뀜 → 빈 Report. 스칼라, 키 추가·삭제, 스칼라 목록 +/-, 순서 바뀜, steps 끼워 넣기, cases 이름 짝, 짝 키 없는 맵 목록, 멤버 그룹 이동, actors, 설정 tiers, 이름 바뀜(고정 키 더해진 것), 같은 내용 둘이면 묶지 않음
- [ ] 구현
- [ ] `go vet ./... && go test ./...`, 커밋 `feat: designdiff(두 설계를 노드·키 단위로 비교)`

### Task 3: 마크다운과 Run

**Files:** Create `internal/designdiff/markdown.go`, `internal/designdiff/run.go`; Test `internal/designdiff/markdown_test.go`, `internal/designdiff/run_test.go`

**Interfaces:**
- `func (r *Report) Markdown(base string) string`: 스펙의 출력 형식. 빈 Report는 `설계 변경 없음 (<base> 대비)\n`.
- `func Run(designDir, base string) (string, error)`: `gitbase.Files(designDir, base)`, 작업 폴더 `*.yaml`을 읽어 Compare·Markdown. 저장소가 아니면 오류.

- [ ] 실패하는 테스트: 스펙 출력 예시와 같은 글. 임시 git 저장소에서 `Run`이 HEAD 대비 수정을 보이고, 없는 ref·저장소 아님은 오류, 기준에 설계가 없으면 모두 추가
- [ ] 구현
- [ ] 커밋 `feat: designdiff 마크다운 출력과 Run`

### Task 4: CLI와 MCP

**Files:** Create `cmd/daemok/diff.go`, Modify `cmd/daemok/main.go`, `internal/mcp/server.go`, `internal/mcp/tools.go`; Test `cmd/daemok/diff_test.go`, `internal/mcp/server_test.go`

- [ ] 실패하는 테스트: `daemok diff --root R`이 0과 마크다운, 없는 ref는 2. MCP `diff` 도구가 목록에 `readOnlyHint`로 있고 결과가 같은 글
- [ ] 구현: 플래그 `--base`(기본 HEAD), `pathFlags`. MCP 인자 `base`
- [ ] 커밋 `feat: daemok diff 명령과 MCP diff 도구`

### Task 5: 문서와 자기 설계, 실사례

- [ ] `design/daemok.yaml`에 `designdiff: { tier: verify, path: internal/designdiff, depends_on: [gitbase, yaml] }`, mcp와 실행 단위에 반영. `go run ./cmd/daemok check` 오류 0
- [ ] README 명령 표, MCP 스펙 도구 표. `tools/doc-lint.sh`
- [ ] tolchain 클론에서 `daemok diff --base <설계를 바꾼 이전 커밋>`을 돌려 출력 확인
- [ ] 커밋 `docs: daemok diff 반영`
