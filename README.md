# korean-prose-lint

[English](README.en.md)

`korean-prose-lint`는 명료하고 일관된 한국어 산문을 위한 결정론적 스타일 린터입니다.
한국어 맞춤법·문법 전체를 검사하거나 문서를 포맷하거나 AI가 문장을 다시 쓰게 하는
도구가 아니라 ESLint·markdownlint에 가까운 도구입니다.

첫 릴리스는 [Vale](https://vale.sh/)에서 실행하는 `KoreanProse` style package와 Vale의
JSON 결과를 해석하는 작은 Agent Skill을 제공합니다. 지원 및 CI 고정 기준은 Vale
3.17.1입니다.

## 설치

1. [Vale 3.17.1](https://github.com/vale-cli/vale/releases/tag/v3.17.1)을 설치하고
   버전을 확인합니다.

   ```console
   vale --version
   ```

   지원하는 출력은 `vale version 3.17.1`입니다. 다른 버전에서도 실행할 수 있지만
   best-effort이며, 결과가 다르면 3.17.1 fixture 행동을 기준으로 비교해야 합니다.

2. `KoreanProse/` 디렉터리를 사용할 저장소의 style 디렉터리로 복사합니다.

   ```console
   mkdir -p styles
   cp -R /path/to/korean-prose-lint/KoreanProse styles/KoreanProse
   ```

3. 검사할 저장소의 `.vale.ini`에 다음 지원 consumer profile을 작성합니다.

   ```ini
   StylesPath = styles
   MinAlertLevel = suggestion

   [*.md]
   BasedOnStyles = KoreanProse
   TokenIgnores = (?i)((?:https?://|www\.)[^\s<]+)
   ```

`TokenIgnores` 항목은 필수입니다. Vale rule YAML만으로 설정할 수 없는 URL 경계에서도
규칙 계약을 보존합니다. 이 profile 없이 style 폴더만 활성화한 실행은 지원하는 backend
구성이 아닙니다.

## 실행

consumer 저장소 루트에서 파일이나 디렉터리를 검사합니다.

```console
vale --no-global --no-exit --output=JSON README.md docs/
```

`--no-exit`는 finding과 backend 실패를 구분하기 위해 사용합니다. Vale가 성공적으로
실행되고 JSON을 해석한 경우에만 빈 결과를 위반 없음으로 판단해야 합니다.

## 규칙

| 규칙 ID | 기본 심각도 | 탐지 대상 |
|---|---|---|
| [`KoreanProse.DoubleSpace`](docs/rules.md#koreanprosedoublespace) | warning | 비공백 문자 사이에서 ASCII U+0020 공백이 두 칸 이상 이어진 경우 |
| [`KoreanProse.SentenceSpacing`](docs/rules.md#koreanprosesentencespacing) | warning | 한글 문장 종결 부호와 다음 한글 사이의 무공백 |
| [`KoreanProse.RepeatedPunctuation`](docs/rules.md#koreanproserepeatedpunctuation) | suggestion | `!`와 `?`가 두 개 이상 이어지는 반복 강조 |
| [`KoreanProse.RedundantExpression`](docs/rules.md#koreanproseredundantexpression) | suggestion | 명시적으로 지원하는 중복 표현 표면형 세 개 |

일반 Markdown 문단, 제목과 목록 항목은 검사합니다. 인라인 코드, fenced·들여쓰기 코드
블록과 consumer profile이 처리하는 URL은 검사하지 않습니다. 수평 tab은 `DoubleSpace`
finding이 아닙니다. 입력은 UTF-8 NFC 한글을 전제로 합니다. 정확한 행동 계약은
[`rules.json`](rules.json)과 각 규칙의 fixture를 참고하세요.

Vale 설정에서 개별 규칙의 심각도나 활성 상태를 바꿀 수 있습니다.

```ini
[*.md]
BasedOnStyles = KoreanProse
TokenIgnores = (?i)((?:https?://|www\.)[^\s<]+)
KoreanProse = error
KoreanProse.DoubleSpace = suggestion
KoreanProse.RedundantExpression = NO
```

`KoreanProse = error`는 먼저 style 전체의 심각도를 바꾸며, 뒤의 규칙별 항목은 개별
규칙을 다시 override하거나 비활성화합니다.

이 override는 Vale 설정 기능입니다. backend 간 v0.1 적합성 계약은 catalog의 기본값을
비교합니다.

규칙별 정상·위반 예시와 경계 조건은 [규칙 안내](docs/rules.md)를 참고하세요.

## GitHub Actions

consumer 저장소의 workflow에서 Vale를 설치한 뒤, 이 저장소의 `KoreanProse/`와
지원 `.vale.ini` profile을 검사 대상 저장소에 제공하고 다음 명령을 실행할 수
있습니다.

```yaml
- name: Run Korean prose lint
  run: vale --no-global --no-exit --output=JSON README.md docs/
```

이 프로젝트의 [test workflow](.github/workflows/test.yml)는 Vale 3.17.1을 checksum
검증하고 전체 conformance suite를 실행하는 패키지 자체 검증 예시입니다.

## Agent Skill

[`skills/korean-prose-lint/`](skills/korean-prose-lint/)를 사용하는 에이전트가 지원하는
skill 디렉터리에 복사한 뒤 `korean-prose-lint`로 한국어 산문을 검사해 달라고 요청합니다.
Skill은 두 번째 lint engine이 아니라 adapter이므로 Vale와 지원 consumer profile이 계속
필요합니다.

Skill은 결과를 다음처럼 구분합니다.

- `completed`: Vale 실행과 JSON 해석이 끝났습니다. finding 0건은 검사한 입력에 보고된
  위반이 없다는 뜻입니다.
- `not_run/backend_unavailable`: Vale를 사용할 수 없어 lint를 실행하지 못했습니다.
- `failed`: 설정·입력·backend·정규화 문제로 신뢰할 수 있는 결과를 만들지 못했습니다.

Skill은 Vale를 자동 설치하거나 LLM으로 규칙을 흉내 내거나 파일을 재작성하거나 자동
수정을 적용하지 않습니다.

## 범위

현재 릴리스는 기술 문서, README, GitHub 이슈와 PR 산문을 대상으로 합니다. 맞춤법·문법
전체 검사, 형태소 분석, 독자 CLI, native backend, 에디터 플러그인과 Package Explorer
배포는 제공하지 않습니다.

## 개발

Vale 3.17.1로 적합성 테스트를 실행합니다.

```console
VALE_BIN=/path/to/vale go test ./...
```

Go 코드는 테스트 harness일 뿐 제품 CLI가 아닙니다. 공개 용어의 정본은
[도메인 언어 사전](docs/domain-language.md)입니다.
