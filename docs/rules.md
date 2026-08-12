# 규칙 안내

[English](rules.en.md)

`KoreanProse`의 규칙은 한국어 기술 문서·README·GitHub 이슈와 PR 산문에서
기계적으로 판정할 수 있는 표면 패턴만 다룹니다. 이 패키지는 맞춤법·문법
전체를 판정하거나 문장을 자동으로 다시 쓰지 않습니다.

규칙의 식별자·기본 심각도·정본 fixture는 저장소 루트의 [`rules.json`](../rules.json)과
[`fixtures/`](../fixtures/)에 있습니다.

## `KoreanProse.DoubleSpace`

| 항목 | 내용 |
|---|---|
| 기본 심각도 | `warning` |
| 탐지 | 비공백 문자 사이의 ASCII U+0020 공백 두 칸 이상 |
| 제외 | 수평 tab, 코드 영역, URL |

```text
문장 사이에  공백이 두 칸 있습니다.
```

```text
문장 사이에 공백이 하나 있습니다.
```

## `KoreanProse.SentenceSpacing`

| 항목 | 내용 |
|---|---|
| 기본 심각도 | `warning` |
| 탐지 | 한글 뒤 문장 종결 부호(`.`, `!`, `?`)와 다음 한글 사이의 무공백 |
| 제외 | 문장 부호 뒤가 한글이 아닌 경우, 코드 영역, URL |

```text
문장.다음 문장입니다.
```

```text
문장. 다음 문장입니다.
```

## `KoreanProse.RepeatedPunctuation`

| 항목 | 내용 |
|---|---|
| 기본 심각도 | `suggestion` |
| 탐지 | `!` 또는 `?`가 두 개 이상 연속된 경우 |
| 제외 | 코드 영역, URL 내부의 구두점 |

```text
정말요?!
```

```text
정말요?
```

## `KoreanProse.RedundantExpression`

| 항목 | 내용 |
|---|---|
| 기본 심각도 | `suggestion` |
| 탐지 | 현재 명시적으로 지원하는 중복 표현 세 가지 |
| 지원 표면형 | `미리 사전에`, `다시 재시도합니다`, `다시 재시도` |

```text
이 값을 미리 사전에 확인합니다.
```

```text
이 값을 사전에 확인합니다.
```

지원 표면형은 의도적으로 닫힌 목록입니다. 새로운 표현을 추가하려면 규칙 YAML,
catalog, valid·invalid fixture와 expectation manifest를 함께 변경해야 합니다.

## Markdown과 URL 경계

일반 문단·제목·목록은 검사하지만 인라인 코드, fenced·들여쓰기 코드 블록은
검사하지 않습니다. URL 경계 보존은 Vale rule YAML이 아니라 consumer의
`TokenIgnores` profile이 담당하므로, [README의 지원 profile](../README.md)을
그대로 사용해야 합니다.
