# 슬라이스: tolchain tx_modules

> [모델](01-model.md)을 실제 코드에 적용한 예시. 코드 기준은 github.com/tolelom/tolchain `main`(2026-06-12)이다.

## 현재 코드

| 항목 | 위치 | 내용 |
|---|---|---|
| 핸들러 계약 | `vm/registry.go:12` | `type Handler func(ctx *Context, payload json.RawMessage) error` |
| 등록 | `vm/registry.go:51` | `vm.Register(typ core.TxType, h Handler)`. 모듈이 `init()`에서 호출 |
| 모듈 로드 | `cmd/node/main.go:27-34` | 모듈 8개를 blank import |
| 디스패치 | `vm/registry.go:36-44` | `map[core.TxType]Handler` 조회. 없으면 런타임 에러, 중복 등록은 panic |
| 트랜잭션 타입 | `core/transaction.go:13-33` | `type TxType string`, 상수 18개 |
| 페이로드 | `core/transaction.go:132-242` | 타입마다 struct 하나 |
| 에러 | `core/errors.go:8-26` | sentinel 7개. 모듈 안에는 `errors.New`로 만든 즉석 에러가 섞여 있다 |
| 형제 의존 | `vm/modules/reward/reward.go:85` | `asset.ValidateSchema` 호출. 형제 간 import는 이것 하나다 |

문서와 코드의 차이:

| 항목 | 문서 | 코드 |
|---|---|---|
| Go 버전 | `CODE_STRUCTURE.md` 1.22, `README.md` 1.25 | `go.mod` 1.24.0 |
| TxType 수 | `CODE_STRUCTURE.md:48` 9개 | 18개 |
| 블록당 tx 상한 500 | - | `config/config.go:19`와 `consensus/poa.go:229`에 따로 정의 |

## 설계 파일

```yaml
enums:
  tx_type:
    in: core
    go: { type: TxType, prefix: Tx }
    values: [transfer, mint_asset, grant_reward]   # 18개 전부

errors:
  ErrInsufficientBalance: { in: core, message: "insufficient balance" }
  ErrUnauthorized:        { in: core, message: "unauthorized" }
  ErrBalanceOverflow:     { in: core, message: "balance overflow" }

consts:
  max_block_txs: { in: config, type: int, value: 500 }

types:
  transfer_payload:
    in: core
    gen: false   # core/transaction.go의 struct를 그대로 둔다
    fields: { to: address, amount: uint64 }

handler_kinds:
  tx_handler:
    in: vm
    go: { name: Handler, register: Register }
    params: { ctx: "*Context", payload: json.RawMessage }
    returns: [error]
    key: tx_type

groups:
  tx_modules:
    kind: tx_handler
    path: vm/modules
    depends_on: [core, crypto, events, vm]
    rule: no_sibling_deps
    members:
      economy:
        handles: [transfer]
      reward:
        handles: [grant_reward]
        exceptions:
          - depends_on: asset
            reason: "보상 지급 시 자산 스키마 검증 재사용"

funcs:
  handle_transfer:
    in: economy
    kind: tx_handler
    handles: transfer
    errors: [ErrInsufficientBalance, ErrUnauthorized, ErrBalanceOverflow]
    cases:
      - { name: "금액 0",       given: { amount: 0 },                error: invalid }
      - { name: "잔액 부족",    given: { balance: 5, amount: 10 },   error: ErrInsufficientBalance }
      - { name: "위임 한도 초과", given: { delegated_max: 5, amount: 10 }, error: ErrUnauthorized }
      - { name: "정상 이체",    given: { balance: 10, amount: 3 },   expect: { from: 7, to: 3 } }
```

`handler_kinds`는 이번 슬라이스에서 새로 필요해진 노드다. 인터페이스가 아니라 함수 타입과 등록 키로 이뤄진 계약을 표현한다. `handles`는 enum 값 하나에 핸들러 하나를 대응시킨다.

## 검사

| 검사 | 결과 |
|---|---|
| `tx_type`의 모든 값이 정확히 한 멤버의 `handles`에 있는가 | 빠지거나 겹치면 생성 전에 에러. 지금은 누락이 런타임 에러, 중복이 런타임 panic이다 |
| 형제 모듈 import | `reward → asset`은 예외로 기록돼 정보로 표시. 다른 형제 import가 생기면 에러 |

모듈 안의 `errors.New`와 `consensus.defaultMaxBlockTxs` 같은 설계에 없는 상수는 코드 본문을 읽어야 잡힌다. MVP의 코드 대조는 import만 읽으므로 이 둘은 MVP 범위 밖이다.

## 생성물

```go
// core/tx_type.gen.go
type TxType string

const (
	TxTransfer    TxType = "transfer"
	TxMintAsset   TxType = "mint_asset"
	TxGrantReward TxType = "grant_reward"
)

// vm/modules/economy/register.gen.go
func init() {
	vm.Register(core.TxTransfer, handleTransfer)
}

var _ vm.Handler = handleTransfer

// vm/modules/economy/handle_transfer_gen_test.go
func TestHandleTransfer(t *testing.T) {
	cases := []struct {
		name   string
		given  caseInput
		expect caseInput
		err    error
	}{
		{name: "금액 0", given: caseInput{"amount": 0}, err: errAny},
		{name: "잔액 부족", given: caseInput{"balance": 5, "amount": 10}, err: core.ErrInsufficientBalance},
		{name: "위임 한도 초과", given: caseInput{"delegated_max": 5, "amount": 10}, err: core.ErrUnauthorized},
		{name: "정상 이체", given: caseInput{"balance": 10, "amount": 3}, expect: caseInput{"from": 7, "to": 3}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := runHandleTransferCase(t, c.given)
			checkErr(t, err, c.err)
			checkExpect(t, got, c.expect)
		})
	}
}
```

- `handleTransfer`의 본문은 손으로 쓰는 파일(`token.go`)에 있다. 이름이나 시그니처가 설계와 다르면 `var _ vm.Handler = handleTransfer`에서 컴파일이 깨진다.
- 테스트 표는 설계의 `cases`에서 생성한다. `given`을 상태로 바꾸고 호출하는 `runHandleTransferCase`는 손으로 쓴다. `caseInput`, `errAny`, `checkErr`, `checkExpect`는 같은 디렉토리의 `daemok_case_gen_test.go`에 생성된다.
- `error: invalid`는 `errAny`로 나와 에러 여부만 확인한다.
- `transfer_payload`는 `gen: false`라 `core/types.gen.go`에 나오지 않는다. 다른 tx 모듈의 멤버는 L4 함수 노드가 없으므로 등록 코드는 `handle` + 값의 CamelCase(`handleMintAsset`)를 쓴다.

## 기존 코드에 도입할 때

- 생성물의 이름이 지금 코드의 이름과 같아야 한다. `go.prefix`로 `transfer`를 `TxTransfer`로 바꾼다.
- 도입 첫 단계는 `core/transaction.go`의 상수 블록과 각 모듈의 `init()`을 지우고 생성 파일로 바꾸는 것이다. 이 두 곳 외의 코드는 바뀌지 않는다. 그러려면 첫 단계의 설계 파일에서 생성하는 노드는 enum `tx_type`과 멤버의 등록뿐이고, 기존 선언이 있는 나머지 L3 노드는 `gen: false`로 둔다.

| 노드 | 기존 선언 |
|---|---|
| `consts` | `config/config.go:19` `DefaultMaxBlockTxs` |
| `errors` | `core/errors.go:8-26` |
| `types` | `core/transaction.go:132-242` 페이로드 struct |
| `handler_kinds.tx_handler` | `vm/registry.go:12` `type Handler` |

- 핸들러 18개의 이름은 모두 `handle<Value>` 규약과 같아서(`handleMintAsset`, `handleRegisterTemplate`) L4 함수 노드 없이 등록 코드가 생성된다.
- `gen: false`를 하나씩 풀어 생성 파일로 바꾸는 것이 그다음 단계다.

## 도입 결과

tolchain `feat/daemok` 브랜치(기준 `main` `29c1d3a`)에 형식 v0.4 설계 파일 `design/tolchain.yaml`을 두고 도입했다.

| 대상 | 전 | 후 |
|---|---|---|
| 트랜잭션 타입 | `core/transaction.go`의 `TxType`과 상수 18개 | `core/tx_type.gen.go`(`Valid()`, `TxTypeValues()` 포함) |
| 핸들러 등록 | 모듈 9개 파일의 `init()` | 멤버마다 `register.gen.go`, 실행 단위의 blank import는 `cmd/node/modules.gen.go` |
| 이벤트 타입 | `events/emitter.go`의 상수 20개와 손으로 맞추는 `AllEventTypes()` | `events/event_type.gen.go`의 상수와 `EventTypeValues()`. `EventTemplateReg`는 `EventTemplateRegistered`로 바뀌고 값 문자열은 그대로다 |
| 이벤트 본문 | 발행 22곳의 `map[string]any` 리터럴 | `events.<이벤트>Data{...}.Map()`. 키 이름과 값 타입을 컴파일러가 확인한다 |
| 블록 생성 | `consensus/poa.go`의 `ProduceBlock` 본문 | 흐름 `produce_block`(단계 18개). 단계 본문은 `consensus/produce_block.go` |
| 블록 적용 | `network/sync.go`의 `handleBlocks`와 `handleBlock`에 같은 순서가 두 번 | 흐름 `apply_block`(단계 13개) 하나를 둘이 부른다. 단계 본문은 `network/apply_block.go` |

- `daemok check`: 오류 0, 경고 2(보류 요구사항 `REQ-random`에 연결된 요소, 케이스가 없는 에러 `ErrBalanceOverflow`), 정보 1(그룹 예외 `reward → asset`). import 대조와 이벤트 상수 참조에서 어긋난 곳이 없다.
- 형식 0.5로 올린 뒤(`a8a41c6`, 첫 줄만 바꿈)에도 `daemok check` 결과는 오류 0, 경고 2, 정보 33으로 같다. `gen: false` 타입 `TransferPayload`의 필드 대조도 통과한다.
- `go test ./...`는 `core`의 `TestHash_OnBehalfOf_BackwardCompatible` 하나만 실패한다. 이 실패는 도입 전 `29c1d3a`에서도 같다.
- indexer는 이벤트 본문을 지금처럼 `ev.Data["키"]`로 읽는다. indexer 테스트가 키 일부가 빠진 이벤트를 건너뛰거나 처리하는 동작을 확인하고 있어, 키가 하나라도 없으면 오류를 내는 `<이벤트>DataFrom`으로 바꾸지 않았다.

### 흐름으로 옮긴 뒤의 차이

단계 순서와 되돌리기는 옮기기 전과 같다. `network/apply_block_test.go`가 실패 경로 다섯 가지의 호출 순서를 옮기기 전 코드와 대조한다.

| 항목 | 전 | 후 |
|---|---|---|
| 되돌리기 중 emitter | 이미 복원된 뒤에는 다시 설정하지 않음 | `discard_events`가 같은 emitter를 한 번 더 설정 |
| Fatal 로그 | 실패 지점마다 다른 문구(`revert failed after exec error` 등) | `block production cannot recover`·`block apply cannot recover`와 단계 이름, 원인 오류 |
| `ProduceBlock`의 오류 | 단계별로 감싼 오류 | 같다. `FlowError`를 벗겨 단계의 오류를 돌려준다 |

두 흐름 사이의 차이는 설계 파일에 그대로 드러난다.

| 항목 | `produce_block` | `apply_block` |
|---|---|---|
| 잠금 해제와 이벤트 flush | `unlock` 뒤 `flush_events` | `flush_events` 뒤 `unlock` |
| `block_commit` 발행 | `emit_block_commit` 단계 | 없음 |
| 실패 뒤 호출하는 쪽 | 오류를 돌려준다 | `handleBlocks`는 스냅샷 실패면 다음 블록으로, 그 밖은 멈춘다 |

### 설계 노드 대응 비율

패키지 최상위 선언(type, const, var, 메서드가 아닌 func. `_test.go`, `init`, `main` 제외) 중 설계 노드에 대응하는 것의 수다. 대응은 생성 파일이 선언한 이름, `gen: false` 노드의 Go 이름(`DefaultMaxBlockTxs`, 에러 7개, `TransferPayload`, `Handler`), 그룹 멤버의 핸들러 함수다. 기준은 `feat/daemok`의 `a97cbb0`이다.

| 패키지 | 최상위 선언 | 설계 대응 | 비율 |
|---|---|---|---|
| `cmd/backup` | 3 | 0 | 0% |
| `cmd/node` | 2 | 0 | 0% |
| `config` | 22 | 1 | 4% |
| `consensus` | 8 | 4 | 50% |
| `core` | 67 | 28 | 41% |
| `crypto` | 9 | 0 | 0% |
| `crypto/certgen` | 7 | 0 | 0% |
| `events` | 75 | 68 | 90% |
| `indexer` | 11 | 0 | 0% |
| `internal/testutil` | 9 | 0 | 0% |
| `metrics` | 17 | 0 | 0% |
| `network` | 28 | 4 | 14% |
| `rpc` | 31 | 0 | 0% |
| `storage` | 25 | 0 | 0% |
| `vm` | 12 | 1 | 8% |
| `vm/modules/*`(8개) | 19 | 18 | 95% |
| `wallet` | 10 | 0 | 0% |
| 합계 | 355 | 124 | 34% |

설계가 거의 덮는 곳은 이벤트와 tx 모듈이고, 덮지 않는 곳은 저장소(`storage`), 네트워크 프로토콜(`network`의 메시지·피어), RPC, 지갑, 설정이다.

## 코드에서 만든 초안과의 비교

`main` `29c1d3a`의 복사본에서 `daemok init`이 만든 초안과 이 문서의 설계 파일(`internal/model/testdata/tolchain/design/tolchain.yaml`)을 모듈 단위로 비교한다. 초안의 `daemok check`는 오류 0, 경고 1(어느 실행 단위에도 없는 모듈 `testutil`)이다.

| 항목 | 세는 규칙 | 수 | 모듈 |
|---|---|---|---|
| tier가 다른 모듈 | 초안의 층 번호와 손으로 쓴 설계의 tier 순번(foundation 0부터 service 4까지)이 다른 모듈. 그룹 멤버 제외 | 6 | `config`, `network`, `wallet`(service, 초안 2층), `vm`(execution, 초안 2층), `indexer`(persistence, 초안 3층), `consensus`(service, 초안 3층) |
| 그룹으로 묶을 패키지 | 손으로 쓴 설계에서 그룹 멤버인 패키지 | 8 | `vm/modules/*` 8개 |
| ID가 다른 모듈 | 같은 경로인데 ID가 다른 모듈 | 0 | |
| 초안에만 있는 모듈 | 손으로 쓴 설계에 없는 패키지 | 1 | `internal/testutil` |

- 모듈의 `depends_on`은 13개 모두 손으로 쓴 설계와 같다.
- 초안의 층은 의존 깊이이고, 손으로 쓴 tier는 역할(persistence, execution, service)이다. 차이 6개가 모두 이 기준 차이에서 나온다.
