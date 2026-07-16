// Command calc is a single-shot command-line calculator: it evaluates one
// mathematical expression, given as a single command-line argument, and
// prints the result. See usageText below for the supported syntax.
package main

import (
	"fmt"
	"io"
	"os"

	"cli-calc/internal/calc"
)

const usageText = `calc: evaluate a single mathematical expression

Usage:
  calc "<expression>"
  calc --help | -h

Example:
  calc "2 + 3 * 4"        -> 14

Operators (highest to lowest precedence):
  ()          grouping
  -x          unary minus
  ^           exponentiation (right-associative)
  * /         multiplication, division (left-associative)
  + -         addition, subtraction (left-associative)

Functions (radians for all trigonometric functions), called as name(x):
  sin cos tan asin acos atan   trigonometric functions, in radians
  sqrt                         square root
  log                          base-10 logarithm
  ln                           natural logarithm
  exp                          e raised to the given power

Constants:
  pi   3.14159...
  e    2.71828...`

// run implements the CLI argument contract (FR-1, FR-9) as a pure function
// of its inputs and outputs, so it's testable with in-memory buffers
// instead of spawning the built binary. main() is just a thin wrapper.
func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 1 && (args[0] == "-h" || args[0] == "--help") {
		fmt.Fprintln(stdout, usageText)
		return 0
	}

	if len(args) == 1 {
		return evaluateAndPrint(args[0], stdout, stderr)
	}

	// Zero args, or two-or-more args (including a stray "--help" among
	// them): FR-1 requires the expression as a single argument, so
	// unquoted multi-token invocations are rejected via the same
	// "missing expression" usage-error path rather than silently joined.
	fmt.Fprintln(stderr, usageText)
	return 1
}

func evaluateAndPrint(expr string, stdout, stderr io.Writer) int {
	result, err := calc.Evaluate(expr)
	if err != nil {
		fmt.Fprintln(stderr, "Error: "+err.Error())
		return 1
	}
	fmt.Fprintln(stdout, calc.FormatResult(result))
	return 0
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}
