# daemok serve 구현 계획

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `daemok serve`가 로컬 웹 지도를 띄우고, 설계 파일을 저장하면 지도가 다시 그려진다. 지도는 git HEAD 기준 대비 변경을 표시한다.

**Architecture:** `internal/gitbase`가 HEAD의 설계 파일을 읽고 `model.LoadSources`가 모델로 만든다. `internal/serve`가 workspace 결과와 기준 모델을 화면용 JSON으로 바꿔 `/api/state`로 주고, 1초 간격 폴링으로 설계 디렉토리 변경을 감지해 `/api/events`(SSE)로 알린다. 화면은 프로토타입의 렌더링 코드를 옮긴 `embed` 페이지이고 JSON을 그리기만 한다.

**Tech Stack:** `net/http`, `embed`, `os/exec`(git)

**Spec:** `docs/superpowers/specs/2026-09-27-daemok-mvp-design.md`(기준과 승인, 구조, 테스트), `docs/superpowers/specs/2026-09-29-daemok-format-v04-design.md`

## Global Constraints

- 기본 주소 `127.0.0.1:7412`, `--addr`로 바꾼다. 종료 시 0.
- 기준은 git HEAD의 설계 파일이다. 파일 목록은 `git ls-tree`, 내용은 `git show HEAD:<path>`. git 저장소가 아니거나 커밋이 없으면 기준 없이 동작한다.
- 화면은 계산하지 않는다. 검사, 생성, 진단의 노드 대응은 서버가 한다. 기준 대비 변경 표시만 두 모델을 비교한다.
- JSON의 노드 목록은 배열로 보내 설계 순서를 지킨다.
- 외부 스크립트·폰트를 불러오지 않는다(로컬 도구).

## Review Focus

- 설계 파일에 문법 오류가 생긴 상태로 저장: 서버가 죽지 않고 진단만 보인다. Task 3에서 테스트한다.
- 설계 디렉토리에 파일이 추가·삭제됨: 폴링이 감지한다. Task 3에서 테스트한다.
- git 저장소 밖: 기준 없음으로 동작한다. Task 1에서 테스트한다.
- HEAD에 설계 디렉토리가 없음(새로 만든 설계): 모든 노드가 추가로 보인다. Task 1에서 테스트한다.
- 코드 파일 진단(Go 파일 위치): 모듈 노드에 붙는다. Task 2에서 테스트한다.

---

## Task 1: model.LoadSources와 gitbase

- [ ] `model.LoadSources(files map[string][]byte) (*Model, Diagnostics)`: 이름순으로 읽는다. `Load`는 파일을 읽어 이것을 부른다. 기존 테스트 통과.
- [ ] `gitbase.Files(designDir string) (files map[string][]byte, ok bool, err error)`: `git -C <dir> rev-parse --is-inside-work-tree`가 실패하면 ok=false. `git -C <dir> ls-tree --full-name --name-only HEAD .`로 목록, `*.yaml`만 `git -C <dir> show HEAD:<path>`. HEAD가 없으면 ok=false. 키는 `HEAD:<full path>`.
- [ ] 테스트: 임시 저장소에서 커밋 전(ok=false), 커밋 뒤 파일, 작업 트리에서 지운 파일도 HEAD에서는 나옴, 저장소 밖(ok=false). `git`이 없으면 Skip.
- [ ] 커밋 `feat: HEAD 설계 파일 읽기`

## Task 2: 화면용 JSON

**Files:** `internal/serve/view.go`, `view_test.go`

- [ ] `workspace.Result`에 `Files []golang.File`(생성 파일 전부)을 더한다.
- [ ] 뷰 모델: 프로토타입 `normalize()`와 같은 필드 이름. 종류별로 배열(`mods`, `groups`, `members`, `enums`, `consts`, `errors`, `types`, `contracts`, `hks`, `funcs`, `reqs`, `actors`, `ifaces`, `exes`)과 v0.4 `eks`, `events`, `flows`. 모듈·그룹·멤버에 `needs: [{contract, target, uses, why}]`, 모듈에 `role`, 계약에 `kind`.
- [ ] 진단 → `issues: [{lv, msg, key, file, line, origin}]`. `lv`는 err·warn·info. `key`는 설계 파일 진단이면 같은 파일에서 위치가 진단 이전인 마지막 노드, Go 파일 진단이면 경로가 그 디렉토리인 모듈·멤버. `origin`은 design 또는 code.
- [ ] 생성 파일 → `files: [{path, lang, code, sources}]`. `sources`는 그 파일을 만든 노드 키(생성 규칙 표의 원본).
- [ ] 테스트: 샘플에서 배열 순서가 설계 순서, 진단의 key 대응 두 종류, sources.
- [ ] 커밋 `feat: serve 화면용 JSON`

## Task 3: 서버

**Files:** `internal/serve/server.go`, `server_test.go`, `internal/serve/web/index.html`, `cmd/daemok/serve.go`

- [ ] `serve.New(design, root)`, `Refresh()`, `Handler()`(`/`, `/api/state`, `/api/events`), `Watch(ctx, interval)`. 상태 JSON에 `version`(갱신마다 1 증가), `base`(기준 모델 또는 null).
- [ ] 폴링 서명: 설계 디렉토리의 `*.yaml` 이름·크기·수정 시각. 바뀌면 `Refresh` 후 구독자에게 알린다.
- [ ] 테스트(`httptest`): `/api/state`의 JSON 구조, 파일을 고치면 `poll()` 뒤 version 증가, 문법 오류 파일에서도 200과 진단, SSE가 변경 뒤 이벤트를 보낸다.
- [ ] `daemok serve [--design] [--root] [--addr]`, SIGINT에 종료하고 0.
- [ ] 커밋 `feat: daemok serve`

## Task 4: 웹 화면

- [ ] 프로토타입(`prototype/daemok-v2.html`)의 CSS, 헤더, 지도·인스펙터·진단·변경 카드, 렌더링 함수를 옮긴다. 편집기, 샘플, JS 검사·생성, js-yaml, 웹 폰트는 뺀다.
- [ ] 상태는 `/api/state`에서 받고 배열을 맵으로 바꾼다. `EventSource('/api/events')`를 받으면 다시 받고 현재 계층·선택을 유지한다.
- [ ] L3에 이벤트 종류·이벤트·흐름 절을, 인스펙터에 그 개요를 더한다. L2에 needs 관계를 점선으로 그린다.
- [ ] 변경 카드는 승인 버튼 없이 "승인은 설계 파일 커밋"을 안내한다.
- [ ] 확인: `daemok serve`를 tolchain 샘플로 띄워 브라우저에서 R~L3을 오가고, 설계 파일을 저장하면 다시 그려지는지, 콘솔 오류가 없는지 본다.
- [ ] 커밋 `feat: 웹 지도`
