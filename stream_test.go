package main

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"unicode/utf8"
)

// fakeTerm makes stdout look like a terminal of the given width.
func fakeTerm(cols int) func() (int, bool) {
	return func() (int, bool) { return cols, true }
}

func runStream(t *testing.T, term func() (int, bool), stdin string, args ...string) (stdout, stderr string, code int) {
	t.Helper()
	var out, errOut bytes.Buffer
	c := &cli{
		stdin:      strings.NewReader(stdin),
		stdout:     &out,
		stderr:     &errOut,
		stdoutTerm: term,
	}
	code = c.run(append([]string{"--stream"}, args...))
	return out.String(), errOut.String(), code
}

func TestStreamFrames(t *testing.T) {
	out, errOut, code := runStream(t, nil, "1\n2\n3\n", "-w", "2")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errOut)
	}
	// One frame per line; the window keeps the last two values.
	if want := "▄\n▁█\n▁█\n"; out != want {
		t.Errorf("stdout = %q, want %q", out, want)
	}
}

func TestStreamRedrawsInPlaceOnTerminal(t *testing.T) {
	out, _, code := runStream(t, fakeTerm(80), "1\n2\n", "-w", "3")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if want := "\r▄\x1b[K\r▁█\x1b[K\n"; out != want {
		t.Errorf("stdout = %q, want %q", out, want)
	}
}

func TestStreamWindow(t *testing.T) {
	// Lines with words mixed in, carrying the values 1..9, 0, 1..9, 0.
	var in strings.Builder
	for i := 1; i <= 20; i++ {
		fmt.Fprintf(&in, "%s %d\n", strings.Repeat("x", i%3), i%10)
	}
	out, _, code := runStream(t, nil, in.String(), "-w", "5")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	frames := strings.Split(strings.TrimSuffix(out, "\n"), "\n")
	if len(frames) != 20 {
		t.Fatalf("got %d frames, want 20", len(frames))
	}
	for i, f := range frames {
		if n, want := utf8.RuneCountInString(f), min(i+1, 5); n != want {
			t.Errorf("frame %d has %d ticks, want %d", i, n, want)
		}
	}
	// The last five values are 6 7 8 9 0.
	if got, want := frames[19], "▆▇██▁"; got != want {
		t.Errorf("last frame = %q, want %q", got, want)
	}
}

func TestStreamWidthFitsTerminal(t *testing.T) {
	in := strings.Repeat("5\n", 200)
	tests := []struct {
		name string
		cols int
		args []string
		want int
	}{
		{"fills terminal", 40, nil, 39},
		{"leaves room for stats", 80, []string{"--stats"}, 80 - 1 - statsRoom},
		{"never too narrow", 30, []string{"--stats"}, minStreamWidth},
		{"explicit width wins", 40, []string{"-w", "7"}, 7},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, _, _ := runStream(t, fakeTerm(tt.cols), in, tt.args...)
			frames := strings.Split(strings.TrimSuffix(out, "\n"), "\r")
			last := strings.TrimSuffix(frames[len(frames)-1], "\x1b[K")
			last, _, _ = strings.Cut(last, "  ")
			if n := utf8.RuneCountInString(last); n != tt.want {
				t.Errorf("window = %d, want %d", n, tt.want)
			}
		})
	}
}

func TestStreamNarrowsForLongStats(t *testing.T) {
	// Stats longer than the room set aside for them would make every frame
	// wrap, so the window gives up columns until the whole line fits.
	var in strings.Builder
	for i := 0; i <= 40; i++ {
		fmt.Fprintf(&in, "%v\n", 12345678.5+float64(i))
	}
	out, _, code := runStream(t, fakeTerm(80), in.String(), "--stats")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	frames := strings.Split(strings.TrimPrefix(strings.TrimSuffix(out, "\n"), "\r"), "\r")
	for i, f := range frames {
		f = strings.TrimSuffix(f, "\x1b[K")
		if n := utf8.RuneCountInString(f); n > 79 {
			t.Errorf("frame %d is %d columns wide: %q", i, n, f)
		}
	}
	last := strings.TrimSuffix(frames[len(frames)-1], "\x1b[K")
	if want := "▁▁▁▁▂▂▂▃▃▃▄▄▄▅▅▅▆▆▆▇▇▇████  min 12345693.5  max 12345718.5  avg 12345706  n 26"; last != want {
		t.Errorf("last frame = %q, want %q", last, want)
	}

	// An explicit --width is left alone.
	out, _, _ = runStream(t, fakeTerm(80), in.String(), "--stats", "-w", "31")
	if !strings.HasSuffix(out, "  n 31\x1b[K\n") {
		t.Errorf("--width 31: stdout ends %q", out[max(0, len(out)-20):])
	}
}

func TestStreamDefaultWidthWithoutTerminal(t *testing.T) {
	out, _, _ := runStream(t, nil, strings.Repeat("1 2\n", 100))
	frames := strings.Split(strings.TrimSuffix(out, "\n"), "\n")
	if n := utf8.RuneCountInString(frames[len(frames)-1]); n != defaultStreamWidth {
		t.Errorf("window = %d, want %d", n, defaultStreamWidth)
	}
}

func TestStreamField(t *testing.T) {
	in := "procs memory\n r  b free\n 1 0 10\n 2 0 20\n"
	out, _, code := runStream(t, nil, in, "--field", "3", "-w", "5")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if want := "▄\n▁█\n"; out != want {
		t.Errorf("stdout = %q, want %q", out, want)
	}
}

func TestStreamStats(t *testing.T) {
	out, _, _ := runStream(t, nil, "1\n3\n", "-w", "5", "--stats")
	want := "▄  min 1  max 1  avg 1  n 1\n▁█  min 1  max 3  avg 2  n 2\n"
	if out != want {
		t.Errorf("stdout = %q, want %q", out, want)
	}
}

func TestStreamErrors(t *testing.T) {
	_, errOut, code := runStream(t, nil, "\nfoo\n")
	if code != 1 || !strings.Contains(errOut, "no numbers in input") {
		t.Errorf("no numbers: exit %d, stderr %q", code, errOut)
	}

	// A bad line stops the stream, but on a terminal the line is finished
	// first so the error starts on a fresh one.
	out, errOut, code := runStream(t, fakeTerm(80), "1\nx\n", "--strict")
	if code != 1 || !strings.Contains(errOut, `line 2: not a number: "x"`) {
		t.Errorf("strict: exit %d, stderr %q", code, errOut)
	}
	if !strings.HasSuffix(out, "\n") {
		t.Errorf("strict: stdout %q doesn't end the line", out)
	}

	_, errOut, code = runStream(t, nil, "", "1", "2")
	if code != 2 || !strings.Contains(errOut, "--stream reads standard input") {
		t.Errorf("args: exit %d, stderr %q", code, errOut)
	}
}

func TestWindowPush(t *testing.T) {
	w := &window{size: 3}
	for i := 1; i <= 5; i++ {
		w.push(float64(i))
	}
	if got := w.vals; len(got) != 3 || got[0] != 3 || got[1] != 4 || got[2] != 5 {
		t.Errorf("window = %v, want [3 4 5]", got)
	}
}
