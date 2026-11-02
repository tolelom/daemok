# daemok mcp

> [개요](../design/00-overview.md)의 구조 절에 "MCP 서버는 같은 바이너리에 나중에 붙인다"고 정한 것을 구현합니다. AI가 설계를 읽고 검사를 돌려 코드가 설계에서 벗어나는지 스스로 확인하게 합니다.

## 명령

`daemok mcp [--design DIR] [--root DIR]`

표준 입력과 표준 출력으로 MCP(JSON-RPC 2.0, 한 줄에 메시지 하나)를 주고받습니다. 기본 경로는 다른 명령과 같습니다. 도구 인자로 경로를 바꿀 수 있습니다.

## 메서드

| 메서드 | 응답 |
|---|---|
| `initialize` | `protocolVersion`(요청한 값이 지원하는 버전 `2025-06-18`, `2025-03-26`, `2024-11-05` 중 하나면 그 값, 아니면 `2025-06-18`), `capabilities.tools`, `serverInfo`(`daemok`, 버전) |
| `notifications/initialized` 등 알림 | 응답 없음 |
| `ping` | 빈 결과 |
| `tools/list` | 아래 도구 |
| `tools/call` | 도구 결과(`content`의 text 하나). 도구 실행 실패는 `isError: true` |
| 그 밖 | JSON-RPC 오류 -32601 |

알 수 없는 도구와 잘못된 인자는 JSON-RPC 오류 -32602입니다. 배치(JSON 배열)는 받지 않고 -32600을 줍니다. 16MB를 넘는 줄은 그 줄만 -32700으로 거부하고 다음 줄을 계속 읽습니다.

## 도구

`design`, `check`, `draft`, `gen`, `diff`는 읽기 전용이고 `annotations.readOnlyHint`를 붙입니다. 설계를 고치는 `edit`은 서버를 `--allow-write`로 띄웠을 때만 목록에 나옵니다.

도구 인자의 `design`, `root`는 제한하지 않습니다. 서버는 클라이언트를 띄운 사용자의 권한으로 로컬에서 돌고, 클라이언트가 여러 프로젝트를 오가며 경로를 넘기기 때문입니다. 그래서 클라이언트가 넘긴 디렉토리의 `*.yaml` 원문을 그대로 돌려줄 수 있습니다.

| 도구 | 인자 | 결과 |
|---|---|---|
| `design` | `module`(선택), `design`, `root` | `module`이 없으면 설계 파일 원문(파일 이름과 내용). 있으면 그 모듈·그룹·멤버의 tier, 경로, summary, `depends_on`, 그걸 쓰는 쪽, 소속 L3·L4 노드 ID를 JSON으로 |
| `check` | `design`, `root` | `daemok check --format json`과 같은 JSON |
| `draft` | `root`, `design`, `l3` | `daemok init --dry-run`과 같은 초안 YAML과 분석 메모 JSON. `l3`가 참이면 `daemok init --l3`와 같은 L3 초안 YAML과 옮기지 않은 선언 JSON |
| `diff` | `base`(기본 `HEAD`), `design`, `root` | `daemok diff`와 같은 설계 변경 요약 마크다운([설계 변경 요약 스펙](2026-10-14-daemok-diff-design.md)) |
| `edit` | `op`, `id`, `value`, `kind`, `code` | 화면 편집과 같은 작업(`tier`, `add_dep`, `remove_dep`, `add_module`, `remove_module`)과 이름 바꾸기(`rename`, [이름 바꾸기 스펙](2026-10-14-daemok-rename-design.md))를 서버의 `--design`에 적용하고 고친 파일 경로를 줌 |

`edit`의 확인은 세 겹입니다. 사용자가 `--allow-write`로 등록해야 도구가 보이고, `readOnlyHint: false`와 `destructiveHint: true`를 붙여 클라이언트가 호출마다 사용자에게 묻게 하며, 고친 내용은 커밋 전까지 지도에 HEAD 대비 변경으로 보입니다. `design`·`root` 인자를 주면 거부하고 서버의 설계 디렉토리만 고칩니다. 화면 편집과 달리, 설계 파일이 심볼릭 링크로 설계 디렉토리 밖을 가리키면 쓰지 않습니다. 호출마다 묻는지는 클라이언트 설정에 달려 있어 서버가 경로를 막습니다. 고친 결과가 새 읽기 오류를 내면 쓰지 않는 규칙은 화면 편집과 같습니다.

## 테스트

| 대상 | 확인 |
|---|---|
| 프로토콜 | initialize와 버전 협상, initialized 알림에 응답 없음, ping, tools/list, 없는 메서드 -32601, 없는 도구 -32602, 깨진 JSON -32700, 배치 -32600, 너무 긴 줄 뒤에도 계속 응답 |
| 도구 | 임시 프로젝트에서 세 도구의 결과 |
| 명령 | `daemok mcp`가 표준 입력이 끝나면 0으로 끝남 |
