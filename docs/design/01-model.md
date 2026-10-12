# 모델

설계 파일 형식 v0.5. [프로토타입](https://claude.ai/artifact/BeDedmG4Vn8i4M9He2sQmE)은 v0.2를 읽으며, 그 뒤에 더해진 `gen` 필드, 알 수 없는 키 오류, 관계 종류, 이벤트, 흐름은 반영하지 않는다.

## 예시

트랜잭션 하나가 처리되는 데 필요한 최소 조각이다. 최상위 키가 노드 종류이고, 그 안에 `ID: { 필드 }`가 나열된다.

```yaml
requirements:                      # R: 왜 만드나
  REQ-transfer: { text: "사용자는 토큰을 다른 주소로 보낼 수 있다" }

modules:                           # L2: 코드가 어느 패키지에 사는가
  core: { tier: domain }
  vm:   { tier: execution, depends_on: [core] }

enums:                             # L3: 값의 종류
  tx_type: { in: core, values: [transfer, mint_asset] }

handler_kinds:                     # L3: tx_type 값 하나당 핸들러 함수 하나라는 계약
  tx_handler: { in: vm, params: { payload: json.RawMessage }, returns: [error], key: tx_type }

groups:                            # L2: 같은 계약을 따르는 모듈 묶음
  tx_modules:
    kind: tx_handler
    depends_on: [core, vm]
    members:
      economy: { handles: [transfer] }
      asset:   { handles: [mint_asset] }

funcs:                             # L4: 함수 하나의 계약과 테스트 표
  handle_transfer:
    in: economy
    kind: tx_handler
    handles: transfer
    satisfies: [REQ-transfer]
    errors: [ErrInsufficientBalance]
    cases:
      - { name: "잔액 부족", given: { balance: 5, amount: 10 }, error: ErrInsufficientBalance }
```

읽는 규칙은 둘이다.

- 모든 노드는 자기 종류 안에서 ID가 하나다. 다른 곳에서는 그 ID를 이름으로만 가리킨다(`depends_on: [core]`, `key: tx_type`, `handles: transfer`).
- L3·L4 노드는 `in`으로 어느 모듈(또는 그룹 멤버)의 코드에 들어가는지 적는다. `tx_type`은 `core` 패키지에, `handle_transfer`는 `vm/modules/economy`에 생긴다.

이 파일에서 도구가 하는 일은 셋이다. `tx_type`의 값을 어느 멤버도 `handles`에 적지 않으면 에러다(검사). `core/tx_type.gen.go`에 상수 블록을, `vm/modules/economy/register.gen.go`에 `vm.Register(core.TxTransfer, handleTransfer)`를 만든다(생성). `handleTransfer`의 본문은 사람이 쓰고, 시그니처가 `tx_handler`와 다르면 컴파일이 깨진다(검증).

## 파일

- `design/` 아래 `*.yaml`을 모두 읽어 하나의 모델로 합친다. 파일을 어떻게 나눌지는 자유다.
- 최상위 키 `version`은 형식 버전이고 `0.5`가 아니면 에러다. `go_module`은 생성 코드의 import 경로 접두사, `ts_out`은 TypeScript 출력 파일 경로다. 여러 파일에 같은 최상위 스칼라 키가 다른 값으로 있으면 에러다.
- 같은 종류 안에서 ID가 두 번 나오면 두 위치를 함께 보여주고 에러를 낸다.
- 어느 노드에서든 형식에 없는 키는 위치와 함께 에러다. 조용히 무시하지 않는다.

## ID와 참조

- ID 이름공간은 종류별로 나뉜다. 인터페이스 `events`와 모듈 `events`, 모듈 `post`와 타입 `post`는 함께 쓸 수 있다.
- 참조 필드마다 받는 종류가 정해져 있다. 같은 필드가 받는 종류끼리는 ID가 겹칠 수 없다.

| 필드 | 받는 종류 |
|---|---|
| `depends_on`, `composes`, 예외의 `depends_on` | 모듈, 그룹, 그룹 멤버 |
| 타입 필드, 파라미터, 반환 | enum, 타입, 내장 타입, 외부 타입(`json.RawMessage` 등) |
| `ref` | 타입 |
| `errors`, 케이스의 `error` | 에러 |
| `satisfies` | 요구사항 |
| `in` | 모듈, 그룹 멤버 |
| `kind`(그룹, 함수) | 핸들러 종류 |
| `key` | enum |
| `max` | 상수 또는 숫자 |
| `needs`, `needs`의 `contract` | 계약 |
| `needs`의 `uses` | 그 계약의 메서드 |
| 이벤트의 `kind` | 이벤트 종류 |
| `publishers`, `subscribers` | 모듈, 그룹 멤버 |
| `on`, `emits` | 이벤트 |
| `calls` | `계약.메서드` |

## 노드 종류

| 층 | 최상위 키 | 주요 필드 |
|---|---|---|
| R | `requirements` | `text`, `status: active \| held \| dropped` |
| L0 | `system`(`actors`, `interfaces` 포함) | `protocol`, `port`, `path`, `used_by`, `spec` |
| L1 | `executables` | `exposes`, `composes` |
| L2 | `tiers`, `modules`, `groups` | `tier`, `depends_on`, `needs`, `implements`, `path`, 모듈의 `role`, 그룹의 `kind`(선택)·`rule`·`members`, 멤버의 `handles`·`exceptions`·`needs`. `path`가 없으면 ID가 경로다. `kind` 없는 그룹은 L2 제약 묶음이라 등록 파일과 blank import를 만들지 않고, 멤버에 `handles`를 적으면 에러다 |
| L3 | `enums`, `consts`, `errors`, `types`, `contracts`, `handler_kinds`, `event_kinds`, `events` | 아래 표 |
| L4 | `funcs`, `flows` | 함수는 `in`, `kind`+`handles` 또는 `of`, `errors`, `cases`. 흐름은 아래 절 |

설계 노드는 모두 `summary`와 `satisfies`를 가질 수 있다. L3·L4 노드는 모두 `in`으로 소속 모듈을 적는다. 이벤트는 예외이고 이벤트 종류의 `in`을 따른다.

L3·L4 노드에 `gen: false`를 적으면 그 노드는 생성에서 빠진다. 코드에 손으로 쓴 선언이 이미 있는 노드를 설계에 올릴 때 쓴다. 참조 해석과 설계 검사, 코드 대조는 `gen` 값과 무관하게 한다.

코드 대조는 `gen: false`인 enum, 상수, 에러, 타입, `call` 계약, 핸들러 종류의 선언이 소유 모듈 패키지에 있는지 본다. 기대하는 이름은 생성했을 때의 이름이고, enum은 타입과 값 상수다. 없으면 오류다. `call` 계약과 타입은 모양도 본다. 생성했다면 나왔을 메서드 시그니처와 필드 타입이 코드 interface·struct와 같아야 하고, 다르거나 설계 메서드·필드가 코드에 없으면 오류, 코드에만 있으면 경고다. 이름은 대소문자와 밑줄을 무시하고 맞추고, 파라미터 이름과 필드 태그는 보지 않는다. 타입식은 다른 패키지를 import 경로의 마지막 요소로 한정해 비교한다. 이벤트 종류, 흐름, 함수는 손으로 쓴 코드가 생성 코드와 같은 이름을 쓴다고 볼 수 없어 보지 않는다. 코드 디렉토리가 없는 모듈도 보지 않는다.

### L3

| 키 | 필드 | Go 생성물 |
|---|---|---|
| `enums` | `values`(문자열 목록), `go.type`, `go.prefix`, `export` | 문자열 타입, 상수 블록, `Valid()` |
| `consts` | `type`(int, string, duration), `value`, `go.name`, `export` | const 블록 |
| `errors` | `message`, `http`, `go.name` | sentinel `var` 블록, `http`가 있으면 `HTTPStatus(err)`. Go 이름은 `go.name`, 없으면 ID |
| `types` | `fields`: 타입 문자열 또는 `{type, ref, many, optional, max}` | struct |
| `contracts` | `kind`(call, query, port, external), `methods`: `{params, returns, errors}`, `go.name` | interface. 반환 끝에 `error`가 붙는다 |
| `handler_kinds` | `params`, `returns`, `key`, `go.name`, `go.register` | 함수 타입, 멤버별 `init()` 등록과 시그니처 확인 |
| `event_kinds` | `key`(enum), `encoding`(map, json), `subscribers` | 이벤트마다 payload struct, map이면 변환 함수 |
| `events` | `kind`, `publishers`, `subscribers`, `data`(`types.fields`와 같은 문법) | event_kind 파일에 들어간다 |

`export: [ts]`가 붙은 상수와 enum은 `ts_out`에 TypeScript 상수로도 생성한다.

### L4

- `funcs`는 핸들러(`kind` + `handles`)이거나 계약 메서드(`of: 계약.메서드`)다.
- `cases`는 `{name, given, expect}` 또는 `{name, given, error}` 행의 목록이다. `given`과 `expect`는 키가 문자열인 맵이고 값은 스칼라 또는 같은 형태의 맵이다. 여기서 테이블 테스트를 생성한다. `error: invalid`는 sentinel 없이 에러 여부만 확인한다.
- 전제·결과 조건을 산문으로 두지 않는다.
- 목록 필드가 비어 있는 것이 의도라면 `[]`로 적는다. 계약 메서드에 `errors`가 없으면 경고한다.

### 관계

관계는 따로 선언하지 않고 참조에서 나온다.

| 종류 | 나오는 곳 |
|---|---|
| 호출 | `depends_on`. 계약 없이 직접 import한다 |
| call, query, port, external | 주는 쪽 `contracts.kind`와 받는 쪽 `needs` |
| 이벤트 | `events`의 `publishers`에서 `subscribers`로 |
| 등록 | `groups`의 멤버에서 `handler_kinds`의 모듈로 |

- `needs`의 항목은 계약 ID이거나 `{contract, uses, why}`다. `uses`를 생략하면 계약의 메서드 전부다.
- `modules.role`은 `data`, `infra`, `adapter` 중 하나다. `port` 계약은 `infra` 모듈에, `external` 계약은 `adapter` 모듈에 둔다.
- `needs`는 import를 허락하지 않는다. 받는 쪽은 생성된 interface를 거쳐 부른다.

### 이벤트

`event_kinds`는 이벤트 묶음이다. `key` enum의 값 하나가 이벤트 하나이고, 이벤트 ID는 그 값이다. `event_kinds`의 `subscribers`는 묶음의 모든 이벤트를 받는다. 이벤트 관계는 import를 만들지 않으므로 tier 방향과 순환 검사에 들어가지 않는다.

### 흐름

`flows`는 단계의 순서, 되돌리기, 커밋 지점을 정한다. 단계의 본문은 손으로 쓴다.

```yaml
flows:
  produce_block:
    in: consensus
    steps:
      - { id: lock, undo: unlock }
      - { id: snapshot, undo: revert }
      - { id: add_block }
      - { id: commit_state, commit: true, fatal: true }
      - { id: unlock }
      - { id: emit_block_commit, emits: [block_commit] }
```

| 필드 | 뜻 |
|---|---|
| `in` | 흐름 함수가 생기는 모듈 또는 그룹 멤버 |
| `on` | 이 흐름을 시작하는 이벤트 |
| `steps` | 순서대로 부르는 단계. 항목은 `{id, undo, commit, fatal, calls, emits, summary}` |
| `undo` | 이 단계가 끝난 뒤 뒤 단계가 실패하면 부를 되돌리기 메서드 이름 |
| `commit` | 이 단계가 성공하면 쌓인 되돌리기를 비운다. 흐름에 하나까지 |
| `fatal` | 이 단계가 실패하면 되돌리지 않고 `Fatal`을 부른다 |

- 단계를 적힌 순서대로 부른다. 커밋 전 단계가 실패하면 쌓인 되돌리기를 역순으로 부르고 오류를 돌려준다. 되돌리기가 실패하면 `Fatal`을 부른다. 커밋 뒤 단계가 실패하면 되돌리지 않고 오류를 돌려준다.
- 단계 사이에 값을 넘기지 않는다. 반복과 조건은 단계 본문 안에 둔다.
- 단계 ID와 `undo`는 소문자 snake_case이고 메서드 이름은 CamelCase다.

## 생성 파일

모든 생성 파일의 첫 줄은 `// Code generated by daemok. DO NOT EDIT.`다. 생성 파일인지는 이 줄로 판정하고, 파일 이름 패턴으로 판정하지 않는다. 경로는 `go_module` 아래 모듈의 `path` 기준이다.

| 원본 | 파일 | 내용 |
|---|---|---|
| enum 하나 | `<모듈>/<enum id>.gen.go` | 타입, 상수 블록, `Valid()`, `<타입>Values()` |
| 모듈의 consts 전부 | `<모듈>/consts.gen.go` | const 블록 |
| 모듈의 errors 전부 | `<모듈>/errors.gen.go` | sentinel `var` 블록, `HTTPStatus` |
| 모듈의 types 전부 | `<모듈>/types.gen.go` | struct |
| 모듈의 `kind: call` 계약 전부 | `<모듈>/contracts.gen.go` | interface |
| handler_kind 하나 | `<모듈>/<handler_kind id>.gen.go` | 함수 타입 |
| `kind` 있는 그룹의 멤버 하나 | `<그룹 path>/<멤버>/register.gen.go` | `init()` 등록, `var _ Handler = f` |
| 실행 단위 하나 | `cmd/<실행 단위>/modules.gen.go` | `composes`에 적은 `kind` 있는 그룹의 멤버 blank import |
| `needs`가 있는 모듈·멤버 | `<path>/needs.gen.go` | 계약마다 `uses`의 메서드만 담은 interface. 멤버는 그룹의 needs를 포함 |
| 실행 단위 하나 | `cmd/<실행 단위>/contracts.gen.go` | 조립하는 받는 쪽·주는 쪽 쌍마다 구현 타입의 interface 만족 확인. query 구현은 `<주는 모듈>/query` |
| event_kind 하나 | `<모듈>/<event_kind id>.gen.go` | 이벤트마다 `<이벤트>Data` struct, `encoding: map`이면 `Map()`과 `<이벤트>DataFrom` |
| `encoding: map` event_kind를 가진 모듈 | `<모듈>/daemok_event.gen.go` | 변환 도우미 |
| 흐름 하나 | `<모듈>/<흐름 id>.gen.go` | `<흐름>Steps` interface, `Run<흐름>` |
| 흐름을 가진 모듈 | `<모듈>/daemok_flow.gen.go` | `FlowError` |
| cases가 있는 함수 하나 | `<모듈>/<함수 id>_gen_test.go` | 테이블 테스트 |
| cases가 있는 함수를 가진 모듈 | `<모듈>/daemok_case_gen_test.go` | `caseInput`, `checkErr`, `checkExpect` 도우미 |
| `export: [ts]`가 붙은 consts·enums 전부 | `ts_out` | TypeScript 상수와 타입. MVP에서는 생성하지 않는다 |

- 이름 변환: ID `handle_transfer`는 함수 `handleTransfer`, enum 값 `mint_asset`은 `go.prefix` + `MintAsset`. `Id`로 끝나는 조각은 `ID`로 바꾼다.
- 멤버의 `handles` 값에 대응하는 L4 함수 노드가 없으면 등록 코드는 `handle` + 값의 CamelCase(`handleMintAsset`)를 쓴다. 함수 노드가 있으면 그 ID를 쓴다.
- 테이블 테스트의 `given`과 `expect`는 `caseInput`(`map[string]any`)으로 나온다. 손으로 쓰는 `run<함수>Case(t, given) (caseInput, error)`가 상태를 만들고 호출하며, `checkExpect`는 `expect`의 키마다 `fmt.Sprint` 결과를 비교한다. 키 오타는 컴파일러가 잡지 못하고 테스트 실패로 드러난다.
- `gen: false`인 노드는 어느 파일에도 나오지 않는다. 모듈의 노드가 전부 `gen: false`면 그 모듈 파일은 생성하지 않는다.

## 설계 검사

| 대상 | 검사 |
|---|---|
| 참조 | 없는 대상, 받을 수 없는 종류 |
| L0 | 구현 모듈이 없는 인터페이스, 노출했지만 구현 모듈을 조립하지 않은 실행 단위 |
| L2 | 상위 tier 의존, 순환, 어느 실행 단위에도 없는 모듈, 그룹 규칙 예외 |
| L3 | 타입을 쓰는 모듈이 그 타입의 모듈에 의존하지 않음, 알 수 없는 타입, 에러 이름과 HTTP 상태 불일치 |
| 핸들러 | 키 enum의 모든 값이 정확히 한 멤버에서 처리되는지 |
| L4 | `handles`가 멤버의 `handles`에 있는지, 케이스의 에러가 `errors`에 있는지, 케이스가 없는 에러 |
| R | 충족 요소가 없는 active 요구사항, held·dropped 요구사항에 매달린 요소 |
| 관계 | `needs`의 관계도 tier 방향과 순환(형제 멤버 포함)에 넣음, 자기 계약을 받는 needs, 같은 계약을 두 번 받음(경고), 빈 `uses`, port·external 계약의 모듈 role, 받은 계약의 시그니처 타입에 대한 의존 |
| 이벤트 | key enum의 값마다 이벤트가 있는지, 한 enum을 여러 이벤트 종류가 쓰는지, 발행자가 있는지, 발행자·구독자가 모듈이나 그룹 멤버인지, 이벤트 종류의 모듈에 의존하는지 |
| 흐름 | 빈 흐름, 단계 ID와 undo 형식, 단계 중복, Go 메서드 이름 충돌, commit 하나, commit 단계와 커밋 뒤 단계의 undo, `calls`의 계약과 `uses`, `emits`의 발행자, `on`의 구독자 |
| 외부 타입 | `모듈.이름` 형식의 외부 타입은 그 모듈에 대한 의존으로 본다 |
| Go 이름 | 생성 코드가 있으면 `go_module` 필요, path 요소는 Go 식별자이고 `testdata`·`vendor`·`.`·`_`로 시작하지 않음, 노드 ID는 소문자 snake_case, enum 값은 Go 상수 이름이 되고 enum 안에서 겹치지 않음, 에러의 Go 이름(`go.name`, 없으면 ID)은 export되는 이름, 필드 이름과 이벤트의 `Map` 충돌, duration 형식, 같은 패키지의 생성 이름 충돌 |
| 코드 | 설계에 없는 의존의 import, needs로만 이어진 모듈의 직접 import, query 패키지의 data 아닌 모듈 import, 관리하지 않는 패키지(정보), 쓰지 않는 depends_on(정보), 발행자·구독자의 이벤트 상수 참조, 적히지 않은 모듈의 이벤트 상수 참조(경고), 쓸 수 있는 설계 에러가 있는 모듈의 함수 본문에서 부르는 즉석 `errors.New`(정보), 쓸 수 있는 enum의 값과 같은 문자열 리터럴을 비교(`==`, `!=`), switch case, `?`가 든 쿼리 문자열 인자 뒤의 인자에 쓴 곳(정보) |

## 어긋남 유형별 대응

| 유형 | 대응 |
|---|---|
| 같은 숫자가 여러 곳에 적힘 | `consts`에 한 번 정의하고, 필요하면 TS로도 내보냄 |
| 관계 차수가 문서마다 다름 | `types` 필드의 `ref`와 `many`로 한 번 정의 |
| 상태값 표기가 문서마다 다름 | `enums`에 한 번 정의 |
| 모듈 이름을 바꿨는데 참조가 옛 이름 | 없는 참조 에러 |
| 보류된 요구사항에 매달린 설계 요소 | `satisfies`와 `status`로 검사 |

## 예시

- [tolchain tx_modules](02-slice-tolchain.md)
- [tolelog 게시글](03-slice-tolelog.md)
- 전체 설계 파일: [prototype/daemok-v2.html](../../prototype/daemok-v2.html)의 `SAMPLES`
