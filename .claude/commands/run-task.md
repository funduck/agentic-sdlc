---
description: Run, resume, or restart a workflow task (drives the pipeline in-session)
argument-hint: <task-id> [--restart]
allowed-tools: Bash, Task
---

Drive a task through the multiagent SDLC workflow. You are a **thin driver**: the orchestrator decides
every step, you dispatch the named subagent, and you relay **only what the orchestrator prints**. Do
not read the workspace files yourself and do not analyze or relay a subagent's returned message — the
orchestrator is the single source of truth.

Arguments: `$ARGUMENTS`

Steps:

1. Parse the task id (first token). If `--restart` is present, note it for step 2.

2. **Get the first instruction.** Run from the repo root (state is persisted under `.agents/` by
   default, regardless of cwd):
   ```bash
   python3 .claude/scripts/orchestrator/orchestrator.py next --task <task-id>
   ```
   (`next` is read-only and launches nothing. `--restart` is a stub-only flag; for a real restart the
   user should `/add-task ... --force` or ask you to reset — do not pass `--restart` to `next`.)

3. **Loop on the instruction's `action`:**
   - **`run_agent`** — dispatch the subagent named in `agent` via the **Task tool**. Build its prompt
     from the instruction's `message` (it already contains the workspace path, project dir, verdict
     path, and any `feedback`). The subagent writes `verdict.json` and returns a one-line ack; **ignore
     that ack**. Then record the verdict and get the next step:
     ```bash
     python3 .claude/scripts/orchestrator/orchestrator.py record --task <task-id>
     ```
     Relay the returned `last_summary` to the user (one line), then repeat step 3 with the new
     instruction.
   - **`await_approval`** — stop. Tell the user the requirements need sign-off and to run
     `/confirm-task <task-id>` (review `requirements.md` first). Design does not start until they do.
   - **`done`** — report the task is complete.
   - **`escalate`** — a loop exhausted its budget. Point the user to `/status-task <task-id>` for the
     history and what to resolve.

4. Keep the running commentary short: one line per stage from `last_summary`, plus the final outcome.
   The user watches each subagent work live in the UI; your job is the concise per-step summary.
