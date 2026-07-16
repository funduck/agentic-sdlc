# Requirements: CLI Calculator with Extended Math Functions

## Overview
Build a command-line calculator that evaluates mathematical expressions, including
extended (scientific) functions such as trigonometric functions, beyond basic
arithmetic.

## Functional Requirements

**FR-1 — Single-expression evaluation.** The calculator accepts one mathematical
expression as a command-line argument and prints the evaluated numeric result to
stdout, exiting with status code 0.
Example: `calc "2 + 3 * 4"` prints `14`.

**FR-2 — Arithmetic operators.** The expression syntax supports the binary operators
`+`, `-`, `*`, `/`, `^` (exponentiation), unary minus, and parentheses `()` for
grouping. Standard operator precedence applies (parentheses > unary minus > `^` >
`*`/`/` > `+`/`-`). `*`, `/`, `+`, `-` are left-associative; `^` is right-associative.

**FR-3 — Extended math functions.** The expression syntax supports these functions,
invoked as `name(argument)`, each taking exactly one numeric argument:
`sin`, `cos`, `tan`, `asin`, `acos`, `atan`, `sqrt`, `log` (base 10), `ln` (natural
log), `exp`.

**FR-4 — Named constants.** The expression syntax supports the named constants `pi`
and `e`.

**FR-5 — Angle unit.** Trigonometric functions (`sin`, `cos`, `tan`, `asin`, `acos`,
`atan`) interpret and return angles in radians. *(Angle unit is a genuine ambiguity —
see Open Questions.)*

**FR-6 — Syntax error handling.** When an expression is syntactically invalid (e.g.
mismatched parentheses, unknown token, unknown function name, missing operand), the
calculator prints nothing to stdout, prints a clear error message identifying the
problem to stderr, and exits with a non-zero status code.

**FR-7 — Domain error handling.** When an expression is mathematically undefined for
the given inputs — division by zero, `sqrt` of a negative number, `log`/`ln` of a
non-positive number, `asin`/`acos` of a value outside `[-1, 1]` — the calculator
prints nothing to stdout, prints a clear error message describing the domain error to
stderr, and exits with a non-zero status code. No complex-number results are
produced; such cases are always treated as domain errors.

**FR-8 — Numeric output formatting.** Results are printed as plain decimal numbers.
An integer-valued result is printed without a decimal point (e.g. `4`, not `4.0`). A
non-integer result is printed with up to 10 significant digits, with trailing zeros
removed.

**FR-9 — Usage help.** Invoking the calculator with no expression argument, or with
`--help`/`-h`, prints usage text describing the supported operators, functions, and
constants. `--help`/`-h` exits with status 0; a missing expression argument exits
with a non-zero status code.

## Non-Functional Requirements

**NFR-1 — Environment.** Runs as a text-mode command-line tool on Linux; no GUI, no
network access.

**NFR-2 — Test coverage.** Every functional requirement above (FR-1 through FR-9)
has at least one automated test covering its normal and error-path behavior.

## Confirmed Assumptions

- **No interactive REPL mode.** Each invocation evaluates exactly one expression
  passed as a command-line argument; there is no persistent interactive prompt loop.
  (The alternative is called out in Open Questions.)
- **Implementation language** Go.
- **Function names are case-sensitive, lowercase** (`sin`, `cos`, …). No degree-unit
  function-name variants (e.g. `sind`) are provided; a caller wanting degrees must
  convert manually via `pi` (pending the angle-unit Open Question).
- **Whitespace within an expression is insignificant** and ignored during parsing.
- **`log` means base-10 logarithm and `ln` means natural logarithm** — the common
  calculator convention. No arbitrary-base logarithm function is required.
- **No persistent state.** No memory/variable feature (e.g. `ans`, `M+`) across
  invocations; each invocation is fully stateless.
- **No configuration file or environment variables**; all behavior is controlled via
  command-line arguments only.
- **Exit code convention:** `0` = success, non-zero (e.g. `1`) = any failure (parse
  error or domain error), used consistently across all failure modes.
- **No scientific notation output**; FR-8's "up to 10 significant digits" rule is
  used for all magnitudes.

