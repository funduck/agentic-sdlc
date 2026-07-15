# Testing guidelines

Shared by the Pre-QA and QA stages. Apply alongside `core.md`. Pre-QA writes these tests *before*
code exists (and uses the act of writing them to surface ambiguity); QA runs them against the
implementation and classifies any failure by root cause.

## Cover the right paths

- **Happy path** — the primary success case for each requirement.
- **Edge & boundary** — empty, zero, one, max, off-by-one, first/last, overflow.
- **Error paths** — invalid input, missing data, failure of a dependency. Assert the *behavior on
  failure*, not just success.
- Trace coverage back to requirements: every acceptance criterion has at least one test.

## Write tests that stay trustworthy

- **Deterministic** — no dependence on time, ordering, network, or randomness. A flaky test is worse
  than no test.
- **Arrange–Act–Assert** — set up, do one thing, assert one outcome. Keep each test readable top to
  bottom.
- **One behavior per test** — a failure name should point straight at what broke.
- **Independent** — tests don't rely on each other's state or run order.

## Classify failures by owner (QA)

Not every failure is the implementation's fault. When a test fails, find the root cause and route it:

- **implementation** — the design is right, the code doesn't match it → Implementation.
- **design** — the code matches the design, but the design itself is flawed → Design.
- **requirement** — the requirement was wrong or misunderstood → Requirements.

Route to the *owner*, not merely the previous stage.
