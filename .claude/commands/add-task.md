---
description: Create a new workflow task and capture its requirements
argument-hint: <task-id> <requirements...>
allowed-tools: Bash
---

Create a new task in the multiagent SDLC workflow.

Arguments: `$ARGUMENTS`

Steps:
1. Parse the arguments: the first token is the task id. If it looks like free text rather than an id, derive a short kebab-case task id from the requirements yourself.
2. Treat the remaining text as the task requirements.
3. Create the task by piping the requirements into the orchestrator (this creates `state/<task-id>/task.md` and an initial `state.json`):

```bash
python3 .claude/scripts/orchestrator/orchestrator.py add --task <task-id> <<'EOF'
<requirements text>
EOF
```

4. Report the created task id and `task.md` path. Do NOT start the run — tell the user to run `/run-task <task-id>` when ready.
