# AGENTS.md

Orientation for any agent (Claude Code or otherwise) working in this repo.

## Start here

Read [README.md](README.md) first — it explains the multiagent SDLC workflow this repo
implements: the six pipeline agents (Requirements → Design → Pre-QA → Implementation → QA →
Review), the routing principles (route to the owner, escalate ambiguities, iteration budgets,
versioned artifacts), and the full state diagram.

**Do not re-implement or bypass the workflow it describes.** The orchestrator, not an LLM, is the
thing enforcing loop budgets and state transitions — control flow lives in code
([.agents/orchestrator.py](.agents/orchestrator.py)), agents are stateless workers that return a
verdict.

## Project structure

```
.agents/
  orchestrator.py       deterministic state machine driving the pipeline (see README "Orchestrator")
  prompts/               one prompt file per pipeline stage (requirements.md, design.md,
                          pre_qa.md, implementation.md, qa.md, review.md) — these define what
                          each agent is asked to do when invoked via `claude -p`
  scenarios/              scripted stub verdict sequences for exercising the orchestrator
                          without any API calls (happy path, budget exhaustion, routing examples)
  state/<task_id>/        per-task persisted state: append-only state.json history plus the
                          versioned task.md / requirements.md / design.md artifacts

.claude/commands/        thin slash-command wrappers around the orchestrator CLI:
  add-task.md             /add-task <task-id> <requirements...>
  run-task.md              /run-task <task-id> [--restart]
  confirm-task.md          /confirm-task <task-id> [--approve | --request-changes "..."]
  status-task.md           /status-task <task-id>
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

All `orchestrator.py` commands are run from `.agents/` so its default `state/` and `prompts/`
paths resolve:

```bash
cd .agents && python3 orchestrator.py add --task <id> <<< "requirements text"
cd .agents && python3 orchestrator.py run --task <id> [--restart]
cd .agents && python3 orchestrator.py status --task <id>
```

For development/testing, the pipeline can be exercised deterministically with stubbed agents and
zero API calls via `--stub-scenario .agents/scenarios/<name>.json` (see README "Orchestrator" for
examples). Real runs invoke agents through `claude -p` using the prompts in `.agents/prompts/`.

## Conventions

- The **requirements document** and **design document** are versioned and each owned by exactly
  one stage (Requirements Agent, Design Agent respectively). Only the owning agent amends its
  document; treat these as the source of truth for a task's current scope, not something to edit
  directly.
- When you discover a defect while working on any stage, classify it (`requirement` / `design` /
  `implementation`) and route it to the owning stage rather than patching it locally — this is the
  core routing principle the orchestrator enforces.
- State in `.agents/state/<task_id>/state.json` is append-only; don't hand-edit it. Use
  `orchestrator.py status` or `/status-task` to inspect it.
