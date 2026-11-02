# 형식 v0.5

> tolchain과 tolelog 백엔드 두 도입 사례에서 공통으로 막힌 형식 제약을 풉니다. 에러의 Go 이름 분리, `gen: false` 계약과 타입의 시그니처 대조, 리터럴 검사의 SQL 자리표시자 인자입니다. 그룹 `kind`를 선택으로 하는 변경은 넣었다가 뺐습니다.

## 근거

| 막힌 곳 | 사례 |
|---|---|
| 에러 ID가 곧 Go 이름이고 설계 전체에서 하나 | tolelog `post`, `comment`, `series`가 같은 이름의 `ErrUnauthorized`, `ErrPostNotFound`를 따로 선언합니다. `post` 쪽만 적었습니다 |
| 그룹은 `kind` 필수 | tolelog의 같은 모양 도메인 다섯(`post`, `comment`, `series`, `tag`, `user`)에 핸들러 등록 방식이 없어 그룹으로 묶지 못했습니다. 0.5에서 다루지 않습니다(아래 그룹 절) |
| 코드 대조가 계약과 타입의 이름만 봄 | tolelog `Service` interface 5개(메서드 약 35개)와 엔티티 필드를 올리지 않았습니다 |
| 리터럴 검사가 비교와 case만 봄 | tolelog 글 상태 리터럴 7곳 중 6곳이 GORM `Where`의 인자라 안 잡혔습니다 |

## 버전

최상위 `version`은 `0.5`여야 합니다. 0.4 파일은 첫 줄만 바꾸면 그대로 읽힙니다. 형식 변경은 키 추가(에러의 `go.name`)뿐입니다.

## 에러의 Go 이름

```yaml
errors:
  ErrUnauthorized:      { in: post, message: "권한 없음", http: 403 }
  comment_unauthorized: { in: comment, message: "권한 없음", http: 403, go: { name: ErrUnauthorized } }
```

- 에러에 `go.name`을 둘 수 있습니다. 없으면 지금처럼 ID가 Go 이름입니다.
- `go.name`이 없으면 ID가 export되는 Go 이름이어야 합니다. 있으면 `go.name`이 export되는 Go 이름이어야 하고 ID에는 Go 이름 규칙을 두지 않습니다.
- ID는 지금처럼 설계 전체에서 하나입니다. 참조(`errors: [...]`, 케이스의 `error`)는 ID로 합니다.
- 같은 패키지 생성 이름 충돌, 생성 코드, 케이스 테스트의 `errors.Is` 대상, 코드 대조는 모두 Go 이름을 씁니다.

## 그룹 `kind`는 필수로 둠

`kind`를 선택으로 하고 `kind` 없는 그룹을 생성 없는 L2 제약 묶음으로 두는 변경을 넣었다가 되돌렸습니다. tolelog의 다섯 도메인은 서로 import하지 않지만(`975ae34` 확인) `depends_on`이 서로 다르고(`tag`는 `dto`만, `user`는 다섯 개), 멤버에는 개별 `depends_on`이 없습니다. 묶으면 그룹 `depends_on`이 합집합이 되어 모듈별 의존 검사가 느슨해지므로 묶지 않았고, 그러면 이 변경을 쓰는 곳이 없습니다. 형제끼리 import하지 않는다는 것은 모듈로 둔 지금도 설계에 없는 의존 오류로 잡힙니다. 같은 이유로 `daemok init` 초안의 그룹 후보도 메모로 둡니다.

## 계약과 타입의 시그니처 대조

코드 대조는 `gen: false`인 `call` 계약과 타입의 선언이 소유 패키지에 있는지 볼 뿐 아니라, 생성했다면 나왔을 모양과 같은지 봅니다. 코드 디렉토리가 없는 모듈은 지금처럼 보지 않습니다.

### 비교 형식

양쪽 타입식을 생성 코드가 소유 패키지 안에서 쓰는 모양으로 바꿔 문자열로 비교합니다. 같은 패키지 이름은 그대로, 설계 모듈(go_module 안) 패키지와 생성기가 쓰는 표준 라이브러리(`encoding/json`, `time`, `net/http`, `context`, `io`)는 import 경로의 마지막 요소로, 그 밖의 패키지는 코드와 설계에 쓴 이름 그대로 한정합니다(`*model.Post`, `[]model.Post`, `json.RawMessage`, `gorm.DeletedAt`, `int64`).

| 쪽 | 바꾸는 방법 |
|---|---|
| 설계 | 생성기와 같은 함수(`argType`, `fieldType`)로 바꿉니다(`golang.ContractShape`, `golang.TypeShape`). 생성 규칙(설계 struct는 접두사가 없으면 포인터, 반환 끝 `error`, `optional`과 `ref`의 포인터)이 한곳에만 있습니다 |
| 코드 | AST 타입식을 읽습니다. 맨 이름은 그대로, `pkg.Name`은 `pkg`가 설계 모듈이나 위 표준 라이브러리를 가리키면(별칭 import 포함) 그 경로의 마지막 요소로 바꾸고, 그 밖이면 코드에 쓴 이름 그대로 둡니다. 설계에는 그 밖의 패키지를 코드와 같은 이름(`gorm.DeletedAt`)으로 적습니다. `*`, `[]`, `map[K]V`, 빈 `interface{}`(`any`로)를 지원합니다 |

마지막 요소가 같은 서로 다른 패키지는 구분하지 않습니다. 함수 타입, 채널, 배열 길이, 가변 인자 같은 나머지 타입식은 대조할 수 없다고 정보로 알리고 그 메서드나 필드는 비교하지 않습니다. 코드 선언이 타입 별칭(`type X = Y`)이거나 제네릭이면 그 계약·타입 전체를 같은 정보로 알리고 비교하지 않습니다.

메서드와 필드는 이름의 대소문자와 밑줄을 무시하고 맞춥니다. 설계 `avatar_url`의 생성 이름은 `AvatarUrl`이고 손으로 쓴 코드는 `AvatarURL`일 수 있습니다. 코드 쪽은 export된 필드만 봅니다.

### 계약

- 코드에서 계약의 Go 이름을 가진 interface 선언을 찾습니다. 없으면 지금처럼 선언이 없다는 오류입니다. interface가 아닌 타입 선언이나 함수·변수·상수 선언이면 오류입니다.
- 설계 메서드마다 같은 Go 이름의 코드 메서드를 찾습니다. 없으면 오류입니다.
- 파라미터와 반환은 순서와 개수, 한정 타입이 같아야 합니다. 다르면 오류이고 기대한 시그니처와 코드의 시그니처를 함께 보여 줍니다. 파라미터 이름은 비교하지 않습니다.
- 코드에만 있는 메서드는 계약마다 경고 하나로 모아 알립니다(`--strict`에서 실패). 처음 도입할 때 일부만 적는 단계를 허용합니다.
- 코드 interface에 다른 interface를 끼워 넣었으면 메서드 집합을 확정할 수 없어 코드에만 있는 메서드 경고를 내지 않고 정보로 알립니다.

### 타입

- 코드에서 타입의 Go 이름을 가진 struct 선언을 찾습니다. struct가 아닌 타입 선언이나 함수·변수·상수 선언이면 오류입니다.
- 설계 필드마다 같은 Go 이름의 코드 필드를 찾고 한정 타입을 비교합니다. 다르면 오류입니다. 태그는 비교하지 않습니다(GORM 태그 등 손으로 쓴 태그가 정당합니다).
- 코드에 없는 설계 필드는 오류입니다. 단, 코드 struct에 끼워 넣은 필드가 있으면 그쪽에서 올 수 있어 타입마다 정보 하나로 낮춥니다.
- 코드에만 있는 export 필드는 타입마다 경고 하나로 모아 알립니다. 끼워 넣은 필드 자체는 보지 않습니다.

## 리터럴 검사의 SQL 자리표시자 인자

지금 검사(쓸 수 있는 enum 값과 같은 문자열 리터럴을 비교나 case에 쓴 곳, 정보)에 자리 하나를 더합니다. GORM·database/sql 메서드(`Where`, `Or`, `Not`, `Having`, `Joins`, `Select`, `Raw`, `Exec`, `Find`, `First`, `Take`, `Last`, `Delete`, `Query`, `QueryRow`, `QueryContext`, `QueryRowContext`, `ExecContext`) 호출의 인자 중 `?`가 든 문자열 리터럴이 있으면, 그 뒤 인자에 있는 enum 값 리터럴도 알립니다. 뒤 인자가 슬라이스 리터럴이면(`IN ?`) 원소를 봅니다. `db.Where("status = ?", "published")`가 해당합니다. 쿼리 문자열이 상수나 변수인 호출, 쿼리 문자열 안에 값을 직접 쓴 `Where("status = 'published'")`, `$1` 형식 자리표시자, 다른 이름의 호출은 보지 않습니다.

## 완료 조건

1. tolchain `feat/daemok`와 tolelog `feat/daemok`의 설계 파일을 0.5로 올리고 `daemok check`가 오류 0입니다. 새 대조가 실제 어긋남을 잡으면 설계를 고치고 결과 문서에 적습니다.
2. tolelog에 `comment`, `series`의 에러를 `go.name`으로 올리고, `post` 계약(`Service`)과 엔티티를 시그니처·필드 대조로 올린 뒤 대응 비율을 다시 잽니다. 다섯 도메인의 그룹 적용은 위 그룹 절의 이유로 하지 않습니다. 결과는 `docs/design/03-slice-tolelog.md`에 적습니다.
3. `docs/design/01-model.md`를 0.5 기준으로 고치고 `04-open-items.md`에서 그룹 `kind` 항목을 지웁니다.
