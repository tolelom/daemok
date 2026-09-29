# 설계 파일 형식 v0.4

> [모델](../../design/01-model.md)의 형식 v0.3에 관계 종류, 이벤트, 흐름을 더한다. 명령과 구조는 [MVP 설계](2026-09-27-daemok-mvp-design.md)를 따른다.

## 완료 조건

1. tolchain의 이벤트 20개와 블록 흐름 둘(`consensus/poa.go`의 블록 생성, `network/sync.go`의 블록 적용)을 v0.4 설계 파일로 쓴다. `daemok gen`이 만든 이벤트 payload 타입과 흐름 함수로 두 흐름의 순서·실패 처리 코드와 `events/emitter.go`의 상수 목록을 대체한 뒤 `go build ./...`와 `go test ./...`가 통과한다.
2. fiber_api_server의 설계 파일에 query 관계(feed → post)와 port 관계(post → cache, user·image → upload)를 쓰고, `daemok check`의 진단이 테스트에 적은 기대 목록과 같다. fiber_api_server 코드는 바꾸지 않는다.
3. external 계약은 daemok `testdata`의 샘플 설계로 확인한다. 생성 결과가 골든 파일과 같고, 스텁 본문과 함께 `go build`와 `go vet`을 통과한다.

## 바뀌는 것

| 대상 | v0.3 | v0.4 |
|---|---|---|
| `version` | `0.3` | `0.4`. 다른 값은 오류 |
| `contracts` | `in`, `methods` | `kind` 추가 |
| `modules` | `tier`, `depends_on`, `implements`, `path` | `role`, `needs` 추가 |
| 그룹, 그룹 멤버 | `depends_on`, 예외 | `needs` 추가 |
| L3 | `enums`, `consts`, `errors`, `types`, `contracts`, `handler_kinds` | `event_kinds`, `events` 추가 |
| L4 | `funcs` | `flows` 추가 |
| enum 생성물 | 타입, 상수 블록, `Valid()` | `<타입>Values()` 추가 |
| 모델 문서 | "실행 흐름과 시퀀스는 모델에 넣지 않는다" | 이 문장을 지우고 흐름 절을 둔다 |

v0.3 설계 파일은 `version`만 바꾸면 v0.4로 읽힌다. 새 키는 모두 생략할 수 있다.

## 관계

관계는 따로 선언하지 않고 참조에서 나온다.

| 종류 | 나오는 곳 |
|---|---|
| 호출 | `depends_on`. 계약 없이 직접 import한다 |
| call, query, port, external | 주는 쪽의 `contracts.kind`와 받는 쪽의 `needs` |
| 이벤트 | `events`의 `publishers`에서 `subscribers`로 |
| 등록 | `groups`의 멤버에서 `handler_kinds`의 모듈로 |

### 형식

```yaml
modules:
  post:  { tier: domain }
  cache: { tier: foundation, role: infra }
  feed:
    tier: domain
    needs:
      - { contract: post_reader, uses: [list_published], why: "피드용 공개 글 목록" }

contracts:
  post_reader:
    in: post
    kind: query
    methods:
      list_published: { params: { page: int }, returns: [post_page] }
```

- `contracts.kind`는 `call`, `query`, `port`, `external` 중 하나다. 생략하면 `call`이다.
- `modules.role`은 `data`, `infra`, `adapter` 중 하나다. 생략하면 도메인 모듈이다.
- `needs`의 항목은 계약 ID이거나 `{contract, uses, why}`다. `uses`를 생략하면 계약의 메서드 전부다.

### 생성

| 원본 | 파일 | 내용 |
|---|---|---|
| `needs`가 있는 모듈 | `<모듈>/needs.gen.go` | 계약마다 `uses`의 메서드만 담은 interface. 이름은 계약 ID의 CamelCase |
| 실행 단위 하나 | `cmd/<실행 단위>/contracts.gen.go` | 실행 단위가 조립하는 모듈 사이의 `needs`마다 `var _ <받는 모듈>.<계약> = (*<구현 패키지>.<계약>)(nil)` |

- 구현 타입은 손으로 쓴다. 이름은 계약 ID의 CamelCase다.
- `query` 계약의 구현은 `<주는 모듈 path>/query` 패키지에, 나머지 종류는 주는 모듈 패키지에 둔다.
- 생성자 호출과 배선 전체는 생성하지 않는다.

### 검사

| 대상 | 검사 |
|---|---|
| 참조 | `needs`의 계약, `uses`의 메서드가 있는지 |
| L2 | `needs`로 생긴 관계도 tier 방향과 순환 검사에 넣는다 |
| L2 | `port` 계약의 모듈은 `role: infra`, `external` 계약의 모듈은 `role: adapter`다 |
| L3 | 계약 시그니처의 타입이 속한 모듈에 받는 쪽이 의존한다. 기존 타입 의존 검사를 `needs`로 넓힌 것이다 |
| import | 받는 쪽이 `needs`로만 이어진 모듈을 직접 import하면 오류다 |
| import | `<모듈>/query` 패키지가 `role: data`가 아닌 설계 모듈을 import하면 오류다 |

## 이벤트

### 형식

```yaml
enums:
  event_type:
    in: events
    go: { type: EventType, prefix: Event }
    values: [block_commit, asset_minted, session_open]

event_kinds:
  chain_event: { in: events, key: event_type, encoding: map, subscribers: [rpc] }

events:
  asset_minted:
    kind: chain_event
    publishers: [asset, reward]
    subscribers: [indexer]
    data: { asset_id: string, template_id: string, owner: string }
  session_open:
    kind: chain_event
    publishers: [session]
    subscribers: [indexer]
    data: { session_id: string, game_id: string, players: { type: string, many: true } }
```

- `event_kinds`는 이벤트 묶음 하나다. `key` enum의 값 하나가 이벤트 하나다. `subscribers`는 묶음의 모든 이벤트를 받는 모듈이다.
- `encoding`은 `map` 또는 `json`이다. `map`은 봉투의 본문이 `map[string]any`, `json`은 `json.RawMessage`인 경우다.
- 이벤트 ID는 `key` enum의 값이다. `data`의 필드 문법은 `types.fields`와 같다.
- `publishers`와 `subscribers`는 모듈이나 그룹 멤버다. 발행자가 여럿일 수 있다.

### 생성

| 원본 | 파일 | 내용 |
|---|---|---|
| event_kind 하나 | `<모듈>/<event_kind id>.gen.go` | 이벤트마다 `<이벤트>Data` struct(필드마다 `json` 태그는 필드 ID). `encoding: map`이면 `Map() map[string]any`와 `<이벤트>DataFrom(map[string]any) (<이벤트>Data, error)` |

- `...DataFrom`은 키가 없거나 타입이 다르면 이벤트 ID와 키를 담은 오류를 돌려준다. JSON을 거친 값(`[]any`, `float64`)도 받는다.
- 봉투 타입(tolchain의 `events.Event`)과 발행·구독 함수는 생성하지 않는다. 발행하는 쪽은 `Data: d.Map()`, 받는 쪽은 `...DataFrom(ev.Data)`를 쓴다.

### 검사

| 대상 | 검사 |
|---|---|
| 이벤트 | `key` enum의 모든 값에 그 묶음의 이벤트가 정확히 하나 있다 |
| 이벤트 | 발행자와 구독자는 event_kind의 모듈에 의존한다 |
| 관계 | 이벤트 관계는 import를 만들지 않으므로 tier 방향과 순환 검사에 넣지 않는다 |
| 코드 | 발행자와 구독자로 적힌 모듈의 코드가 이벤트 상수(`<패키지>.<prefix><값>`)를 쓰지 않으면 오류다. 묶음의 `subscribers`는 `<타입>Values()`를 쓰면 된다 |
| 코드 | 이벤트 상수를 쓰는 모듈이 발행자나 구독자로 적혀 있지 않으면 경고다 |

코드 검사는 `go/parser`로 파일 전체를 읽어 선택자 표현식을 찾는다. `_test.go`는 제외한다.

## 흐름

흐름은 단계의 순서, 되돌리기, 커밋 지점을 정한다. 단계의 본문은 손으로 쓴다.

### 형식

```yaml
flows:
  produce_block:
    in: consensus
    steps:
      - { id: check_proposer }
      - { id: select_txs }
      - { id: lock, undo: unlock }
      - { id: snapshot, undo: revert }
      - { id: buffer_events, undo: discard_events }
      - { id: execute_txs }
      - { id: restore_emitter }
      - { id: compute_root }
      - { id: sign }
      - { id: validate }
      - { id: add_block }
      - { id: commit_state, commit: true, fatal: true }
      - { id: unlock }
      - { id: flush_events }
      - { id: emit_block_commit, emits: [block_commit] }
      - { id: remove_from_mempool }
```

| 필드 | 뜻 |
|---|---|
| `in` | 흐름 함수가 생기는 모듈 또는 그룹 멤버 |
| `on` | 이 흐름을 시작하는 이벤트. 흐름의 모듈은 그 이벤트의 구독자다 |
| `steps` | 순서대로 부르는 단계. 항목은 `{id, undo, commit, fatal, calls, emits, summary}` |
| `undo` | 이 단계가 끝난 뒤 뒤 단계가 실패하면 부를 되돌리기 메서드 이름. 단계 ID와 같아도 된다 |
| `commit` | 이 단계가 성공하면 쌓인 되돌리기를 비운다. 흐름에 하나까지 |
| `fatal` | 이 단계가 실패하면 되돌리지 않고 `Fatal`을 부른다 |
| `calls` | 단계가 부르는 계약 메서드(`계약.메서드`) |
| `emits` | 단계가 발행하는 이벤트 |

단계 사이에 값을 넘기지 않는다. 단계가 공유하는 상태는 단계 메서드를 가진 손으로 쓴 타입이 들고 있다. 반복과 조건은 단계 본문 안에 둔다.

### 실행 규칙

- 단계를 적힌 순서대로 부른다.
- `undo`가 있는 단계가 성공하면 그 되돌리기를 쌓는다.
- 커밋 전 단계가 실패하면 쌓인 되돌리기를 역순으로 부르고 오류를 돌려준다. 실패한 단계 자신의 되돌리기는 부르지 않는다.
- 되돌리기가 실패하면 `Fatal`을 부른다.
- `fatal` 단계가 실패하면 되돌리지 않고 `Fatal`을 부른다.
- 커밋 뒤 단계가 실패하면 되돌리지 않고 오류를 돌려준다.
- 돌려주는 오류는 `*FlowError{Flow, Step, Err}`이고 `Unwrap`이 원래 오류를 준다. 호출하는 쪽은 `Step`으로 실패 지점에 따라 다르게 처리한다.

### 생성

| 원본 | 파일 | 내용 |
|---|---|---|
| 흐름 하나 | `<모듈>/<흐름 id>.gen.go` | `<흐름>Steps` interface(단계와 되돌리기마다 `func() error`, 그리고 `Fatal(step string, err error)`), `Run<흐름>(s <흐름>Steps) error` |
| 흐름을 가진 모듈 | `<모듈>/daemok_flow.gen.go` | `FlowError` 타입 |

메서드 이름은 단계 ID의 CamelCase다.

### 검사

| 대상 | 검사 |
|---|---|
| 흐름 | 단계 ID가 흐름 안에서 하나다. 메서드 이름이 `Fatal`과 겹치지 않는다 |
| 흐름 | `commit` 단계는 하나까지다. 커밋 뒤 단계에 `undo`가 없다 |
| 참조 | `calls`의 계약·메서드, `emits`와 `on`의 이벤트가 있다 |
| 관계 | `calls`의 계약은 흐름의 모듈이 `needs`로 받거나 흐름의 모듈에 있다 |
| 관계 | `emits`의 이벤트는 흐름의 모듈을 발행자로, `on`의 이벤트는 흐름의 모듈을 구독자로 적는다 |

## tolchain 적용

블록 생성(`produce_block`, `consensus`)과 블록 적용(`apply_block`, `network`)은 따로 적은 흐름 둘이다. 같은 이름의 단계는 같은 뜻이고, 본문은 각 모듈의 타입이 구현한다.

지금 두 구현은 세 곳이 다르다. 도입할 때 설계 파일의 단계 순서와 `emits`로 정한다.

| 항목 | `consensus/poa.go` | `network/sync.go` |
|---|---|---|
| 잠금 해제와 이벤트 flush | 해제 뒤 flush | flush 뒤 해제 |
| `block_commit` 발행 | 한다 | 하지 않는다. `cmd/node/main.go:217`의 구독자가 블록 전파와 pruning을 함께 한다 |
| 실패 뒤 호출하는 쪽 | 오류를 돌려준다 | 스냅샷 실패는 `continue`, 나머지는 `return` |

`events/emitter.go`의 `EventType` 상수와 `AllEventTypes()`는 `event_type` enum 생성물로 바뀐다. 상수 이름은 값의 CamelCase라서 `EventTemplateReg`(값 `template_registered`)는 `EventTemplateRegistered`가 된다. 값 문자열은 그대로다.

## 구현 순서

MVP 설계의 구현 순서를 다음으로 바꾼다.

| 단계 | 결과 |
|---|---|
| 1~3 | 끝남(yaml, model, check) |
| 4 | model·check의 v0.4: 관계, 이벤트, 흐름의 읽기와 설계 검사. 샘플과 [모델](../../design/01-model.md)을 v0.4로 옮긴다 |
| 5 | `internal/gen/golang`, `daemok gen`, 생성 파일 최신 여부. v0.3 생성물과 v0.4 생성물(needs, contracts, event_kind, 흐름, enum `Values()`) |
| 6 | `internal/codescan`: import 대조와 이벤트 상수 참조 |
| 7 | `internal/gitbase`, `internal/serve`, 웹 화면 |
| 8 | tolchain 도입, fiber_api_server 설계 파일, 완료 조건 확인 |

## 범위 밖

- 흐름의 시간 목표(단계 사이 지연 기준)
- 흐름의 조기 종료(성공으로 끝나는 중간 단계)
- 단계 사이의 값 전달, 흐름 안의 반복과 조건
- 생성자 호출을 포함한 배선 전체 생성
- 이벤트 봉투 타입과 발행·구독 함수 생성
- TypeScript 이벤트 타입

## 위험

| 위험 | 대응 |
|---|---|
| 흐름 형식이 프로그래밍 언어처럼 커짐 | 선형 단계, 되돌리기, 커밋, fatal까지만 둔다. 값 전달과 분기는 받지 않는다 |
| 두 흐름이 같은 단계를 다른 순서로 둬도 도구가 모름 | 차이는 설계 파일에 드러나고 지도에서 나란히 본다. 공유 정의는 두 번째 사례가 생길 때 검토한다 |
| 이벤트 상수 참조 검사가 발행과 구독을 구분하지 못함 | 선언한 모듈이 상수를 쓰는지만 본다. 역할 구분은 설계가 맡는다 |
| query·port·external이 개인 프로젝트에서 충분히 쓰이지 않음 | fiber_api_server와 testdata 샘플로 검사와 생성을 확인한다 |
