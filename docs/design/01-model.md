# 모델

설계 파일 형식 v0.2. [프로토타입](https://claude.ai/artifact/BeDedmG4Vn8i4M9He2sQmE)이 이 형식을 읽는다.

## 파일

- `design/` 아래 `*.yaml`을 모두 읽어 하나의 모델로 합친다. 파일을 어떻게 나눌지는 자유다.
- 최상위 키 `go_module`은 생성 코드의 import 경로 접두사, `ts_out`은 TypeScript 출력 파일 경로다.
- 같은 종류 안에서 ID가 두 번 나오면 두 위치를 함께 보여주고 에러를 낸다.

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

## 노드 종류

| 층 | 최상위 키 | 주요 필드 |
|---|---|---|
| R | `requirements` | `text`, `status: active \| held \| dropped` |
| L0 | `system`(`actors`, `interfaces` 포함) | `protocol`, `port`, `path`, `used_by`, `spec` |
| L1 | `executables` | `exposes`, `composes` |
| L2 | `tiers`, `modules`, `groups` | `tier`, `depends_on`, `implements`, `path`, 그룹의 `kind`·`rule`·`members`, 멤버의 `handles`·`exceptions` |
| L3 | `enums`, `consts`, `errors`, `types`, `contracts`, `handler_kinds` | 아래 표 |
| L4 | `funcs` | `in`, `kind`+`handles` 또는 `of`, `errors`, `cases` |

설계 노드는 모두 `summary`와 `satisfies`를 가질 수 있다. L3·L4 노드는 모두 `in`으로 소속 모듈을 적는다.

### L3

| 키 | 필드 | Go 생성물 |
|---|---|---|
| `enums` | `values`, `go.type`, `go.prefix`, `export` | 문자열 타입, 상수 블록, `Valid()` |
| `consts` | `type`(int, string, duration), `value`, `go.name`, `export` | const 블록 |
| `errors` | `message`, `http` | sentinel `var` 블록, `http`가 있으면 `HTTPStatus(err)` |
| `types` | `fields`: 타입 문자열 또는 `{type, ref, many, optional, max}` | struct |
| `contracts` | `methods`: `{params, returns, errors}`, `go.name` | interface. 반환 끝에 `error`가 붙는다 |
| `handler_kinds` | `params`, `returns`, `key`, `go.name`, `go.register` | 함수 타입, 멤버별 `init()` 등록과 시그니처 확인 |

`export: [ts]`가 붙은 상수와 enum은 `ts_out`에 TypeScript 상수로도 생성한다.

### L4

- `funcs`는 핸들러(`kind` + `handles`)이거나 계약 메서드(`of: 계약.메서드`)다.
- `cases`는 `{name, given, expect}` 또는 `{name, given, error}` 행의 목록이다. 여기서 테이블 테스트를 생성한다. `error: invalid`는 sentinel 없이 에러 여부만 확인한다.
- 전제·결과 조건을 산문으로 두지 않는다.
- 목록 필드가 비어 있는 것이 의도라면 `[]`로 적는다. 계약 메서드에 `errors`가 없으면 경고한다.

실행 흐름과 시퀀스는 모델에 넣지 않는다. `cases`와 요구사항으로 표현한다.

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
