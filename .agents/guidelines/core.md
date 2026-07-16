# Core engineering guidelines

Cross-cutting principles every SDLC agent applies, whatever the stage. Read this first, then your
stage-specific guideline. Keep the bar high but stay pragmatic — these are defaults, not dogma.

## Simplicity

- **KISS** — prefer the simplest thing that fully solves the problem. Complexity must earn its place.
- **YAGNI** — build what the requirements ask for, not what they might ask for later. No speculative
  abstraction, config, or extensibility "just in case".
- **DRY, but not prematurely** — remove real duplication of *knowledge*; don't hoist an abstraction
  from two lines that merely look alike. A little repetition beats the wrong abstraction.

## Structure

- **SOLID where it pays** — single responsibility per unit; depend on interfaces/abstractions at real
  boundaries; keep modules open to extension without invasive edits. Apply with judgment, not ritual.
- **Separation of concerns** — one module/function does one job. Keep I/O, business logic, and
  presentation from bleeding into each other.
- **Small, focused functions** — a function should do one thing at one level of abstraction. If you
  can't name it clearly, it's doing too much.

## Clarity

- **Names carry intent** — say what a thing *is* or *does*; avoid abbreviations and cleverness. Code
  is read far more than it's written.
- **Match the surrounding code** — mirror the existing style, idioms, naming, and structure of the
  repo. Consistency beats personal preference.
- **Comment the *why*, not the *what*** — explain intent, trade-offs, and non-obvious constraints; let
  clear code explain the mechanics.

## Robustness

- **Handle errors explicitly** — validate inputs at boundaries, fail fast with clear messages, don't
  swallow exceptions. Make invalid states hard to represent.
- **No dead code** — delete unused code, commented-out blocks, and TODOs you can just do now.

## Security

- **Never hardcode secrets** — no credentials, tokens, or keys in code or logs.
- **Trust no input** — validate and sanitize anything crossing a boundary (user, network, file).
- **Least privilege** — request only the access a task needs.

## Testing mindset

- Design and write so the result is **testable**: pure logic separable from side effects.
- Every behavior worth relying on is worth a test. See `testing.md` for how QA/Pre-QA apply this.

## Stay in your lane

- When you find a defect you don't own, **route it to the owner** (requirement / design /
  implementation) rather than papering over it locally. Don't invent facts to fill a gap you can't own.
