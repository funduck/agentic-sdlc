---
description: Run, resume, or restart a workflow task (real agents)
argument-hint: <task-id> [--restart]
allowed-tools: Bash
---

Run a task through the multiagent SDLC workflow using the real agents (`claude -p`).

Arguments: `$ARGUMENTS`

Steps:
1. Parse the task id (first token). If `--restart` is present, pass it through to start fresh from the REQUIREMENTS stage.
2. Run the orchestrator from the `.agents/` directory so its default `state/` and `prompts/` paths resolve:

```bash
cd .agents && python3 orchestrator.py run --task <task-id> [--restart]
```

3. Note: re-running an interrupted task without `--restart` automatically resumes from where it left off (state is persisted after every stage). If the task already finished, the command reports that and does nothing unless `--restart` is given.
4. Relay the final `state`/`status` JSON to the user.
   - If the status is `awaiting_approval` (state `CONFIRM_REQUIREMENTS`), the run paused for the human
     requirements gate: tell the user to run `/confirm-task <task-id>` to review and approve (or request
     changes). Design does not start until they do.
   - If the status is `escalated`, point them to the state history for what to resolve.
