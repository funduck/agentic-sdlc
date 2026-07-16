package main

import (
	"bytes"
	"strings"
	"testing"
)

// FR-1: a single expression argument prints the result to stdout and
// exits 0.
func TestRunEvaluatesExpression(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"2 + 3 * 4"}, &stdout, &stderr)

	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if got := stdout.String(); got != "14\n" {
		t.Errorf("stdout = %q, want %q", got, "14\n")
	}
	if stderr.Len() != 0 {
		t.Errorf("stderr = %q, want empty", stderr.String())
	}
}

// FR-6: a syntax error prints nothing to stdout, an error to stderr, and
// exits non-zero.
func TestRunSyntaxError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"2 @ 3"}, &stdout, &stderr)

	if code == 0 {
		t.Error("exit code = 0, want non-zero")
	}
	if stdout.Len() != 0 {
		t.Errorf("stdout = %q, want empty", stdout.String())
	}
	if !strings.Contains(stderr.String(), "unexpected character") {
		t.Errorf("stderr = %q, want it to mention the bad character", stderr.String())
	}
}

// FR-7: a domain error prints nothing to stdout, an error to stderr, and
// exits non-zero.
func TestRunDomainError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"1 / 0"}, &stdout, &stderr)

	if code == 0 {
		t.Error("exit code = 0, want non-zero")
	}
	if stdout.Len() != 0 {
		t.Errorf("stdout = %q, want empty", stdout.String())
	}
	if !strings.Contains(stderr.String(), "division by zero") {
		t.Errorf("stderr = %q, want it to mention division by zero", stderr.String())
	}
}

// FR-9: --help and -h print usage to stdout and exit 0.
func TestRunHelp(t *testing.T) {
	for _, flag := range []string{"--help", "-h"} {
		var stdout, stderr bytes.Buffer
		code := run([]string{flag}, &stdout, &stderr)

		if code != 0 {
			t.Errorf("run(%q): exit code = %d, want 0", flag, code)
		}
		if stderr.Len() != 0 {
			t.Errorf("run(%q): stderr = %q, want empty", flag, stderr.String())
		}
		out := stdout.String()
		for _, want := range []string{"calc", "sin", "pi", "e"} {
			if !strings.Contains(out, want) {
				t.Errorf("run(%q): usage text missing %q:\n%s", flag, want, out)
			}
		}
	}
}

// FR-9: no expression argument prints usage to stderr and exits non-zero.
func TestRunNoArgs(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{}, &stdout, &stderr)

	if code == 0 {
		t.Error("exit code = 0, want non-zero")
	}
	if stdout.Len() != 0 {
		t.Errorf("stdout = %q, want empty", stdout.String())
	}
	if stderr.Len() == 0 {
		t.Error("stderr is empty, want usage text")
	}
}

// FR-9 / design.md §4: two-or-more arguments are rejected the same way as
// zero arguments — even when one of the extra tokens is literally "--help" —
// because FR-1 requires the expression as a single (quoted) argument.
func TestRunMultipleArgsIsUsageError(t *testing.T) {
	cases := [][]string{
		{"2+2", "extra"},
		{"2+2", "--help"},
	}
	for _, args := range cases {
		var stdout, stderr bytes.Buffer
		code := run(args, &stdout, &stderr)

		if code == 0 {
			t.Errorf("run(%v): exit code = 0, want non-zero", args)
		}
		if stdout.Len() != 0 {
			t.Errorf("run(%v): stdout = %q, want empty", args, stdout.String())
		}
		if stderr.Len() == 0 {
			t.Errorf("run(%v): stderr is empty, want usage text", args)
		}
	}
}

// Every stdout/stderr write ends with a single trailing newline
// (design.md §4a).
func TestRunOutputEndsWithNewline(t *testing.T) {
	var stdout, stderr bytes.Buffer
	run([]string{"2 + 2"}, &stdout, &stderr)
	if got := stdout.String(); !strings.HasSuffix(got, "\n") {
		t.Errorf("stdout = %q, want trailing newline", got)
	}

	stdout.Reset()
	stderr.Reset()
	run([]string{"1 / 0"}, &stdout, &stderr)
	if got := stderr.String(); !strings.HasSuffix(got, "\n") {
		t.Errorf("stderr = %q, want trailing newline", got)
	}
}
