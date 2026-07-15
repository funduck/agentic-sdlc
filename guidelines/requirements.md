# Requirements guidelines

Deeper guidance for the Requirements stage. Apply alongside `core.md`.

## Write good requirements — INVEST

Each requirement should be:

- **Independent** — standalone, minimal coupling to others.
- **Negotiable** — a statement of need, not a locked-in solution.
- **Valuable** — traceable to real user or business value.
- **Estimable** — clear enough that effort can be judged.
- **Small** — a single, cohesive capability; split anything sprawling.
- **Testable** — has an observable acceptance criterion. If you can't write a test for it, it's not a
  requirement yet.

## Say what, not how

- Capture the *problem* and the *observable outcome*, not an implementation. Leave "how" to Design.
- No premature technology, schema, or algorithm choices unless the task genuinely constrains them.

## Be unambiguous

- One reading only. Kill vague words ("fast", "user-friendly", "handle errors") — replace with
  measurable criteria.
- Define every term that could be read two ways.

## Surface assumptions, don't bury them

- Every decision the task didn't state explicitly goes in **Assumptions** — visible, minimal, only
  what's needed to proceed.
- Anything material you genuinely can't decide goes in **Open Questions** — never silently guess a
  core behavior. Surfacing an assumption for the human to confirm is the goal, not a failure.
