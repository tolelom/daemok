# daemok mcp

> [개요](../../design/00-overview.md)의 구조 절에 "MCP 서버는 같은 바이너리에 나중에 붙인다"고 정한 것을 구현합니다. AI가 설계를 읽고 검사를 돌려 코드가 설계에서 벗어나는지 스스로 확인하게 합니다.

## 명령

`daemok mcp [--design DIR] [--root DIR]`

표준 입력과 표준 출력으로 MCP(JSON-RPC 2.0, 한 줄에 메시지 하나)를 주고받습니다. 기본 경로는 다른 명령과 같습니다. 도구 인자로 경로를 바꿀 수 있습니다.

## 메서드

| 메서드 | 응답 |
|---|---|
| `initialize` | `protocolVersion`(요청의 값, 없으면 `2025-06-18`), `capabilities.tools`, `serverInfo`(`daemok`, 버전) |
| `notifications/initialized` 등 알림 | 응답 없음 |
| `ping` | 빈 결과 |
| `tools/list` | 아래 도구 |
| `tools/call` | 도구 결과(`content`의 text 하나). 도구 실행 실패는 `isError: true` |
| 그 밖 | JSON-RPC 오류 -32601 |

알 수 없는 도구와 잘못된 인자는 JSON-RPC 오류 -32602입니다.

## 도구

모두 읽기 전용입니다. 파일 쓰기는 CLI와 화면만 합니다.

| 도구 | 인자 | 결과 |
|---|---|---|
| `design` | `module`(선택), `design`, `root` | `module`이 없으면 설계 파일 원문(파일 이름과 내용). 있으면 그 모듈·그룹·멤버의 tier, 경로, summary, `depends_on`, 그걸 쓰는 쪽, 소속 L3·L4 노드 ID를 JSON으로 |
| `check` | `design`, `root` | `daemok check --format json`과 같은 JSON |
| `draft` | `root` | `daemok init --dry-run`과 같은 초안 YAML과 분석 메모 JSON |

## 테스트

| 대상 | 확인 |
|---|---|
| 프로토콜 | initialize, initialized 알림에 응답 없음, ping, tools/list, 없는 메서드 -32601, 없는 도구 -32602, 깨진 JSON -32700 |
| 도구 | 임시 프로젝트에서 세 도구의 결과 |
| 명령 | `daemok mcp`가 표준 입력이 끝나면 0으로 끝남 |
