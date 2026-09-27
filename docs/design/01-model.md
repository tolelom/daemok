# 모델

> 검토 중인 초안. 승인 전까지 바뀔 수 있다.

## 노드와 참조

- 설계의 모든 요소는 ID가 붙은 노드다.
- ID 네임스페이스는 전역 하나다. 종류가 달라도 같은 ID를 쓸 수 없다.
- 참조 필드마다 받을 수 있는 노드 종류가 정해져 있다. `depends_on`은 모듈과 그룹만 받는다. 종류가 틀리거나 대상이 없으면 에러다.
- `design/` 아래 YAML 파일을 모두 읽어 하나의 모델로 합친다. 파일을 어떻게 나눌지는 자유다. 같은 ID가 두 번 나오면 두 위치를 함께 보여주고 에러를 낸다.
- 모든 설계 노드는 `summary`와 `satisfies: [요구사항 ID]`를 가질 수 있다.

## 노드 종류

| 층 | 종류 | 주요 필드 |
|---|---|---|
| R | `requirement` | `text`, `status: active \| held \| dropped` |
| L0 | `system`, `actor`, `interface` | `protocol`, `used_by`, `spec`(OpenAPI 파일 경로) |
| L1 | `executable` | `exposes`, `composes` |
| L2 | `tiers`, `module`, `group` | `tier`, `depends_on`, `implements`, `path`, 그룹의 `kind`·`rule`·`members`·`exceptions` |
| L3 | `type`, `enum`, `const`, `contract` | `in: <모듈>`, 필드 타입, `ref`와 `many` |
| L4 | `func` | `params`, `returns`, `errors`, `pre`, `post` |

`func`는 `contract`의 메서드이거나 모듈에 속한 독립 함수다.

## 어긋남 유형별 대응

| 유형 | 대응 |
|---|---|
| 같은 숫자가 여러 곳에 적힘 | `const`에 한 번 정의하고 참조 |
| 관계 차수가 문서마다 다름 | `type` 필드의 `ref`와 `many`로 한 번 정의 |
| 상태값 표기가 문서마다 다름 | `enum`에 한 번 정의 |
| 모듈 이름을 바꿨는데 참조가 옛 이름 | 없는 참조 에러 |
| 보류된 요구사항에 매달린 설계 요소 | `satisfies`와 `status`로 검사 |
| 요구사항이 없는 설계 요소 | `satisfies`가 빈 노드를 경고 |

## 예시

tolchain 구조를 빌린 형식 예시다. 값은 실제 tolchain과 다르다.

```yaml
requirement:
  REQ-reward-batch: { text: "보상은 블록마다 한 번에 모아 지급한다", status: active }

enum:
  tx_error: { in: core, values: [bad_signature, insufficient_balance, unknown_type] }

const:
  block_interval: { in: consensus, type: duration, value: 2s }

type:
  block: { in: core, fields: { height: uint64, txs: { ref: transaction, many: true } } }

contract:
  tx_module:
    in: vm
    methods:
      apply: { params: { st: state, tx: transaction }, errors: [tx_error] }

groups:
  tx_modules:
    members:
      reward: { summary: "배치 보상", satisfies: [REQ-reward-batch] }
```

실행 흐름과 시퀀스는 모델에 넣지 않는다. 함수의 `pre`·`post`와 요구사항으로 표현한다.
