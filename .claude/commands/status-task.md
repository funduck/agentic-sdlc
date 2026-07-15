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
cd .agents && python3 orchestrator.py status --task <task-id>
```

3. Relay the summary: current stage, status, loop counters, and the recent transition history.
