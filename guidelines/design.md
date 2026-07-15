# Design guidelines

Deeper guidance for the Design stage. Apply alongside `core.md`.

## Design to the requirements — no more

- Cover every requirement; add nothing the requirements don't call for (**YAGNI**). Extensibility is a
  cost; add a seam only where a real, near-term need justifies it.
- If designing reveals a gap or contradiction in the requirements you can't resolve, **route it back**
  to Requirements — don't invent a requirement to paper over it.

## Apply SOLID at the boundaries

- **Single responsibility** per component; a component should have one reason to change.
- **Depend on abstractions** at real seams (storage, external services, I/O) so pieces are swappable
  and testable — but don't abstract a boundary that doesn't exist yet.
- Keep interfaces small and focused; prefer composition over deep inheritance.

## Clear boundaries and data flow

- Define modules/components, their responsibilities, and the contracts between them.
- Make the data flow explicit: inputs, outputs, ownership of state, and where side effects happen.
- Isolate side effects (I/O, network, persistence) from pure logic.

## Design for testability

- Structure so behavior can be exercised in isolation (dependency seams, pure cores).
- Note the key test points so Pre-QA/QA know where to aim.

## Document trade-offs

- State the alternatives you weighed and why you chose this one. A design the next agent can't reason
  about is a liability.
