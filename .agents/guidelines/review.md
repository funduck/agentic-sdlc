# Review guidelines

Deeper guidance for the Review stage. Apply alongside `core.md`. Review is the last gate before
ship — judge the whole against requirements and design, and route any improvement to its owner.

## What to inspect

- **Correctness** — does it actually satisfy every requirement and match the design? Reason about
  edge cases and failure modes, not just the happy path.
- **Readability & maintainability** — clear names, small functions, no needless complexity, would the
  next developer understand it quickly?
- **Security** — no secrets, inputs validated, no obvious injection/leak; least privilege.
- **Test adequacy** — do the tests cover the real behavior and edge cases, and do they pass? Missing
  coverage is a finding.
- **Consistency** — does it match the repo's conventions and idioms?

## How to review

- **Distinguish blocking from nit.** Only block on things that affect correctness, security, or
  maintainability. Don't hold up a task for style preferences — mention nits, don't gate on them.
- **Route to the owner.** An improvement in code goes to Implementation; a flawed design goes to
  Design; a wrong requirement goes to Requirements. Don't fix another stage's problem in place.
- Be specific: point at the exact concern and why it matters, so the owning stage can act without
  guessing.
