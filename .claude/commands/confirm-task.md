---
description: Review and sign off on a task's requirements at the approval gate
argument-hint: <task-id> [--approve | --request-changes "<feedback>"]
allowed-tools: Bash, Read, Task
---

Resolve the CONFIRM_REQUIREMENTS human-approval gate for a workflow task. A task pauses here (status
`awaiting_approval`) after the Requirements Agent runs; Design does not start until you approve.

Arguments: `$ARGUMENTS`

Steps:

1. Parse the task id (first token).

2. **Show the requirements first.** Read `.agents/state/<task-id>/requirements.md` and present it to
   the user — especially the **Assumptions** and **Open Questions** sections — so they can judge before
   signing off. (Confirm the task is at the gate with `/status-task <task-id>` if unsure.) This is the
   one place the driver reads a workspace file directly, because the human is the reviewer here.

3. **Resolve the gate.** `confirm` resolves it and prints the *next* instruction (it launches nothing):

   Approve (proceed to Design):
   ```bash
   python3 .agents/orchestrator/orchestrator.py confirm --task <task-id> --approve
   ```

   Request changes (send back to Requirements with feedback, then return to the gate):
   ```bash
   python3 .agents/orchestrator/orchestrator.py confirm --task <task-id> --request-changes "<feedback>"
   ```

4. **Continue the driver loop** using the instruction `confirm` printed, exactly as `/run-task` does:
   dispatch the `run_agent` subagent via the Task tool, `record` its verdict, relay `last_summary`, and
   repeat until `done`, `escalate`, or `await_approval`. If it comes back `await_approval` again (after
   a changes round), review the updated `requirements.md` and repeat from step 2.
