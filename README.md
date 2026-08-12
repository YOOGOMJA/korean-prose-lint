# korean-prose-lint

[한국어](README.ko.md)

`korean-prose-lint` is a deterministic style linter for clear, consistent
Korean prose. It is closer to ESLint or markdownlint than a complete Korean
spelling and grammar checker, formatter, or AI rewriting prompt.

The initial release provides a `KoreanProse` [Vale](https://vale.sh/) style
package and a small Agent Skill that runs Vale and interprets its JSON output.
Vale 3.17.1 is the supported and CI-pinned baseline.

## Install

1. Install [Vale 3.17.1](https://github.com/vale-cli/vale/releases/tag/v3.17.1)
   and confirm the version:

   ```console
   vale --version
   ```

   The supported output is `vale version 3.17.1`. Other versions may run, but
   are best-effort and should be compared with the 3.17.1 fixture behavior when
   results differ.

2. Copy the `KoreanProse/` directory into a local styles directory:

   ```console
   mkdir -p styles
   cp -R /path/to/korean-prose-lint/KoreanProse styles/KoreanProse
   ```

3. Add this supported consumer profile as `.vale.ini` in the repository to
   lint:

   ```ini
   StylesPath = styles
   MinAlertLevel = suggestion

   [*.md]
   BasedOnStyles = KoreanProse
   TokenIgnores = (?i)((?:https?://|www\.)[^\s<]+)
   ```

The `TokenIgnores` entry is required. It preserves the rule contract for URL
boundaries that Vale rule YAML cannot configure by itself. A style-only setup
without this profile is not a supported backend configuration.

## Run

Lint files or directories from the consumer repository root:

```console
vale --no-global --no-exit --output=JSON README.md docs/
```

`--no-exit` keeps findings separate from backend failures. Do not interpret an
empty output as clean unless Vale ran successfully and its JSON was decoded.

## Rules

| Rule ID | Default severity | Reports |
|---|---|---|
| `KoreanProse.DoubleSpace` | warning | Runs of two or more ASCII U+0020 spaces between non-whitespace characters |
| `KoreanProse.SentenceSpacing` | warning | No space between Korean sentence-ending punctuation and the next Hangul character |
| `KoreanProse.RepeatedPunctuation` | suggestion | Runs of two or more `!` and `?` characters |
| `KoreanProse.RedundantExpression` | suggestion | Three explicitly supported redundant-expression surface forms |

Ordinary Markdown paragraphs, headings, and list items are checked. Code spans,
fenced and indented code blocks, and URLs covered by the consumer profile are
excluded. A horizontal tab is not a `DoubleSpace` finding. Inputs are expected
to be UTF-8 with NFC Korean text. See [`rules.json`](rules.json) and each rule's
fixtures for the exact behavior contract.

Vale supports rule-specific configuration after `BasedOnStyles`:

```ini
[*.md]
BasedOnStyles = KoreanProse
TokenIgnores = (?i)((?:https?://|www\.)[^\s<]+)
KoreanProse = error
KoreanProse.DoubleSpace = suggestion
KoreanProse.RedundantExpression = NO
```

`KoreanProse = error` changes the whole style first; the following rule-specific
entries override or disable individual rules.

These overrides are Vale configuration features; the cross-backend v0.1
conformance contract compares the catalog defaults.

## Agent Skill

Copy [`skills/korean-prose-lint/`](skills/korean-prose-lint/) into a skill
directory supported by your agent, then ask the agent to lint Korean prose with
`korean-prose-lint`. The Skill is an adapter, not a second lint engine: Vale and
the supported consumer profile are still required.

The Skill distinguishes these outcomes:

- `completed`: Vale ran and JSON decoding succeeded. Zero findings means the
  checked inputs had no reported violations.
- `not_run/backend_unavailable`: Vale was unavailable, so no lint occurred.
- `failed`: configuration, input, backend, or normalization prevented a
  trustworthy result.

It does not install Vale, imitate the rules with an LLM, rewrite files, or apply
automatic fixes.

## Scope

The current release targets technical documentation, README files, and GitHub
issue or pull-request prose. It does not provide comprehensive spelling or
grammar checking, morphological analysis, a standalone CLI, a native backend,
editor plugins, or Package Explorer distribution.

## Development

Run the conformance suite with Vale 3.17.1:

```console
VALE_BIN=/path/to/vale go test ./...
```

The Go code is a test harness only. Public terminology is defined in the
[domain language glossary](docs/domain-language.md).
