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
    values:
      transfer:      { payload: transfer_payload }
      mint_asset:    { payload: mint_asset_payload }
      grant_reward:  { payload: grant_reward_payload }
      # 18개 전부

errors:
  ErrInsufficientBalance: { in: core, message: "insufficient balance" }
  ErrUnauthorized:        { in: core, message: "unauthorized" }
  ErrBalanceOverflow:     { in: core, message: "balance overflow" }

consts:
  max_block_txs: { in: config, type: int, value: 500 }

types:
  transfer_payload:
    in: core
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
| 설계에 없는 에러 | 모듈 안의 `errors.New`는 경고. 코드를 읽어야 하므로 `daemok check`의 코드 대조에서 한다 |
| 같은 값의 상수가 두 곳에 정의 | `consensus.defaultMaxBlockTxs`는 설계에 없는 상수라 경고. 이것도 코드 대조에서 한다 |

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
		name  string
		given transferGiven
		err   error
	}{
		{"잔액 부족", transferGiven{balance: 5, amount: 10}, core.ErrInsufficientBalance},
		{"위임 한도 초과", transferGiven{delegatedMax: 5, amount: 10}, core.ErrUnauthorized},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) { runTransferCase(t, c.given, c.err) })
	}
}
```

- `handleTransfer`의 본문은 손으로 쓰는 파일(`token.go`)에 있다. 이름이나 시그니처가 설계와 다르면 `var _ vm.Handler = handleTransfer`에서 컴파일이 깨진다.
- 테스트 표는 설계의 `cases`에서 생성한다. `given`을 상태로 바꾸는 `runTransferCase`는 손으로 쓴다.
- `error: invalid`처럼 sentinel이 없는 경우는 에러 여부만 확인한다.

## 기존 코드에 도입할 때

- 생성물의 이름이 지금 코드의 이름과 같아야 한다. `go.prefix`로 `transfer`를 `TxTransfer`로 바꾼다.
- 도입 첫 단계는 `core/transaction.go`의 상수 블록과 각 모듈의 `init()`을 지우고 생성 파일로 바꾸는 것이다. 이 두 곳 외의 코드는 바뀌지 않는다.
