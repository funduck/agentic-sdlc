#!/usr/bin/env python3
"""Deterministic orchestrator for the multiagent SDLC workflow.

Control flow (which stage runs next, and whether a loop has exhausted its
budget) lives here as plain, testable code. The orchestrator is a pure
*instruction printer*: it never launches agents itself.

A single Claude session drives the pipeline. It asks the orchestrator for the
next step (`next`), dispatches the named subagent (defined in `.claude/agents/`)
via the Task tool, and feeds the subagent's verdict back (`record`). Each
subagent writes its substance to files (its owned artifact + `verdict.json`) and
returns only a terse ack; the orchestrator reads the verdict file, updates
externally-persisted state, and decides retry-vs-advance-vs-escalate. The
orchestrator's stdout is the single source of truth the driver acts on — the
main session never has to parse an agent's raw output, and an LLM never counts
its own attempts.

For development the whole state machine can be exercised with scripted verdicts
(`run --stub-scenario`), zero API calls and no subagents.
"""

from __future__ import annotations

import argparse
import json
import sys
from datetime import datetime, timezone
from pathlib import Path

# --- States ---------------------------------------------------------------

STATE_REQUIREMENTS = "REQUIREMENTS"
STATE_DESIGN = "DESIGN"
STATE_PRE_QA = "PRE_QA"
STATE_IMPLEMENTATION = "IMPLEMENTATION"
STATE_QA = "QA"
STATE_REVIEW = "REVIEW"
STATE_CONFIRM_REQUIREMENTS = "CONFIRM_REQUIREMENTS"  # human-approval gate (pauses the run)
STATE_DONE = "DONE"
STATE_ESCALATE_USER = "ESCALATE_USER"

TERMINAL_STATES = {STATE_DONE, STATE_ESCALATE_USER}
# Non-terminal states the run loop stops on to wait for a human decision.
PAUSE_STATES = {STATE_CONFIRM_REQUIREMENTS}
START_STATE = STATE_REQUIREMENTS

# Each running (non-terminal) state has a subagent and takes part in routing.
STAGES = [STATE_REQUIREMENTS, STATE_DESIGN, STATE_PRE_QA, STATE_IMPLEMENTATION, STATE_QA, STATE_REVIEW]

# Which subagent (in .claude/agents/) runs each stage.
AGENT = {
    STATE_REQUIREMENTS: "sdlc-requirements",
    STATE_DESIGN: "sdlc-design",
    STATE_PRE_QA: "sdlc-pre-qa",
    STATE_IMPLEMENTATION: "sdlc-implementation",
    STATE_QA: "sdlc-qa",
    STATE_REVIEW: "sdlc-review",
}

# Which stage owns each defect type — "route to the owner, not the neighbor".
OWNER = {
    "requirement": STATE_REQUIREMENTS,
    "design": STATE_DESIGN,
    "implementation": STATE_IMPLEMENTATION,
}

DEFAULT_BUDGET = 3

# This file lives at <repo_root>/.claude/scripts/orchestrator/orchestrator.py.
REPO_ROOT = Path(__file__).resolve().parent.parent.parent.parent

# Where agents run (and write product code) by default: the repo root.
DEFAULT_PROJECT_DIR = str(REPO_ROOT)

# Per-task state persists under `.agents/` at the repo root and is committed to
# the repo as a work artifact, regardless of the orchestrator's own location.
DEFAULT_STATE_ROOT = str(REPO_ROOT / ".agents")

# --- Verdicts -------------------------------------------------------------
DECISION_ADVANCE = "advance"
DECISION_NEEDS_WORK = "needs_work"
DECISION_UNCLEAR = "unclear"


# --- Transition table -----------------------------------------------------

def transition(state: str, verdict: dict) -> tuple[str, str | None]:
    """Map (state, verdict) -> (next_state, backward_edge_key | None).

    The edge key is non-None only for a backward (retry) transition; it names
    the loop whose budget should be charged, e.g. "qa->implementation".
    """
    decision = verdict.get("decision")
    defect = verdict.get("defect_type")

    if state == STATE_REQUIREMENTS:
        # Requirements never advance straight to Design: a human must approve the
        # requirements (including every assumption) at the CONFIRM_REQUIREMENTS gate.
        return STATE_CONFIRM_REQUIREMENTS, None

    if state == STATE_DESIGN:
        if decision == DECISION_ADVANCE:
            return STATE_PRE_QA, None
        # Design can only bounce work back to requirements.
        return STATE_REQUIREMENTS, "design->requirements"

    if state == STATE_PRE_QA:
        if decision == DECISION_ADVANCE:
            return STATE_IMPLEMENTATION, None
        target = OWNER.get(defect, STATE_DESIGN)
        return target, f"pre_qa->{target.lower()}"

    if state == STATE_IMPLEMENTATION:
        # Implementation does not self-judge; it hands off to QA.
        return STATE_QA, None

    if state == STATE_QA:
        if decision == DECISION_ADVANCE:
            return STATE_REVIEW, None
        target = OWNER.get(defect, STATE_IMPLEMENTATION)
        return target, f"qa->{target.lower()}"

    if state == STATE_REVIEW:
        if decision == DECISION_ADVANCE:
            return STATE_DONE, None
        target = OWNER.get(defect, STATE_IMPLEMENTATION)
        return target, f"review->{target.lower()}"

    raise ValueError(f"no transitions defined for state {state!r}")


# --- State persistence ----------------------------------------------------

def task_dir(root: Path, task_id: str) -> Path:
    return root / task_id


def state_path(root: Path, task_id: str) -> Path:
    return task_dir(root, task_id) / "state.json"


def verdict_path(root: Path, task_id: str) -> Path:
    return task_dir(root, task_id).resolve() / "verdict.json"


def load_state(root: Path, task_id: str) -> dict:
    path = state_path(root, task_id)
    if path.exists():
        return json.loads(path.read_text())
    return {
        "task_id": task_id,
        "current_state": START_STATE,
        "status": "running",
        "counters": {},
        "history": [],
    }


def save_state(root: Path, task_id: str, state: dict) -> None:
    d = task_dir(root, task_id)
    d.mkdir(parents=True, exist_ok=True)
    state_path(root, task_id).write_text(json.dumps(state, indent=2) + "\n")


def now() -> str:
    return datetime.now(timezone.utc).isoformat()


def apply_status(state: dict) -> None:
    """Derive the human-facing status field from the current state."""
    cur = state["current_state"]
    if cur == STATE_DONE:
        state["status"] = "done"
    elif cur == STATE_ESCALATE_USER:
        state["status"] = "escalated"
    elif cur in PAUSE_STATES:
        state["status"] = "awaiting_approval"
    else:
        state["status"] = "running"


# --- Feedback -------------------------------------------------------------

def incoming_feedback(state: dict, current: str) -> str | None:
    """Feedback the current stage must address, or None.

    When the most recent transition routed *into* ``current`` because of a
    problem (a needs_work/unclear bounce, or user-requested changes at the
    confirm gate), its summary is the feedback that stage should act on. On a
    normal forward advance there is nothing to address.
    """
    history = state.get("history", [])
    if not history:
        return None
    last = history[-1]
    if last.get("to") != current:
        return None
    verdict = last.get("verdict") or {}
    if verdict.get("decision") in {DECISION_NEEDS_WORK, DECISION_UNCLEAR, "changes_requested"}:
        return verdict.get("summary")
    return None


# --- Instruction printing -------------------------------------------------

def _dispatch_message(agent: str, workspace: Path, project_dir: Path, feedback: str | None) -> str:
    msg = (
        f"Run the {agent} subagent. Its working directory is the project repo ({project_dir}), where "
        f"product code goes. Task artifacts live in the workspace {workspace} (task.md / "
        f"requirements.md / design.md, addressed by absolute path). The subagent must write its verdict "
        f"to {workspace / 'verdict.json'} and return only a one-line ack — do not relay its output; the "
        f"orchestrator reads the verdict on `record`."
    )
    if feedback:
        msg += f" Feedback it must address first: {feedback}"
    return msg


def next_instruction(root: Path, task_id: str, state: dict, project_dir: Path) -> dict:
    """Compute the next action for the driver, given persisted state. Pure/read-only."""
    current = state["current_state"]

    if current == STATE_DONE:
        return {"action": "done", "state": current,
                "message": "Task complete. Nothing more to run."}

    if current == STATE_ESCALATE_USER:
        return {"action": "escalate", "state": current,
                "message": "A loop exhausted its budget; the task escalated to you. Inspect the "
                           "history with `status` to decide how to resolve it."}

    if current in PAUSE_STATES:
        d = task_dir(root, task_id).resolve()
        return {"action": "await_approval", "state": current,
                "requirements_file": str(d / "requirements.md"),
                "message": "Requirements need human sign-off. Review requirements.md (Assumptions & "
                           "Open Questions), then run `/confirm-task` to --approve or --request-changes."}

    # A running stage: dispatch its subagent.
    d = task_dir(root, task_id).resolve()
    project_dir = project_dir.resolve()
    feedback = incoming_feedback(state, current)
    agent = AGENT[current]
    return {
        "action": "run_agent",
        "state": current,
        "agent": agent,
        "workspace": str(d),
        "project_dir": str(project_dir),
        "feedback": feedback,
        "verdict_file": str(d / "verdict.json"),
        "message": _dispatch_message(agent, d, project_dir, feedback),
    }


# --- Stub simulator (dev/CI, no API calls) --------------------------------

def run_agent_stub(stage: str, scenario: list[dict], step: int) -> dict:
    """Return the next scripted verdict for a stubbed agent run."""
    if step >= len(scenario):
        raise RuntimeError(
            f"stub scenario exhausted at step {step} (state {stage}); "
            "add more verdicts or expect a terminal state sooner"
        )
    verdict = scenario[step]
    # A light sanity check that the scenario is aligned with the run.
    if "state" in verdict and verdict["state"] != stage:
        raise RuntimeError(
            f"scenario step {step} expects state {verdict['state']!r} "
            f"but orchestrator is in {stage!r}"
        )
    return {k: v for k, v in verdict.items() if k != "state"}


def simulate(
    root: Path,
    task_id: str,
    budget: int,
    scenario: list[dict],
    max_steps: int = 100,
) -> dict:
    """Drive the state machine from scripted verdicts — no agents, no API calls.

    This is the CI/dev path that keeps the transition table and budget logic
    exercisable deterministically. Real runs go through `next`/`record` driven by
    a Claude session. The human-approval gate auto-approves here so the whole
    machine stays reachable without a human.
    """
    state = load_state(root, task_id)
    step = 0            # loop-iteration guard (bounds runaway loops)
    scenario_step = 0   # index into the stub scenario (only advances per agent verdict)

    while state["current_state"] not in TERMINAL_STATES and step < max_steps:
        current = state["current_state"]

        if current in PAUSE_STATES:
            # A gate consumes no scenario verdict; auto-approve to keep going.
            state["history"].append({
                "ts": now(),
                "from": current,
                "to": STATE_DESIGN,
                "verdict": {"decision": "approved", "summary": "auto-approved (stub)"},
            })
            state["current_state"] = STATE_DESIGN
            save_state(root, task_id, state)
            step += 1
            continue

        verdict = run_agent_stub(current, scenario, scenario_step)
        scenario_step += 1
        next_state, edge = transition(current, verdict)

        if edge is not None:
            counters = state["counters"]
            counters[edge] = counters.get(edge, 0) + 1
            if counters[edge] > budget:
                next_state = STATE_ESCALATE_USER

        state["history"].append({
            "ts": now(),
            "from": current,
            "to": next_state,
            "verdict": verdict,
        })
        state["current_state"] = next_state
        save_state(root, task_id, state)
        step += 1

    apply_status(state)
    save_state(root, task_id, state)
    return state


# --- Subcommands ----------------------------------------------------------

def cmd_add(args) -> int:
    """Create a task workspace and capture its requirements in task.md."""
    root = Path(args.state_root)
    d = task_dir(root, args.task)
    task_md = d / "task.md"

    if task_md.exists() and not args.force:
        print(f"task.md already exists at {task_md}; use --force to overwrite", file=sys.stderr)
        return 1

    if args.from_file:
        requirements = Path(args.from_file).read_text()
    else:
        requirements = sys.stdin.read()

    if not requirements.strip():
        print("no requirements text provided (pass --from PATH or pipe via stdin)", file=sys.stderr)
        return 1

    d.mkdir(parents=True, exist_ok=True)
    task_md.write_text(requirements if requirements.endswith("\n") else requirements + "\n")

    # Initialize a fresh state record (load_state returns a default when absent).
    state = load_state(root, args.task)
    save_state(root, args.task, state)

    print(json.dumps({
        "task_id": args.task,
        "task_md": str(task_md),
        "current_state": state["current_state"],
    }, indent=2))
    return 0


def cmd_next(args) -> int:
    """Print the next instruction for the driver (read-only; launches nothing)."""
    root = Path(args.state_root)
    if not state_path(root, args.task).exists():
        print(f"no state for task {args.task!r} under {root}; run `add` first", file=sys.stderr)
        return 1
    state = load_state(root, args.task)
    instr = next_instruction(root, args.task, state, Path(args.project_dir))
    print(json.dumps(instr, indent=2))
    return 0


def cmd_record(args) -> int:
    """Record a subagent's verdict, advance the state machine, print the next instruction.

    This is the narrator: it reads the verdict file the subagent wrote, applies the
    transition + budget, persists, and emits the next step plus `last_summary` (the
    verdict's human-readable summary) for the driver to relay.
    """
    root = Path(args.state_root)
    if not state_path(root, args.task).exists():
        print(f"no state for task {args.task!r} under {root}; run `add` first", file=sys.stderr)
        return 1

    state = load_state(root, args.task)
    current = state["current_state"]
    if current in TERMINAL_STATES:
        print(f"task {args.task!r} already finished (state {current}); nothing to record",
              file=sys.stderr)
        return 1
    if current in PAUSE_STATES:
        print(f"task {args.task!r} is at the {current} gate; resolve it with `confirm`, not `record`",
              file=sys.stderr)
        return 1

    vpath = Path(args.verdict) if args.verdict else verdict_path(root, args.task)
    if not vpath.exists():
        print(f"no verdict at {vpath}; the {current} subagent must write one before you record",
              file=sys.stderr)
        return 1
    verdict = json.loads(vpath.read_text())

    next_state, edge = transition(current, verdict)
    if edge is not None:
        counters = state["counters"]
        counters[edge] = counters.get(edge, 0) + 1
        if counters[edge] > args.budget:
            next_state = STATE_ESCALATE_USER

    state["history"].append({
        "ts": now(), "from": current, "to": next_state, "verdict": verdict,
    })
    state["current_state"] = next_state
    apply_status(state)
    save_state(root, args.task, state)
    vpath.unlink(missing_ok=True)  # verdict is now recorded in history; clear it for the next stage

    instr = next_instruction(root, args.task, state, Path(args.project_dir))
    instr["last_summary"] = verdict.get("summary")
    instr["recorded"] = {
        "from": current, "to": next_state,
        "decision": verdict.get("decision"), "defect_type": verdict.get("defect_type"),
    }
    print(json.dumps(instr, indent=2))
    return 0


def cmd_confirm(args) -> int:
    """Resolve the human-approval gate, then print the next instruction.

    ``--approve`` proceeds to Design; ``--request-changes`` routes back to
    Requirements (which receives the change text as feedback) and will return to
    the gate. Unlike the old flow, this does not run any agent — it just resolves
    the gate so the driver loop can continue in-session.
    """
    root = Path(args.state_root)
    if not state_path(root, args.task).exists():
        print(f"no state for task {args.task!r} under {root}; run `add` first", file=sys.stderr)
        return 1

    state = load_state(root, args.task)
    if state["current_state"] not in PAUSE_STATES:
        print(
            f"task {args.task!r} is not awaiting approval "
            f"(current state: {state['current_state']}, status: {state['status']})",
            file=sys.stderr,
        )
        return 1

    current = state["current_state"]
    if args.approve:
        next_state = STATE_DESIGN
        verdict = {"decision": "approved", "defect_type": None,
                   "summary": "user approved requirements"}
    else:
        next_state = STATE_REQUIREMENTS
        verdict = {"decision": "changes_requested", "defect_type": None,
                   "summary": args.request_changes}

    state["history"].append({
        "ts": now(), "from": current, "to": next_state, "verdict": verdict,
    })
    state["current_state"] = next_state
    apply_status(state)
    save_state(root, args.task, state)

    instr = next_instruction(root, args.task, state, Path(args.project_dir))
    instr["last_summary"] = verdict["summary"]
    print(json.dumps(instr, indent=2))
    return 0


# Statuses that represent a normal end to a stub `run` (as opposed to an
# unexpected interruption). `awaiting_approval` is a healthy pause.
RUN_OK_STATUSES = {"done", "escalated", "awaiting_approval"}


def _run_summary(final: dict) -> dict:
    summary = {
        "task_id": final["task_id"],
        "final_state": final["current_state"],
        "status": final["status"],
        "counters": final["counters"],
        "steps": len(final["history"]),
    }
    if final["status"] == "awaiting_approval":
        summary["note"] = "requirements need your sign-off; run `confirm` (or /confirm-task)"
    return summary


def cmd_run(args) -> int:
    """Drive the stub simulator (dev/CI only). Real runs use `next`/`record`."""
    root = Path(args.state_root)
    if not args.stub_scenario:
        print(
            "`run` now only drives the stub simulator (pass --stub-scenario). For a real run, use "
            "/run-task: it asks the orchestrator for each step via `next`/`record` and dispatches "
            "the subagents itself.",
            file=sys.stderr,
        )
        return 2

    scenario = json.loads(Path(args.stub_scenario).read_text())
    d = task_dir(root, args.task)

    if args.restart:
        fresh = {
            "task_id": args.task,
            "current_state": START_STATE,
            "status": "running",
            "counters": {},
            "history": [],
        }
        d.mkdir(parents=True, exist_ok=True)
        save_state(root, args.task, fresh)
    else:
        existing = load_state(root, args.task)
        if existing["current_state"] in TERMINAL_STATES:
            print(json.dumps({
                "task_id": args.task,
                "final_state": existing["current_state"],
                "status": existing["status"],
                "note": "task already finished; pass --restart to run again",
            }, indent=2))
            return 0

    final = simulate(root, args.task, args.budget, scenario)
    print(json.dumps(_run_summary(final), indent=2))
    return 0 if final["status"] in RUN_OK_STATUSES else 1


def cmd_status(args) -> int:
    """Print a compact summary of a task's persisted state."""
    root = Path(args.state_root)
    if not state_path(root, args.task).exists():
        print(f"no state for task {args.task!r} under {root}; run `add` first", file=sys.stderr)
        return 1

    state = load_state(root, args.task)
    history = state.get("history", [])
    tail = history[-5:]
    print(json.dumps({
        "task_id": state["task_id"],
        "current_state": state["current_state"],
        "status": state["status"],
        "counters": state["counters"],
        "steps": len(history),
        "recent": [
            {"from": h["from"], "to": h["to"], "summary": h["verdict"].get("summary")}
            for h in tail
        ],
    }, indent=2))
    return 0


# --- CLI ------------------------------------------------------------------

def _add_common(p) -> None:
    p.add_argument("--task", required=True, help="task id (also the state dir name)")
    p.add_argument(
        "--state-root",
        default=DEFAULT_STATE_ROOT,
        help="root dir for per-task state (default: .agents/ at the repo root)",
    )


def _add_project_dir(p) -> None:
    p.add_argument(
        "--project-dir",
        default=DEFAULT_PROJECT_DIR,
        help="working dir for agents — where product code is written (default: repo root)",
    )


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    sub = parser.add_subparsers(dest="command", required=True)

    p_add = sub.add_parser("add", help="create a task workspace and capture its requirements")
    _add_common(p_add)
    p_add.add_argument("--from", dest="from_file", help="read requirements from this file (default: stdin)")
    p_add.add_argument("--force", action="store_true", help="overwrite an existing task.md")
    p_add.set_defaults(func=cmd_add)

    p_next = sub.add_parser("next", help="print the next instruction for the driver (launches nothing)")
    _add_common(p_next)
    _add_project_dir(p_next)
    p_next.set_defaults(func=cmd_next)

    p_record = sub.add_parser("record", help="record a subagent verdict, advance state, print next step")
    _add_common(p_record)
    _add_project_dir(p_record)
    p_record.add_argument("--verdict", help="path to the verdict JSON (default: <workspace>/verdict.json)")
    p_record.add_argument("--budget", type=int, default=DEFAULT_BUDGET, help="max attempts per loop")
    p_record.set_defaults(func=cmd_record)

    p_confirm = sub.add_parser("confirm", help="resolve a requirements approval gate, then print next step")
    _add_common(p_confirm)
    _add_project_dir(p_confirm)
    g_confirm = p_confirm.add_mutually_exclusive_group(required=True)
    g_confirm.add_argument("--approve", action="store_true", help="approve the requirements; proceed to Design")
    g_confirm.add_argument(
        "--request-changes",
        metavar="TEXT",
        help="send the requirements back with this feedback for another round",
    )
    p_confirm.set_defaults(func=cmd_confirm)

    p_run = sub.add_parser("run", help="drive the stub simulator (dev/CI; requires --stub-scenario)")
    _add_common(p_run)
    p_run.add_argument("--budget", type=int, default=DEFAULT_BUDGET, help="max attempts per loop")
    p_run.add_argument("--restart", action="store_true", help="reset progress and run from the start")
    p_run.add_argument(
        "--stub-scenario",
        required=False,
        help="path to a JSON list of scripted verdicts (stub mode; no API calls)",
    )
    p_run.set_defaults(func=cmd_run)

    p_status = sub.add_parser("status", help="print a task's current state and recent history")
    _add_common(p_status)
    p_status.set_defaults(func=cmd_status)

    args = parser.parse_args(argv)
    return args.func(args)


if __name__ == "__main__":
    sys.exit(main())
