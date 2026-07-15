---
description: Review and sign off on a task's requirements at the approval gate
argument-hint: <task-id> [--approve | --request-changes "<feedback>"]
allowed-tools: Bash, Read
---

Resolve the CONFIRM_REQUIREMENTS human-approval gate for a workflow task. A task pauses here (status
`awaiting_approval`) after the Requirements Agent runs; Design does not start until you approve.

Arguments: `$ARGUMENTS`

Steps:
1. Parse the task id (first token).
2. **Show the requirements first.** Read `.agents/state/<task-id>/requirements.md` and present it to
   the user — especially the **Assumptions** and **Open Questions** sections — so they can judge before
   signing off. (Confirm the task is actually at the gate with `/status-task <task-id>` if unsure.)
3. Resolve the gate based on the user's decision:

   Approve (proceed to Design, then auto-continue the pipeline):
   ```bash
   cd .agents && python3 orchestrator.py confirm --task <task-id> --approve
   ```

   Request changes (send back to Requirements with feedback, then return to the gate):
   ```bash
   cd .agents && python3 orchestrator.py confirm --task <task-id> --request-changes "<feedback>"
   ```

4. Relay the resulting `state`/`status` JSON. If it comes back `awaiting_approval` again (after a
   changes round), review the updated `requirements.md` and repeat. On `escalated`, point the user to
   the state history.
