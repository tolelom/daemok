# daemok

설계를 유일한 원본으로 두고, 탑다운으로 한 계층씩 시각적으로 확인하며 확정하고, 코드는 설계에서 한 방향으로 생성하는 도구.

이름은 대목(大木)에서 왔다. 한옥에서 설계부터 시공까지 맡는 목수를 가리킨다.

설계 문서는 [docs/design](docs/design/)에, 설계 파일 형식은 [모델](docs/design/01-model.md)에 있다.

## 설치

```
go install github.com/tolelom/daemok/cmd/daemok@latest
```

Go 1.27 이상. 외부 의존성은 없다.

## 명령

설계 디렉토리는 `design/`(`*.yaml`), 코드 루트는 현재 디렉토리가 기본이다.

| 명령 | 하는 일 | 주요 옵션 |
|---|---|---|
| `daemok check` | 설계 검사, import 대조, 생성 파일 최신 여부를 출력한다 | `--design DIR`, `--root DIR`, `--strict`(경고도 실패) |
| `daemok gen` | 설계에서 Go 생성 파일을 쓰고, 설계에 없는 생성 파일을 지운다 | `--design DIR`, `--root DIR`, `--dry-run` |
| `daemok init` | 설계 파일이 없는 Go 모듈의 코드에서 L2 초안(모듈, tier, 의존)을 `design/`에 쓴다 | `--root DIR`, `--design DIR`(기본 `<root>/design`) |
| `daemok serve` | 설계 지도를 로컬 웹 화면으로 띄운다 | `--design DIR`, `--root DIR`, `--addr HOST:PORT`(기본 `127.0.0.1:7412`), `--workspace DIR` |

종료 코드는 0 통과, 1 문제 있음, 2 진행할 수 없음이다.

## 화면

`daemok serve`가 출력한 주소를 브라우저로 연다.

- 지도, 모듈, 흐름, 요구사항 탭으로 설계를 본다. 설계 파일을 저장하면 다시 그려지고, git HEAD 대비 바뀐 노드가 표시된다.
- 머리의 프로젝트 이름을 누르면 다른 프로젝트를 연다. 경로를 입력하거나, 최근 목록이나 `--workspace` 아래의 Go 모듈 목록에서 고른다.
- 설계 파일이 없는 Go 모듈을 열면 분석 모드다. 코드에서 추정한 초안을 지도로 보여 주고, 초안 탭에서 `design/`에 저장한다.
- 프로젝트 열기와 초안 저장은 루프백 주소로 띄웠을 때만 된다.

## 기존 프로젝트에 도입

1. `daemok init`으로 초안을 만든다. tier 이름과 그룹은 사람이 정한다.
2. `daemok check`로 설계와 코드가 맞는지 본다.
3. 생성할 노드를 설계에 더하고 `daemok gen`으로 생성 파일을 쓴다. 이미 손으로 쓴 선언이 있는 노드는 `gen: false`로 둔다.

tolchain에 도입한 기록은 [tolchain 슬라이스](docs/design/02-slice-tolchain.md)에 있다.
