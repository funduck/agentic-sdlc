# AGENTS.md

Orientation for any agent (Claude Code or otherwise) working in this repo.

## Start here

Read [README.md](README.md) first — it explains the multiagent SDLC workflow this repo
implements: the six pipeline agents (Requirements → Design → Pre-QA → Implementation → QA →
Review), the routing principles (route to the owner, escalate ambiguities, iteration budgets,
versioned artifacts), and the full state diagram.

**Do not re-implement or bypass the workflow it describes.** The orchestrator, not an LLM, is the
thing enforcing loop budgets and state transitions — control flow lives in code
([.claude/scripts/orchestrator/orchestrator.py](.claude/scripts/orchestrator/orchestrator.py)). It is a pure **instruction printer**: it never
launches agents. A single Claude session drives the pipeline — it asks for the next step (`next`),
dispatches the stage's subagent via the Task tool, and feeds the verdict back (`record`). The subagents
are stateless workers that write files and return a verdict.

## Project structure

```
guidelines/              engineering best practices referenced by the subagents: core.md (shared:
                          KISS/DRY/YAGNI/SOLID/…) + per-domain requirements/design/testing/
                          implementation/review files

.agents/
  <task_id>/              per-task persisted state: append-only state.json history plus the
                          versioned task.md / requirements.md / design.md artifacts and verdict.json —
                          committed to the repo as work artifacts

.claude/agents/          idiomatic subagents, one per pipeline stage (sdlc-requirements, sdlc-design,
                          sdlc-pre-qa, sdlc-implementation, sdlc-qa, sdlc-review) — the persona +
                          verdict contract, each referencing the guidelines it applies

.claude/commands/        slash-command wrappers around the orchestrator CLI:
  add-task.md             /add-task <task-id> <requirements...>
  run-task.md              /run-task <task-id>            (drives next → subagent → record)
  confirm-task.md          /confirm-task <task-id> [--approve | --request-changes "..."]
  status-task.md           /status-task <task-id>

.claude/scripts/orchestrator/
  orchestrator.py         deterministic state machine; prints next-step instructions (see README)
  test/scenarios/         scripted stub verdict sequences for exercising the orchestrator
                          without any API calls (happy path, budget exhaustion, routing examples)
```

**Human-approval gate.** After the Requirements Agent runs, the pipeline always pauses at the
`CONFIRM_REQUIREMENTS` state (status `awaiting_approval`) and waits for a human to approve
`requirements.md` — assumptions and all — before Design starts. Resolve it with `/confirm-task` (or
`orchestrator.py confirm`): `--approve` proceeds to Design; `--request-changes "..."` sends it back to
Requirements with that feedback and returns to the gate. This is a deliberate design invariant — do
not route around it or auto-advance requirements.

Equivalent skills (`add-task`, `run-task`, `status-task`) are also registered for direct
invocation.

## Running the workflow

Real runs happen inside a Claude session via the slash commands — `/run-task` drives the loop by
asking the orchestrator for each step and dispatching the stage's subagent. Under the hood, all
`orchestrator.py` commands default to persisting state under `.agents/` at the repo root, regardless
of the caller's cwd:

```bash
python3 .claude/scripts/orchestrator/orchestrator.py add --task <id> <<< "requirements text"
python3 .claude/scripts/orchestrator/orchestrator.py next --task <id>       # read-only: what to run next
python3 .claude/scripts/orchestrator/orchestrator.py record --task <id>     # record verdict.json, advance, print next
python3 .claude/scripts/orchestrator/orchestrator.py confirm --task <id> --approve
python3 .claude/scripts/orchestrator/orchestrator.py status --task <id>
```

The driver is **thin**: it relays only what the orchestrator prints (`last_summary` + the next
instruction) and never parses a subagent's raw output — the orchestrator is the single source of
routing truth. For development/testing, the state machine can be exercised deterministically with
zero API calls and no subagents via `run --stub-scenario .claude/scripts/orchestrator/test/scenarios/<name>.json`
(see README "Orchestrator" for examples).

## Conventions

- The **requirements document** and **design document** are versioned and each owned by exactly
  one stage (Requirements Agent, Design Agent respectively). Only the owning agent amends its
  document; treat these as the source of truth for a task's current scope, not something to edit
  directly.
- When you discover a defect while working on any stage, classify it (`requirement` / `design` /
  `implementation`) and route it to the owning stage rather than patching it locally — this is the
  core routing principle the orchestrator enforces.
- State in `.agents/<task_id>/state.json` is append-only; don't hand-edit it. Use
  `orchestrator.py status` or `/status-task` to inspect it.
