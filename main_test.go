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
		{name: "ascii", stdin: "1 2 3 4 5 6 7 8", args: []string{"--ascii"}, out: "_.-~=^*#\n"},
		{name: "ascii constant", args: []string{"--ascii", "2", "2"}, out: "~~\n"},
		{name: "stats", args: []string{"--stats", "1", "5", "22", "13", "53"}, out: "▁▁▄▂█  min 1  max 53  avg 18.8  n 5\n"},
		{name: "width avg", stdin: "1 9 2 8 3 7 4 6", args: []string{"-w", "4"}, out: "▄▄▄▄\n"},
		{name: "width max", stdin: "1 9 2 8 3 7 4 6", args: []string{"--width", "4", "--agg", "max"}, out: "█▆▃▁\n"},
		{name: "width min", stdin: "1 9 2 8 3 7 4 6", args: []string{"--width=4", "--agg=min"}, out: "▁▃▆█\n"},
		{name: "width stretch", args: []string{"-w", "6", "1", "2", "3"}, out: "▁▁▅▅██\n"},
		{name: "width keeps raw stats", stdin: "1 9 2 8 3 7 4 6", args: []string{"-w", "2", "--stats"}, out: "▄▄  min 1  max 9  avg 5  n 8\n"},
		{name: "width with pinned scale", stdin: "0 0 10 10", args: []string{"-w", "2", "--min", "0", "--max", "20"}, out: "▁▅\n"},

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
		{name: "zero width", args: []string{"-w", "0", "1"}, code: 2, errText: "--width must be at least 1"},
		{name: "negative width", args: []string{"--width", "-3", "1"}, code: 2, errText: "--width must be at least 1"},
		{name: "bad agg", args: []string{"--agg", "median", "1"}, code: 2, errText: `--agg must be one of avg, max, min, sum, not "median"`},
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

func TestFieldFromStdin(t *testing.T) {
	out, errOut, code := runCLI(t, "x 1 5\ny 2 3\nheader\nz 3 9\n", "--field", "2")
	if code != 0 || out != "▃▁█\n" {
		t.Errorf("exit %d, stdout %q, stderr %q", code, out, errOut)
	}

	_, errOut, code = runCLI(t, "1 5\n2\n", "--field", "2", "--strict")
	if code != 1 || !strings.Contains(errOut, "line 2: no field 2: line has 1 number(s)") {
		t.Errorf("strict short line: exit %d, stderr %q", code, errOut)
	}

	for _, args := range [][]string{{"--field", "0"}, {"--field=-1"}} {
		if _, errOut, code := runCLI(t, "1", args...); code != 2 || !strings.Contains(errOut, "--field must be at least 1") {
			t.Errorf("%v: exit %d, stderr %q", args, code, errOut)
		}
	}
	if _, errOut, code := runCLI(t, "", "--field", "1", "1", "2"); code != 2 || !strings.Contains(errOut, "--field reads standard input") {
		t.Errorf("--field with args: exit %d, stderr %q", code, errOut)
	}
}

func TestColor(t *testing.T) {
	run := func(noColor string, args ...string) string {
		var out bytes.Buffer
		c := &cli{
			stdin:  strings.NewReader(""),
			stdout: &out,
			stderr: &bytes.Buffer{},
			getenv: func(k string) string {
				if k == "NO_COLOR" {
					return noColor
				}
				return ""
			},
		}
		if code := c.run(args); code != 0 {
			t.Fatalf("%v: exit %d", args, code)
		}
		return out.String()
	}

	if got, want := run("", "--color", "1", "2", "2"), "\x1b[38;5;46m▁\x1b[38;5;196m██\x1b[0m\n"; got != want {
		t.Errorf("--color = %q, want %q", got, want)
	}
	if got, want := run("", "--color", "4", "4"), "\x1b[38;5;154m▄▄\x1b[0m\n"; got != want {
		t.Errorf("--color constant = %q, want %q", got, want)
	}
	if got, want := run("1", "--color", "1", "2"), "▁█\n"; got != want {
		t.Errorf("NO_COLOR set: %q, want %q", got, want)
	}
	if got, want := run("", "1", "2"), "▁█\n"; got != want {
		t.Errorf("no --color: %q, want %q", got, want)
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
