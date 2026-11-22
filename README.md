# daemok

> 설계를 유일한 원본으로 두고, 탑다운으로 한 계층씩 시각적으로 확인하며 확정하고, 코드는 설계에서 한 방향으로 생성하는 도구

Go 1.27 · MCP

## 기능

- 설계 검사, 코드 import 대조(설계에 `cpp`가 있으면 C++ include 대조), 생성 파일 최신 여부 확인
- 설계에서 Go 생성 파일과 TypeScript 파일 생성, 설계에 없는 Go 생성 파일 정리
- 기존 Go 모듈 코드에서 설계 초안 추출(모듈, tier, 의존, 손으로 쓴 선언의 L3 노드)
- 패키지마다 설계 노드에 대응하는 선언 비율 표
- 기준 ref와 작업 폴더의 설계를 노드·키 단위로 비교해 PR 설명에 붙일 마크다운 출력
- 설계 노드 ID와 그 참조를 설계 파일 전체에서 바꾸기
- 지도, 모듈, 흐름, 요구사항 탭으로 설계를 보는 로컬 웹 화면. 저장하면 다시 그리고 git HEAD 대비 바뀐 노드 표시
- 화면에서 tier와 의존 편집, 초안 저장, 생성 결과 미리보기. 설계 파일의 해당 줄만 바꾸고 주석과 정렬 유지
- AI 클라이언트가 설계를 읽고 검사하는 MCP 서버. 허용하면 설계 편집

## 시작하기

Go 1.27 이상이 필요합니다. 외부 의존성은 없습니다.

```bash
go install github.com/tolelom/daemok/cmd/daemok@latest
```

### 사용법

코드 루트(`--root`)는 현재 디렉토리, 설계 디렉토리(`--design`)는 `<root>/design`(`*.yaml`)이 기본입니다. 두 옵션은 `version`을 뺀 모든 명령이 받습니다.

| 명령 | 하는 일 | 그 밖의 옵션 |
|---|---|---|
| `daemok check` | 설계 검사, import·include 대조, 생성 파일 최신 여부 출력 | `--strict`(경고도 실패), `--format json` |
| `daemok gen` | 설계에서 Go 생성 파일과 `ts_out` TS 파일을 쓰고, 설계에 없는 Go 생성 파일 삭제 | `--dry-run` |
| `daemok init` | Go 모듈 코드에서 L2 초안(모듈, tier, 의존)을 `design/`에 기록. 설계 파일이 이미 있으면 쓰지 않음 | `--dry-run`(출력만), `--l3`(설계에 없는 손으로 쓴 선언을 `gen: false` L3 노드 초안으로 출력) |
| `daemok coverage` | 패키지마다 최상위 선언 중 설계 노드에 대응하는 비율과 함수를 뺀 비율 출력 | |
| `daemok diff` | 기준 ref와 작업 폴더의 설계 비교. 주석·형식·파일 위치만 바뀐 것은 변경으로 보지 않고, 고정 키만 다른 삭제·추가 짝은 이름 바뀜으로 묶음 | `--base REF`(기본 `HEAD`) |
| `daemok rename <종류> <옛 ID> <새 ID>` | 설계 노드 ID와 참조 변경. 기본은 코드 이름을 고정 키(`go.name`, `go.type`, `path`)로 유지하고 생성 코드가 그대로일 때만 기록 | `--code`(생성 코드 이름도 새 ID로) |
| `daemok mcp` | 표준 입출력 MCP 서버 | `--allow-write`(설계를 고치는 `edit` 도구 추가) |
| `daemok serve` | 설계 지도를 로컬 웹 화면으로 띄우기 | `--addr HOST:PORT`, `--workspace DIR` |
| `daemok version` | 버전과 빌드한 커밋 출력 | |

종료 코드는 0 통과, 1 문제 있음, 2 진행할 수 없음입니다.

### 화면

`daemok serve`가 출력한 주소를 브라우저로 엽니다. 기본 주소는 `--addr`로 바꿀 수 있습니다.

| 접속 | 주소 |
|---|---|
| 브라우저 | http://127.0.0.1:7412 |

- 머리의 프로젝트 이름을 누르면 다른 프로젝트를 엽니다. 경로를 입력하거나 최근 목록, `--workspace` 아래의 Go 모듈 목록에서 고릅니다.
- 설계 파일이 없는 Go 모듈은 코드에서 추정한 초안을 지도로 보여 주고, 초안 탭에서 `design/`에 저장합니다.
- 생성 탭은 `daemok gen`이 쓰고 지울 파일과 생성할 내용을 보여 주기만 합니다.
- 프로젝트 열기, 초안 저장, 편집은 루프백 주소로 띄웠을 때만 됩니다. 승인은 커밋입니다.

### AI 연동

MCP를 지원하는 클라이언트의 서버 설정에 `daemok mcp`를 등록합니다.

```json
{
  "mcpServers": {
    "daemok": { "command": "daemok", "args": ["mcp", "--root", "/path/to/project"] }
  }
}
```

| 도구 | 주는 것 |
|---|---|
| `design` | 설계 파일, 또는 모듈 하나의 의존·소속 노드 |
| `check` | 진단 JSON |
| `draft` | 코드에서 만든 초안. `l3: true`면 L3 초안 |
| `gen` | 쓰고 지울 생성 파일과 상태. `path`를 주면 그 파일의 생성할 내용 |
| `diff` | 설계 비교 |
| `edit` | `--allow-write`일 때만. 설계의 L2 수정과 노드 이름 바꾸기 |

도구는 인자로 받은 경로를 제한하지 않으므로, 클라이언트가 넘긴 디렉토리의 `*.yaml`을 읽어 돌려줄 수 있습니다. `edit`는 서버의 `--design`만 고치고, 클라이언트는 호출마다 사용자에게 묻습니다.

### 기존 프로젝트에 도입

1. `daemok init`으로 초안을 만듭니다. tier 이름과 그룹은 사람이 정합니다.
2. `daemok check`로 설계와 코드가 맞는지 봅니다.
3. `daemok init --l3 > design/l3.yaml`로 손으로 쓴 에러·상수·enum·struct·interface와 설계 에러를 반환하는 함수를 설계에 올립니다. 초안은 설계에 더해도 검사를 통과하는 노드만 담습니다.
4. 생성할 노드를 설계에 더하고 `daemok gen`으로 생성 파일을 씁니다. 이미 손으로 쓴 선언이 있는 노드는 `gen: false`로 둡니다.

## 문서

| 문서 | 내용 |
|---|---|
| `docs/design/00-overview.md` | 문제와 확정된 결정 |
| `docs/design/01-model.md` | 설계 파일 형식 |
| `docs/design/02-slice-tolchain.md` | tolchain 도입 기록 |
| `docs/design/03-slice-tolelog.md` | tolelog 도입 기록 |
| `docs/design/04-open-items.md` | 정하지 않은 것과 미룬 작업 |
| `docs/specs/` | 기능별 설계 스펙 |
| `docs/code-notes.md` | 파일·선언별 코드 메모 |
| `design/daemok.yaml` | daemok 자신의 설계 |

## 기여

검사 명령과 코드 규칙은 `CONTRIBUTING.md`에 있습니다.
