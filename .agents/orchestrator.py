#!/usr/bin/env python3
"""Deterministic orchestrator for the multiagent SDLC workflow.

Control flow (which stage runs next, and whether a loop has exhausted its
budget) lives here as plain, testable code. The Claude agents are stateless
workers: each is invoked, does one job, and returns a machine-readable
*verdict*. This script reads the verdict, updates externally-persisted state,
and decides retry-vs-advance-vs-escalate. An LLM never counts its own attempts.

Agents are invoked via the Claude CLI in headless mode (`claude -p`); no SDK.
For development the agents are stubbed with scripted verdict sequences so the
whole state machine can be exercised deterministically with zero API calls.
"""

from __future__ import annotations

import argparse
import json
import subprocess
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
STATE_DONE = "DONE"                    
STATE_ESCALATE_USER = "ESCALATE_USER"  

TERMINAL_STATES = {STATE_DONE, STATE_ESCALATE_USER}
START_STATE = STATE_REQUIREMENTS

# Each running (non-terminal) state has a prompt file and a stub key.
STAGES = [STATE_REQUIREMENTS, STATE_DESIGN, STATE_PRE_QA, STATE_IMPLEMENTATION, STATE_QA, STATE_REVIEW]

# Which stage owns each defect type — "route to the owner, not the neighbor".
OWNER = {
    "requirement": STATE_REQUIREMENTS,
    "design": STATE_DESIGN,
    "implementation": STATE_IMPLEMENTATION,
}

DEFAULT_BUDGET = 3

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
        if decision == DECISION_ADVANCE:
            return STATE_DESIGN, None
        # Genuine ambiguity only the human can resolve.
        return STATE_ESCALATE_USER, None

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


# --- Agent invocation -----------------------------------------------------

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


def run_agent_cli(stage: str, root: Path, task_id: str, prompts_dir: Path) -> dict:
    """Invoke the real agent via `claude -p` and read back its verdict file.

    The agent is instructed (via prompts/<stage>.md) to write its verdict to
    <task_dir>/verdict.json, which avoids fragile parsing of free-text stdout.
    """
    d = task_dir(root, task_id)
    verdict_file = d / "verdict.json"
    if verdict_file.exists():
        verdict_file.unlink()

    prompt_file = prompts_dir / f"{stage.lower()}.md"
    system_prompt = prompt_file.read_text() if prompt_file.exists() else ""
    user_prompt = (
        f"You are the {stage} agent. The task workspace is {d}. "
        f"Read task.md and any requirements.md / design.md there, do your job, "
        f"update your owned document if applicable, then write your verdict to "
        f"{verdict_file} as JSON: "
        f'{{"decision": "advance|needs_work|unclear", '
        f'"defect_type": "requirement|design|implementation|null", '
        f'"summary": "..."}}.'
    )
    cmd = [
        "claude", "-p", user_prompt,
        "--output-format", "json",
        "--append-system-prompt", system_prompt,
    ]
    subprocess.run(cmd, check=True, cwd=d)

    if not verdict_file.exists():
        raise RuntimeError(f"{stage} agent did not write {verdict_file}")
    return json.loads(verdict_file.read_text())


# --- Orchestration loop ---------------------------------------------------

def run(
    root: Path,
    task_id: str,
    budget: int,
    scenario: list[dict] | None,
    prompts_dir: Path,
    max_steps: int = 100,
) -> dict:
    state = load_state(root, task_id)
    step = 0

    while state["current_state"] not in TERMINAL_STATES and step < max_steps:
        current = state["current_state"]

        if scenario is not None:
            verdict = run_agent_stub(current, scenario, step)
        else:
            verdict = run_agent_cli(current, root, task_id, prompts_dir)

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

    if state["current_state"] == STATE_DONE:
        state["status"] = "done"
    elif state["current_state"] == STATE_ESCALATE_USER:
        state["status"] = "escalated"
    save_state(root, task_id, state)
    return state


# --- CLI ------------------------------------------------------------------

def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--task", required=True, help="task id (also the state dir name)")
    parser.add_argument("--state-root", default="state", help="root dir for per-task state")
    parser.add_argument("--prompts-dir", default="prompts", help="dir with per-stage prompts")
    parser.add_argument("--budget", type=int, default=DEFAULT_BUDGET, help="max attempts per loop")
    parser.add_argument(
        "--stub-scenario",
        help="path to a JSON list of scripted verdicts (stub mode; no API calls)",
    )
    args = parser.parse_args(argv)

    scenario = None
    if args.stub_scenario:
        scenario = json.loads(Path(args.stub_scenario).read_text())

    root = Path(args.state_root)
    final = run(
        root=root,
        task_id=args.task,
        budget=args.budget,
        scenario=scenario,
        prompts_dir=Path(args.prompts_dir),
    )

    print(json.dumps({
        "task_id": final["task_id"],
        "final_state": final["current_state"],
        "status": final["status"],
        "counters": final["counters"],
        "steps": len(final["history"]),
    }, indent=2))
    return 0 if final["status"] in {"done", "escalated"} else 1


if __name__ == "__main__":
    sys.exit(main())
