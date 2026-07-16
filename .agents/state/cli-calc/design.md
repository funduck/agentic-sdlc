# Design: CLI Calculator with Extended Math Functions

Traces to `requirements.md` (FR-1 .. FR-9, NFR-1, NFR-2). Implementation language: **Go** (per
Confirmed Assumptions).

## 1. Overview

A single-shot CLI that parses one expression string into an AST via a hand-written
recursive-descent (precedence-climbing) parser, evaluates the AST with domain-checked math
operations, and formats the numeric result. No REPL, no persistent state, no I/O beyond
argv/stdout/stderr (NFR-1).

The design keeps parsing/evaluation as a pure, dependency-free core (`internal/calc`) and pushes
all I/O (argv, stdout, stderr, process exit) to a thin CLI shell (`cmd/calc`), so the math/parsing
logic is unit-testable without spawning processes, and the CLI wiring is testable without stdio
mocking gymnastics (see "Testability" below).

## 2. Module layout

```
go.mod                          module "cli-calc"
cmd/calc/
  main.go                       thin entry point: os.Args -> run() -> os.Exit()
  main_test.go                  CLI-contract tests (streams, exit codes)
internal/calc/
  token.go                      token kinds
  lexer.go                      Tokenize(input string) ([]token, error)
  ast.go                        AST node types
  parser.go                     Parse(tokens []token) (node, error)
  eval.go                       Eval(n node) (float64, error) + function/constant tables
  calc.go                       Evaluate(input string) (float64, error)  (Tokenize+Parse+Eval)
  format.go                     FormatResult(v float64) string
  errors.go                     SyntaxError, DomainError types
  *_test.go                     unit tests per file above
```

`internal/` prevents the parsing/eval package from becoming a public API surface no requirement
asked for (YAGNI).

## 3. Data flow

```
argv ──> main.run(args, stdout, stderr) ──┬─ help/usage path ─> write usage ─> exit code
                                           └─ expression path:
                                                calc.Evaluate(expr string)
                                                  │
                                                  ├─ Tokenize  -> []token           (*SyntaxError on bad char)
                                                  ├─ Parse     -> AST               (*SyntaxError on structure)
                                                  └─ Eval      -> float64           (*DomainError on undefined math)
                                                  │
                                                  ├─ error   -> stderr "Error: <msg>", exit 1
                                                  └─ ok      -> calc.FormatResult(v) -> stdout, exit 0
```

`run(args []string, stdout, stderr io.Writer) int` is extracted out of `main()` specifically as a
test seam: CLI tests call it directly with buffers instead of spawning the binary, per the
"design for testability" guideline. `main()` itself is just
`os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))`.

## 4. CLI argument contract (FR-1, FR-9)

- Exactly one argument equal to `-h` or `--help` → write usage text to **stdout**, exit `0`.
- Exactly one argument, anything else → treat the whole string as the expression (do **not**
  special-case a leading `-`, since `-5 + 2` is a legal expression per FR-2's unary minus — only
  the literal strings `-h`/`--help` are flags).
- Zero arguments → write usage text to **stderr**, exit non-zero (FR-9).
- Two or more arguments → same as zero arguments (usage to stderr, non-zero exit). FR-1 specifies
  the expression arrives as *one* argument; requiring the caller to quote it and rejecting
  unquoted multi-token invocations is the literal, unambiguous reading, and re-uses the exact
  "no expression" usage-error path rather than inventing a new behavior (e.g. silently
  joining tokens) that no FR calls for.
- Usage text content (FR-9): program name, one example invocation, the operator set
  (`+ - * / ^`, unary `-`, parentheses), the function list with the log/ln and radians notes,
  and the constants `pi`, `e`. Exact wording is an implementation detail.

## 4a. Stdout/stderr conventions

All text written by the CLI — the numeric result (FR-1/FR-8), error messages (FR-6/FR-7), and usage
text (FR-9) — ends with a single trailing `\n`, following standard Unix CLI convention (every
built-in and coreutils tool does this; omitting it would make output awkward to pipe/compare in a
shell). This isn't called out explicitly in any FR, so it's recorded here as a design decision
rather than left implicit: `fmt.Fprintln`-style output (not `Fprint`) is used at every stdout/stderr
write site in `cmd/calc`.

## 5. Grammar (FR-2, FR-3, FR-4)

Whitespace is insignificant and skipped by the lexer (Confirmed Assumption).

```
expression = addExpr
addExpr    = mulExpr { ("+" | "-") mulExpr }        (left-assoc)
mulExpr    = powExpr { ("*" | "/") powExpr }        (left-assoc)
powExpr    = unaryExpr [ "^" powExpr ]              (right-assoc)
unaryExpr  = "-" unaryExpr | atom
atom       = number
           | constant
           | ident "(" expression ")"               (function call)
           | "(" expression ")"
number     = digit+ ( "." digit+ )?
constant   = "pi" | "e"
ident      = letter+                                (function name)
```

This directly encodes FR-2's stated precedence order — **parentheses > unary minus > `^` >
`*`/`/` > `+`/`-`** — by nesting `unaryExpr` *inside* `powExpr` rather than the other way round:
`unaryExpr` is tried first (so unary minus binds tighter than `^`), and `powExpr` recurses into
itself on the right for right-associativity (`2^3^2` = `2^(3^2)` = `512`, `-2^2` = `(-2)^2` = `4`
per the stated precedence).

Number literals require at least one digit on both sides of an optional single `.` (`.5` and `5.`
are lexer errors, not shorthand for `0.5`/`5.0`) — a plain, unambiguous literal syntax; the
requirements don't specify literal syntax beyond "numeric argument", so this is the simplest
choice consistent with FR-6's "unknown token" error class.

Known function names (FR-3): `sin, cos, tan, asin, acos, atan, sqrt, log, ln, exp` — all
unary, `name(argument)`. Known constants (FR-4): `pi = math.Pi`, `e = math.E`, resolved directly
to a numeric AST leaf at parse time (no separate `ConstantNode` — they're fixed values, so a
dedicated node type would be an abstraction with no second use, against YAGNI).

**`ident` disambiguation (constant vs. function-call lookahead):** the grammar block above lists
`constant` and `ident "(" expression ")"` as sibling `atom` alternatives without stating how the
parser picks between them; the rule is single-token lookahead on whatever follows the identifier
text: if the next token is `(`, parse it as a function call (look the name up in the function
table only; `unknown function "<name>"` if absent — this is why `pi(3)` is a function-call error,
not a constant-plus-syntax-error, per the §7 table); otherwise look the name up in the constant
table (`unknown identifier "<name>"` if absent). A name is never checked against both tables — the
trailing `(` alone decides which table applies, so `pi` and `pi(` never collide.

## 6. AST (`ast.go`)

```go
type node interface{ isNode() }

type numberNode   struct{ value float64 }
type unaryMinus    struct{ operand node }
type binaryOp      struct{ op byte; left, right node }   // op in {'+','-','*','/','^'}
type functionCall  struct{ name string; arg node }
```

## 7. Parser error productions → FR-6 (syntax errors)

Every FR-6 example maps to one parser/lexer failure mode; all return `*calc.SyntaxError`
(exported, carries a human-readable `Msg`), never a generic error, so tests can assert on error
*kind* as well as exit behavior:

| FR-6 case                | Where it's caught | Example message |
|---|---|---|
| Unknown token / bad character | lexer | `unexpected character '@'` |
| Unknown function name | parser, on `ident "("` where `ident` isn't in the function table | `unknown function "foo"` |
| Unknown bare identifier | parser, on `ident` not followed by `(` and not `pi`/`e` | `unknown identifier "foo"` |
| Mismatched parentheses (unclosed) | parser, expects `)` and hits EOF | `expected ')'` |
| Mismatched parentheses (extra `)`) | top-level `Evaluate`, tokens remain after a complete `expression` parse | `unexpected token ')'` |
| Missing operand (e.g. `2 +`, `sin()`, `* 3`) | parser's `atom` production hits EOF/unexpected token where a value was expected | `missing operand` |

`calc.Evaluate` treats "leftover tokens after a full parse" as its own check (a plain
recursive-descent parser only guarantees a *prefix* is well-formed; the caller must confirm the
whole input was consumed) — this is what catches trailing garbage like `2+3)` or `2 2`.

## 8. Evaluator → FR-7 (domain errors)

`Eval` walks the AST; every case below returns `*calc.DomainError` (also exported/typed, same
reasoning as `SyntaxError`):

| Operation | Domain check | Message |
|---|---|---|
| `/` (binaryOp) | divisor `== 0` | `division by zero` |
| `sqrt` | arg `< 0` | `sqrt of negative number` |
| `log` | arg `<= 0` | `log of non-positive number` |
| `ln` | arg `<= 0` | `ln of non-positive number` |
| `asin`, `acos` | arg `< -1` or `> 1` | `argument out of domain [-1, 1]` |

`sin, cos, tan, atan, exp, ^` have no enumerated domain restriction in FR-7 and are computed
directly via Go's `math` package (`math.Sin`, `math.Tan`, `math.Pow`, …). `atan`/`sin`/`cos` are
total functions on `float64`; `tan(pi/2)` doesn't hit a literal domain error because floating-point
`pi` is never exactly `π/2`'s asymptote, so it evaluates to a large finite float, not `Inf` — no
special case needed.

**Design decision (closing a corner case, not a requirements gap):** if any operation not covered
by the table above still produces `±Inf`/`NaN` (e.g. `exp(1000)` overflowing `float64`), `Eval`
reports it as a `*DomainError` with message `result out of range`, using the exact same
reporting path as the enumerated FR-7 cases. This is necessary because FR-8's formatting rules
have no representation for a non-finite value — printing `+Inf` would violate FR-8 — and treating
overflow as a domain error is a straightforward, non-controversial extension of FR-7's existing
pattern, not a new product decision that needs Requirements' input.

Function/constant dispatch tables (single source of truth, used by both the parser's "is this a
known name" check and the evaluator's dispatch):

```go
var functions = map[string]func(float64) (float64, error){ "sin": ..., "cos": ..., ... }
var constants = map[string]float64{ "pi": math.Pi, "e": math.E }
```

## 9. Output formatting (`format.go`, FR-8)

```
const zeroThreshold = 1e-9   // §9a: snap near-zero floating-point noise to a clean "0"

FormatResult(v float64) string:
    if math.Abs(v) < zeroThreshold:  return "0"          // subsumes the old exact-v==0 case
    if v == math.Trunc(v):           return strconv.FormatFloat(v, 'f', -1, 64)   // shortest exact int repr, no ".0"
    exp := floor(log10(abs(v)))       // order of magnitude
    if exp >= 9:                      // integer part alone would need > 10 significant digits
        scale := 10^(exp - 9)
        rounded := math.Round(v/scale) * scale
        return strconv.FormatFloat(rounded, 'f', 0, 64)
    decimals := max(0, 9 - exp)
    s := strconv.FormatFloat(v, 'f', decimals, 64)
    return trimTrailingZeros(s)        // strip trailing "0"s, then a bare trailing "."
```

- Integer-valued results (including negative integers and `0`) print with no decimal point
  (`4`, `-4`, `0`), satisfying FR-8's first sentence. `strconv.FormatFloat(v, 'f', -1, 64)` is Go's
  shortest round-tripping decimal, which is exact and has no trailing `.0` for integer-valued
  floats.
- Non-integer results are rounded to 10 significant digits and printed in fixed notation, never
  scientific notation (Confirmed Assumption), by computing the decimal-point position from the
  value's order of magnitude instead of using `%g` (which switches to exponential form for very
  large/small magnitudes).
- The `exp >= 9` branch handles magnitudes whose integer part alone already exceeds 10 digits
  (e.g. a value like `12345678901.5`): FR-8 caps significant digits at 10 while also forbidding
  scientific notation, so the only way to honor both is to round the value itself to 10
  significant figures (trailing zeros in the integer part) rather than truncating digits after
  printing.
- **Known edge case for Pre-QA/QA to exercise:** rounding can push a value across a power-of-ten
  boundary (e.g. a value whose first 10 significant digits round up from `9.999999999` to
  `10.00000000`), which shifts `exp` by one after rounding. The implementation must format based
  on the *rounded* value's magnitude, not the pre-rounding `exp`, to avoid an off-by-one in digit
  count. This is called out explicitly for test coverage rather than solved in pseudocode here.

### 9a. Near-zero snapping (closes a design gap: transcendental "quasi-zero" results)

**Problem this closes:** `float64` constants like `math.Pi` are finite binary approximations of
irrational values, so a mathematically-exact-zero trig result computed from them is rarely an
*exact* `float64` zero. `cos(pi/2)` evaluates to `6.123233995736766e-17` and `tan(pi)` to
`-1.2246467991473532e-16` — genuine nonzero floats, not bugs in `math.Cos`/`math.Tan`. The
original pseudocode's `v == 0` fast path only catches *exact* zero, so these fall through to the
significant-digit branch and print as a ~20-28 character string of leading zeros followed by
floating-point noise. `cos(pi/2)` and `tan(pi)` are the first inputs anyone will try against this
task's headline feature (trig functions), so this reads as a bug to any user or tester, even
though FR-8's literal wording ("up to 10 significant digits") is technically satisfied either way.

**Decision:** `FormatResult` treats any result with `|v| < zeroThreshold` (`1e-9`) as `0`,
replacing the old exact-equality check (`1e-9` is itself `< zeroThreshold`, so the old case is
still covered).

**Why `1e-9`, and why in `FormatResult` rather than scoped to the six trig functions:**
- Float64 rounding error from evaluating a transcendental function at an input with representation
  error (e.g. `math.Pi/2` vs. true `π/2`) is bounded by a small multiple of machine epsilon
  (`~2.22e-16`) for arguments of the O(1)-to-O(100) magnitude this calculator's expressions
  realistically produce (literals, `pi`/`e`, and simple combinations thereof). `1e-9` clears that
  noise floor by roughly seven orders of magnitude — comfortably enough to be robust to slightly
  larger inputs (e.g. `cos(1000*pi/2)`) without needing a per-call, magnitude-aware tolerance.
- Applying the check globally in `FormatResult` (rather than only to `sin`/`cos`/`tan`/`asin`/
  `acos`/`atan` output in `eval.go`) is simpler (KISS) and covers the general case correctly: *any*
  computation chain that happens to land near zero because of accumulated float64 error (not just
  a single trig call — e.g. `cos(pi/2) * 1000000` is still a manifestation of the same underlying
  noise) benefits from the same clean-up, and no requirement distinguishes "trig noise" from
  "any other near-zero float noise" as a separate category needing different treatment.
- **Trade-off, stated explicitly:** a *genuine*, deliberately-computed result smaller in magnitude
  than `1e-9` (e.g. a hypothetical `sin(1e-10)`) would also print as `0` under this rule, which is
  formally a loss of information relative to raw `float64` output. No FR or test case in this task
  calls for sub-`1e-9`-magnitude precision, so this is judged an acceptable, deliberate
  simplification rather than a defect — noted here so Pre-QA/QA don't mistake it for an
  unconsidered oversight if they construct such a case.
- **Alternative considered and rejected:** scoping the epsilon check to only the trig-function
  call sites in `eval.go` (rather than globally in `FormatResult`) would shrink the blast radius of
  the trade-off above, but requires threading a "this value came from a transcendental function"
  flag through `Eval`'s return path for no requirement-driven benefit (YAGNI) — the global check in
  `FormatResult` is a single guard clause, is where all other FR-8 formatting decisions already
  live, and is trivially unit-testable in isolation (`format_test.go`, no `Eval` involvement
  needed).

## 10. Error types (`errors.go`)

```go
type SyntaxError struct{ Msg string }   // FR-6
type DomainError struct{ Msg string }   // FR-7
func (e *SyntaxError) Error() string { return e.Msg }
func (e *DomainError) Error() string { return e.Msg }
```

Two distinct exported types (rather than one generic `error`) because FR-6 and FR-7 are
distinguishable failure *categories* with different messages, and Pre-QA/QA need to assert "this
input is a syntax error" vs "this input is a domain error" independently of exact wording — even
though both currently map to the same exit code (Confirmed Assumption: any failure → non-zero).
`main.run` doesn't need to branch on the type today (both print `Error: <msg>` to stderr and
return the same exit code); the split exists at the `internal/calc` boundary for testability, not
because the CLI layer needs two code paths — no speculative exit-code-per-error-type mechanism is
added (YAGNI).

## 11. Testability (NFR-2)

- `internal/calc` is pure (no I/O), so `Tokenize`, `Parse`, `Eval`, `Evaluate`, and `FormatResult`
  are all directly unit-testable with table-driven tests — this is where the bulk of FR-2..FR-8
  coverage belongs (happy path + error path per FR, per the testing guideline).
- `cmd/calc`'s `run(args, stdout, stderr) int` seam covers what only exists at the CLI boundary:
  FR-1's "prints to stdout and exits 0", FR-6/FR-7's "prints nothing to stdout, error to stderr,
  non-zero exit", and FR-9's help/usage/exit-code behavior — without spawning a subprocess.
- Suggested traceability (Pre-QA can refine into concrete cases):
  - FR-1 → `cmd/calc` happy-path test asserting stdout content + exit 0.
  - FR-2 → `internal/calc` parser/eval tests per operator, precedence, and associativity example
    (including the `-2^2` and `2^3^2` cases from §5).
  - FR-3 → `internal/calc` eval test per function, one in-domain case each.
  - FR-4 → `internal/calc` eval test for `pi`, `e`, and their use inside expressions.
  - FR-5 → `internal/calc` eval test confirming radians (e.g. `sin(pi/2) == 1`).
  - FR-6 → one test per row of the table in §7, asserting `*SyntaxError` + message content.
  - FR-7 → one test per row of the table in §8, asserting `*DomainError` + message content.
  - FR-8 → `format_test.go` covering: integer results, negative integers, zero, 10-sig-digit
    rounding, trailing-zero trimming, the large-magnitude rounding branch, the power-of-ten
    rounding boundary noted in §9, and the near-zero snap-to-`0` rule from §9a (e.g.
    `cos(pi/2)`, `tan(pi)` → `0`, exercised end-to-end via `internal/calc.Evaluate` since the
    bug this guards against only manifests through real transcendental-function float64 error,
    not from a hand-picked `FormatResult` input).
  - FR-9 → `cmd/calc` tests for `--help`, `-h`, zero args, and multi-arg invocations.

## 12. Alternatives considered

- **Parser generator / grammar library** vs. hand-written recursive descent — rejected: the
  grammar is small and fixed (FR-2/FR-3/FR-4 are closed lists), so a generator adds a build step
  and dependency for no real gain (YAGNI). Recursive descent also makes the exact FR-6 error
  productions easy to name and test individually.
- **`big.Float`/arbitrary precision** vs. `float64` — rejected: no FR asks for precision beyond
  "up to 10 significant digits"; `float64` easily carries that, and Go's standard `math` package
  (needed for `sin`/`log`/etc. anyway) is `float64`-based.
- **Distinguishing syntax vs. domain errors with different exit codes** — rejected per the
  Confirmed Assumption that all failures share one non-zero exit code; the type distinction is
  kept internally for message clarity and testability only (see §10).
- **Resolving `pi`/`e` as a `ConstantNode` evaluated at eval time** vs. resolving them to a numeric
  literal at parse time — chose the latter: constants have no runtime-dependent behavior, so
  eval-time resolution would be an indirection with no payoff.
- **Near-zero formatting: global epsilon snap in `FormatResult` vs. no snap (raw float64 output)
  vs. snap scoped to only the trig functions** — chose the global snap; see §9a for the full
  rationale and trade-offs (this closes the `cos(pi/2)`/`tan(pi)` quasi-zero display gap).

## 13. Non-blocking documentation notes

- `requirements.md`'s FR-5 and the angle-unit line in Confirmed Assumptions both reference "see Open
  Questions", but the current document has no Open Questions section and FR-5's operative text
  already states radians unambiguously. This design proceeds with radians (§8, FR-5) and treats the
  dangling cross-reference as stale wording rather than a blocking ambiguity — nothing in the design
  depends on resolving it.
- The stdout-trailing-newline convention (§4a) and the `ident` constant-vs-function-call lookahead
  rule (§5) were both underspecified in earlier drafts of this design; both are now pinned down
  explicitly above rather than left for Implementation to guess.
