---
name: korean-prose-lint
description: >-
  Run the KoreanProse Vale style package and interpret its JSON findings for
  Korean technical prose, README files, GitHub issues, and pull requests. Use
  this skill whenever a user asks to lint, check, or review Korean prose with
  korean-prose-lint or KoreanProse rules. It requires Vale and must report an
  unavailable or failed backend instead of imitating the rules in the prompt.
---

# Korean prose lint

Use Vale as the backend and explain its results. The `KoreanProse` rule files
remain the source of executable detection behavior; do not reproduce their
patterns in this skill.

## Use this skill for

- Deterministic style linting of Korean technical prose and Markdown.
- Explaining `KoreanProse.<RuleName>` findings from Vale JSON.
- Distinguishing a clean lint run from a run that could not start or finish.

Do not use it as a general spelling or grammar checker, formatter, rewriting
assistant, or autofixer. If Vale is unavailable, do not silently substitute an
LLM-only review and call it a successful lint run.

## Run the backend

1. Identify the repository root, the intended `.vale.ini`, and whether the
   request supplies files or direct text. Do not guess a different config or
   broaden the input set.
2. Confirm the config and every requested file exist and are readable before
   invoking Vale. Classify a missing or unreadable config as
   `failed/configuration_error`; classify a missing or unreadable input as
   `failed/input_error`.
3. Resolve `vale` without installing software or changing the user's system.
   If no executable is available, stop with `not_run/backend_unavailable` and
   provide a Vale installation link plus the exact command to retry.
4. Run `vale --version`. Record the actual version. Vale 3.17.1 is required as
   the installed backend for the supported baseline. Version 3.17.1 is the
   supported baseline; for another version, warn about the mismatch but still
   run and interpret the result unless Vale itself fails. If its result is
   known to differ from the 3.17.1 fixture behavior, treat the 3.17.1 fixture as
   the compatibility reference and report the discrepancy.
5. Confirm the active config includes the required KoreanProse consumer
   profile, including its documented Markdown `TokenIgnores`. Use `vale
   --config=<path> --no-global ls-config` when the effective configuration is
   uncertain.
6. For files, execute from the repository root:

   ```bash
   vale --config=<path> --no-global --no-exit --output=JSON -- <inputs...>
   ```

   For direct Markdown text, pipe the exact text to Vale and assign a synthetic
   Markdown path so syntax-aware rules run:

   ```bash
   printf '%s' "$INPUT_TEXT" | vale --config=<path> --no-global --no-exit \
     --output=JSON --ext=.md --path='<stdin>.md'
   ```

   Do not interpolate untrusted text into a shell command; provide it through a
   safely passed value or process stdin. Normalize Vale's `<stdin>.md` output
   key to the contract path `<stdin>`.

   `--no-exit` prevents findings from becoming a process failure. It does not
   make configuration, input, or backend errors successful.
7. Capture stdout, stderr, and the exit code separately. A nonzero backend exit
   is `failed`: use `configuration_error` when Vale identifies invalid or
   unloadable configuration, `input_error` for an input failure not caught in
   step 2, and `backend_error` otherwise.
8. Decode stdout as Vale JSON only after a successful process exit. Invalid or
   structurally unusable JSON is `failed/normalization_error`; do not report it
   as zero findings.

## Interpret findings

Project each Vale alert into these terms:

- `path`: repository-relative POSIX path
- `rule_id`: Vale `Check`
- `severity`: Vale `Severity`
- `message`: Vale `Message`
- `line`: 1-based line
- `span_start`, `span_end`: Vale's 1-based inclusive `Span`
- `match`: Vale `Match`

Keep duplicate findings and sort explanations by path, line, span start, span
end, then rule ID. Explain the rule ID, location, matched text, severity, and
message. Do not infer an automatic replacement or edit files.

## Report the lint result

Always distinguish these outcomes:

- `completed`: Vale ran and JSON normalization succeeded. An empty `findings`
  array means the checked inputs had no reported violations.
- `not_run`: the backend was unavailable and linting never started. Use reason
  `backend_unavailable`.
- `failed`: linting did not produce a trustworthy result. Use one of
  `configuration_error`, `input_error`, `backend_error`, or
  `normalization_error`, and include a concise `detail` plus a retry action.

Every reported lint result contains `status`, `backend.name`, and `findings`.
`findings` is always an array. For `not_run` and `failed`, discard any partial
alerts and return `findings: []`; include `reason`. For `completed`, omit
`reason`, including when `findings` is empty. Include `backend.version` only
when it was observed.

Report `backend.name` as `vale` and include `backend.version` when it was
observed. If the version differs from 3.17.1, keep the actual version and add a
support-baseline warning; do not rewrite it to the expected version.

For `completed`, summarize the number of checked inputs and findings, then list
findings in normalized order. For `not_run` or `failed`, lead with the status and
reason so the user cannot confuse an empty result with a clean document.
