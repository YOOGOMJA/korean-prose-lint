# 도메인 언어 사전

이 문서는 `korean-prose-lint`의 문서, JSON 필드, 타입, 함수와 모듈 이름에 사용하는
도메인 언어의 정본이다. 같은 개념에 여러 이름을 붙이지 않고, Vale의 구현 용어와
프로젝트의 도메인 용어를 구분하기 위해 사용한다.

이 사전은 실행 schema가 아니다. 여기에 이름이 있다는 이유만으로 타입, interface,
모듈이나 확장 지점을 만들지 않는다.

## 사용 규칙

1. 공개 타입, 필드, 함수, 모듈과 규칙 이름을 만들기 전에 이 문서를 확인한다.
2. 같은 개념에는 표의 canonical term을 이름의 어휘 뿌리로 사용한다. 새 동의어를
   만들지 않는다.
3. 새 도메인 개념이 코드에 필요하면 구현과 같은 변경에서 정의를 추가한다.
4. 단순 구현 세부, 라이브러리 용어와 한 번만 쓰는 이름은 추가하지 않는다.
5. 사전과 코드·설계의 의미가 어긋나면 한쪽을 암묵적으로 우선하지 않고 같은 변경에서
   함께 바로잡는다.
6. 사전 추가는 구현 허가가 아니다. 현재 계약이나 fixture가 요구하지 않는 개념은
   YAGNI에 따라 구현하지 않는다.
7. 표의 canonical term은 타입 생성 지시가 아니다. 타입·함수·필드가 실제로 필요할
   때만 언어 관례에 따라 `Finding`, `lint_result`, `normalizeFinding`처럼 파생한다.
   `rule_id`, `contract_version`처럼 표에 명시된 공개 JSON 필드만 정확한 identifier로
   고정한다.

## 핵심 규칙 언어

| 한국어 | Canonical term | 정의 | 구분·사용 지침 |
|---|---|---|---|
| 규칙 | `rule` | 한국어 산문에서 한 종류의 위반을 탐지하고 하나의 공개 ID를 갖는 단위 | Vale YAML 파일 자체가 아니라 그 파일이 구현하는 도메인 개념이다. |
| 규칙 ID | `rule ID`, `rule_id` | `KoreanProse.<PascalCaseName>` 형식의 안정된 공개 식별자 | 설정, 문서와 finding의 연결 기준이다. 백엔드가 바뀌어도 유지한다. |
| 규칙 계약 | `rule contract` | 규칙의 의미와 관찰 가능한 동작을 정의하는 `rules.json`, fixture, `expected.json`의 결합 | 수식어 없는 `Contract` 타입을 만들지 않는다. 이 개념 전체가 실제 타입일 필요도 없다. |
| 규칙 카탈로그 | `rule catalog` | 규칙 ID, summary, rationale, scope, 기본 심각도, 예외와 fix 안전성을 담은 `rules.json` | 탐지 패턴이나 실행 연산을 담는 규칙 DSL이 아니다. |
| 규칙 구현 | `rule implementation` | 특정 백엔드에서 규칙 계약을 실행 가능하게 표현한 것 | v0.1에서는 `KoreanProse/*.yml`이다. 규칙 의미의 정본이 아니다. |
| 계약 버전 | `contract version`, `contract_version` | 한 백엔드가 준수한다고 선언하는 rule contract의 버전 | 제품 버전이나 Vale 버전과 구분한다. |
| 적용 범위 | `scope` | 규칙이 검사하는 문서 영역의 의미적 범위 | Vale의 구체적인 `scope` 문법과 동일하다고 가정하지 않는다. |
| 예외 | `exception` | 규칙이 의도적으로 탐지하지 않는 경계 | 테스트 누락이나 억제와 구분하고 정상 fixture로 검증한다. |
| 수정 안전성 | `fix safety`, `fix_safety` | 자동 수정 가능성을 `none`, `review`, `safe`로 표현한 규칙 속성 | v0.1이 자동 수정을 제공한다는 뜻이 아니다. |
| fixture | `fixture` | 규칙 동작을 검증하는 실제 UTF-8 NFC 입력 문서 | 테스트 설정이나 기대 결과를 뜻하지 않는다. `test sample`을 동의어로 만들지 않는다. |
| 기대 결과 manifest | `expectation manifest` | fixture별 기대 finding을 선언하는 `expected.json` | 패턴이나 실행 연산이 없는 적합성 자료다. `rule manifest`라고 부르지 않는다. |

## 진단과 실행 결과

| 한국어 | Canonical term | 정의 | 구분·사용 지침 |
|---|---|---|---|
| finding | `finding` | 정규화된 규칙 위반 한 건 | 프로젝트 코드에서는 `diagnostic`, `issue`, `violation`을 같은 뜻의 이름으로 섞지 않는다. |
| Vale alert | `Vale alert` | Vale가 JSON으로 내보내는 백엔드 원시 진단 | normalizer의 입력이다. 공통 finding과 동일시하지 않는다. |
| 린트 결과 | `lint result` | run status, backend 정보와 finding 배열을 묶은 실행 결과 | 수식어 없는 `Result`를 도메인 이름으로 사용하지 않는다. |
| 실행 상태 | `run status` | `completed`, `not_run`, `failed` 중 하나인 실행 완료 상태 | finding의 severity와 다르다. |
| 심각도 | `severity` | finding의 기본 중요도인 `suggestion`, `warning`, `error` | 실행 성공·실패나 CI 종료 코드를 뜻하지 않는다. |
| 입력 경로 | `path` | lint 실행 기준 디렉터리에 대한 POSIX 상대 경로 | 직접 텍스트 입력은 `<stdin>`을 사용한다. 절대 경로나 OS별 구분자를 노출하지 않는다. |
| 행 | `line` | finding이 시작하는 1-based 입력 행 | 문서 전체 byte offset과 구분한다. |
| 매치 | `match` | 입력에서 규칙이 실제로 탐지한 문자열 | 사용자에게 제안하는 대체 문자열이 아니다. |
| span | `span` | 한 줄 안에서 match가 차지하는 1-based Unicode code point 열 범위 | v0.1에서는 양끝을 포함하고 여러 줄 match를 허용하지 않는다. byte offset이나 화면 폭이 아니다. |
| 실행 사유 | `reason` | `not_run` 또는 `failed`의 기계 판독 가능한 원인 | 자유 서술은 `detail`에 둔다. |

## 실행과 통합

| 한국어 | Canonical term | 정의 | v0.1 상태·구분 |
|---|---|---|---|
| 백엔드 | `backend` | 입력을 실제로 검사해 원시 진단을 만드는 실행 주체 | Vale가 담당한다. Skill이나 CI를 backend라고 부르지 않는다. |
| 정규화기 | `normalizer` | 백엔드 원시 출력을 공통 finding과 lint result로 변환하는 경계 | v0.1 테스트와 Skill 경계에서 사용한다. 규칙을 탐지하거나 사용자 메시지를 작성하지 않는다. |
| 어댑터 | `adapter` | 실행 요청이나 결과 전달을 외부 소비자와 연결하는 계층 | Skill과 CI가 해당한다. 백엔드를 adapter라고 부르지 않는다. |
| 적합성 테스트 | `conformance test` | 정규화된 실제 finding을 expectation manifest와 비교하는 테스트 | v0.1 Go 테스트가 담당한다. Vale 출력 문자열 전체를 고정하는 golden test와 구분한다. |
| 호환 백엔드 | `compatible backend` | 선언한 contract version의 모든 비폐기 규칙과 fixture를 만족하는 백엔드 | Vale의 적합성을 설명하는 현재 용어이며 일부 규칙만 지원하는 구현에는 쓰지 않는다. |
| 스타일 패키지 | `style package` | 사용자가 Vale에 설치하는 `KoreanProse/` 배포 단위 | v0.1 배포 단위다. 저장소 전체나 Skill 패키지와 구분한다. |
| 억제 | `suppression` | 특정 finding을 보고하지 않도록 하는 백엔드 기능 | Vale에는 존재하지만 v0.1 compatible backend 판정 범위 밖이다. |
| 심각도 override | `severity override` | 소비자가 규칙의 기본 심각도를 설정에서 바꾸는 기능 | Vale에는 존재하지만 `rules.json` 기본값 변경과 다르며 v0.1 compatible backend 판정 범위 밖이다. |

## 예약된 미래 용어

다음 용어는 설계 경계를 설명하기 위해 예약하지만 v0.1의 구현 구성요소는 아니다.

| 한국어 | Canonical term | 의미 | v0.1 제약 |
|---|---|---|---|
| 독립 백엔드 | `native backend` | Vale 없이 규칙 계약을 실행하는 저장소의 일반 라이브러리·CLI | interface, crate, package나 빈 디렉터리를 미리 만들지 않는다. |
| 중립 실행 schema | `engine-neutral execution schema` | 여러 백엔드 구현을 생성할 수 있는 미래의 실행 규칙 정본 | 현재 `rules.json`을 이 이름으로 부르지 않고 schema를 미리 설계하지 않는다. |
| Vale compiler | `Vale compiler` | 중립 실행 schema를 Vale YAML로 변환하는 미래 구성요소 | 두 번째 백엔드와 코드 생성 필요가 확인되기 전에는 만들지 않는다. |

## 피해야 할 모호한 이름

- `Contract`: 등록된 `rule contract`처럼 대상을 붙인다. 다른 계약 개념이 실제로
  필요하면 이 사전에 qualified term을 먼저 정의한다.
- `Result`: 실행 전체에는 `lint result`, 위반 한 건에는 `finding`을 이름의 뿌리로 쓴다.
- `Engine`: 실행 주체는 기본적으로 `backend`를 쓴다. 외부 라이브러리의 고유 명칭일
  때만 engine을 유지한다.
- `RuleDefinition`: 의미 카탈로그인지 백엔드 구현인지 모호하므로 각각 `rule catalog`와
  `rule implementation`으로 구분한다.
- `Diagnostic`: 외부 프로토콜의 고유 타입을 다룰 때만 사용하고 프로젝트 공통 값은
  `finding`을 쓴다.
