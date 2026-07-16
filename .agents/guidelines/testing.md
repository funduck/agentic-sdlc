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
- **test** — the code and design are both right; the *test case itself* is wrong — a bad expected
  value, an unverified assumption, or a flaky test → Pre-QA (which authored the tests). Example: a
  test asserting `10 - 0.0000000005 == 10` when float64 correctly yields `9.999999999`. Fix the test,
  don't bend the code to a wrong expectation.

Route to the *owner*, not merely the previous stage.

## Don't re-litigate accepted deviations

Before classifying a mismatch as a failure, check the task's `deviations.md` (if present). It records
behaviors already reviewed and **accepted** as trade-offs (e.g. float64 precision limits), with the
rationale. A listed deviation is *not* a failure — confirm it still holds and move on; don't re-derive
the analysis from scratch. If you accept a *new* deviation, add a row there rather than re-explaining
it in every downstream verdict.
