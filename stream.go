package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
)

const (
	// defaultStreamWidth is the window size when stdout isn't a terminal
	// and --width isn't given.
	defaultStreamWidth = 60
	// statsRoom is how many columns are left free for --stats when the
	// window is sized to fit the terminal. Longer stats narrow the window
	// further as they appear.
	statsRoom = 48
	// minStreamWidth keeps the window usable on very narrow terminals.
	minStreamWidth = 10
)

// window holds the most recent values, oldest first.
type window struct {
	vals []float64
	size int
}

func (w *window) push(v float64) {
	if len(w.vals) < w.size {
		w.vals = append(w.vals, v)
		return
	}
	copy(w.vals, w.vals[1:])
	w.vals[len(w.vals)-1] = v
}

// fit narrows the window for good, dropping the oldest values, until
// width(w.vals) is at most cols or only minStreamWidth values are left.
func (w *window) fit(cols int, width func([]float64) int) {
	for len(w.vals) > minStreamWidth && width(w.vals) > cols {
		w.vals = w.vals[1:]
		w.size = len(w.vals)
	}
}

// streamWidth picks the window size: --width if given, otherwise whatever
// fits on one line of the terminal.
func (o *options) streamWidth(cols int, tty bool) int {
	if o.width > 0 {
		return o.width
	}
	if !tty {
		return defaultStreamWidth
	}
	w := cols - 1 // writing the last column makes some terminals wrap
	if o.stats {
		w -= statsRoom
	}
	return max(w, minStreamWidth)
}

// frameWidth is how many columns a frame of vals takes up on screen.
func (o *options) frameWidth(vals []float64) int {
	n := len(vals)
	if o.stats {
		n += len("  " + summarize(vals).String())
	}
	return n
}

type readResult struct {
	line string
	err  error
}

// readLines sends each line of r to out until r is exhausted or done is
// closed. The last result carries the read error, io.EOF included.
func readLines(r io.Reader, out chan<- readResult, done <-chan struct{}) {
	br := bufio.NewReader(r)
	for {
		line, err := br.ReadString('\n')
		select {
		case out <- readResult{line, err}:
		case <-done:
			return
		}
		if err != nil {
			return
		}
	}
}

// stream reads input line by line and redraws the latest window of values
// after every line that adds any. On a terminal the sparkline is redrawn in
// place with a carriage return; otherwise every frame is written on a line
// of its own.
func (c *cli) stream(o *options, p parser) int {
	var cols int
	var tty bool
	if c.stdoutTerm != nil {
		cols, tty = c.stdoutTerm()
	}
	win := &window{size: o.streamWidth(cols, tty)}
	// The room left for --stats is a guess; if the real stats turn out
	// longer, the window has to give way or every frame would wrap and
	// leave a stale copy behind.
	fit := tty && o.width == 0 && o.stats

	lines := make(chan readResult)
	done := make(chan struct{})
	defer close(done)
	go readLines(c.stdin, lines, done)

	// Only trap Ctrl-C when redrawing in place, so the shell prompt doesn't
	// land on the end of the sparkline. A nil channel never fires.
	var interrupt chan os.Signal
	if tty {
		interrupt = make(chan os.Signal, 1)
		signal.Notify(interrupt, os.Interrupt)
		defer signal.Stop(interrupt)
	}

	drawn := false
	// endLine finishes an in-place redraw so the next prompt starts on a
	// fresh line. Callers that already have an error to report ignore a
	// failure here; the success path must not.
	endLine := func() error {
		if tty && drawn {
			_, err := fmt.Fprintln(c.stdout)
			return err
		}
		return nil
	}
	for n := 1; ; n++ {
		var r readResult
		select {
		case r = <-lines:
		case <-interrupt:
			_ = endLine()
			return 130
		}

		vals, err := p.parseLine(r.line)
		if err != nil {
			_ = endLine()
			return c.fail(fmt.Errorf("line %d: %w", n, err))
		}
		for _, v := range vals {
			win.push(v)
		}
		if len(vals) > 0 {
			if fit {
				win.fit(cols-1, o.frameWidth)
			}
			if err := c.frame(o.draw(win.vals, win.vals), tty); err != nil {
				// A later frame can fail after an earlier one was drawn.
				// Finish that line so the error isn't appended to it.
				_ = endLine()
				return c.fail(err)
			}
			drawn = true
		}

		if r.err == io.EOF {
			break
		}
		if r.err != nil {
			_ = endLine()
			return c.fail(r.err)
		}
	}
	if !drawn {
		return c.fail(errors.New("no numbers in input"))
	}
	if err := endLine(); err != nil {
		return c.fail(err)
	}
	return 0
}

// frame writes one redraw of the sparkline.
func (c *cli) frame(line string, tty bool) error {
	if tty {
		// Return to the start of the line, draw, and clear whatever is
		// left of a longer previous frame.
		_, err := io.WriteString(c.stdout, "\r"+line+"\x1b[K")
		return err
	}
	_, err := io.WriteString(c.stdout, line+"\n")
	return err
}
