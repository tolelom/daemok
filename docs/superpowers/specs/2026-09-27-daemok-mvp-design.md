# daemok MVP 설계

> 문제, 원칙, 계층은 [개요](../../design/00-overview.md)에, 설계 파일 형식과 설계 검사는 [모델](../../design/01-model.md)에 있다. 이 문서는 MVP의 범위, 명령, 구조를 정한다.

## 완료 조건

tolchain 저장소에 다음이 된다.

1. tolchain 설계 파일(L0~L3, tx_modules의 L4 일부)에 대해 `daemok check`가 설계 검사와 import 대조를 돌리고 결과를 줄 번호와 함께 출력한다.
2. `daemok gen`이 만든 파일로 `core/transaction.go`의 TxType 상수 블록과 각 tx 모듈의 `init()` 등록을 대체한 뒤 `go build ./...`와 `go test ./...`가 통과한다.
3. `daemok serve`로 연 브라우저 지도에서 R~L3을 오가고, 설계 파일을 저장하면 지도가 다시 그려진다.
4. 도입 뒤 tolchain의 패키지 최상위 선언(type, const, var, func) 중 설계 노드에 대응하는 것(생성됐거나 `gen: false`로 적힌 것, 핸들러 함수)의 비율을 패키지별로 [슬라이스 문서](../../design/02-slice-tolchain.md)에 기록한다. 이 비율이 형식을 넓힐지 정하는 근거다.

## 범위

| 포함 | 제외 |
|---|---|
| 설계 파일 읽기(YAML 부분집합), 여러 파일 병합 | MCP 서버 |
| 설계 검사 전체([모델](../../design/01-model.md#설계-검사)) | 그림에서 편집 |
| Go import 대조, 생성 파일 최신 여부 | 라우트, OpenAPI 생성 |
| Go 생성, TS 상수·enum 생성 | 코드에서 설계 추출 |
| 로컬 웹 지도, git HEAD 대비 변경 표시 | TS 상수·enum 외의 다른 언어 생성, Go 외 언어의 import 대조 |
| | 즉석 `errors.New`, 중복 상수 같은 코드 본문 검사 |

## 명령

설계 디렉토리는 기본 `design/`, 코드 루트는 기본 현재 디렉토리다. `--design`, `--root`로 바꾼다.

| 명령 | 동작 | 종료 코드 |
|---|---|---|
| `daemok check` | 설계 검사, import 대조, 생성 파일 최신 여부 | 0 통과, 1 오류 있음, 2 진행 불가 |
| `daemok check --strict` | 경고도 실패로 본다 | 위와 같음 |
| `daemok gen` | 생성 파일을 쓰고, 이전에 만들었지만 이제 나오지 않는 생성 파일을 지운다 | 0 성공, 1 쓰기 거부, 2 진행 불가 |
| `daemok gen --dry-run` | 쓸 파일과 지울 파일 목록만 출력 | 0 성공, 2 진행 불가 |
| `daemok serve` | `127.0.0.1:7412`에서 지도 서빙. `--addr`로 변경 | 종료 시 0 |

종료 코드 2는 설계 파일을 읽지 못했거나 설계 검사에 오류가 있어 명령을 진행할 수 없는 경우다. `gen`은 이때 아무것도 쓰지 않는다.

출력 형식은 `design/tx.yaml:12:5: 오류: tx_type.transfer를 처리하는 모듈이 없음`처럼 파일, 줄, 열, 수준, 메시지 순이다. 코드 대조 결과는 설계 파일 대신 Go 파일 위치를 가리킨다.

## 기준과 승인

- 기준은 git HEAD의 설계 파일이다. `git show HEAD:<path>`로 읽는다.
- 지도는 작업 중인 설계와 기준의 차이를 추가, 수정, 삭제로 표시한다.
- 승인은 설계 파일을 커밋하는 것이다. 도구는 별도 상태를 저장하지 않는다.
- git 저장소가 아니면 차이 표시 없이 동작한다.

## 생성 규칙

- 파일 위치, 이름, 이름 변환, 테이블 테스트 형태는 [모델의 생성 파일 절](../../design/01-model.md#생성-파일)을 따른다. 패키지 이름은 모듈 경로의 마지막 요소다.
- 생성 헤더가 없는 기존 파일을 덮어써야 하면 `gen`은 그 경로를 알리고 거부한다. 거부할 파일이 하나라도 있으면 아무것도 쓰지 않는다.
- `gen: false`인 노드는 생성하지 않는다. 코드 대조에는 포함한다.
- 출력은 `go/format.Source`를 거친다.
- 같은 설계에서는 항상 같은 바이트가 나온다. 순서는 설계 파일 이름순, 한 파일 안에서는 적힌 순서다.
- 지울 파일은 설계에 있는 모듈 경로, `cmd/<실행 단위>`, `ts_out`이 가리키는 디렉토리 안에서 생성 헤더가 있고 이번 생성 결과에 없는 파일이다. 그 밖의 디렉토리는 보지 않는다.
- 본문은 손으로 쓰는 파일에 있다. 핸들러 시그니처는 생성 파일의 `var _ Handler = handleX` 줄에서 컴파일러가 확인한다.
- 테이블 테스트의 `run<함수>Case`는 손으로 쓴다.

## import 대조

- `go/parser`를 `ImportsOnly` 모드로 써서 설계에 있는 모듈 경로의 Go 파일을 읽는다. `_test.go`는 제외한다.
- import 경로가 `go_module/<모듈 경로>`와 맞으면 그 모듈로 본다. 설계에 없는 내부 패키지는 관리하지 않는 패키지로 정보만 표시한다.
- 모듈 A가 B를 import하는데 설계상 A가 B에 의존할 수 없으면 오류다. 그룹 멤버는 그룹의 `depends_on`과 자기 `exceptions`를 따른다.
- 설계상 의존은 있는데 실제 import가 없으면 정보로 표시한다.

## 생성 파일 최신 여부

`check`는 `gen`이 쓸 내용을 메모리에서 만들어 디스크의 생성 파일과 바이트 단위로 비교한다. 내용이 다르거나 파일이 없으면 오류이고, 생성 헤더가 있는데 이번 결과에 없는 파일도 오류다. 메시지는 `daemok gen`을 안내한다. 비교 범위는 생성 규칙의 지울 파일 범위와 같다.

## 다른 언어

설계 모델에는 언어별 요소가 `go:` 같은 이름 규칙에만 있다. 언어 하나를 지원하려면 두 가지가 필요하다.

| 부분 | 내용 | Go 표준 라이브러리로 구현 |
|---|---|---|
| 생성기 | 모델에서 소스 텍스트를 만드는 템플릿 | 된다. 텍스트 생성이다 |
| import 스캐너 | 모듈 간 의존을 읽는다 | 된다. 줄 단위 스캔으로 충분하다 |

import 스캐너가 읽을 대상:

| 언어 | 의존 선언 |
|---|---|
| C, C++ | `#include` |
| UE5 C++ | `*.Build.cs`의 `PublicDependencyModuleNames`, `PrivateDependencyModuleNames` |
| TypeScript | `import ... from`, `export ... from` |
| Python | `import`, `from ... import` |

타입이나 시그니처가 설계와 맞는지는 파서를 만들어 확인하지 않는다. 생성 코드 안의 확인 구문을 대상 언어의 컴파일러가 검사한다. Go는 `var _ Handler = f`, C++은 `static_assert`, TypeScript는 `satisfies`를 쓴다. tree-sitter 같은 범용 파서는 cgo가 필요해서 외부 의존성 없는 조건과 맞지 않는다.

## 구조

Go 표준 라이브러리만 쓴다. MVP의 지도는 `embed`로 넣는 로컬 웹 페이지이며, 설계 상태를 확인하는 임시 수단이다. 최종 화면이 웹인지 프로그램 안의 화면인지는 정하지 않았고, 어느 쪽이든 `serve`가 주는 JSON을 그린다.

| 패키지 | 책임 | 의존 |
|---|---|---|
| `cmd/daemok` | 인자 해석, 명령 실행 | 아래 전부 |
| `internal/yaml` | YAML 부분집합 파싱. 키 순서와 줄·열 위치 보존 | 없음 |
| `internal/model` | 노드 타입, 파일 읽기와 병합, 참조 해석 | yaml |
| `internal/check` | 설계 검사 | model |
| `internal/codescan` | Go import 읽기, 설계와 대조 | model |
| `internal/gen/golang` | Go 생성 | model |
| `internal/gen/ts` | TS 생성 | model |
| `internal/gitbase` | HEAD의 설계 파일 읽기 | 없음(`git` 실행) |
| `internal/serve` | HTTP, 파일 폴링, SSE로 새로고침 알림, 웹 화면 | model, check, gen, gitbase |

`serve`는 `/api/state`에서 모델, 기준 모델, 검사 결과, 생성 파일을 JSON으로 준다. 웹 화면은 계산하지 않고 이 JSON을 그린다. 파일 감시는 1초 간격으로 수정 시각을 비교한다.

## YAML 부분집합

| 지원 | 지원하지 않음 |
|---|---|
| 블록 맵, 블록 리스트 | 앵커, 별칭, 병합 키 |
| 한 줄 `{}`와 `[]`(서로 중첩 가능), 여러 줄로 이어지는 `[]` | 여러 줄 문자열(`\|`, `>`) |
| 따옴표 없는 문자열, `"..."` 문자열 | 태그(`!!`) |
| 정수, `true`/`false`, `null` | 한 파일 안 여러 문서(`---`) |
| `#` 주석 | 탭 들여쓰기 |

- 지원하지 않는 문법은 `파일:줄:열`과 함께 오류다.
- 한 맵 안의 중복 키는 오류다. 형식에 없는 키는 모델 단계에서 오류다.
- 숫자처럼 보이는 값 중 단위가 붙은 것(`2s`, `168h`)은 문자열이다.

## 테스트

- 구현은 TDD로 한다.
- `internal/yaml`: 지원 문법별 파싱 결과, 지원하지 않는 문법별 오류 위치.
- `internal/model`, `internal/check`: 규칙마다 통과하는 설계와 걸리는 설계를 `testdata`에 둔다.
- `internal/gen`: `testdata/<샘플>/design`에서 생성한 결과를 골든 파일과 비교한다. 생성된 Go는 임시 모듈에 손으로 쓴 스텁 본문과 함께 넣고 `go build`와 `go vet`을 통과해야 한다. 샘플은 [프로토타입](../../../prototype/daemok-v2.html) `SAMPLES`의 tolchain과 tolelog 설계 파일을 v0.3에 맞춰 옮긴 것이다.
- `internal/codescan`: `testdata` 안의 작은 Go 모듈로 허용, 위반, 예외, 관리하지 않는 패키지를 확인한다.
- `internal/serve`: `httptest`로 `/api/state` 응답을 확인한다.
- 웹 화면은 자동 테스트를 두지 않는다. 배포 전에 headless Chrome으로 렌더링과 콘솔 오류를 확인한다.

## 구현 순서

| 단계 | 결과 |
|---|---|
| 1 | `internal/yaml` |
| 2 | `internal/model`: 두 샘플을 읽는다 |
| 3 | `internal/check`, `daemok check`의 설계 검사 |
| 4 | `internal/gen/golang`, `daemok gen` |
| 5 | `internal/gen/ts` |
| 6 | `internal/codescan`, 생성 파일 최신 여부 |
| 7 | `internal/gitbase`, `internal/serve`, 웹 화면 이식 |
| 8 | tolchain 도입과 완료 조건 확인 |

3단계부터 tolchain 설계 파일에 `daemok check`를 돌려볼 수 있다.

## 위험

| 위험 | 대응 |
|---|---|
| YAML 부분집합이 사용자가 기대하는 YAML과 다름 | 지원하지 않는 문법을 조용히 넘기지 않고 위치와 함께 오류를 낸다 |
| 도입 시 생성 파일이 기존 선언과 충돌 | `gen`이 헤더 없는 파일을 덮어쓰지 않는다. 첫 단계에서 바꿀 선언은 사람이 지우고, 나머지는 `gen: false`로 남긴다 |
| 웹 화면이 커져 순수 JS로 유지하기 어려움 | 계산은 서버가 하고 화면은 JSON을 그리기만 한다 |
