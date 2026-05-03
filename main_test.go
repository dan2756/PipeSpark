package main

import (
	"bytes"
	"strings"
	"testing"
)

func runCLI(t *testing.T, stdin string, args ...string) (stdout, stderr string, code int) {
	t.Helper()
	var out, errOut bytes.Buffer
	c := &cli{stdin: strings.NewReader(stdin), stdout: &out, stderr: &errOut}
	code = c.run(args)
	return out.String(), errOut.String(), code
}

func TestRun(t *testing.T) {
	tests := []struct {
		name    string
		stdin   string
		args    []string
		out     string
		code    int
		errText string
	}{
		{name: "stdin", stdin: "1\n2\n3\n4\n5\n6\n7\n8\n", out: "▁▂▃▄▅▆▇█\n"},
		{name: "args", args: []string{"1", "5", "22", "13", "53"}, out: "▁▁▄▂█\n"},
		{name: "commas", stdin: "1,2,3\n", out: "▁▅█\n"},
		{name: "args win over stdin", stdin: "9 9 9", args: []string{"1", "2"}, out: "▁█\n"},
		{name: "negative args", args: []string{"-3", "-2", "-1", "0"}, out: "▁▃▆█\n"},
		{name: "double dash", args: []string{"--", "-1", "1"}, out: "▁█\n"},
		{name: "skips words", stdin: "cpu 10\ncpu 20\n", out: "▁█\n"},
		{name: "constant", stdin: "5 5 5 5", out: "▄▄▄▄\n"},
		{name: "single", args: []string{"7"}, out: "▄\n"},
		{name: "clamped", args: []string{"--min", "0", "--max", "10", "-5", "5", "15"}, out: "▁▅█\n"},
		{name: "flags after numbers", args: []string{"-5", "0", "--min", "-10", "--max", "10"}, out: "▃▅\n"},
		{name: "min only", args: []string{"--min=0", "5", "10"}, out: "▅█\n"},
		{name: "data below pinned min", args: []string{"--min", "100", "1", "2"}, out: "▁▁\n"},
		{name: "data above pinned max", args: []string{"--max", "0", "1", "2"}, out: "██\n"},

		{name: "strict", stdin: "1 two 3\n", args: []string{"--strict"}, code: 1, errText: `line 1: not a number: "two"`},
		{name: "strict args", args: []string{"--strict", "1", "x"}, code: 1, errText: `not a number: "x"`},
		{name: "nan", stdin: "1 NaN 3", code: 1, errText: "NaN and Inf are not allowed"},
		{name: "inf", stdin: "1\n+Inf\n", code: 1, errText: `line 2: "+Inf"`},
		{name: "overflow", stdin: "1e999", code: 1, errText: "out of range"},
		{name: "empty", code: 1, errText: "no numbers in input"},
		{name: "only words", stdin: "hello world\n", code: 1, errText: "no numbers in input"},

		{name: "min equals max", args: []string{"--min", "5", "--max", "5", "1"}, code: 2, errText: "--min must be less than --max"},
		{name: "bad min", args: []string{"--min", "x", "1"}, code: 2, errText: "invalid value"},
		{name: "min needs value", args: []string{"--min"}, code: 2, errText: "needs an argument"},
		{name: "unknown flag", args: []string{"--nope"}, code: 2, errText: "not defined"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, errOut, code := runCLI(t, tt.stdin, tt.args...)
			if code != tt.code {
				t.Errorf("exit code = %d, want %d (stderr %q)", code, tt.code, errOut)
			}
			if out != tt.out {
				t.Errorf("stdout = %q, want %q", out, tt.out)
			}
			if tt.errText != "" && !strings.Contains(errOut, tt.errText) {
				t.Errorf("stderr = %q, want it to contain %q", errOut, tt.errText)
			}
			if tt.errText == "" && errOut != "" {
				t.Errorf("unexpected stderr %q", errOut)
			}
		})
	}
}

func TestHelpAndVersion(t *testing.T) {
	out, _, code := runCLI(t, "", "--help")
	if code != 0 || !strings.HasPrefix(out, "Usage: spark") {
		t.Errorf("--help: code %d, out %q", code, out)
	}
	out, _, code = runCLI(t, "", "-h")
	if code != 0 || !strings.HasPrefix(out, "Usage: spark") {
		t.Errorf("-h: code %d, out %q", code, out)
	}
	out, _, code = runCLI(t, "", "--version")
	if code != 0 || !strings.HasPrefix(out, "spark ") {
		t.Errorf("--version: code %d, out %q", code, out)
	}
}
