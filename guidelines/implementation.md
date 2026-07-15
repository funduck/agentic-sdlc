# Implementation guidelines

Deeper guidance for the Implementation stage. Apply alongside `core.md`.

## Build the design, not your own idea of it

- Implement what `design.md` specifies. If the design is wrong or missing something you can't resolve,
  **route it back** to Design rather than improvising a different solution.

## Write clean code

- **Small functions**, one job each, one level of abstraction. Extract rather than nest deeply.
- **Intent-revealing names** for variables, functions, types. No cryptic abbreviations.
- **Match the repo** — follow the existing structure, style, idioms, and conventions. New code should
  look like it was always there.
- **No dead code** — no commented-out blocks, unused params, or leftover scaffolding.

## Handle errors honestly

- Validate inputs at boundaries; fail fast with a clear message. Don't swallow errors or hide them
  behind a default that masks the problem.
- Free/clean up resources reliably.

## Keep it secure

- No hardcoded secrets. Validate and sanitize external input. Don't log sensitive data.

## Leave it testable and verified

- Structure so the logic can be unit-tested (pure cores, injected dependencies).
- Actually run what you wrote before handing off — build/lint/smoke it. Implementation doesn't
  self-judge the result, but it must not hand QA something that doesn't run.
