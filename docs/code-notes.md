# 코드 노트

운영 코드 주석에 있던 설명을 파일·선언별로 모았다. 운영 코드(_test.go가 아닌 .go)에는 주석을 두지 않는다(//go:embed 같은 지시문은 예외). 설계 메모는 여기에 둔다. 2026-10-20에 원문 그대로 옮겼으므로 그 시점의 코드를 설명한다.

## cmd/daemok/check.go

- `printDiags`: printDiags는 진단을 한 줄씩 쓴다. 같은 수준·같은 설명("설명: 대상"의 설명)이 3개 이상이면 머리 한 줄 아래에 위치와 대상만 들여 쓴다.

## cmd/daemok/coverage.go

- `runCoverage`: runCoverage는 패키지마다 최상위 선언 중 설계 노드에 대응하는 것의 비율과, 함수를 뺀 비율을 출력한다. 설계를 읽지 못하거나 설계 검사 오류가 있으면 2, 파싱하지 못한 Go 파일이 있으면 표를 내고 1이다.

## cmd/daemok/diff.go

- `runDiff`: runDiff는 기준 ref 대비 설계 변경을 PR 설명용 마크다운으로 출력한다. 기준을 읽지 못하면 2다.

## cmd/daemok/gen.go

- `runGen`: runGen은 생성 파일을 쓰고 지운다. 설계를 읽지 못하거나 검사 오류가 있으면 2, 생성 경로에 헤더 없는 파일이 있으면 아무것도 쓰지 않고 1이다.

## cmd/daemok/init.go

- `runInit`: runInit은 설계 파일이 없는 Go 모듈의 코드에서 초안을 만들어 설계 디렉토리에 쓴다. 이미 설계 파일이 있으면 쓰지 않고 1, Go 모듈이 아니거나 쓰지 못하면 2다. --dry-run은 쓰지 않고 출력만 한다. --l3는 설계가 있는 저장소에서 L3 초안을 stdout에, 옮기지 않은 선언을 stderr에 낸다. 쓰지 않는다.

## cmd/daemok/main.go

- `package main`: daemok은 설계 파일을 읽어 검사하고 코드를 생성하는 CLI다.

## cmd/daemok/mcp.go

- `runMCP`: runMCP는 표준 입출력으로 MCP 서버를 돌린다. 입력이 끝나면 0이다.

## cmd/daemok/paths.go

- `pathFlags`: pathFlags는 모든 명령이 같은 뜻으로 쓰는 --design과 --root를 등록한다.
- `parse`: parse는 플래그를 읽고, 플래그가 아닌 인자가 남으면 거부한다. flag는 첫 위치 인자에서 멈추므로 남은 인자를 두면 그 뒤의 플래그까지 조용히 버려진다.
- `designDir`: designDir는 --design이 비어 있으면 코드 루트 아래 design이다.

## cmd/daemok/rename.go

- `runRename`: runRename은 설계 노드 하나의 ID와 그 참조를 바꾼다. 기본은 코드 이름을 고정 키로 유지한다. 인자 오류는 2, 거부는 1이다.
- `runRename`: flag는 첫 위치 인자에서 멈추므로 위치 인자를 하나씩 떼며 나머지 플래그를 계속 읽는다.

## cmd/daemok/serve.go

- `runServe`: runServe는 설계 지도를 로컬 웹 페이지로 띄운다. 인터럽트를 받으면 0으로 끝난다.

## cmd/daemok/version.go

- `runVersion`: runVersion은 모듈 버전과, 저장소에서 빌드했으면 커밋을 출력한다.

## internal/check/check.go

- `package check`: Package check는 모델만 보고 설계 검사를 한다. 코드는 읽지 않는다.
- `checker.satisfiers`: 요구사항 ID → 충족한다고 적은 요소
- `Run`: Run은 설계 검사 전부를 돌려 위치순으로 정렬한 진단을 준다.
- `checker.eachMeta`: eachMeta는 satisfies를 가질 수 있는 모든 노드를 돈다.
- `checker.depsOf`: depsOf는 id가 설계상 의존하는 ID 목록이다. 멤버는 그룹의 의존과 자기 예외를 따른다.
- `checker.layerDepsOf`: layerDepsOf는 depsOf에서 query needs를 뺀 것이다. 순환 검사가 쓴다.
- `checker.ownNeeds`: ownNeeds는 모듈·그룹·멤버가 자기 노드에 적은 needs다.
- `checker.needsOf`: needsOf는 id가 받는 계약이다. 멤버는 그룹의 needs와 자기 needs를 받는다.
- `checker.layerTargets`: needTargets는 needs가 가리키는 계약의 모듈을 참조 위치와 함께 준다. layerTargets는 tier 방향과 순환에 넣는 needs 대상이다. query 구현은 data 모듈만 import하는 끝 패키지라 다시 부르는 쪽이 없으므로 query 계약은 뺀다.
- `checker.unitIDs`: unitIDs는 모듈, 그룹, 멤버 ID 전부다.
- `checker.unitPos`: unitPos는 모듈·그룹·멤버 ID의 정의 위치다.

## internal/check/events.go

- `checker.events`: events는 key enum 값마다 이벤트가 있는지, 발행자·구독자가 이벤트 종류의 모듈에 의존하는지 본다.
- `checker.events`: key enum → 처음 쓴 이벤트 종류
- `checker.dependsOnKind`: dependsOnKind는 발행자·구독자가 이벤트 종류의 모듈에 의존하는지 본다. 없는 모듈은 참조 검사가 이미 잡으므로 넘긴다.

## internal/check/flows.go

- `checker.flows`: flows는 단계 이름, commit 위치, 흐름의 모듈과 calls·emits·on의 관계를 본다.
- `checker.flows`: Go 메서드 이름 → 처음 쓴 단계·undo 이름
- `checker.usesMethod`: usesMethod는 id가 받은 계약에서 메서드를 쓸 수 있는지다. uses를 생략한 needs가 하나라도 있으면 전부 쓸 수 있다.
- `checker.receives`: receives는 id가 needs로 계약을 받는지다.

## internal/check/handlers.go

- `checker.handlers`: handlers는 핸들러 종류의 키 enum 값마다 처리하는 멤버가 정확히 하나인지 본다.

## internal/check/l0.go

- `checker.l0`: l0은 인터페이스마다 구현 모듈이 있고, 노출하는 실행 단위가 그 모듈을 조립하는지 본다.
- `checker.reachable`: reachable은 composes에서 의존을 따라 닿는 모듈·그룹·멤버 ID 집합이다. 그룹은 멤버를 포함한다.

## internal/check/l2.go

- `checker.tierDirection`: tierDirection은 자기보다 위 tier에 의존하면 오류다. 같은 tier는 허용한다. query needs는 보지 않는다.
- `checker.selfDependency`: selfDependency는 자기 자신을 가리키는 의존을 오류로 낸다. 멤버가 자기 그룹에 의존하는 것도 포함한다. cycles는 그룹으로 접은 뒤 자기 간선을 버리므로 여기서 따로 잡는다.
- `checker.cycles`: cycles는 모듈과 그룹의 의존 그래프에서 순환을 찾는다. 순환마다 한 번 보고한다. 멤버는 그룹으로 접는다: 멤버를 가리키는 의존은 그 그룹으로, 멤버의 예외는 그룹의 의존으로 본다. 그룹에 의존하면 멤버에도 의존할 수 있으므로 멤버 예외를 거치는 순환도 여기서 잡힌다. 형제 멤버 사이의 순환은 siblingCycles가 잡는다.
- `checker.cycles`: 형제 멤버로 가는 예외는 그룹 자신으로 접히므로 뺀다.
- `checker.siblingCycles`: siblingCycles는 그룹마다 형제 멤버를 가리키는 예외만으로 순환을 찾는다.
- `checker.findCycles`: findCycles는 ids 안에서 edges를 따라 DFS로 순환을 찾아 순환마다 한 번 보고한다.
- `checker.orphans`: orphans는 어느 실행 단위에서도 닿지 않는 모듈·그룹을 경고한다. 실행 단위가 없으면 건너뛴다.
- `checker.exceptions`: exceptions는 멤버 예외를 정보로 남기고, 그룹이 이미 의존하는 대상이면 경고한다.
- `checker.contractRoles`: contractRoles는 port·external 계약이 알맞은 role의 모듈에 있는지 본다.
- `checker.contractRoles`: query 구현은 <모듈 path>/query에 두므로 path가 이미 /query로 끝나면 query/query가 된다.

## internal/check/l3.go

- `checker.typeDeps`: typeDeps는 노드가 쓰는 enum·타입의 모듈에 그 노드의 모듈이 의존하는지 본다.
- `checker.typeDeps`: needs로 받은 계약의 시그니처 타입에도 받는 쪽이 의존해야 한다.
- `checker.useNeededType`: useNeededType은 받은 계약의 시그니처 타입에 받는 쪽이 의존하는지 본다. 위치는 받는 쪽의 needs 항목이다.
- `checker.typeOwner`: typeOwner는 r이 설계 enum·struct거나 "모듈.이름" 외부 타입이면 그 모듈과 이름을 준다. map은 값, 키 순으로 본다.
- `checker.typeOwner`: 값 타입이 먼저다. 키는 enum만 모듈에 속한다.
- `checker.useType`: useType은 owner가 r이 가리키는 enum·타입의 모듈에 의존하는지 본다.
- `checker.errorHTTP`: errorHTTP는 에러 이름의 접미사가 뜻하는 HTTP 상태와 http 값이 다르면 경고한다.

## internal/check/l4.go

- `checker.l4`: l4는 함수 계약이 소속 멤버·계약과 맞고, 케이스의 에러가 errors와 맞는지 본다.
- `checker.l4`: gen: false면 케이스 테스트를 만들지 않는다

## internal/check/names.go

- `checker.names`: names는 생성 코드가 컴파일되도록 Go 이름과 경로의 규칙을 본다.
- `checker.goModule`: goModule은 생성 코드가 다른 패키지를 import할 수 있는 설계에 go_module이 있는지 본다.
- `checker.cppDirs`: cppDirs는 include 디렉토리가 있고 코드 루트 안의 상대 경로인지 본다.
- `checker.paths`: paths는 모듈·그룹 path와 멤버 ID가 Go 패키지 경로로 쓸 수 있고 생성 파일 범위 안인지 본다.
- `checker.nodeIDs`: nodeIDs는 Go 이름과 파일 이름이 되는 ID가 소문자 snake_case인지, 타입·enum ID가 내장 타입 이름과 겹치지 않는지 본다.
- `checker.enumValues`: enumValues는 enum 값이 Go 상수 이름이 되는지, enum 안에서 상수 이름이 겹치지 않는지 본다.
- `checker.errorIDs`: errorIDs는 에러의 Go 이름이 export되는 이름인지 본다. go.name이 없으면 ID가 Go 이름이다.
- `checker.funcNames`: funcNames는 일반 함수의 go.name이 export되는 이름인지 본다.
- `checker.fieldNames`: fieldNames는 struct와 이벤트 data의 필드가 Go 필드 이름이 되고 겹치지 않는지 본다.
- `checker.packageNames`: packageNames는 같은 패키지(모듈·멤버)에 생성되는 최상위 이름이 겹치지 않는지 본다.
- `checker.packageNames`: 이름 → 처음 만든 노드
- `checker.packageNames`: 그룹의 needs는 멤버 패키지에 생긴다

## internal/check/refs.go

- `checker.refs`: refs는 모든 참조 필드가 존재하는, 받을 수 있는 종류의 대상을 가리키는지 본다.
- `checker.pubSub`: pubSub은 발행자·구독자가 모듈이나 그룹 멤버인지 본다.
- `checker.handleValue`: handleValue는 handles 값이 핸들러 종류의 키 enum에 있는지 본다. 핸들러 종류나 enum이 없으면 다른 곳에서 이미 오류이므로 조용히 넘긴다.

## internal/check/req.go

- `checker.requirements`: requirements는 active 요구사항에 충족 요소가 있는지, held·dropped 요구사항에 요소가 매달려 있지 않은지 본다.

## internal/check/ts.go

- `checker.ts`: ts는 export 대상, ts_out 형식, TS로 내보낼 수 없는 상수와 타입 필드, TS 파일 안의 이름 충돌을 본다.

## internal/codescan/cpp.go

- `include`: include는 소스의 #include 한 줄이다. 열은 #의 바이트 위치다.
- `includes`: includes는 주석을 걷어 낸 소스에서 줄 머리의 #include를 찾는다. 문자열·문자 리터럴 안의 주석 표시는 주석으로 보지 않는다. 리터럴은 줄 끝에서 끝난다고 본다.
- `stripComments`: stripComments는 주석 글자를 공백으로 바꾼다. 줄바꿈은 남겨 줄·열이 그대로다.
- `stripComments`: 영숫자 뒤의 '는 숫자 구분자(1'000)로 본다. u'x' 같은 접두사 붙은 문자 리터럴은 리터럴로 보지 않는다
- `scanner.cppScan`: cppScan은 모듈·멤버 폴더의 C++ 파일에서 include를 읽어 depends_on과 대조한다. 모듈 path는 각 include 디렉토리 아래 폴더이고, 하위 폴더도 그 모듈이다. 겹치면 가장 깊은 폴더가 이긴다.
- `scanner.cppScan`: 폴더 → 모듈·멤버 ID
- `continued`: continued는 i의 줄바꿈 앞이 줄 잇기 \인지다.

## internal/codescan/deps.go

- `scanner.needsOf`: needsOf는 id가 받는 계약이다. 멤버는 그룹의 needs도 받는다.
- `scanner.receivesFrom`: receivesFrom은 id가 target 모듈의 계약을 needs로 받는지다.
- `scanner.queryPackages`: queryPackages는 query 계약을 가진 모듈의 query 패키지가 data 모듈 말고 다른 설계 모듈을 import하는지 본다.
- `scanner.unusedDeps`: unusedDeps는 모듈의 depends_on 대상을 import하지 않으면 정보로 알린다. 그룹 대상은 멤버 하나를 import하면 된다. 코드 디렉토리가 없는 모듈은 보지 않는다. cpp가 있으면 include 디렉토리 아래 폴더도 코드 디렉토리다.

## internal/codescan/events.go

- `scanner.collectRefs`: collectRefs는 이벤트 종류 모듈의 이름을 쓴 식별자를 모은다. 다른 패키지에서는 import 이름(별칭 포함)의 선택자, 이벤트 종류 모듈 안에서는 맨 식별자다.
- `scanner.collectRefs`: 이 파일에서 이벤트 종류 모듈을 가리키는 이름
- `scanner.collectRefs`: 이벤트 종류 모듈 안이거나 그 모듈을 도트 import하면 맨 식별자를 센다
- `scanner.eventRefs`: eventRefs는 발행자·구독자가 이벤트 상수를 쓰는지, 적히지 않은 모듈이 쓰는지 본다.

## internal/codescan/handwritten.go

- `scanner.handWritten`: handWritten은 gen: false 노드가 생성 코드 대신 둔다고 한 선언이 소유 모듈 패키지에 있는지 본다. 손으로 쓴 코드에 생성 코드와 같은 이름이 있다고 볼 수 있는 노드만 본다. 이벤트 종류, 흐름, 함수는 보지 않는다. enum은 타입과 값 상수만 보고 목록 함수는 보지 않는다. 코드 디렉토리가 없는 모듈은 보지 않는다.
- `scanner.handWritten`: 일반 함수의 본문은 gen과 관계없이 손으로 쓴다

## internal/codescan/literals.go

- `adHocCall`: adHocCall은 즉석 errors.New 호출 하나다. msg는 인자가 문자열 리터럴일 때만 있다.
- `scanner.adHocErrors`: adHocErrors는 쓸 수 있는 설계 에러가 있는 모듈의 함수 본문에서 errors.New를 부르면 정보로 알린다. 상황마다 메시지가 다른 정당한 경우가 섞여 있어 경고(--strict 실패)로 두지 않는다. 패키지 수준 var 선언(설계 에러를 손으로 둔 gen: false 선언)과 fmt.Errorf의 감싸기는 보지 않는다.
- `scanner.reportAdHoc`: reportAdHoc은 즉석 errors.New를 메시지와 함께 알린다. 쓸 수 있는 설계 에러와 메시지가 같으면 그 에러를, 같은 메시지가 여러 곳이면 그 수를 붙인다. 앞은 바로 바꿔 쓸 후보이고 뒤는 sentinel로 둘 후보다.
- `scanner.enumLiterals`: enumLiterals는 쓸 수 있는 설계 enum의 값과 같은 문자열 리터럴을 함수 본문의 비교(==, !=), switch case, SQL 메서드의 자리표시자 값(placeholderArgs)에서 쓰면 정보로 알린다. enum 상수를 쓰면 값이 바뀔 때 컴파일러가 잡는다. 로그 키나 외부 프로토콜 문자열처럼 다른 자리의 같은 문자열은 enum과 상관없을 때가 많아 보지 않는다.
- `scanner.enumLiterals`: 값 → 상수 이름
- `sqlMethods`: sqlMethods는 쿼리 문자열과 자리표시자 값을 함께 받는 GORM·database/sql 메서드다.
- `placeholderArgs`: placeholderArgs는 SQL 메서드 호출에서 ?가 든 문자열 리터럴 인자 뒤의 인자다. db.Where("status = ?", "published")의 값 자리다. IN ?에 넘긴 슬라이스 리터럴은 원소를 준다.

## internal/codescan/scan.go

- `package codescan`: Package codescan은 코드의 import와 이벤트 상수 참조를 설계와 대조한다.
- `scanner.units`: import 경로 → 모듈·멤버 ID
- `scanner.imports`: 모듈·멤버 ID → 그 패키지가 import한 설계 단위
- `scanner.refs`: 모듈·멤버 ID → 쓴 이벤트 상수와 목록 함수의 이름
- `scanner.decls`: 모듈·멤버 ID → 패키지 최상위 선언 이름
- `scanner.typeSpecs`: 모듈·멤버 ID → 타입 이름 → 선언과 그 파일
- `scanner.adHoc`: 즉석 errors.New 호출. 모두 모은 뒤 같은 메시지 수를 붙여 알린다.
- `unit`: unit은 코드 디렉토리를 가진 모듈이나 그룹 멤버다.
- `Scan`: Scan은 설계 검사를 통과한 모델의 모듈·멤버 코드를 읽어 진단을 준다.
- `scanner.goFiles`: goFiles는 모듈·멤버 디렉토리의 Go 파일이다. 파일 선택은 gosrc.Files를 따른다.
- `scanner.depend`: depend는 id가 target을 쓴 것을 기록하고 설계가 허용하지 않으면 알린다. verb는 import나 include다.

## internal/codescan/shape.go

- `matchName`: matchName은 설계 이름과 코드 이름을 대조하는 키다. 생성 이름은 AvatarUrl, 손으로 쓴 이름은 AvatarURL일 수 있다.
- `typeExpr`: typeExpr는 코드의 타입식을 생성 코드와 같은 모양으로 쓴다. 설계 모듈 패키지는 import 경로의 마지막 요소로, 그 밖의 패키지는 코드에 쓴 이름 그대로 한정한다. 생성 코드에 나오지 않는 타입식(함수, 채널, 배열 길이, 가변 인자, 제네릭 등)이면 false다.
- `importBase`: importBase는 파일에서 패키지를 가리키는 이름이 설계 모듈(go_module 안) 패키지나 생성기가 쓰는 표준 라이브러리면 그 import 경로의 마지막 요소로 바꾼다. 생성기가 그 이름으로 한정하기 때문이다. 별칭 없는 import의 패키지 이름은 경로의 마지막 요소로 본다. go_module 밖 패키지는 설계에 코드와 같은 이름으로 적으므로 그대로 둔다.
- `fieldTypes`: fieldTypes는 파라미터·반환 목록을 타입 하나씩 편다. a, b int는 int 두 개다.
- `scanner.shapes`: shapes는 gen: false인 call 계약과 타입의 코드 선언이 생성했다면 나왔을 모양과 같은지 본다. 선언이 없는 경우는 handWritten이 알린다. 타입이 아닌 선언(함수, 변수, 상수)이면 오류다.
- `jsonTag`: jsonTag는 struct 필드 태그의 json 부분을 `json:"값"`으로 준다. 없으면 "태그 없음"이다.
- `scanner.uncomparable`: uncomparable은 타입 별칭과 제네릭 선언이면 정보로 알리고 true를 준다. 생성 코드에 없는 모양이다.
- `scanner.typeShape`: json은 태그의 json 값. 태그가 없으면 "태그 없음"

## internal/coverage/coverage.go

- `package coverage`: Package coverage는 패키지마다 최상위 선언 중 설계 노드에 대응하는 것의 수를 센다. 이 비율은 설계 형식을 넓힐지 정하는 근거다.
- `Row`: Row는 패키지 하나의 결과다. Pkg는 코드 루트 기준 슬래시 경로이고 루트는 "."다. Funcs와 FuncsMatched는 Decls·Matched 중 메서드가 아닌 함수다. 설계는 에러 계약이 있는 함수만 두므로 함수를 뺀 비율을 따로 볼 수 있게 한다.
- `Count`: Count는 root 아래 패키지마다 최상위 선언 수와 그중 설계가 만드는 이름의 수를 준다. 설계가 만드는 이름은 모든 L3·L4 노드를 gen: true로 보고 생성한 코드에서 얻는다. 모델은 바꾸지 않는다. 파싱하지 못한 Go 파일은 세지 않고 함께 돌려준다.
- `DesignNames`: designNames는 패키지 디렉토리마다 설계가 만드는 최상위 이름이다. 등록 파일이 가리키는 핸들러 함수와 일반 함수도 넣는다.
- `DesignNames`: 일반 함수는 생성물이 없고 손으로 쓴 선언을 가리킨다
- `handlerRefs`: handlerRefs는 등록 파일의 `_ 패키지.종류 = 함수` 줄이 가리키는 손으로 쓴 함수 이름이다.

## internal/designdiff/collect.go

- `package designdiff`: Package designdiff는 두 설계 파일 묶음을 노드와 키 단위로 비교해 변경 요약을 만든다. 비교는 YAML 값으로 해서 주석, 정렬, 형식, 노드가 든 파일은 변경으로 보지 않는다.
- `kinds`: kinds는 노드 종류와 출력 순서다. 설계 파일의 키 순서를 따른다.
- `settings`: settings는 노드가 아닌 최상위 설정 키다.
- `node.group`: 멤버의 소속 그룹
- `design`: design은 설계 하나의 노드(종류 → ID → 값)와 설정이다.
- `collect`: collect는 파일들을 이름순으로 읽어 노드를 모은다. 같은 종류·ID가 두 번 나오면 어느 쪽 변경을 볼지 정할 수 없어 오류다.
- `without`: without은 맵 n에서 keys를 뺀 사본이다. 맵이 아니면 그대로 준다.

## internal/designdiff/compare.go

- `Modified`: Modified는 바뀐 노드와 키 경로별 변경 줄이다.
- `Report`: Report는 기준 대비 작업 설계의 변경이다. 각 목록은 종류 순서, 같은 종류 안에서는 ID 순이다.
- `Report.Empty`: Empty는 변경이 없는지다.
- `pairKeys`: pairKeys는 맵 목록의 원소를 짝지을 키다. 키 경로의 마지막 이름으로 찾는다.
- `orderedMaps`: orderedMaps는 키 순서에 뜻이 있는 맵이다. 생성 struct 필드와 함수 파라미터 순서가 된다.
- `pinKeys`: pinKeys는 daemok rename이 종류마다 더하는 코드 이름 고정 키다.
- `pinValue`: pinValue는 daemok rename 기본 모드가 old 노드에 더하는 고정 값이다. 경로는 ID, 에러는 ID, 그 밖은 Go 이름이다.
- `Compare`: Compare는 기준 설계 파일들과 작업 설계 파일들을 비교한다. 어느 파일이든 YAML로 읽지 못하거나 같은 종류·ID가 두 곳에 있으면 오류다.
- `Compare`: 새 ID → 옛 ID
- `Compare`: 이름 바뀜 짝도 값 차이(고정 키 추가, 코드 이름 변경)는 새 ID의 수정으로 보인다.
- `pairRenames`: pairRenames는 삭제·추가 노드 가운데 값이 같은 짝이 서로 하나뿐인 것을 [옛, 새]로 준다. 새 노드에만 daemok rename이 더하는 고정 키가 옛 ID의 값으로 있으면 그 키는 빼고 비교한다. 빈 노드는 무엇과도 같아 짝짓지 않는다.
- `unpinned`: unpinned는 새 노드 값 nv에서, 옛 값 ov에 없던 고정 키가 old의 고정 값으로 있으면 그 키를 뺀다. 비게 된 go 맵도 뺀다.
- `norm`: norm은 디코더가 없는 것과 같게 읽는 값(null, 빈 목록, 빈 맵)을 nil로 바꾼다.
- `typed`: typed는 따옴표 없이 쓰면 문자열이 아닌 값(정수, 불리언, null)으로 읽히는 글인지다.
- `sameScalar`: sameScalar는 두 스칼라가 같은 값인지다. 따옴표는 따옴표를 떼면 문자열이 아닌 값이 될 때만 뜻이 있다.
- `diffValue`: diffValue는 path의 두 값의 차이를 줄로 준다. 같으면 nil이다. 노드 자체(path가 빈 값)는 "값"으로 적는다.
- `keyed`: keyed는 모든 원소가 key를 스칼라로 갖고 그 값이 겹치지 않는지다.
- `render`: render는 값을 한 줄로 쓴다. 따옴표가 뜻이 있거나 구분 글자·줄바꿈이 든 문자열은 이스케이프해 따옴표로 감싼다.

## internal/designdiff/markdown.go

- `code`: code는 s를 코드 스팬으로 감싼다. s에 백틱이 있으면 더 긴 백틱 묶음으로 감싼다.
- `Report.Markdown`: Markdown은 PR 설명에 붙일 변경 요약이다. base는 제목에 쓰는 기준 ref다.

## internal/designdiff/run.go

- `Run`: Run은 designDir의 base 시점 설계와 작업 폴더 설계를 비교해 마크다운을 준다. 기준을 읽지 못하거나 설계 파일을 YAML로 읽지 못하면 오류다.
- `Run`: Glob은 디렉토리 이름의 [ ] 를 패턴으로 읽으므로 목록을 직접 거른다.

## internal/edit/apply.go

- `ErrBreaksLoad`: ErrBreaksLoad는 고친 결과가 편집 전에 없던 설계 읽기 오류를 낸다는 뜻이다. 쓰지 않는다.
- `ErrIO`: ErrIO는 설계 파일을 읽거나 쓰지 못했다는 뜻이다.
- `Do`: Do는 설계 파일 하나의 내용에 작업 하나를 적용한다.
- `Ops`: Ops는 Do가 받는 작업 이름이다.
- `ErrOutsideDir`: ErrOutsideDir는 고칠 설계 파일이 심볼릭 링크로 설계 디렉토리 밖을 가리킨다는 뜻이다. 쓰지 않는다.
- `Apply`: Apply는 설계 디렉토리의 *.yaml 가운데 대상이 있는 파일에 작업을 적용해 쓰고 그 경로를 준다. 고친 결과가 편집 전에 없던 읽기 오류를 내면 쓰지 않는다. 설계 검사 오류는 막지 않는다. 설계 파일이 심볼릭 링크면 가리키는 파일을 고친다.
- `ApplyConfined`: ApplyConfined는 Apply와 같되, 설계 파일이 심볼릭 링크로 설계 디렉토리 밖을 가리키면 쓰지 않는다. 사람이 화면에서 누르지 않는 경로(MCP)가 설계 디렉토리 밖을 고치지 못하게 한다.
- `apply`: 다른 파일에 있을 수 있는 경우만 다음 파일을 본다.
- `apply`: 편집 전부터 있던 읽기 오류는 빼고, 편집이 새로 만든 오류만 본다.
- `loadErrors`: loadErrors는 설계 파일들을 읽을 때 나는 오류 메시지와 그 수다.
- `WriteAtomic`: WriteAtomic은 같은 디렉토리의 임시 파일에 쓴 뒤 이름을 바꾼다. 심볼릭 링크면 링크는 두고 가리키는 파일을 바꾼다.
- `renameRetry`: renameRetry는 Windows에서 다른 쪽(감시, 에디터, 백신)이 대상을 잠깐 열고 있어 이름 바꾸기가 실패하면 짧게 기다렸다 다시 한다.
- `Inside`: Inside는 path의 실제 위치가 dir의 실제 위치 아래인지 본다.

## internal/edit/edit.go

- `package edit`: Package edit는 설계 파일의 L2 모듈과 그룹을 줄 단위로 고친다. 파일 전체를 다시 쓰지 않고 파서가 남긴 노드 시작 위치에서 원본 텍스트를 읽어 해당 토큰이나 줄만 바꾼다. 주석과 정렬은 그대로다.
- `plainID`: plainID는 따옴표 없이 YAML에 그대로 쓸 수 있는 ID다. 모델은 모듈 ID를 snake_case로 강제하지 않는다.
- `ErrModuleNotInFile`: ErrModuleNotInFile은 고칠 모듈이 이 파일에 없다는 뜻이다. 다른 설계 파일에 있을 수 있다.
- `ErrNoModuleBlock`: ErrNoModuleBlock은 이 파일에 모듈을 더할 modules 블록이 없다는 뜻이다. 다른 설계 파일에 있을 수 있다.
- `SetTier`: SetTier는 모듈의 tier 값을 바꾼다.
- `AddDep`: AddDep는 모듈의 depends_on에 dep를 더한다. depends_on이 없으면 만든다.
- `RemoveDep`: RemoveDep는 모듈의 depends_on에서 dep를 지운다. 비면 []로 남긴다.
- `RemoveDep`: 항목 줄만 지우고 키 뒤에 []를 넣는다. 키 줄과 사이의 주석은 그대로 둔다.
- `AddModule`: AddModule은 modules 블록의 마지막 모듈 아래에 `id: { tier: T }` 줄을 더한다.
- `RemoveModule`: RemoveModule은 modules 블록에서 모듈 항목을 지운다. 키 줄부터 그 항목의 마지막 내용 줄까지다. 다른 곳에 남은 참조는 고치지 않는다. 설계 검사가 없는 참조로 알린다.
- `text.starts`: 줄마다 시작 바이트. BOM이 있으면 첫 줄은 BOM 뒤에서 시작한다(파서가 BOM을 떼고 열을 센다).
- `text.val`: 고칠 모듈이나 그룹의 값
- `open`: 모듈과 그룹은 depends_on이 함께 가리키는 이름공간이라 ID가 겹치지 않는다.
- `text.lineEnd`: lineEnd는 줄바꿈까지 포함한 줄의 끝이다.
- `text.flow`: flow는 노드가 { 나 [ 로 시작하는 플로우 형식인지다.
- `text.tokenEnd`: tokenEnd는 s에서 시작하는 스칼라의 끝이다. 따옴표 문자열은 닫는 따옴표 뒤다.
- `text.closing`: closing은 open 위치의 { 나 [ 와 짝이 맞는 닫는 괄호 위치다.
- `text.blockEnd`: blockEnd는 first 줄에서 시작하는 들여쓰기 indent 이상의 블록에서 마지막 내용 줄의 끝이다. 빈 줄과 주석 줄은 블록을 끝내지 않지만 내용으로 세지 않는다.
- `text.insertLine`: insertLine은 at에 한 줄을 넣는다. 파일 끝에 줄바꿈이 없으면 먼저 줄을 바꾼다. 줄바꿈은 넣는 자리 앞줄의 것을 따른다.
- `text.afterColon`: afterColon은 항목 키 뒤 콜론의 바로 다음 위치다.

## internal/edit/pin.go

- `ErrNodeNotInFile`: ErrNodeNotInFile은 고정 키를 더할 노드가 이 파일에 없다는 뜻이다.
- `ErrKeyExists`: ErrKeyExists는 더할 키가 노드에 이미 있다는 뜻이다.
- `Pin`: Pin은 path가 가리키는 맵 노드에 key: value(sub가 비면) 또는 key: { sub: value }를 더한다. key 맵이 이미 있으면 그 안에 sub: value를 더한다. 같은 키가 이미 있으면 거부한다. 흐름 형식이면 닫는 괄호 앞에, 블록 형식이면 노드의 마지막 내용 줄 뒤에 같은 들여쓰기로 넣는다.
- `Pin`: 블록 맵의 위치는 첫 항목 위치라 들여쓰기는 첫 항목 열에서 얻는다.

## internal/edit/replace.go

- `Replacement`: Replacement는 설계 파일 한 위치의 스칼라 토큰을 바꾸는 일이다. Old는 원문 토큰(따옴표면 안쪽)이다.
- `plainStop`: plainStop은 평문 토큰 뒤에 올 수 있는 글자다. 블록 형식 평문에는 [ ]가 들어갈 수 있어 토큰 끝을 원문에서 찾지 않고 Old 길이로 자른 뒤 다음 글자가 경계인지 본다.
- `Replace`: Replace는 위치마다 원문 토큰이 Old인지 확인하고 New로 바꾼다. 따옴표 스타일은 그대로 둔다. 이스케이프가 든 따옴표 토큰은 원문과 값이 달라 바꾸지 않는다. 뒤의 위치부터 바꿔 앞의 교체가 뒤 위치를 밀지 않는다.

## internal/gen/golang/cases.go

- `literal`: literal은 given·expect 노드를 Go 리터럴로 바꾼다. 맨 위 맵은 typ, 안쪽 맵은 map[string]any다.

## internal/gen/golang/events.go

- `numberKinds`: numberKinds는 JSON을 거친 값(float64)과 Go에서 직접 넣은 정수를 모두 받는 변환 도우미의 대상이다.
- `eventHelpers`: eventHelpers는 모듈 디렉토리별로 쓰인 변환 도우미다.
- `gen.kindEvents`: kindEvents는 이벤트 종류의 이벤트를 key enum 값 순서로 준다.
- `gen.optionalValue`: optionalValue는 optional 필드를 포인터로 두고 map에는 값으로 넣는지다.
- `gen.eventMap`: eventMap은 map 본문을 만드는 메서드다. optional 필드는 값이 있을 때만 역참조해서 넣어, map에는 JSON을 거친 것과 같은 모양의 값만 들어간다.
- `gen.eventMap`: ref·many·포인터 타입의 optional은 nil이 아닐 때만 넣는다
- `gen.eventFrom`: eventFrom은 map 본문에서 payload를 읽는 함수를 만든다. 키가 없거나 타입이 다르면 오류다. optional 필드는 키가 없거나 nil이면 건너뛴다. 내장 숫자·문자열·bool, 문자열 목록, 설계 enum은 JSON을 거친 값도 받고, 그 밖의 타입은 타입 단언만 한다.
- `gen.assign`: assign은 필드 하나를 target에 읽어 넣는 문장이다. 실패하면 d와 오류를 돌려준다.
- `gen.assign`: 생성한 enum이면 Valid()로 설계에 없는 값을 거른다
- `gen.helperKind`: helperKind는 필드가 변환 도우미를 쓰면 그 종류를 준다. optional 여부는 호출하는 쪽이 다룬다.
- `numberHelper`: numberHelper는 JSON을 거친 float64와 Go에서 넣은 정수를 받는다. 부호·범위를 넘거나 정수가 아닌 값은 오류다. float64를 거쳐 옮기지 않아 큰 정수도 그대로다.

## internal/gen/golang/file.go

- `Header`: Header는 모든 생성 파일의 첫 줄이다. 생성 파일인지는 이 줄로만 판정한다.
- `IsDaemokGenerated`: IsDaemokGenerated는 daemok이 만든 파일인지다. 첫 줄이 정확히 Header일 때만 참이고 줄 끝 \r은 무시한다. 빈 줄이나 BOM으로 시작하거나 헤더가 둘째 줄 이후에 있으면 거짓이다. 이 판정으로 고아 생성 파일을 지우므로 손으로 쓴 파일을 생성 파일로 보지 않게 엄격하게 본다. daemok은 늘 첫 줄에 헤더를 쓴다.
- `IsDaemokGeneratedFile`: IsDaemokGeneratedFile은 파싱한 파일에 IsDaemokGenerated와 같은 판정을 한다. 파일 맨 앞의 줄 주석이 첫 줄이다.
- `File`: File은 코드 루트 기준 경로(슬래시 구분)와 내용이다.
- `goFile.imports`: import 경로 → 이름. 이름이 경로의 마지막 요소와 같으면 별칭 없이 쓴다
- `goFile.shape`: shape이면 별칭을 붙이지 않고 항상 경로의 마지막 요소를 쓴다. 코드 대조용 모양 계산에만 쓴다.
- `goFile.use`: use는 경로를 import하고 코드에서 쓸 패키지 이름을 준다. 마지막 요소가 이미 다른 경로에 쓰였으면 경로 요소를 이어 붙인 별칭을 쓴다.
- `goFile.blank`: blank는 이름 없이(_) import한다.
- `localNames`: localNames는 생성 함수 안에서 쓰는 지역 이름이다. import 이름으로 쓰지 않는다.
- `goFile.render`: 표준 라이브러리(첫 요소에 점이 없는 경로)를 앞 그룹에, 나머지를 뒤 그룹에 둔다.

## internal/gen/golang/flows.go

- `flowInterface`: flowInterface는 단계와 되돌리기 메서드를 처음 나온 순서로 중복 없이 담는다.
- `flowRun`: flowRun은 단계를 순서대로 부르는 함수다. 커밋 전 단계가 실패하면 쌓인 되돌리기를 역순으로 부르고, fatal 단계는 되돌리지 않고 Fatal을 부른다.

## internal/gen/golang/gen.go

- `package golang`: Package golang은 모델에서 Go 생성 파일을 만든다. 디스크는 읽거나 쓰지 않는다.
- `Generate`: Generate는 설계 검사를 통과한 모델에서 생성 파일을 경로순으로 만든다.
- `byDir`: byDir는 노드를 소속 디렉토리별로 처음 나온 순서를 지켜 묶는다.
- `gen.consts`: 앞자리 0을 8진수로 읽지 않게 한다
- `JSONTag`: JSONTag는 설계 필드의 struct 태그다. optional이면 omitempty를 붙인다.
- `gen.structFields`: structFields는 설계 필드를 struct 필드 줄로 바꾼다.
- `gen.fieldType`: fieldType은 설계 필드의 Go 타입이다. ref는 포인터(many면 슬라이스), optional이면 포인터다.
- `gen.methodTypes`: methodTypes는 계약 메서드의 파라미터와 반환 타입이다. 반환 끝에 error가 붙는다.
- `gen.methodSig`: methodSig는 계약 메서드의 시그니처 줄이다. 반환 끝에 error가 붙는다.
- `gen.methodSig`: 주석이라 import하지 않고 패키지 이름만 붙인다
- `gen.handlerFunc`: handlerFunc는 멤버가 값 v를 처리하는 함수 이름이다. L4 함수 노드가 있으면 그 ID를 쓴다.
- `gen.registrations`: registrations는 그룹 멤버마다 핸들러 등록 파일을 만든다. 핸들러 종류가 gen: false여도 만든다.
- `gen.reachable`: reachable은 실행 단위가 조립하는 모듈·그룹·멤버다. composes에서 depends_on, needs, 그룹 멤버를 따라간다. 설계 검사의 도달 판정과 같은 규칙이다.
- `gen.executableModules`: executableModules는 실행 단위가 조립하는 그룹의 멤버를 blank import한다.
- `comment`: comment는 여러 줄 text의 줄마다 indent와 "// "를 붙인다.

## internal/gen/golang/needs.go

- `gen.unitNeeds`: unitNeeds는 모듈과 멤버가 받는 계약이다. 멤버는 그룹의 needs를 먼저 받는다. 같은 계약이 두 번 나오면 uses를 합치고, 한쪽이라도 uses가 비면 메서드 전부다.
- `gen.needs`: needs는 받는 쪽 패키지에 uses의 메서드만 담은 interface를 만든다.
- `gen.executableContracts`: executableContracts는 실행 단위가 조립하는 받는 쪽·주는 쪽 쌍마다 구현 타입이 받는 쪽 interface를 만족하는지 확인하는 줄을 만든다. call 계약은 구현 타입이 정해져 있지 않아 뺀다.

## internal/gen/golang/shape.go

- `MethodShape`: MethodShape는 계약 메서드를 생성했다면 나왔을 이름과 타입이다.
- `FieldShape`: FieldShape는 타입 필드를 생성했다면 나왔을 이름과 타입이다.
- `shapeFile`: shapeFile은 소유 패키지 안의 빈 파일이다. 타입식은 그 패키지에서 쓰는 모양이 되고 다른 패키지는 별칭 없이 import 경로의 마지막 요소로 한정된다.
- `ContractShape`: ContractShape는 계약의 메서드를 생성 규칙대로 바꾼 것이다. 코드 대조가 gen: false 계약에 쓴다.
- `TypeShape`: TypeShape는 타입의 필드를 생성 규칙대로 바꾼 것이다. 코드 대조가 gen: false 타입에 쓴다.

## internal/gen/golang/types.go

- `stdPackages`: stdPackages는 외부 타입 접두사 중 표준 라이브러리로 import하는 것이다.
- `StdImport`: StdImport는 생성 코드가 외부 타입 접두사로 import하는 표준 라이브러리 경로인지다.
- `gen`: gen은 한 번의 생성 상태다.
- `gen.ownerPath`: ownerPath는 모듈·멤버 ID의 코드 경로다.
- `gen.file`: file은 경로의 파일을 만들거나 이미 있는 것을 준다. 패키지 이름은 디렉토리의 마지막 요소다.
- `gen.mainFile`: mainFile은 실행 단위의 main 패키지 파일이다.
- `gen.goType`: goType은 설계 타입 문자열을 from 디렉토리의 파일 f에서 쓸 Go 타입으로 바꾼다.
- `gen.qualify`: qualify는 owner 모듈의 이름을 fromDir에서 쓸 수 있게 패키지를 붙인다.
- `gen.argType`: argType은 계약 시그니처의 타입이다. 설계 struct는 접두사가 없으면 포인터로 쓴다.

## internal/gen/ts/ts.go

- `package ts`: Package ts는 export: [ts]가 붙은 상수, enum, 타입을 ts_out 파일 하나의 TypeScript로 만든다.
- `Header`: Header는 생성 헤더다. Go 생성 파일의 헤더와 같아야 한다(workspace 테스트가 확인).
- `Generate`: Generate는 ts_out에 쓸 내용을 준다. ts_out이 없거나 내보낼 것이 없으면 false다. gen: false는 Go 선언을 손으로 썼다는 뜻이라 TS에는 영향이 없다.
- `Generate`: 앞자리 0은 TS 모듈에서 문법 오류다
- `Nodes`: Nodes는 TS로 내보내는 노드다. 각각 ID 순이다.
- `Exported`: Exported는 export에 ts가 있는 상수, enum, 타입을 준다.
- `ConstName`: ConstName은 상수와 enum 값 목록의 TS 이름이다.
- `TypeName`: TypeName은 enum과 타입의 TS 타입 이름이다.

## internal/gen/ts/types.go

- `builtin`: 설계 내장 타입의 TS 타입. 없는 내장 타입(duration, error)은 옮기지 않는다.
- `FieldType`: FieldType은 설계 필드의 TS 타입이다. Go에서 nil이 될 수 있는 값(포인터, 슬라이스, bytes, map)에 | null을 붙인다. optional 필드는 생성 코드가 omitempty로 키를 빼므로 TS에서 키를 선택으로 쓰고(optional이 true), 타입은 키가 있을 때의 값이다. omitempty는 nil 포인터와 빈 슬라이스·bytes를 빼므로 그 null은 붙이지 않는다. 옮길 수 없는 타입이거나 내보내지 않는 enum·타입을 가리키면 오류다.
- `FieldType`: 생성 코드가 포인터를 붙인다

## internal/gitbase/gitbase.go

- `package gitbase`: Package gitbase는 git HEAD에 있는 설계 파일과 저장소의 worktree를 읽는다. 설계의 기준은 HEAD다.
- `Files`: Files는 designDir의 ref 시점 *.yaml 파일을 준다. 키는 "<ref>:<저장소 기준 경로>"다. git 저장소가 아니거나 커밋이 없어 HEAD가 없으면 ok가 거짓이다. HEAD가 아닌 ref가 없으면 오류다. ref 시점에 설계 디렉토리가 없으면 빈 목록이다.
- `Files`: -z는 경로를 따옴표로 감싸지 않고 NUL로 구분한다. 비ASCII 경로도 그대로 받는다.
- `Worktree`: Worktree는 저장소의 작업 폴더 하나다. Branch는 짧은 브랜치 이름이고 detached면 비어 있다.
- `Worktrees`: Worktrees는 dir이 속한 저장소의 worktree를 git이 알려 주는 순서(주 worktree 먼저)로 준다. git 저장소가 아니면 nil이다.
- `Branch`: Branch는 dir의 현재 브랜치 이름이다. git 저장소가 아니거나 detached면 비어 있다.
- `HasHead`: HasHead는 dir이 커밋이 하나 이상 있는 git 저장소 안인지 본다.
- `Dirty`: Dirty는 dir 작업 폴더의 커밋 안 된 변경(추적 안 하는 파일 포함) 수다. 저장소가 아니면 0이다.
- `Dirty`: 이름 바꾸기·복사는 원래 이름을 다음 칸에 하나 더 준다.
- `NewWorktree`: NewWorktree는 dir이 속한 저장소의 HEAD에서 branch를 새로 만들어 <주 worktree>/.worktrees/<branch의 /를 -로>에 worktree로 꺼내고 그 경로를 준다. .git/info/exclude에 /.worktrees/를 한 번 더해 주 worktree의 status에 보이지 않게 한다. 브랜치나 경로가 이미 있으면 아무것도 하지 않고 오류를 준다.
- `Prefix`: Prefix는 dir의 저장소 최상위 기준 상대 경로다. 최상위면 빈 문자열이다.
- `RemoveWorktree`: RemoveWorktree는 NewWorktree가 만든 worktree와 브랜치를 지운다. 만든 뒤 실패했을 때 되돌린다.
- `excludeWorktrees`: excludeWorktrees는 저장소의 info/exclude에 /.worktrees/ 줄이 없으면 더한다.
- `gitOut`: gitOut은 git을 돌려 표준 출력과 표준 오류를 함께 준다. 오류 메시지를 보이려고 쓴다.

## internal/gosrc/gosrc.go

- `package gosrc`: Package gosrc는 디렉토리의 Go 소스 파일을 고르고 읽는다. codescan과 infer가 같은 파일 집합을 보도록 한 곳에 둔다.
- `ParseError`: ParseError는 읽거나 파싱하지 못한 파일이다.
- `Files`: Files는 dir의 Go 파일을 문법 트리로 준다. _test.go와 //go:build ignore 파일은 뺀다. OS·아키텍처 조건은 모두 참으로 보아 어느 플랫폼의 파일이든 읽는다. 디렉토리가 없으면 비어 있다. build 조건을 보려고 mode에 parser.ParseComments를 더한다.
- `ImportsOnly`: ImportsOnly는 파일의 package 절과 import만 읽는다. 본문에 문법 오류가 있어도 읽힌다. 빌드 제약으로 빠지는 파일이면 nil이다.
- `TopNames`: TopNames는 type, const, var, 메서드가 아닌 func의 이름이다. _, init, main은 뺀다.
- `SkipDir`: SkipDir는 코드를 읽지 않는 디렉토리다.
- `SkipPackageDir`: SkipPackageDir는 패키지를 찾을 때 내려가지 않는 디렉토리다. go 도구처럼 _로 시작하는 디렉토리도 뺀다.
- `ignored`: ignored는 파일의 build 조건이 ignore 태그 없이는 참이 될 수 없는지다.
- `satisfiable`: satisfiable은 ignore 태그를 거짓으로 두고 나머지 태그를 어떻게 정해 참이 될 수 있는지다. 태그가 많으면 판단하지 않고 참으로 본다.

## internal/infer/collect.go

- `package infer`: Package infer는 Go 모듈의 코드에서 L2 설계 초안을 만든다. 기존 프로젝트를 처음 가져올 때만 쓴다.
- `pkg`: pkg는 Go 파일이 있는 디렉토리 하나다. 경로는 루트 기준 슬래시 경로이고 루트는 "."다.
- `pkg.imports`: 같은 모듈 안의 패키지 디렉토리, 정렬
- `pkg.ext`: 표준 라이브러리가 아닌 다른 모듈의 import 경로, 정렬
- `pkg.doc`: 패키지 doc 주석의 첫 문장
- `pkg.hints`: Module은 build가 채운다
- `pkg.name`: package 절의 이름
- `pkg.decls`: export된 최상위 선언 이름 → 종류
- `Skip`: Skip은 초안에 넣지 않은 디렉토리나 파일과 그 이유다.
- `readGoMod`: readGoMod는 go.mod의 module 경로와 직접 require(// indirect가 아닌 것)를 준다.
- `collect`: 하위 디렉토리 하나를 읽지 못해도 나머지는 분석한다.
- `collect`: 본문만 깨진 파일(작업 중인 파일)은 import만 읽어 의존에 넣는다. 힌트는 읽은 파일에서만 낸다.
- `collect`: 첫 경로 요소에 점이 없으면 표준 라이브러리다.

## internal/infer/draft.go

- `Exe`: Exe는 초안의 실행 단위다.
- `Mod`: Mod는 초안의 모듈이다. Path는 ID와 다를 때만 있다.
- `Draft`: Draft는 코드에서 추정한 L2 설계다. tier 이름은 tier0부터 tier(Tiers-1)까지다.
- `FanIn`: FanIn은 모듈을 import하는 모듈·실행 단위의 수다.
- `Report`: Report는 분석 모드 화면과 daemok init이 보여 주는 메모다.
- `Report.Cycles`: Cycles는 서로 import하는 패키지 묶음의 모듈 ID다. 모든 플랫폼의 파일을 합쳐 보면 생길 수 있다.
- `Report.GroupCandidates`: GroupCandidates는 그룹으로 묶을 만한 형제 패키지다. 그룹은 핸들러 종류(L3)가 있어야 해서 초안에 넣지 않고 제안만 한다.
- `Report.External`: External은 go.mod의 직접 require와 그 모듈을 import하는 모듈·실행 단위다.
- `Report.Hints`: Hints는 L3 노드(gen: false)로 올릴 만한 손으로 쓴 선언이다.
- `Report.Uses`: Uses는 모듈·실행 단위가 다른 모듈에서 쓰는 export 이름이다.
- `External`: External은 외부 모듈 하나와 그걸 쓰는 모듈·실행 단위 ID다.
- `GroupCandidate`: GroupCandidate는 같은 부모 디렉토리 아래 형제 모듈 3개 이상이 형제 밖의 모듈을 공통으로 쓰는 묶음이다.
- `GroupCandidate.Path`: 부모 디렉토리
- `GroupCandidate.Members`: 모듈 ID
- `GroupCandidate.Common`: 모든 형제가 쓰는 형제 밖 모듈
- `GroupCandidate.Exceptions`: 형제끼리의 의존 "a → b". 그룹 예외 후보
- `Analysis`: Analysis는 Analyze의 결과다.
- `Analyze`: Analyze는 root의 go.mod와 Go 패키지에서 초안을 만든다. 파일을 쓰지 않는다.
- `build`: 순환 묶음의 패키지는 같은 tier다. 묶음 밖 의존의 층 중 가장 큰 값 + 1이다.
- `assignIDs`: assignIDs는 디렉토리마다 ID를 정한다. 마지막 경로 요소부터 쓰고, 겹치면 겹친 것끼리 경로 요소를 하나씩 앞으로 더 붙인다. 경로 전체로도 겹치면 디렉토리 순서대로 _2, _3을 붙인다. 루트(".")는 rootID다.
- `snake`: snake는 이름을 소문자 snake_case로 바꾼다. 소문자나 숫자 뒤의 대문자 앞, 대문자 연속 뒤에 소문자가 오면 그 연속의 마지막 대문자 앞에 _를 넣는다. -와 .은 _가 된다.
- `components`: components는 의존 그래프의 강한 연결 요소(Tarjan)다. comp는 노드마다 요소 번호, comps는 요소마다 노드다.
- `groupCandidates`: groupCandidates는 부모 디렉토리가 같은 형제 모듈 3개 이상이 형제 밖 모듈을 하나 이상 공통으로 쓰면 후보로 낸다. 루트 바로 아래 디렉토리는 프로젝트의 큰 구조라 보지 않는다.
- `external`: external은 require마다 그 모듈(또는 하위 패키지)을 import하는 초안 노드를 모은다. require가 겹치면(a/b와 a/b/c) 가장 긴 것에 붙인다.

## internal/infer/hints.go

- `Hint`: Hint는 손으로 쓴 선언 중 설계의 L3 노드로 올릴 만한 것이다. 초안에는 넣지 않고 gen: false 노드 후보로 알린다.
- `Hint.Kind`: 에러, enum
- `Hint.Type`: enum의 Go 타입
- `Hint.Where`: 코드 루트 기준 "경로:줄"
- `fileHints`: fileHints는 파일 하나에서 에러 변수 묶음과 enum처럼 쓰는 상수 블록을 찾는다. types는 같은 패키지에 선언된, 기반 타입이 string인 타입 이름이다.
- `fileHints`: 설계 노드는 exported Go 이름을 만들어서 unexported 타입·값은 gen: false로 올려도 대응하지 않는다. init --l3와 같은 기준이다.
- `fileHints`: 같은 타입을 적었거나 iota처럼 타입과 값을 생략한 줄만 값으로 본다.
- `stringTypes`: stringTypes는 파일들의 최상위 타입 중 기반 타입이 string인 것이다. 설계 enum은 문자열 값이라 이것만 enum 후보다. init --l3와 같은 기준이다(별칭과 제네릭 제외).

## internal/infer/render.go

- `Render`: Render는 초안을 설계 파일 텍스트로 쓴다. 노드 하나를 한 줄 플로우 맵으로 쓴다.
- `quote`: quote는 큰따옴표 문자열로 쓴다. 줄바꿈과 탭은 공백 하나로 바꾼다.

## internal/infer/uses.go

- `Use`: Use는 모듈·실행 단위 하나가 같은 Go 모듈의 다른 모듈에서 가져다 쓰는 export 이름이다.
- `Name`: Name은 쓰인 이름과 그 선언의 종류(함수, 타입, 인터페이스, 상수, 에러, 변수)다.
- `fileRef`: fileRef는 파일 하나의 같은 모듈 import와 그 파일에 나온 "식별자.이름" 선택자다.
- `fileRef.sels`: 식별자 → 이름
- `fileImport.dep`: 패키지 디렉토리
- `fileImport.alias`: 없으면 ""
- `exportedDecls`: exportedDecls는 패키지의 export된 최상위 선언 이름과 종류다. 메서드는 넣지 않는다.
- `selectors`: selectors는 파일에 나온 "식별자.이름" 쌍이다. 식별자가 지역 변수인지는 가리지 않고, 쓰는 쪽에서 대상 패키지의 export 선언과 맞는 이름만 남긴다.
- `uses`: uses는 ids(디렉토리 → 모듈·실행 단위 ID)에 있는 패키지끼리 쓰는 이름을 모은다. 이름 없이 import만 하는 관계(빈 import 등)는 넣지 않는다.
- `fileRefOf`: fileRefOf는 파일의 같은 모듈 import와 선택자를 모은다. "_"와 "." import는 이름으로 쓰이지 않아 뺀다.

## internal/infer/write.go

- `HasDesignFiles`: HasDesignFiles는 designDir에 *.yaml이 있는지다.
- `Write`: Write는 designDir에 *.yaml이 없을 때만 초안을 <system>.yaml로 새로 만든다. 만든 파일 경로를 준다.

## internal/lift/collect.go

- `collector.unit`: unit은 모듈 하나의 패키지에서 설계에 없는 exported 선언을 옮긴다. known은 설계가 만드는 이름이다.
- `collector.unit`: local은 이 패키지에서 설계 ID로 가리킬 수 있는 타입 이름이다. 설계에 있는 타입·enum과 이번에 함께 옮기는 것.
- `collector.unit`: 내장 타입 이름이 되는 ID는 설계 검사가 막으므로 가리키지 않는다.
- `docs`: docs는 선언 이름마다 문서 주석이다. 블록 안 선언은 자기 주석, 괄호 없는 단독 선언은 선언 주석이다.
- `summary`: summary는 문서 주석 첫 단락의 첫 문장이다. 160자가 넘으면 요약으로 보지 않는다.

## internal/lift/dedupe.go

- `platformVariants`: platformVariants는 한 모듈에 같은 Go 이름의 같은 종류 선언이 여러 번 있으면(build 조건이 다른 플랫폼 파일) 내용이 모두 같을 때 하나만 남기고, 다르면 모두 뺀다. 한 플랫폼의 값을 설계에 적으면 다른 플랫폼에서 틀린다.
- `qualified`: qualified는 필드 타입 문자열 안의 "모듈.Go이름"이다. 앞 글자가 이름의 일부면 맞추지 않는다.
- `relink`: relink는 다른 모듈 패키지의 타입을 코드 이름(core.Block)으로 가리킨 곳을, 그 타입·enum이 같은 초안에 남아 있으면 ID로 바꾼다. 수집할 때는 다른 모듈의 어떤 선언이 초안에 남을지 몰라 코드 이름으로 둔다. 요약 문장, 에러 메시지, 상수 값처럼 글자 그대로인 항목은 바꾸지 않는다.
- `dedupe`: dedupe는 초안 안에서 같은 종류의 ID가 겹치면(모듈마다 있는 Config, Service) 모듈 ID를 앞에 붙이고 go.name으로 Go 이름을 지킨다. 에러는 ID가 Go 이름이고 enum은 go 블록이 달라 바꾸지 않는다. 그래도 겹치는 노드는 첫 것만 남긴다. 초안 YAML에 같은 키가 두 번 나오면 설계를 읽을 수 없다.
- `literalField`: literalField는 값이 타입이 아니라 글자 그대로인 항목이다.

## internal/lift/funcs.go

- `pendingFunc`: pendingFunc는 모든 모듈의 에러를 모은 뒤 판단할 함수다. 다른 모듈의 초안 에러를 가리킬 수 있다.
- `errorRef`: errorRef는 모듈과 Go 이름으로 가리킨 에러다.
- `collector.funcs`: funcs는 반환문에서 설계 에러나 초안 에러를 가리키는 함수를 일반 함수 노드로 옮기고, 나머지는 수만 센다. 에러는 모듈과 Go 이름으로 들고 있다가 dedupe 뒤 resolveFuncErrors가 ID로 푼다. 에러 ID는 모듈을 붙이지 않아, 미리 풀면 빠진 초안 에러 대신 같은 ID의 다른 모듈 에러를 가리킨다.
- `resolveFuncErrors`: resolveFuncErrors는 funcs 노드의 에러를 설계 에러와 dedupe에서 남은 초안 에러의 ID로 푼다. 설계의 다른 에러와 ID가 겹치는 초안 에러는 검증에서 빠지므로 풀지 않는다. 반환하는 에러를 하나라도 풀지 못하면 에러 계약이 모자라므로 함수를 뺀다.
- `collector.returnedRefs`: returnedRefs는 함수 본문의 반환식에 나오는 이름이다. 같은 패키지 이름과 설계 모듈 패키지를 한정한 이름을 준다. 클로저 안의 return은 그 클로저의 반환이라 보지 않는다.

## internal/lift/lift.go

- `package lift`: Package lift는 설계 모듈 패키지에 손으로 쓴 선언 중 설계에 없는 것을 gen: false L3 노드 초안으로 만든다. 초안은 기존 설계와 합쳐 검사했을 때 새 오류·경고를 내지 않는 노드만 남긴다.
- `Node`: Node는 초안 노드 하나다. Line은 "ID: { ... }" 한 줄이다.
- `Node.Kind`: errors, consts, enums, types, contracts, funcs
- `Node.Name`: Go 이름
- `Node.Where`: 코드 루트 기준 "경로:줄"
- `Node.errRefs`: errRefs는 funcs 노드가 반환하는 에러다. dedupe 뒤 resolveFuncErrors가 ID로 푼다.
- `Skip`: Skip은 옮기지 않은 선언과 그 이유다.
- `Result`: Result는 초안 YAML과 노드, 옮기지 않은 선언이다. Funcs는 설계에 없고 설계 에러를 반환하지 않아 옮기지 않은 exported 함수의 수다.
- `Result.Coverage`: Coverage는 초안을 더하기 전과 뒤에 설계에 대응하는 최상위 선언 수다(daemok coverage의 합계).
- `Coverage`: Coverage는 최상위 선언 수와, 초안 전후의 설계 대응 선언 수다.
- `ErrDesign`: ErrDesign은 설계를 읽지 못했거나 설계 검사 오류가 있다는 뜻이다.
- `draftName`: draftName은 검증할 때 초안에 붙이는 설계 파일 이름의 끝이다.
- `draftKey`: draftKey는 검증할 때 초안에 붙이는 경로다. 설계는 경로 순으로 합쳐지고 중복 ID 진단은 뒤에 나온 쪽에 붙으므로, 초안이 언제나 마지막에 오도록 가장 뒤의 설계 경로에 이름을 덧붙인다.
- `Draft`: Draft는 designDir의 설계와 root의 코드로 초안을 만든다.
- `coverageDelta`: coverageDelta는 설계만으로 센 coverage와 초안을 더해 센 coverage다.
- `units`: units는 모듈과 그룹 멤버를 ID 순으로 준다.
- `collector.docs`: 선언 이름에서 그 선언의 문서 주석으로
- `skipLess`: skipLess는 경로, 줄 번호(숫자), 이름 순이다.

## internal/lift/names.go

- `quote`: quote는 설계 YAML의 큰따옴표 문자열로 쓴다. 파서가 받는 이스케이프(\" \\ \n \t)만 쓴다.
- `snakeID`: snakeID는 Go 이름을 snake_case ID로 바꾼다. GoName(ID)가 원래 이름과 다르면 goName에 원래 이름을 준다. 대문자 연속(HTTPPort)은 마지막 대문자 앞에서 끊는다.
- `snakeID`: 숫자 뒤의 대문자는 숫자 앞이 소문자일 때만 끊는다(Sha256Size, P2PPort).
- `snakeID`: 약어 뒤의 복수 s(IDs, URLsByID)는 약어에 붙인다.

## internal/lift/types.go

- `collector.designTypes`: designTypes는 모듈에 속한 설계 타입·enum의 Go 이름에서 설계 ID로 가는 표다.
- `collector.unitOf`: unitOf는 import 경로가 설계 모듈·멤버 패키지면 그 ID를 준다.
- `collector.structType`: structType은 struct를 타입 노드로 옮긴다. 필드를 모두 설계 타입 문자열로 바꿀 수 있어야 한다.
- `omitempty`: omitempty는 필드 태그의 json 옵션에 omitempty가 있는지 본다.
- `optionalField`: optionalField는 omitempty인 필드의 설계 타입 문자열을 optional 필드로 바꾼다. 생성 코드는 optional 필드에 omitempty를 붙이고, 포인터·슬라이스가 아니면 포인터로 감싼다. 그래서 *T는 T로, []T와 bytes는 그대로 쓴다. 값 타입은 생성 코드가 포인터로 바꿔 모양이 달라지므로 그대로 둔다.
- `collector.contract`: contract는 interface를 call 계약으로 옮긴다. 메서드는 반환 끝이 error여야 한다(생성 계약이 error를 붙인다).
- `collector.method`: method는 메서드 하나를 "이름: { params, returns, errors: [] }"로 쓴다. 이름 없는 파라미터는 arg1부터 붙인다.
- `collector.fieldType`: fieldType은 필드의 Go 타입식을 설계 타입 문자열로 바꾼다. 바꿀 수 없으면 이유를 준다. *, [가 들어가면 YAML 평문이 될 수 없어 따옴표로 감싼다.
- `collector.typeString`: typeString은 fieldType의 따옴표 없는 설계 타입 문자열이다.
- `importPath`: importPath는 파일에서 패키지 이름이 가리키는 import 경로다. 별칭이 없으면 경로의 마지막 요소로 맞춘다.

## internal/lift/validate.go

- `validate`: validate는 초안을 설계에 합쳐 돌리고, 초안 줄을 가리키는 오류·경고가 나온 노드를 빼고 다시 돌린다. 초안 밖에 새 오류·경고가 생기면 어느 노드 탓인지 알 수 없어 오류로 끝낸다.
- `validate`: 한 라운드는 진단이 난 노드를 모두 빼므로 라운드 수는 참조 연쇄의 깊이다. YAML 문법 오류처럼 첫 오류에서 멈추는 진단이면 노드를 하나씩만 빼게 되어 상한에서 끝낸다.
- `header`: header는 초안 파일 머리 주석의 줄 수와 내용이다. lineNodes가 줄 번호를 맞추는 데 쓴다.
- `lineNodes`: lineNodes는 render가 쓴 줄 번호(1부터)에서 노드 번호로 가는 표다.

## internal/lift/values.go

- `collector.enums`: enums는 string 타입과 그 타입의 문자열 리터럴 상수를 enum 노드로 옮긴다. 옮겼든 못 옮겼든 다룬 타입과 값 상수 이름을 돌려줘 다른 종류로 다시 알리지 않게 한다.
- `collector.enums`: go.prefix를 비우면 타입 이름이 접두사가 되므로 설계로 적을 수 없다.
- `enumValue`: enumValue는 YAML 평문으로 그대로 쓸 수 있는 enum 값이다. 설계 검사의 값 문자 규칙보다 좁다.
- `collector.errorVar`: errorVar는 var ErrX = errors.New("문자열")를 에러 노드로 옮긴다.
- `makesError`: makesError는 var 값이 errors 패키지 함수나 fmt.Errorf 호출인지 본다.
- `collector.constant`: constant는 정수·문자열 리터럴과 N * time.Unit 상수를 상수 노드로 옮긴다.
- `constValue`: Go 리터럴 규칙(0x, 0o, 앞자리 0의 8진수, 밑줄)으로 읽어 10진수로 쓴다.
- `typeName`: typeName은 타입 식이 이름이나 패키지.이름이면 그 글자를 준다.

## internal/mcp/server.go

- `package mcp`: Package mcp는 표준 입출력으로 MCP(JSON-RPC 2.0, 한 줄에 메시지 하나)를 주고받아 AI가 설계를 읽고 검사를 돌리게 한다. edit 외의 도구는 읽기 전용이고, edit는 --allow-write일 때만 있다.
- `defaultProtocol`: defaultProtocol은 지원하는 MCP 프로토콜 중 최신이다. 클라이언트가 모르는 버전을 요청하면 이것으로 답한다.
- `maxLine`: maxLine은 메시지 한 줄의 최대 바이트다. 넘는 줄은 그 줄만 거부한다.
- `Server`: Server는 기본 설계 디렉토리와 코드 루트를 가진다. 도구 인자로 바꿀 수 있다.
- `Server.AllowWrite`: AllowWrite면 설계 파일을 고치는 edit 도구를 낸다. 사용자가 서버를 등록할 때 --allow-write로 켠다.
- `Server.Serve`: Serve는 r가 끝날 때까지 한 줄에 하나씩 요청을 읽고 w에 응답을 쓴다. 알림(id 없음)에는 응답하지 않는다.
- `readLine`: readLine은 줄바꿈까지 한 줄을 읽는다. maxLine을 넘으면 나머지를 버리고 tooLong을 준다.
- `Server.callTool`: callTool은 tools/call 요청 하나를 처리한다. 도구 오류는 isError 결과로, 요청 형식 오류는 rpcError로 준다.
- `Server.callTool`: 쓰기는 사용자가 등록한 서버의 설계 디렉토리에만 한다. 도구 인자로 경로를 바꾸지 못한다.
- `Server.toolPaths`: toolPaths는 도구 인자의 design·root로 쓸 설계 디렉토리와 코드 루트를 정한다. root만 주면 설계는 그 root 아래 design이고, 둘 다 없으면 서버의 --design, --root다.

## internal/mcp/tools.go

- `unit.Kind`: module, group, member
- `draftL3Tool`: draftL3Tool은 L3 초안 YAML과, 옮기지 않은 선언·설계 에러를 반환하지 않는 함수 수의 JSON을 ---로 나눠 준다.

## internal/model/decode.go

- `decoder.oneOf`: oneOf는 key 값이 choices 중 하나인지 본다. 키가 없으면 def.
- `decoder.need`: need는 key가 있고 null이 아닌지 본다. 오류 위치는 노드의 ID 키(pos)다.
- `decoder.goOpts`: goOpts는 go 블록을 읽는다. keys가 있으면 그 키만 받는다.
- `decoder.idMap`: idMap은 ID를 키로 하는 맵을 돈다. 값이 맵이 아닌 항목은 오류로 남기고 건너뛴다.

## internal/model/decode_l4.go

- `merger.caseMap`: caseMap은 given·expect가 스칼라 또는 맵만 값으로 갖는 맵인지 확인한다.
- `merger.steps`: 빈 목록과 형식 오류(nil)를 구분한다

## internal/model/index.go

- `Model.TierIndex`: TierIndex는 tiers 목록에서의 순서다. 작을수록 아래 tier. 없으면 -1.
- `Owner`: Owner는 L3·L4 노드가 속하는 모듈 또는 그룹 멤버다.
- `Owner.Module`: 모듈이면 채워진다
- `Owner.Member`: 멤버면 채워진다
- `Model.CanDepend`: CanDepend는 설계상 from이 to를 import해도 되는지다. from·to는 모듈, 그룹, 멤버 ID. 그룹에 의존하면 그 멤버에도 의존할 수 있다.
- `MapType`: map[K]V. K는 내장 타입이나 enum, V는 설계 타입 문자열
- `Builtins`: Builtins는 설계 타입 이름과 Go 타입의 대응이다.
- `SplitType`: SplitType은 "*[]post"를 ("*[]", "post")로 나눈다.
- `MapParts`: MapParts는 "map[K]V"를 K와 V로 나눈다. K 안의 대괄호 짝을 맞춘다.
- `Model.ResolveType`: ResolveType은 타입 이름의 종류와 접두사를 뗀 이름을 준다. 대문자로 시작하거나 '.'이 있으면 외부 타입으로 본다.
- `GoName`: GoName은 snake_case ID를 CamelCase로 바꾼다. 약어 조각(id, url)과 그 복수(ids)는 ID, URL, IDs가 된다.
- `initialisms`: initialisms는 Go 이름에서 모두 대문자로 쓰는 약어다. staticcheck ST1003의 기본 목록이다.
- `Enum.GoType`: GoType은 enum의 Go 타입 이름이다.
- `Enum.GoPrefix`: GoPrefix는 enum 상수 이름의 접두사다.
- `Enum.ConstName`: ConstName은 enum 값 하나의 Go 상수 이름이다.
- `Model.ImportPath`: ImportPath는 모듈·멤버의 Go import 경로다. 없는 ID면 빈 문자열이다.
- `Type.GoName`: GoName은 타입의 Go 이름이다.
- `Contract.GoName`: GoName은 계약 interface의 Go 이름이다.
- `HandlerKind.GoName`: GoName은 핸들러 함수 타입의 Go 이름이다.
- `Func.Free`: Free는 핸들러도 계약 메서드도 아닌 일반 함수인지 본다. 본문은 손으로 쓰고 설계는 에러 계약과 케이스만 둔다.
- `Func.GoName`: GoName은 일반 함수의 Go 이름이다.
- `ErrorDef.GoName`: GoName은 에러 sentinel의 Go 이름이다. go.name이 없으면 ID가 Go 이름이다.
- `Const.GoName`: GoName은 상수의 Go 이름이다.

## internal/model/load.go

- `Load`: Load는 dir의 *.yaml을 이름순으로 읽어 한 모델로 합친다. 오류가 있어도 읽은 만큼의 모델을 돌려준다.
- `LoadSources`: LoadSources는 이름과 내용의 설계 파일들을 이름순으로 한 모델로 합친다. 이름은 진단 위치에 쓰인다.
- `merger.ids`: 종류 → ID → 처음 위치
- `merger.once`: 한 번만 나올 수 있는 최상위 키
- `merger.first`: 처음 읽은 파일의 위치. version 부재 오류에 쓴다
- `merger.unique`: unique는 종류 안의 ID 중복을 잡는다. 처음 보는 ID면 true.
- `merger.finish`: finish는 version이 있는지, 한 참조 필드가 받는 종류끼리 ID가 겹치는지 본다.

## internal/model/model.go

- `package model`: Package model은 설계 파일을 노드 타입으로 옮기고 여러 파일을 하나로 합친다.
- `Ref`: Ref는 다른 노드를 가리키는 이름과 그 위치다. 해석은 check가 한다.
- `Meta.Pos`: ID 키 위치
- `Requirement.Status`: active, held, dropped
- `Module.Role`: data, infra, adapter. 비면 도메인 모듈
- `Need`: Need는 받는 쪽이 적은 계약 하나다.
- `Need.Uses`: 비면 계약의 메서드 전부
- `Owned`: Owned는 모듈이나 멤버에 속하는 L3·L4 노드의 공통 부분이다.
- `Owned.Gen`: 기본 true. false면 생성에서 뺀다
- `Const.Type`: int, string, duration
- `ErrorDef.HTTP`: 0이면 없음
- `Field.Type`: type 또는 ref 중 하나만 채워진다
- `Field.Max`: 상수 ID 또는 숫자
- `Method.HasErrors`: errors 키가 있었는지
- `Contract.Kind`: call, query, port, external
- `Case.Given`: 맵
- `Case.Expect`: 맵 또는 nil
- `Case.Error`: 에러 ID, "invalid", 또는 비어 있음
- `Func.Of`: "계약.메서드"
- `Func.Go`: 일반 함수의 go.name
- `EventKind.Encoding`: map, json
- `EventKind.Subscribers`: 묶음의 모든 이벤트를 받는 모듈
- `Step.Calls`: 계약.메서드
- `CPP`: CPP는 C++ include 검사 설정이다. 모듈 path는 각 include 디렉토리 아래 폴더다.
- `CPP.IncludeDirs`: 코드 루트 기준 상대 경로
- `Model.TSOutPos`: ts_out 값 위치

## internal/rename/kinds.go

- `package rename`: Package rename은 설계 노드 하나의 ID와 그 참조를 설계 파일 전체에서 바꾼다. 기본은 코드 이름을 고정 키로 유지하고, 생성 결과와 진단이 그대로인지 확인한 뒤 쓴다.
- `PinKind`: PinKind는 코드 이름을 옛 값으로 고정하는 방법이다.
- `PinNone`: 코드와 무관
- `PinGoName`: go: { name: ... }
- `PinGoType`: go: { type: ... }
- `PinPath`: path: ...
- `PinImpossible`: 형식에 코드 이름 키가 없음
- `Kinds`: Kinds는 이름을 바꿀 수 있는 종류 이름이다.
- `namespace`: namespace는 ID가 서로 겹칠 수 없는 종류 묶음이다. depends_on과 타입 문자열이 묶음을 함께 가리킨다.
- `metas`: metas는 kind의 노드마다 Meta를 준다. tiers는 Meta가 없어 주지 않는다.
- `exists`: exists는 kind에 id 노드가 있는지다.

## internal/rename/occur.go

- `collector.ref`: ref는 이름이 옛 ID인 참조를 바꾼다.
- `collector.typ`: typ은 타입 문자열 안의 옛 ID 토큰을 바꾼다.
- `collector.method`: method는 "계약.메서드" 참조의 계약 부분을 바꾼다.
- `retype`: retype은 타입 문자열에서 식별자 토큰이 정확히 old인 것만 new로 바꾼다. 경계는 [ ] * , 와 문자열 끝이다.
- `definition`: definition은 kind의 id 노드의 ID 키 위치다.
- `Occurrences`: Occurrences는 kind의 old 노드 정의 키와 그 노드를 가리키는 모든 참조 자리를 준다. 참조의 종류는 check의 해석을 따른다.
- `Occurrences`: 이벤트 종류의 key enum 값은 이벤트 ID다.
- `signatures`: signatures는 enum·타입을 가리키는 타입 문자열과 ref, key다.
- `deps`: deps는 모듈·그룹·멤버를 함께 가리키는 참조다. 셋은 ID가 겹치지 않는다.

## internal/rename/run.go

- `Options`: Options는 이름 바꾸기 방식이다. Code면 고정 키 없이 생성 코드 이름도 새 ID를 따른다. Confined면 설계 디렉토리 밖을 가리키는 심볼릭 링크 파일을 고치지 않는다(MCP).
- `Result`: Result는 바꾼 결과다. Pinned는 더한 고정 키, Moves는 --code에서 옮겨야 할 디렉토리다.
- `yamlID`: yamlID는 따옴표 없이 YAML에 쓸 수 있는 ID다. 종류별 형식 규칙은 설계 검사가 본다.
- `yamlKeyword`: yamlKeyword는 따옴표 없이 쓰면 YAML이 문자열이 아닌 값(null, 불리언)으로 읽는 이름이다. 참조 자리에 쓰면 참조가 빈 값이 되어 검사도 잡지 못한다.
- `fileNamed`: fileNamed는 생성 파일 이름을 ID로 짓는 종류다.
- `buildSuffix`: buildSuffix는 id로 지은 파일 이름을 go/build가 GOOS·GOARCH 제약으로 읽는지다. 규칙은 go/build의 goodOSArchFile과 같다.
- `failAfter`: failAfter는 테스트에서 쓴 파일 수가 이 값이 되면 다음 쓰기를 실패로 본다. -1이면 끈다.
- `Run`: Run은 designDir의 설계에서 kind의 old 노드 ID와 그 참조를 new로 바꾼다. 편집 전에 없던 읽기 오류, 늘어난 설계 검사 진단, 기본 모드에서 달라진 생성 결과가 있으면 쓰지 않는다.
- `Run`: 경로를 ID와 같은 값으로 적어 둔 노드는 그 키가 이미 고정이다.
- `pinFor`: pinFor는 더할 고정 키다. 코드 이름 키가 이미 있거나 --code(gen: false 제외)면 비운다.
- `pinFor`: go.name은 일반 함수에만 쓴다. 핸들러·계약 함수의 코드 이름은 생성 결과 비교가 지킨다.
- `pinPath`: pinPath는 바꾼 뒤 노드까지의 키 목록이다.
- `moves`: moves는 --code에서 경로가 ID에서 나오는 노드의 옮길 디렉토리다. 손으로 쓴 파일은 daemok이 옮기지 않는다.
- `verify`: verify는 바꾼 설계가 읽기 오류나 검사 진단을 늘리지 않는지, 기본 모드면 생성 코드가 같은지 본다. 기본 모드에서 이름만 바뀌는 생성 파일 목록을 준다.
- `verify`: TS 파일은 ts_out 디렉토리가 없으면 Files에 빠지므로 따로 비교한다. TS 이름은 고정 키가 아니라 ID에서 나온다.
- `sameCode`: sameCode는 생성 파일이 패키지 디렉토리와 내용으로 같은지 본다. 파일 이름만 바뀐 것은 코드 이름에 영향이 없어 받고(enum, 핸들러 종류 파일은 ID로 이름을 짓는다), 그 목록을 준다.
- `replaceWord`: replaceWord는 s에서 앞뒤가 식별자 글자가 아닌 old만 new로 바꾼다.
- `writeAll`: writeAll은 파일마다 원자적으로 쓰고, 실패하면 이미 쓴 파일을 원래 내용으로 되돌린다.

## internal/serve/edit.go

- `Server.editDesign`: editDesign은 화면에서 온 L2 편집을 설계 파일에 줄 단위로 적용한다. 화면이 본 버전과 다르면 쓰지 않고, 고친 결과가 읽기 오류를 내면 쓰지 않는다.
- `Server.editDesign`: 버전 확인부터 쓰기까지 한 번에 하나만 한다. 같은 버전의 두 번째 요청은 409가 된다.
- `Server.editDesign`: 화면이 본 상태 뒤로 설계 파일 내용이 바뀌었을 때만 거부한다. 코드만 바뀐 버전은 받는다.

## internal/serve/gen.go

- `Server.genPreview`: genPreview는 daemok gen이 할 일을 파일마다 준다. 쓰지 않는다. 설계 대조를 다시 돌리므로 요청할 때만 만든다.
- `Server.genContent`: genContent는 생성 파일 하나의 내용을 준다. 이번 생성 결과에 있는 경로만 받는다.

## internal/serve/guard.go

- `Server.guard`: guard는 루프백에서 들을 때 다른 사이트가 브라우저를 통해 보내는 요청을 막는다. Host가 루프백이 아니면 DNS rebinding으로 보고, POST의 Origin이 Host와 다르면 CSRF로 본다.

## internal/serve/issues.go

- `nodePos`: nodePos는 화면 키와 그 노드의 설계 파일 위치다.
- `nodePositions`: nodePositions는 모든 노드의 키와 위치를 파일·줄·열 순으로 준다.
- `issueKey`: issueKey는 진단이 가리키는 화면 노드 키다. 설계 파일 진단이면 같은 파일에서 진단 위치 이전의 마지막 노드, Go 파일 진단이면 경로 접두사가 가장 긴 모듈·멤버다.

## internal/serve/l3.go

- `Server.l3Draft`: l3Draft는 설계에 없는 손으로 쓴 선언의 L3 초안을 준다. 검증에 설계 대조를 다시 돌리므로 요청할 때만 만든다.
- `Server.l3Save`: l3Save는 초안을 다시 만들어 설계 디렉토리의 새 파일(l3.yaml, l3-2.yaml, ...)로 쓴다. 있는 파일은 덮어쓰지 않는다.
- `createFree`: createFree는 l3.yaml부터 이름이 비어 있는 첫 파일을 만든다. O_EXCL이라 동시에 저장해도 덮어쓰지 않는다.

## internal/serve/projects.go

- `Options`: Options는 프로젝트 열기에 쓰는 설정이다.
- `Options.Workspace`: 프로젝트를 찾을 작업 폴더. 비면 목록이 없다
- `Options.ConfigDir`: recent.json을 둘 디렉토리. 비면 저장하지 않는다
- `Options.Local`: 루프백 주소에서 들을 때 참. 참일 때만 열기·저장을 허용하고 Host·Origin을 검사한다
- `Server.SetOptions`: SetOptions는 설정을 바꾸고 상태를 다시 만든다. 상태의 canEdit이 설정을 따른다.
- `Server.Open`: Open은 path를 현재 프로젝트로 연다. 설계 디렉토리는 path/design이다. 디렉토리가 아니거나 설계 파일과 go.mod가 모두 없으면 바꾸지 않고 오류를 준다.
- `expandHome`: expandHome은 ~와 ~/로 시작하는 경로의 ~를 홈 디렉토리로 바꾼다.
- `findModules`: findModules는 dir 아래 깊이 maxDepth까지에서 go.mod가 있는 디렉토리를 찾는다. 모듈 아래로는 내려가지 않는다.
- `vProject.Mode`: check(설계 있음), analyze(go.mod만), 없거나 열 수 없으면 빈 값
- `project`: project는 경로 하나의 목록 항목이다. 브랜치를 모르면 branch를 git으로 읽는다.
- `repoName`: repoName은 root가 주 worktree가 아닌 worktree면 주 worktree의 폴더 이름을 준다. worktree 폴더 이름은 저장소를 알려 주지 못한다(.worktrees/daemok).
- `samePath`: samePath는 심볼릭 링크를 푼 두 경로가 같은지 본다. git은 링크를 푼 경로를 준다.
- `Server.projects`: worktree가 하나뿐이면 지금 연 것뿐이라 보이지 않는다.
- `Server.initDraft`: 화면 알림이 짧도록 코드 루트 기준 슬래시 경로로 준다.
- `draftBranch`: draftBranch는 초안을 새 worktree에 저장할 때 제안하는 브랜치 이름이다.
- `Server.initWorktree`: initWorktree는 분석 모드의 초안을 지금 저장소 HEAD에서 만든 새 worktree의 design/에 저장하고 그 worktree를 연다. 원래 작업 폴더에는 쓰지 않는다. 초안은 새 worktree의 코드(HEAD)로 다시 추정한다.
- `Server.initWorktree`: 만든 뒤 실패하면 worktree와 브랜치를 지워 같은 이름으로 다시 할 수 있게 한다.

## internal/serve/server.go

- `package serve`: Package serve는 설계 지도를 로컬 웹 페이지로 보여준다. 계산은 서버가 하고 화면은 /api/state를 그린다.
- `state`: state는 /api/state의 본문이다.
- `state.designSum`: 설계 파일 내용 요약. 내보내지 않는다.
- `state.Branch`: git 브랜치. 저장소가 아니거나 detached면 비어 있다
- `state.Repo`: 주 worktree가 아닌 worktree면 주 worktree의 폴더 이름
- `state.Mode`: check, analyze, none
- `state.CanEdit`: 루프백이고 대조 모드면 화면에서 L2를 고칠 수 있다
- `vDraft`: vDraft는 분석 모드의 초안이다.
- `vDraft.Git`: 새 worktree에 저장할 수 있을 때만
- `vDraftGit`: vDraftGit은 초안을 새 worktree에 저장할 때 보이는 정보다.
- `vDraftGit.Branch`: 제안하는 브랜치 이름
- `vDraftGit.Dirty`: 새 worktree에 들어가지 않는 커밋 안 된 변경 수
- `Server`: Server는 설계 디렉토리 하나의 지도를 제공한다.
- `Server.mode`: 마지막으로 낸 상태의 모드
- `Server.designSum`: 설계 파일 내용의 요약과, 그 내용이 처음 나간 상태 버전. 편집의 409 판정에 쓴다.
- `Server.refreshMu`: refreshMu는 상태 만들기를 한 번에 하나로 한다. 늦게 끝난 옛 상태가 새 버전 번호로 나가지 않게 한다.
- `New`: 화면이 프로젝트 이름을 보이도록 절대경로로 둔다. 실패하면 받은 그대로 쓴다.
- `New`: 첫 Poll이 반드시 상태를 만들도록 서명을 어떤 실제 서명과도 다른 값으로 둔다.
- `Server.Refresh`: Refresh는 설계나 코드를 다시 읽고 상태를 새로 만든 뒤 구독자에게 알린다. 설계 파일이 있으면 대조 모드, 없고 코드 루트가 Go 모듈이면 분석 모드다.
- `refreshBuilt`: refreshBuilt는 상태를 다 만들고 내기 직전에 불린다. 테스트가 상태 만들기를 겹치게 하려고 바꾼다.
- `designSum`: designSum은 설계 디렉토리 *.yaml의 이름과 내용 요약이다.
- `Server.designVersions`: designVersions는 지금 설계 내용이 처음 나간 버전과 마지막 버전이다.
- `shellQuote`: shellQuote는 공백이 있는 경로를 큰따옴표로 감싼다.
- `Server.publish`: publish는 상태에 버전을 붙여 저장하고 구독자에게 알린다. 그사이 다른 프로젝트를 열었으면 버린다. 늦게 끝난 옛 프로젝트의 상태가 새 상태를 덮지 않게 한다.
- `Server.Version`: Version은 마지막 상태의 번호다.
- `signature`: signature는 설계 디렉토리의 *.yaml과 코드 루트의 .go 파일의 이름·크기·수정 시각이다. 파일 추가·삭제·수정을 잡는다. 코드 쪽은 생성 파일 범위와 같은 디렉토리만 본다.
- `Server.Poll`: Poll은 설계나 코드가 바뀌었으면 Refresh하고 true를 준다. 도중에 프로젝트가 바뀌면 아무것도 하지 않는다.
- `Server.Watch`: Watch는 ctx가 끝날 때까지 interval마다 Poll한다.
- `Server.events`: events는 상태가 바뀔 때마다 버전 번호를 SSE로 보낸다.

## internal/serve/view.go

- `vSystem`: 화면용 모델. 필드 이름은 프로토타입의 normalize()와 같고, 노드 목록은 설계 순서를 지키려고 배열이다.
- `vStep.OnFailUndo`: OnFailUndo는 이 단계가 실패하면 부르는 되돌리기를 부르는 순서대로 담는다.
- `vRelation`: vRelation은 화면이 그리는 관계 하나다. 종류는 depends, exception, call, query, port, external, event, register.
- `vIssue.Origin`: design, code
- `vIssue.Group`: Group은 메시지의 "설명: 대상"에서 설명 부분이다. 화면이 같은 원인의 진단을 묶는다.
- `vIssue.Abs`: Abs는 에디터로 열 수 있는 File의 절대경로다.
- `nodeValue`: nodeValue는 cases의 given·expect 노드를 JSON 값으로 바꾼다.
- `toView`: toView는 모델을 화면용 모델로 바꾼다. m이 nil이면 nil이다.
- `toView`: 실패 경로는 생성되는 Run 함수와 같은 규칙이다. 커밋 전 실패는 쌓인 되돌리기를 역순으로, fatal 단계는 되돌리지 않고 Fatal, 커밋 뒤 실패는 아무것도 되돌리지 않는다.
- `toIssues`: toIssues는 진단을 화면용으로 바꾼다. code에 든 진단은 위치와 상관없이 코드 대조로 분류한다.
- `fileSources`: fileSources는 생성 파일 경로마다 그 파일을 만든 노드 키다. 생성 규칙 표의 원본과 같다.
- `toRelations`: toRelations는 설계의 참조에서 관계 목록을 만든다. 화면은 이 목록을 그리기만 한다.

## internal/workspace/preview.go

- `FileStatus`: FileStatus는 gen이 생성 파일 하나에 할 일이다. Status는 same(그대로), new(새로 씀), changed(내용이 바뀜), delete(설계에 없어 지움), refuse(생성 경로에 생성 헤더 없는 파일 등이 있어 쓰지 않음)다.
- `Preview`: Preview는 Plan이 있는 결과에서 생성 파일마다 상태를 경로 순으로 준다. 쓰지 않는다.
- `Content`: Content는 이번 생성 결과에서 path의 생성할 내용을 준다.

## internal/workspace/report.go

- `ReportJSON`: ReportJSON은 진단을 CI와 도구가 읽는 JSON으로 만든다. group은 메시지의 "설명: 대상"에서 설명 부분이다. check --format json과 MCP check 도구가 쓴다.

## internal/workspace/workspace.go

- `package workspace`: Package workspace는 설계 읽기, 설계 검사, 생성, 디스크의 생성 파일 대조를 묶는다. check·gen 명령과 serve가 같은 진단 목록을 쓰도록 한 곳에 둔다.
- `Plan`: Plan은 gen이 할 일이다. 경로는 코드 루트 기준 슬래시 경로다.
- `Plan.Write`: 없거나 내용이 다른 생성 파일
- `Plan.Delete`: 생성 헤더가 있고 이번 결과에 없는 파일
- `Plan.Refuse`: 쓰지 않는 생성 경로
- `Refusal`: Refusal은 생성 경로를 쓰지 않는 이유다.
- `Refusal.Reason`: 생성 헤더가 없는 파일, 디렉토리, 심볼릭 링크
- `Result`: Result는 한 번 돌린 결과다.
- `Result.Fatal`: 설계 파일을 읽지 못했다
- `Result.Plan`: 설계 검사와 생성이 성공했을 때만 있다
- `Result.Files`: 생성 파일 전부. 설계 검사와 생성이 성공했을 때만 있다
- `Result.Others`: Others는 Diags에서 생성 파일 대조를 뺀 진단이다. gen이 출력한다.
- `Result.Code`: Code는 Diags 가운데 코드를 읽어 낸 진단(생성 파일 대조와 codescan)이다.
- `Run`: Run은 designDir의 설계를 읽고 검사하고 생성해서 root의 파일과 대조한다.
- `Check`: Check는 코드 대조까지 하는 Run이다. 코드 루트가 없으면 대조할 코드가 없어 통과로 보이므로 진행하지 않는다. gen은 없는 루트에 처음 생성할 수 있어야 하므로 Run을 그대로 쓴다.
- `RunSources`: RunSources는 디스크가 아닌 설계 파일 내용으로 Run과 같은 일을 한다. 키는 진단 위치에 쓰는 파일 이름이다.
- `withTS`: withTS는 TS 생성 파일을 생성 목록에 더하고, 디스크의 것이 생성 파일이면 대조 대상에 넣는다. ts_out은 코드 루트 밖일 수 있어 ScanGenerated가 찾지 않는다. 상위 디렉토리가 없으면 프론트 저장소가 없는 곳으로 보고 쓰지 않는다. 경고는 gen도 출력하도록 대조 진단이 아닌 diags에 넣는다.
- `blocked`: blocked는 생성 경로에 쓰면 안 되는 이유를 준다. 쓸 수 있으면 빈 문자열이다. 코드 루트 아래 경로 요소 중 심볼릭 링크가 있거나, 그 자리에 디렉토리나 생성 헤더 없는 파일이 있으면 막는다.
- `DefaultDesignDir`: DefaultDesignDir는 설계 디렉토리를 따로 주지 않았을 때 쓰는 코드 루트 아래 design이다.
- `SkipDir`: SkipDir는 생성 파일 범위와 코드 감시에서 빼는 디렉토리다.
- `ScanGenerated`: ScanGenerated는 root 아래에서 첫 줄이 생성 헤더인 .go 파일을 찾아 슬래시 경로와 내용(CRLF를 LF로 바꾼 것)을 준다.
- `ScanGenerated`: 아직 없는 코드 루트: 처음 생성할 때 만든다
- `Apply`: Apply는 계획대로 파일을 쓰고 지운다. 거부할 파일이 있으면 호출하지 않는다.
- `removeEmptyDirs`: removeEmptyDirs는 dir부터 코드 루트 바로 아래까지 비게 된 디렉토리를 지운다.

## internal/yaml/block.go

- `Parse`: Parse는 src를 파싱한다. 빈 파일은 빈 맵이다. 실패하면 *Error를 돌려준다.
- `parser.splitKey`: splitKey는 "key: rest"를 나눈다. rest가 비면 값이 다음 줄에 있거나 null이다. restCol은 rest가 시작하는 열이다.
- `parser.splitKey`: ':'의 인덱스
- `colonAfterKey`: colonAfterKey는 뒤에 공백이 오거나 줄 끝인 첫 ':'의 인덱스를 준다. 없으면 -1.
- `parser.parseSeq`: 같은 열의 다음 키. 부모 맵이 이어 읽는다
- `parser.parseSeq`: "- key: v"는 줄을 key부터 시작하는 것으로 바꿔 맵으로 읽는다. 이어지는 줄은 같은 열에 맞춰야 한다.
- `parser.parseInline`: parseInline은 "key: " 뒤나 "- " 뒤의 값을 읽는다. 플로우 컬렉션, 따옴표 문자열, 평문 스칼라.
- `parser.continuePlain`: continuePlain은 평문 스칼라 n 뒤에서 indent보다 깊게 들여 쓴 줄을 이어 붙인다. 줄바꿈은 공백 하나가 된다. 키나 목록 항목처럼 보이는 줄은 이어 붙이지 않아 들여쓰기 오류로 남는다.

## internal/yaml/flow.go

- `fc`: fc는 위치가 붙은 바이트다. 여러 줄 [ ]를 한 스트림으로 이어 읽기 위해 쓴다.
- `flowDepth`: flowDepth는 따옴표 밖의 괄호 깊이를 누적한다. 줄 끝에서 따옴표가 열려 있으면 오류다.
- `flow.plainText`: plainText는 stops 중 하나 앞까지 읽고 오른쪽 공백을 뗀다.
- `flow.plain`: 플로우 문맥의 평문 스칼라에는 ": "가 올 수 없다. 대개 쉼표가 빠진 것이다.
- `flow.afterItem`: afterItem은 항목 뒤의 ',' 또는 닫는 괄호를 처리한다. 닫혔으면 true.

## internal/yaml/scalar.go

- `unquote`: unquote는 s[0] == '"'인 문자열을 읽어 값, 소비한 바이트 수, 오류 메시지를 준다. 오류일 때 n은 오류 위치 오프셋이다.
- `unsupportedScalar`: unsupportedScalar는 평문 스칼라가 지원하지 않는 문법으로 시작하면 메시지를 준다.

## internal/yaml/scan.go

- `line.indent`: 앞 공백 수
- `line.text`: 들여쓰기와 주석을 뺀 본문. 오른쪽 공백 제거
- `stripComment`: stripComment는 따옴표 밖에 있고 줄 처음이거나 앞이 공백인 '#'부터 자른다.

## internal/yaml/yaml.go

- `package yaml`: Package yaml은 daemok 설계 파일이 쓰는 YAML 부분집합을 파싱한다. 모든 노드가 파일·줄·열 위치를 갖는다. 열은 바이트 기준 1-base다.
- `Pos.Less`: Less는 파일, 줄, 열 순으로 p가 q보다 앞인지다.
- `Node.Value`: Scalar. 따옴표 문자열은 이스케이프를 풀어 담는다.
- `Node.Quoted`: Scalar가 "..."였는지
- `Entry.Pos`: 키 위치
