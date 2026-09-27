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

tiers: [foundation, domain, service, transport]

modules:
  model: { tier: domain }
  post:  { tier: service,   depends_on: [model] }
  dto:   { tier: transport, depends_on: [model] }
  router: { tier: transport, depends_on: [post, dto] }

enums:
  post_status:
    in: model
    values: [published, draft]
    default: published
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
| 상태값 리터럴 | `post_status` 값과 같은 문자열 리터럴이 모듈 안에 있으면 경고 |
| 에러와 HTTP 상태 | `ErrUnauthorized`가 403이면 경고. 매핑 코드(`HTTPStatus`)를 생성하므로 핸들러마다 따로 적지 않는다 |

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

## 범위 밖

- 라우트(메서드, 경로, 인증, 리미터)는 설계에 두지 않는다. API 계약은 L0 인터페이스의 `spec`이 가리키는 OpenAPI 파일이 갖는다. swagger에서 PATCH `/posts/{id}`가 빠진 것 같은 어긋남은 잡지 못한다.
- 태그가 `tags` 컬럼과 `post_tags` 조인 테이블에 이중 저장되는 구조는 설계 파일로 옮겨도 그대로 드러날 뿐 고쳐지지 않는다.
