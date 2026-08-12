# Rule guide

[한국어](rules.md)

`KoreanProse` covers only mechanically detectable surface patterns in Korean
technical documentation, README files, and GitHub issue or pull-request prose.
It is not a complete spelling or grammar checker and does not rewrite text.

Rule identifiers, default severities, and canonical fixtures live in
[`rules.json`](../rules.json) and [`fixtures/`](../fixtures/).

## `KoreanProse.DoubleSpace`

| Item | Description |
|---|---|
| Default severity | `warning` |
| Reports | Two or more ASCII U+0020 spaces between non-whitespace characters |
| Excludes | Horizontal tabs, code regions, and URLs |

```text
There are  two spaces here.
```

```text
There is one space here.
```

## `KoreanProse.SentenceSpacing`

| Item | Description |
|---|---|
| Default severity | `warning` |
| Reports | No space between Korean sentence-ending punctuation (`.`, `!`, `?`) and the next Hangul character |
| Excludes | Non-Hangul text after punctuation, code regions, and URLs |

```text
문장.다음 문장입니다.
```

```text
문장. 다음 문장입니다.
```

## `KoreanProse.RepeatedPunctuation`

| Item | Description |
|---|---|
| Default severity | `suggestion` |
| Reports | Two or more consecutive `!` or `?` characters |
| Excludes | Code regions and punctuation inside URLs |

```text
정말요?!
```

```text
정말요?
```

## `KoreanProse.RedundantExpression`

| Item | Description |
|---|---|
| Default severity | `suggestion` |
| Reports | Three explicitly supported redundant-expression surface forms |
| Supported forms | `미리 사전에`, `다시 재시도합니다`, `다시 재시도` |

```text
이 값을 미리 사전에 확인합니다.
```

```text
이 값을 사전에 확인합니다.
```

The supported forms are intentionally a closed list. Adding a new form requires
updating the rule YAML, catalog, valid and invalid fixtures, and expectation
manifest together.

## Markdown and URL boundaries

Ordinary paragraphs, headings, and list items are checked. Inline code, fenced
and indented code blocks are excluded. URL boundaries are preserved by the
consumer's `TokenIgnores` profile rather than by Vale rule YAML, so use the
[supported profile in the README](../README.en.md) unchanged.
