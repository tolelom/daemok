# C++ include 검사

> C++ 프로젝트의 `#include`를 읽어 모듈 사이 의존이 설계의 `depends_on`을 따르는지 검사합니다. 생성기는 만들지 않습니다.

## 목적

[개요](../../design/00-overview.md)의 「다른 언어」 절은 언어 하나를 지원하는 데 생성기와 import 스캐너가 필요하다고 정했습니다. UE5 C++ 프로젝트(claw-machine-shop-simulator)를 측정하니 UE 모듈은 하나이고, 도메인 폴더(`Core`, `UI`, `Player` 등 14개) 사이 include 간선이 74개, 서로 include하는 폴더 짝이 20개였습니다. 생성할 대상은 정해지지 않았고 경계 검사는 바로 쓸 데가 있어, 스캐너만 먼저 만듭니다.

## 설계 형식

최상위 키 `cpp`를 더합니다.

```yaml
cpp:
  include_dirs:
    - unreal/Source/Game/Public
    - unreal/Source/Game/Private
modules:
  core: { tier: base, path: Core }
  ui: { tier: view, path: UI, depends_on: [core] }
```

- `include_dirs`는 코드 루트 기준 상대 경로 목록이고 비어 있으면 안 됩니다. `..`로 코드 루트를 벗어나거나 절대 경로면 오류입니다.
- `cpp`가 있으면 모듈의 `path`와 그룹 멤버 경로는 각 include 디렉토리 아래 폴더입니다. 위 `core`는 `Public/Core`와 `Private/Core`를 함께 가집니다.
- 모듈 폴더 아래 하위 폴더도 그 모듈입니다. 한 파일이 여러 모듈 폴더 아래에 있으면 가장 깊은 폴더의 모듈입니다.
- `cpp`는 여러 파일 중 한 곳에만 둡니다.
- path 요소 규칙은 지금과 같습니다.

## 검사

`daemok check`의 코드 대조 단계에서 `cpp`가 있을 때만 돕니다. Go import 검사는 그대로 돌고, Go 파일이 없으면 아무것도 내지 않습니다.

| 단계 | 규칙 |
|---|---|
| 파일 | 모듈·멤버 폴더 아래의 `.h`, `.hh`, `.hpp`, `.hxx`, `.inl`, `.c`, `.cc`, `.cpp`, `.cxx`. 심볼릭 링크는 따라가지 않음 |
| 읽기 | 주석(`//`, `/* */`)을 걷어 낸 뒤 줄 머리의 `#include "x"`, `#include <x>`. 문자열·문자 리터럴 안의 주석 표시는 주석이 아님. `#if` 분기는 보지 않고 모두 의존으로 셈 |
| 해석 | `"x"`는 포함한 파일의 폴더, 그다음 `include_dirs` 순서. `<x>`는 `include_dirs`만. 디스크에 있는 일반 파일로 처음 찾은 것. 찾지 못하거나 코드 루트 밖이면 외부 헤더로 보고 넘김 |
| 판정 | 찾은 파일이 다른 모듈·멤버에 속하고 `depends_on`이 허용하지 않으면 오류 `설계에 없는 의존을 include한다: a → b`. 어느 모듈에도 속하지 않는 코드 루트 안 파일은 대상 파일마다 한 번 정보 `관리하지 않는 파일을 include한다: a → 경로` |

- 진단 위치는 `#include` 줄의 파일, 줄, 열입니다.
- include로 이어진 대상은 Go import와 같이 `depends_on` 사용 여부에 셉니다. 쓰지 않는 `depends_on`은 지금처럼 정보로 알립니다.
- include 디렉토리가 디스크에 없으면 오류 `include 디렉토리가 없다: 경로`입니다.
- 모듈 폴더를 읽을 수 없으면 오류로 알리고 다음 폴더를 읽습니다.

## 설계 변경 요약

`daemok diff`는 `cpp`를 설정으로 비교합니다.

## 하지 않는 것

- C++ 코드 생성과 `static_assert` 확인 구문
- `*.Build.cs`의 UE 모듈 의존 검사
- C++ 저장소용 `daemok init` 초안
- `#if` 분기 해석, 매크로로 만든 include 경로

## 테스트

| 대상 | 확인 |
|---|---|
| 형식 | `cpp` 읽기, 빈 `include_dirs`, 절대 경로, `..` 경로, 모르는 키, 두 파일에 `cpp` |
| 읽기 | 주석 안 include, 문자열 안 `//` 뒤 include 줄, `# include` 공백, `<x>` |
| 해석 | 같은 폴더 우선, include 디렉토리 순서, 없는 헤더는 외부, Public·Private 두 폴더가 한 모듈, 하위 폴더, 가장 깊은 모듈 |
| 판정 | 허용 의존은 진단 없음, 허용하지 않은 의존은 오류와 위치, 관리하지 않는 파일 정보는 한 번, 쓰지 않는 `depends_on` 정보, 없는 include 디렉토리 오류 |
| 실사례 | claw-machine-shop-simulator 체크아웃에 의도한 계층의 임시 설계로 `daemok check`를 돌려 진단 수와 내용 확인. 설계 파일은 어느 저장소에도 커밋하지 않음 |
