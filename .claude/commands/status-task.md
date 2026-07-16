---
description: Show the current state and recent history of a workflow task
argument-hint: <task-id>
allowed-tools: Bash
---

Summarize the state of a multiagent workflow task.

Arguments: `$ARGUMENTS`

Steps:
1. Parse the task id (first token).
2. Run:

```bash
python3 .agents/orchestrator/orchestrator.py status --task <task-id>
```

3. Relay the summary: current stage, status, loop counters, and the recent transition history.
4. If the status is `awaiting_approval` (state `CONFIRM_REQUIREMENTS`), tell the user the requirements
   need their sign-off and to run `/confirm-task <task-id>` to review and approve or request changes.
