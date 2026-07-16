# Pre-QA test cases: CLI Calculator with Extended Math Functions

Written against `requirements.md` + `design.md` before any code exists. Organized by FR; each row is
a concrete input/expected-behavior pair intended to become a table-driven test in
`internal/calc/*_test.go` / `cmd/calc/main_test.go` once implementation starts.

Legend: `stdout` / `stderr` / `exit` columns describe the `cmd/calc` CLI-boundary contract;
`internal/calc` unit tests assert the equivalent at the `Evaluate`/`Eval` level (`*SyntaxError`,
`*DomainError`, or a `float64` result).

## Re-review note (round 2)

Round 1 of this review found a blocking design gap: canonical trig inputs like `cos(pi/2)` did not
evaluate to a clean `0` under the formatting algorithm in the original design.md §9 (no
near-zero/epsilon handling). Design added §9a ("Near-zero snapping", `|v| < 1e-9` → `"0"`) plus
§4a (trailing-newline convention) and an explicit `ident` lookahead-disambiguation rule in §5. All
three are re-verified below (FR3-T04/T06, FR1's newline note, FR4's lookahead note) and now check
out. See `verdict.json` for the round-2 disposition.

## FR-1 — Single-expression evaluation

| # | Input (argv) | stdout | exit |
|---|---|---|---|
| FR1-T01 | `2 + 3 * 4` | `14` | 0 |
| FR1-T02 | `(2 + 3) * 4` | `20` | 0 |
| FR1-T03 | `""` (empty string arg) | (nothing) | non-zero — treated as FR-6 "missing operand" syntax error, not FR-1 success. Confirms empty-string is a valid *argument* (one arg supplied) but an invalid *expression*. |
| FR1-T04 | `"   "` (whitespace-only arg) | (nothing) | non-zero, same reasoning as T03 (whitespace insignificant → equivalent to empty). |

**Resolved (round 2):** design.md §4a now states explicitly that every stdout/stderr write (result,
error message, usage text) ends with a single trailing `\n` (`fmt.Fprintln`-style). FR1-T01/T02
above (and every stdout-asserting case in this file) should assert the trailing newline is present,
not just the visible content.

## FR-2 — Arithmetic operators / precedence / associativity

| # | Input | Expected | Notes |
|---|---|---|---|
| FR2-T01 | `2 + 3 * 4` | `14` | `*` before `+` |
| FR2-T02 | `2 * 3 + 4` | `10` | |
| FR2-T03 | `2 - 3 - 4` | `-5` | `-` left-assoc: `(2-3)-4`, not `2-(3-4)=3` |
| FR2-T04 | `20 / 4 / 5` | `1` | `/` left-assoc: `(20/4)/5=1`, not `20/(4/5)=25` |
| FR2-T05 | `2 ^ 3 ^ 2` | `512` | `^` right-assoc: `2^(3^2)=2^9`, not `(2^3)^2=64` |
| FR2-T06 | `-2 ^ 2` | `4` | unary minus binds *tighter* than `^` per FR-2's explicit order (`(-2)^2`), diverges from Python/JS convention (`-2**2==-4`) but matches FR-2's literal precedence list and Excel-style convention — worth a named regression test since it's genuinely surprising to some users and easy for an implementer to "fix" incorrectly. |
| FR2-T07 | `-(2 ^ 2)` | `-4` | explicit parens force the "other" reading, confirming both are reachable |
| FR2-T08 | `2 ^ -3` | `0.125` | unary minus as the RHS of `^` |
| FR2-T09 | `--5` | `5` | double unary minus (`unaryExpr = "-" unaryExpr \| atom` recurses) |
| FR2-T10 | `2 + 3 * (4 - 1)` | `11` | parentheses > all |
| FR2-T11 | `0 ^ 0` | `1` | Go's `math.Pow(0,0)==1`; confirms no special-case domain error here (FR-7's table doesn't list `^`) |
| FR2-T12 | `10 - 0.0000000005` | see FR-8 rounding-boundary tests below | carries a rounding-across-magnitude-boundary case |

## FR-3 — Extended math functions

One clean in-domain case per function (FR-3 traceability), plus the special-angle cases that
surfaced the blocking issue.

| # | Input | Expected | Notes |
|---|---|---|---|
| FR3-T01 | `sin(0)` | `0` | |
| FR3-T02 | `sin(pi/2)` | `1` | `math.Sin(math.Pi/2)` happens to be exactly `1.0` in float64 — safe positive case |
| FR3-T03 | `cos(0)` | `1` | |
| FR3-T04 | `cos(pi/2)` | `0` | `math.Cos(math.Pi/2) == 6.123233995736766e-17` (well-known float64 artifact: `math.Pi/2` isn't exactly π/2), snapped to `"0"` by §9a's `|v| < 1e-9` rule — **regression test for the round-1 fix; must not regress to the raw ~28-char quasi-zero string.** |
| FR3-T05 | `tan(0)` | `0` | |
| FR3-T06 | `tan(pi)` | `0` | same class as T04 (`math.Tan(math.Pi) ≈ -1.2246467991473532e-16`), also snapped via §9a. |
| FR3-T07 | `asin(1)` | `1.570796327` (`pi/2` to 10 sig figs) | boundary of domain, must NOT be a domain error |
| FR3-T08 | `acos(1)` | `0` | |
| FR3-T09 | `atan(1)` | `0.7853981634` (`pi/4`) | |
| FR3-T10 | `sqrt(4)` | `2` | |
| FR3-T11 | `sqrt(2)` | `1.414213562` | 10 sig figs |
| FR3-T12 | `log(100)` | `2` | base-10 |
| FR3-T13 | `ln(e)` (nested with FR-4 constant `e`) | `1` (`math.Log(math.E) == 1` exactly in Go) | |
| FR3-T14 | `exp(1)` | `2.718281828` (`e` to 10 sig figs) | |
| FR3-T15 | `exp(0)` | `1` | |
| FR3-T16 | `SIN(0)` | syntax error, `unknown function "SIN"` (or `unknown identifier` if not followed by `(`) | case-sensitivity per Confirmed Assumptions |

### RESOLVED (round 2): FR3-T04 / FR3-T06 — near-zero trig results now render as "0"

Round 1 found that `cos(pi/2)`, `tan(pi)`, etc. — the single most obvious inputs anyone would try
against this task's headline feature (`task.md`: "cli calc with extended math functions like
trigonometric") — rendered as ugly ~20-28 character quasi-zero decimals instead of `0`, because
`math.Pi/2` isn't exactly π/2 in float64 and the old `FormatResult` only fast-pathed *exact* zero.

Design's fix (§9a): `FormatResult` now snaps any `|v| < 1e-9` to `"0"`, applied globally (not
scoped to the six trig functions), with the rationale, magnitude analysis (float64 rounding noise
is bounded by a small multiple of machine epsilon for the O(1)-O(100) magnitudes this calculator
realistically produces), and an explicitly acknowledged trade-off (a deliberately-computed
sub-`1e-9` result, e.g. a hypothetical `sin(1e-10)`, would also be displayed as `0` — accepted,
since no FR/test in this task calls for that precision). This is sound: FR3-T04/T06 above now
expect `0`, and FR7-T15 (`exp(-1000)` → `0`, genuine underflow, not a domain error) continues to
exercise the same snap-to-zero path without conflict.

## FR-4 — Named constants

| # | Input | Expected |
|---|---|---|
| FR4-T01 | `pi` | `3.141592654` |
| FR4-T02 | `e` | `2.718281828` |
| FR4-T03 | `2 * pi` | `6.283185307` |
| FR4-T04 | `pi + e` | `5.859874482` |
| FR4-T05 | `PI` | syntax error, `unknown identifier "PI"` — case sensitivity |
| FR4-T06 | `pi(3)` | syntax error, `unknown function "pi"` — `pi`/`e` are not in the function table, so `ident "("` lookahead routes here even though `pi` is a known *constant* name (see design §5/§7 ambiguity note below) |

**Resolved (round 2):** §5 now states the lookahead rule explicitly ("`ident` disambiguation"
paragraph) — a trailing `(` always routes to the function table only, never the constant table, so
`pi` and `pi(` never collide. FR4-T06 above is the concrete regression test for this rule.

## FR-5 — Angle unit (radians)

| # | Input | Expected |
|---|---|---|
| FR5-T01 | `sin(pi/2)` | `1` (see FR3-T02) |
| FR5-T02 | `atan(1) * 4` | `3.141592654` (≈ `pi`, confirms radians not degrees — if this were degrees, `atan(1)==45` and the result would be `180`) |

## FR-6 — Syntax error handling

One case per design §7 row, plus a couple of adjacent probes.

| # | Input | stdout | stderr contains | exit |
|---|---|---|---|---|
| FR6-T01 | `2 @ 3` | (nothing) | unexpected character `@` | non-zero |
| FR6-T02 | `foo(1)` | (nothing) | unknown function "foo" | non-zero |
| FR6-T03 | `foo` | (nothing) | unknown identifier "foo" | non-zero |
| FR6-T04 | `(2 + 3` | (nothing) | expected ')' | non-zero |
| FR6-T05 | `2 + 3)` | (nothing) | unexpected token | non-zero |
| FR6-T06 | `2 +` | (nothing) | missing operand | non-zero |
| FR6-T07 | `sin()` | (nothing) | missing operand | non-zero |
| FR6-T08 | `* 3` | (nothing) | missing operand | non-zero |
| FR6-T09 | `2 2` | (nothing) | unexpected token | non-zero — trailing-garbage / no-implicit-multiplication check |
| FR6-T10 | `+5` | (nothing) | missing operand (or equivalent "unexpected token '+'") | non-zero — confirms no unary *plus* exists (FR-2 only grants unary minus) |
| FR6-T11 | `1e10` | (nothing) | unexpected token (lexes as `1`, `e`, `10` — no scientific-notation literal syntax) | non-zero — confirms the deliberate simplification in design §5's number grammar |
| FR6-T12 | `sin(1,2)` | (nothing) | unexpected character ',' | non-zero — no multi-arg function call syntax |

For every FR6-T*, also assert: stdout is empty (FR-6's "prints nothing to stdout") and the
`internal/calc.Evaluate` error is of type `*calc.SyntaxError`.

## FR-7 — Domain error handling

One case per design §8 row, plus overflow/boundary probes.

| # | Input | stderr contains | exit |
|---|---|---|---|
| FR7-T01 | `1 / 0` | division by zero | non-zero |
| FR7-T02 | `5 / -0` | division by zero | non-zero — negative-zero divisor (`-0.0 == 0.0` in IEEE 754) must still trip the check |
| FR7-T03 | `0 / 0` | division by zero | non-zero — divisor-based check fires before a NaN could even be computed |
| FR7-T04 | `sqrt(-1)` | sqrt of negative number | non-zero |
| FR7-T05 | `sqrt(0)` | (no error) `0` on stdout | 0 — boundary: 0 is valid, not negative |
| FR7-T06 | `log(0)` | log of non-positive number | non-zero — boundary: 0 is non-positive |
| FR7-T07 | `log(-5)` | log of non-positive number | non-zero |
| FR7-T08 | `ln(0)` | ln of non-positive number | non-zero |
| FR7-T09 | `asin(1.0000001)` | argument out of domain [-1, 1] | non-zero |
| FR7-T10 | `asin(1)` | (no error) `1.570796327` | 0 — boundary: exactly 1 is valid |
| FR7-T11 | `acos(-1)` | (no error) `3.141592654` | 0 — boundary: exactly -1 is valid |
| FR7-T12 | `exp(1000)` | result out of range | non-zero — overflow to `+Inf`, design §8's closing decision |
| FR7-T13 | `10 ^ 400` | result out of range | non-zero — overflow via `^` |
| FR7-T14 | `(-8) ^ (1 / 3)` | result out of range | non-zero — negative base, non-integer exponent → `NaN` in Go's `math.Pow`, caught by the same generic "non-finite result" rule. (Message "out of range" is a slight semantic mismatch for a NaN-not-Inf case — cosmetic only, not blocking.) |
| FR7-T15 | `exp(-1000)` | (no error) `0` | 0 — underflow to `0` is a valid finite result, not a domain error |

For every FR7-T* error case, also assert: stdout is empty and the `internal/calc.Evaluate` error is
of type `*calc.DomainError`.

## FR-8 — Numeric output formatting

| # | Input / value | Expected | Notes |
|---|---|---|---|
| FR8-T01 | `2 + 2` | `4` | integer, no decimal point |
| FR8-T02 | `0 - 4` | `-4` | negative integer |
| FR8-T03 | `2 - 2` | `0` | zero, no decimal point |
| FR8-T04 | `0 * -1` | `0` | negative-zero result must print as `0`, not `-0` |
| FR8-T05 | `1 / 3` | `0.3333333333` | 10 significant digits, all after a leading `0` |
| FR8-T06 | `2 / 3` | `0.6666666667` | rounding at the 10th sig fig |
| FR8-T07 | `pi` | `3.141592654` | 1 integer digit + 9 decimals = 10 sig figs |
| FR8-T08 | `1 / 8` | `0.125` | trailing zeros trimmed (would otherwise be `0.1250000000`) |
| FR8-T09 | `12345678901.5` | `12345678900` | integer part alone > 10 digits → rounds to 10 sig figs, magnitude preserved via trailing zero (design §9's `exp >= 9` branch) |
| FR8-T10 | `1234567890.5` | `1234567891` | 10-digit integer boundary (`exp == 9`); both the "plain" and "big-magnitude" code paths must agree here |
| FR8-T11 | `10 - 0.0000000005` | `10` | power-of-ten rounding-boundary case flagged explicitly in design §9 — rounding the fractional part all the way up must correctly re-derive the new (larger) magnitude, not print something like `10.000000000` un-trimmed or a wrong digit count |
| FR8-T12 | `99.999999996` | `100` | same class of boundary case one order of magnitude down, to confirm the fix generalizes and isn't special-cased to one example |
| FR8-T13 | `2 ^ 60` | `1152921504606846976` (all 19 digits, unrounded) | **non-blocking edge case, noted for QA awareness, not a defect:** design §9's `v == math.Trunc(v)` branch is checked *before* the `exp >= 9` significant-digit cap, so any exact-integer result — however many digits — bypasses the 10-sig-fig rounding that non-integer results of the same magnitude get (contrast with FR8-T09, where `12345678901.5`, one decimal off from being an integer, *does* get rounded to 10 sig figs). This is a literal, defensible reading of FR-8 ("An integer-valued result is printed without a decimal point" states no digit cap; "up to 10 significant digits" is stated only for non-integer results), so it is not routed back as a design defect. Two second-order consequences worth QA verifying explicitly rather than assuming: (1) results whose magnitude exceeds float64's exact-integer range (`2^53 ≈ 9.007e15`) will `Trunc(v) == v` (true of every float at that magnitude) and print in full via `strconv.FormatFloat(v, 'f', -1, 64)`, but the digits beyond ~15-17 significant figures reflect the *nearest representable float64*, not necessarily the true mathematical integer (e.g. `3 ^ 40`, which needs ~64 bits to represent exactly) — this is an accepted float64-precision trade-off per design §12, not a formatting bug, but the printed digit string can look like false precision; (2) confirm this asymmetry is intentional/acceptable to Requirements if a future task revisits FR-8, rather than assuming that inconsistency is desirable. |

## FR-9 — Usage help

| # | Input (argv) | stdout | stderr | exit |
|---|---|---|---|---|
| FR9-T01 | `--help` | usage text (contains program name, an example, operators `+ - * / ^`, the function list, `pi`/`e`) | (nothing) | 0 |
| FR9-T02 | `-h` | same usage text | (nothing) | 0 |
| FR9-T03 | (no args) | (nothing) | usage text | non-zero |
| FR9-T04 | `2+2 extra` (two args) | (nothing) | usage text | non-zero — FR-1 requires the expression as *one* argument; unquoted multi-token input is rejected, not silently joined |
| FR9-T05 | `2+2 --help` (two args, one of which is `--help`) | (nothing) | usage text | non-zero — confirms design §4's rule that the single-arg `-h`/`--help` short-circuit does NOT apply once there are 2+ args, even if one token is literally `--help` |

## Traceability check (NFR-2)

Every FR-1..FR-9 above has at least one happy-path and (where applicable) one error-path case in
this file, satisfying NFR-2's letter. FR-4/FR-5 don't have a distinct "error path" of their own
(an unknown constant name is an FR-6 concern, not an FR-4 one) — this mirrors design §11's own
traceability table and is not a gap.
