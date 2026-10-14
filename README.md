# daemok

설계를 유일한 원본으로 두고, 탑다운으로 한 계층씩 시각적으로 확인하며 확정하고, 코드는 설계에서 한 방향으로 생성하는 도구.

이름은 대목(大木)에서 왔다. 한옥에서 설계부터 시공까지 맡는 목수를 가리킨다.

설계 문서는 [docs/design](docs/design/)에, 설계 파일 형식은 [모델](docs/design/01-model.md)에 있다. 정하지 않은 것과 미룬 작업은 [미결](docs/design/04-open-items.md)에 있다.

## 설치

```
go install github.com/tolelom/daemok/cmd/daemok@latest
```

Go 1.27 이상. 외부 의존성은 없다.

## 명령

코드 루트(`--root`)는 현재 디렉토리, 설계 디렉토리(`--design`)는 `<root>/design`(`*.yaml`)이 기본이다.

| 명령 | 하는 일 | 주요 옵션 |
|---|---|---|
| `daemok check` | 설계 검사, import 대조, 생성 파일 최신 여부를 출력한다 | `--design DIR`, `--root DIR`, `--strict`(경고도 실패), `--format json` |
| `daemok gen` | 설계에서 Go 생성 파일과 `ts_out` TS 파일을 쓰고, 설계에 없는 Go 생성 파일을 지운다 | `--design DIR`, `--root DIR`, `--dry-run` |
| `daemok init` | Go 모듈의 코드에서 L2 초안(모듈, tier, 의존)을 `design/`에 쓴다. 설계 파일이 이미 있으면 쓰지 않는다. `--dry-run`은 설계 파일과 관계없이 출력만 한다. `--l3`는 설계가 있을 때 설계에 없는 손으로 쓴 선언을 `gen: false` L3 노드 초안으로 출력한다 | `--root DIR`, `--design DIR`, `--dry-run`, `--l3` |
| `daemok coverage` | 패키지마다 최상위 선언 중 설계 노드에 대응하는 것의 비율과 함수를 뺀 비율을 표로 출력한다 | `--design DIR`, `--root DIR` |
| `daemok rename` | 설계 노드 하나의 ID와 그 참조를 설계 파일 전체에서 바꾼다(`daemok rename types post article`). 기본은 코드 이름을 고정 키(`go.name`, `go.type`, `path`)로 유지하고 생성 코드가 그대로일 때만 쓴다. `--code`는 생성 코드 이름도 새 ID를 따르게 한다 | `--code`, `--design DIR`, `--root DIR` |
| `daemok mcp` | 표준 입출력으로 MCP 서버를 돌린다. 읽기 도구는 `design`, `check`, `draft`, `gen`이고, `--allow-write`면 설계를 고치는 `edit`도 낸다 | `--design DIR`, `--root DIR`, `--allow-write` |
| `daemok serve` | 설계 지도를 로컬 웹 화면으로 띄운다 | `--design DIR`, `--root DIR`, `--addr HOST:PORT`(기본 `127.0.0.1:7412`), `--workspace DIR` |

`daemok version`은 버전과 빌드한 커밋을 출력한다. 종료 코드는 0 통과, 1 문제 있음, 2 진행할 수 없음이다.

## 화면

`daemok serve`가 출력한 주소를 브라우저로 연다.

- 지도, 모듈, 흐름, 요구사항 탭으로 설계를 본다. 설계 파일을 저장하면 다시 그려지고, git HEAD 대비 바뀐 노드가 표시된다.
- 머리의 프로젝트 이름을 누르면 다른 프로젝트를 연다. 경로를 입력하거나, 최근 목록이나 `--workspace` 아래의 Go 모듈 목록에서 고른다.
- 설계 파일이 없는 Go 모듈을 열면 분석 모드다. 코드에서 추정한 초안을 지도로 보여 주고, 초안 탭에서 `design/`에 저장한다.
- 모듈 탭에서 모듈·그룹의 tier와 `depends_on`을 고치고 모듈을 지우며, 지도 오른쪽 칸에서 모듈을 더한다. 설계 파일의 해당 줄만 바꾸고 주석과 정렬은 그대로 둔다. 승인은 커밋이다.
- L3 초안 탭에서 설계에 없는 손으로 쓴 선언의 초안(`daemok init --l3`와 같다)을 보고 `design/`에 새 파일로 저장한다.
- 생성 탭에서 `daemok gen`이 쓰고 지울 파일과 그 상태(새 파일, 바뀜, 지울 파일, 쓰지 않음)를 보고, 파일을 누르면 생성할 내용을 본다. 이 탭은 쓰지 않는다.
- 프로젝트 열기, 초안 저장, 편집은 루프백 주소로 띄웠을 때만 된다.
- 색은 시스템의 다크·라이트 설정을 따른다.

## AI 연동

MCP를 지원하는 클라이언트의 서버 설정에 `daemok mcp`를 등록한다.

```json
{
  "mcpServers": {
    "daemok": { "command": "daemok", "args": ["mcp", "--root", "/path/to/project"] }
  }
}
```

`design`은 설계 파일이나 모듈 하나의 의존·소속 노드를, `check`는 진단 JSON을, `draft`는 코드에서 만든 초안을 준다(`l3: true`면 설계에 없는 손으로 쓴 선언의 L3 초안). `gen`은 `daemok gen`이 쓰고 지울 생성 파일과 상태를, `path`를 주면 그 파일의 생성할 내용을 준다. AI가 코드를 고친 뒤 `check`로 설계와 어긋나지 않았는지 확인하는 데 쓴다. 도구는 인자로 받은 경로를 제한하지 않으므로, 클라이언트가 넘긴 디렉토리의 `*.yaml`을 읽어 돌려줄 수 있다.

`args`에 `--allow-write`를 더하면 `edit` 도구로 AI가 설계의 L2를 고치고 노드 이름을 바꿀 수 있다. 서버의 `--design`만 고치고, 클라이언트는 호출마다 사용자에게 묻고, 승인은 커밋이다.

## 기존 프로젝트에 도입

1. `daemok init`으로 초안을 만든다. tier 이름과 그룹은 사람이 정한다.
2. `daemok check`로 설계와 코드가 맞는지 본다.
3. `daemok init --l3 > design/l3.yaml`로 손으로 쓴 에러·상수·enum·struct·interface와 설계 에러를 반환하는 함수를 설계에 올린다. 초안은 설계에 더해도 검사를 통과하는 노드만 담는다.
4. 생성할 노드를 설계에 더하고 `daemok gen`으로 생성 파일을 쓴다. 이미 손으로 쓴 선언이 있는 노드는 `gen: false`로 둔다.

tolchain에 도입한 기록은 [tolchain 슬라이스](docs/design/02-slice-tolchain.md)에 있다.
