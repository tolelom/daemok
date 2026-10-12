# 슬라이스: tolelog 게시글

> [모델](01-model.md)을 실제 코드에 적용한 예시. 코드 기준은 github.com/tolelom/fiber_api_server와 github.com/tolelom/tolelog `main`(2026-08-26)이다.

## 현재 코드

| 항목 | 위치 | 내용 |
|---|---|---|
| 실행 파일 | `cmd/server/main.go` | 하나 |
| 도메인 패키지 | `internal/{post,comment,user,series,tag,feed,image,sitemap}` | 각각 handler와 service. repository는 sitemap에만 있고 나머지 service는 GORM을 직접 호출 |
| 조립 | `internal/router/router.go:84-103` | 모든 도메인의 DI |
| 서비스 계약 | `internal/post/service.go:23-34` | `Service` interface, 메서드 10개 |
| 에러 | `internal/post/service.go:15-21` | sentinel 5개 |
| 엔티티 | `internal/model/post.go:9-27` | `Post` |
| 요청 DTO | `internal/dto/post.go:20-34` | `CreatePostRequest`, `UpdatePostRequest` |
| API 문서 | `docs/swagger.json` | swaggo가 코드 주석에서 생성 |

어긋난 곳:

| 유형 | 사례 |
|---|---|
| 상태값에 정의가 없음 | `Post.Status`에 상수도 검증도 없다. `"published"`가 `model/post.go:16`, `post/service.go:76`, `post/service_query.go:58·105·151`, `sitemap/repository.go:21`에 리터럴로 있다. DTO에 `oneof`가 없어 임의 문자열이 저장된다 |
| 백엔드와 프론트의 제한값이 다름 | 시리즈 제목·설명이 백엔드 `max=255`/`max=2000`(`dto/series.go:9-10`), 프론트 `maxLength={100}`/`{500}`(`SeriesFormModal.tsx:88,102`) |
| 같은 값이 양쪽에 따로 있음 | 페이지 크기 10, 이미지 5MB와 허용 형식, 아이디 4-20자, 비밀번호 6-128자, slug 60, 댓글 2000 |
| 계층 방향 위반 | 서비스 계약 `UpdatePost`가 전송용 `dto.UpdatePostRequest`를 받는다. 핸들러가 `model.Post`를 직접 만든다 |
| 에러 이름과 HTTP 상태가 다름 | `ErrUnauthorized`가 403으로 나간다 |
| 문서가 낡음 | `CLAUDE.md`의 모델 목록 4개(실제 6개), 라우트 그룹 누락, swagger에 PATCH `/posts/{id}` 없음 |

## 설계 파일

```yaml
requirements:
  REQ-post-draft: { text: "작성자는 글을 임시저장하고 본인만 볼 수 있다", status: active }

tiers: [foundation, domain, support, service, transport]

modules:
  model:    { tier: domain }
  httputil: { tier: support }
  post:     { tier: service,   depends_on: [model] }
  dto:      { tier: transport, depends_on: [model] }
  router:   { tier: transport, depends_on: [post, dto] }

enums:
  post_status:
    in: model
    values: [published, draft]
    satisfies: [REQ-post-draft]

consts:
  page_size_default: { in: httputil, type: int, value: 10, export: [ts] }
  page_size_max:     { in: httputil, type: int, value: 100 }
  series_title_max:  { in: dto, type: int, value: 255, export: [ts] }
  post_title_max:    { in: dto, type: int, value: 255, export: [ts] }

errors:
  ErrPostNotFound: { in: post, message: "post not found", http: 404 }
  ErrUnauthorized: { in: post, message: "unauthorized",   http: 403 }  # 이름과 상태가 달라 경고

types:
  post:
    in: model
    fields:
      title:  { type: string, max: post_title_max }
      status: { type: post_status }
      user:   { ref: user }
      tags:   { ref: tag, many: true }
```

## 검사

| 검사 | 결과 |
|---|---|
| 계층 방향 | `post`(service) → `dto`(transport)는 상위 tier 의존이라 에러 |
| 에러와 HTTP 상태 | `ErrUnauthorized`가 403이면 경고. 매핑 코드(`HTTPStatus`)를 생성하므로 핸들러마다 따로 적지 않는다 |

`post_status` 값과 같은 문자열 리터럴은 함수 본문의 비교(`==`, `!=`)와 switch case에 쓴 것만 정보로 알린다. 쿼리 인자처럼 다른 자리에 쓴 리터럴은 보지 않는다.

## 생성물

```go
// internal/model/post_status.gen.go
type PostStatus string

const (
	PostStatusPublished PostStatus = "published"
	PostStatusDraft     PostStatus = "draft"
)

func (s PostStatus) Valid() bool {
	return s == PostStatusPublished || s == PostStatusDraft
}

// internal/dto/limits.gen.go
const (
	SeriesTitleMax = 255
	PostTitleMax   = 255
)
```

```ts
// tolelog/src/generated/limits.gen.ts
export const PAGE_SIZE_DEFAULT = 10;
export const SERIES_TITLE_MAX = 255;
export const POST_TITLE_MAX = 255;
```

`export: [ts]`가 붙은 상수와 enum은 TypeScript 상수로도 생성한다. 백엔드와 프론트가 같은 값을 따로 들고 있던 곳이 한 정의로 모인다.

## 도입 결과

fiber_api_server `feat/daemok` 브랜치(기준 `main` `6df1865`)에 설계 파일 `design/fiber.yaml`을 두고 도입했다. 배포 중인 서비스라 동작이 바뀌지 않는 범위로 정했다. 모듈 관계는 지금 코드대로 `depends_on`으로 적었고, 생성물로 바꾼 것은 `post_status` enum 하나다. 손으로 쓴 sentinel 에러 22개와 `model`의 GORM 엔티티 6개는 `gen: false`로 올렸다.

| 대상 | 전 | 후 |
|---|---|---|
| 글 상태 | `Post.Status string`과 리터럴 `"published"`·`"draft"` 7곳(`post` 6곳, `sitemap` 1곳) | `internal/model/post_status.gen.go`의 `PostStatus`. 필드 타입이 바뀌어 다른 문자열 변수를 넣으면 컴파일 오류다. 테스트 픽스처 `WithStatus`도 이 타입을 받는다 |

- `daemok check`: 오류 0, 경고 0, 정보 0. `go vet ./...`와 `go test ./...`는 통과한다. CI의 golangci-lint는 돌리지 않았다.
- 형식 0.4에서 문자열 리터럴 검사는 7곳 중 `==` 비교 1곳(`internal/post/service.go:107`)만 잡았다. 나머지 6곳 중 5곳은 GORM `Where`의 인자이고 1곳은 대입(`service.go:76`)이다.
- `gen: false` 노드 28개의 선언은 코드 대조가 모두 찾았다. 에러 이름 하나와 타입 이름 하나를 틀리게 바꾼 사본에서는 그 둘을 오류로 낸다.
- 필드 타입을 바꾸자 컴파일러가 `internal/testutil/fixtures.go`의 문자열 변수 하나를 잡았다. 리터럴 검사로는 잡히지 않는 자리다.

### 형식 0.5 적용

`feat/daemok` `382fdec`에서 설계 파일을 형식 0.5로 올렸다.

| 대상 | 0.4 | 0.5 |
|---|---|---|
| 같은 이름의 sentinel | `ErrUnauthorized`, `ErrPostNotFound`는 `post` 쪽만 적음 | `comment`, `series` 쪽 4개를 따로 ID를 두고 `go.name`으로 적음. 에러 26개 |
| 엔티티 | 이름만 대조. GORM 관리 필드는 적지 않음 | 엔티티 6개의 필드 전부를 적고 필드 타입까지 대조. 연관 `User`는 값 필드라 `ref`가 아니라 `type: user`로 적음 |
| post 서비스 계약 | 올리지 않음 | `post_service`(`Service`) 메서드 10개를 시그니처까지 대조. `errors`는 코드의 `return` 기준 |

- `daemok check`: 오류 0, 경고 0, 정보 0(`85fd642`).
- `IsLiked(postID, userID uint) bool`은 계약 메서드의 반환 끝 `error` 규칙 때문에 처음에는 적지 못해 경고로 남았다. 이 메서드는 `Count`의 오류를 버려 조회가 실패하면 `false`를 돌려주고 있었다. `(bool, error)`로 바꾸고 좋아요 상태 조회 handler는 오류 시 500을 내게 했다(`85fd642`). 테스트 `TestIsLiked_DBError`, `TestGetLikeStatus_ServiceError`가 이 동작을 고정한다.
- 엔티티 대조를 처음 돌렸을 때 `series`, `post`, `comment`의 `user`를 `ref`(포인터)로 적은 것이 코드의 값 필드와 달라 오류 3개가 나왔다. 0.4에서는 필드를 대조하지 않아 드러나지 않았다.
- 같은 설계를 도입 전 코드(`main` `6df1865`)에 대면 리터럴 검사가 `Where` 인자 5곳을 포함해 6곳을 잡는다. 대입 1곳은 보지 않는다. 엔티티 대조는 `Post.Status`가 `string`이라 설계의 `PostStatus`와 다르다는 오류를 낸다.

### 그룹

`post`, `comment`, `series`, `tag`, `user`는 handler와 service로 된 같은 모양의 도메인이고 서로 import하지 않는다(`975ae34`). 형식 0.5에서는 `kind` 없는 그룹으로 묶을 수 있지만 묶지 않았다. 다섯 모듈의 `depends_on`이 서로 다르고(`tag`는 `dto`만, `user`는 `dto`, `model`, `upload`, `utils`, `validate`) 멤버에는 개별 `depends_on`이 없어, 묶으면 그룹 `depends_on`이 합집합이 되어 모듈별 의존 검사가 느슨해진다. 형제끼리 import하지 않는다는 것은 모듈로 둔 지금도 설계에 없는 의존 오류로 잡힌다.

### 설계 노드 대응 비율

`daemok coverage` 기준 최상위 선언 197개 중 37개(18%)다. 형식 0.4에서는 32개(16%)였고, 늘어난 5개는 `comment`, `series`의 에러 4개와 `post`의 `Service`다. 이 비율은 최상위 선언만 세므로 0.5에서 대조하게 된 엔티티 필드와 계약 메서드는 들어가지 않는다.

| 패키지 | 최상위 선언 | 설계 대응 | 비율 |
|---|---|---|---|
| `internal/model` | 10 | 10 | 100% |
| `internal/user` | 11 | 6 | 54% |
| `internal/comment` | 9 | 4 | 44% |
| `internal/series` | 9 | 4 | 44% |
| `internal/post` | 22 | 6 | 27% |
| `internal/utils` | 24 | 6 | 25% |
| `internal/httputil` | 5 | 1 | 20% |
| 그 밖의 14개 | 107 | 0 | 0% |
| 합계 | 197 | 37 | 18% |

가장 큰 `internal/dto`(43)는 요청·응답 DTO이고 설계에 올리지 않았다.

## 범위 밖

- 라우트(메서드, 경로, 인증, 리미터)는 설계에 두지 않는다. API 계약은 L0 인터페이스의 `spec`이 가리키는 OpenAPI 파일이 갖는다. swagger에서 PATCH `/posts/{id}`가 빠진 것 같은 어긋남은 잡지 못한다.
- 태그가 `tags` 컬럼과 `post_tags` 조인 테이블에 이중 저장되는 구조는 설계 파일로 옮겨도 그대로 드러날 뿐 고쳐지지 않는다.
