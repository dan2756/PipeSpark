// Command spark turns a series of numbers into a sparkline.
//
//	$ seq 1 8 | spark
//	▁▂▃▄▅▆▇█
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"runtime/debug"
	"strconv"
	"strings"

	"golang.org/x/term"
)

// version is set at build time with -ldflags "-X main.version=v1.2.3".
// When empty, the module version from the build info is used instead.
var version = ""

const usage = `Usage: spark [flags] [numbers...]
       command | spark [flags]

Turns a series of numbers into a sparkline. Numbers may be separated by
whitespace, commas or '=' (so key=value pairs work). Tokens that aren't
numbers are skipped unless --strict is given; NaN and Inf are always
rejected.

Flags:
      --min N       pin the bottom of the scale; lower values are clamped
      --max N       pin the top of the scale; higher values are clamped
  -w, --width N     resample the series to exactly N columns
      --agg FUNC    how --width combines values: avg, max, min or sum
                    (default avg)
      --ascii       draw with ASCII characters (_.-~=^*#) instead of blocks
      --stats       print min, max, avg and count after the sparkline
      --color       color bars from green (low) to red (high); ignored
                    when NO_COLOR is set
      --strict      treat anything that isn't a number as an error
      --field K     take only the K-th number from each line of input
      --stream      keep reading and redraw the last --width values in
                    place (width defaults to fit the terminal)
  -h, --help        show this help
      --version     print the version

Examples:
  seq 1 8 | spark
  spark 1 5 22 13 53
  spark -3 -1 0 2 5
  echo 1,2,3 | spark
  spark --min 0 --max 100 < cpu.txt
  spark -w 60 --agg max --stats < latency.log
  vmstat 1 | spark --stream --field 13 --min 0 --max 100
  ping example.com | spark --stream --field 4 --stats
`

// maxWidth caps --width, which sets how much is allocated up front; no
// sparkline needs to be this wide.
const maxWidth = 1_000_000

// cli holds what the program talks to, so tests can swap it out.
type cli struct {
	stdin  io.Reader
	stdout io.Writer
	stderr io.Writer
	getenv func(string) string
	// stdinTerm reports whether stdin is a terminal.
	stdinTerm func() bool
	// stdoutTerm reports whether stdout is a terminal and, if so, how many
	// columns wide it is.
	stdoutTerm func() (cols int, ok bool)
}

// options are the parsed command-line flags.
type options struct {
	lo, hi bound
	strict bool
	width  int
	agg    string
	ascii  bool
	stats  bool
	color  bool
	field  int
	stream bool

	aggFn aggregate // resolved from agg
}

// bound is an optional end of the scale given on the command line.
type bound struct {
	v   float64
	set bool
}

func (b *bound) String() string {
	if b == nil || !b.set {
		return ""
	}
	return strconv.FormatFloat(b.v, 'g', -1, 64)
}

func (b *bound) Set(s string) error {
	v, err := strconv.ParseFloat(s, 64)
	// Overflow (1e999) is a range error, not a syntax error. ParseFloat
	// reports it as ±Inf plus ErrRange; named Inf and NaN parse cleanly.
	if err != nil && !errors.Is(err, strconv.ErrRange) {
		return errors.New("not a number")
	}
	if math.IsNaN(v) || math.IsInf(v, 0) {
		if errors.Is(err, strconv.ErrRange) {
			return errors.New("out of range")
		}
		return errors.New("must be a finite number")
	}
	b.v, b.set = v, true
	return nil
}

func main() {
	c := &cli{
		stdin:      os.Stdin,
		stdout:     os.Stdout,
		stderr:     os.Stderr,
		getenv:     os.Getenv,
		stdinTerm:  func() bool { return term.IsTerminal(int(os.Stdin.Fd())) },
		stdoutTerm: func() (int, bool) { return terminalWidth(os.Stdout) },
	}
	os.Exit(c.run(os.Args[1:]))
}

// terminalWidth reports whether f is a terminal and, if so, its width.
func terminalWidth(f *os.File) (int, bool) {
	fd := int(f.Fd())
	if !term.IsTerminal(fd) {
		return 0, false
	}
	cols, _, err := term.GetSize(fd)
	if err != nil || cols <= 0 {
		return 80, true
	}
	return cols, true
}

func (c *cli) run(args []string) int {
	var o options
	fs := flag.NewFlagSet("spark", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.Var(&o.lo, "min", "")
	fs.Var(&o.hi, "max", "")
	fs.BoolVar(&o.strict, "strict", false, "")
	fs.IntVar(&o.width, "w", 0, "")
	fs.IntVar(&o.width, "width", 0, "")
	fs.StringVar(&o.agg, "agg", "avg", "")
	fs.BoolVar(&o.ascii, "ascii", false, "")
	fs.BoolVar(&o.stats, "stats", false, "")
	fs.BoolVar(&o.color, "color", false, "")
	fs.IntVar(&o.field, "field", 0, "")
	fs.BoolVar(&o.stream, "stream", false, "")
	showVersion := fs.Bool("version", false, "")

	flagArgs, nums := splitArgs(fs, args)
	if err := fs.Parse(flagArgs); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fmt.Fprint(c.stdout, usage)
			return 0
		}
		return c.usageError(err)
	}
	nums = append(nums, fs.Args()...)
	if *showVersion {
		fmt.Fprintln(c.stdout, "spark", versionString())
		return 0
	}
	if err := o.validate(fs); err != nil {
		return c.usageError(err)
	}
	if len(nums) > 0 && (o.stream || o.field > 0) {
		name := "--stream"
		if !o.stream {
			name = "--field"
		}
		return c.usageError(fmt.Errorf("%s reads standard input; it can't be used with numbers as arguments", name))
	}
	if o.color && c.env("NO_COLOR") != "" {
		o.color = false
	}
	if len(nums) == 0 && c.stdinTerm != nil && c.stdinTerm() {
		// Nothing piped in and nothing on the command line: waiting for
		// someone to type numbers is rarely what was meant.
		fmt.Fprint(c.stderr, usage)
		return 2
	}

	p := parser{strict: o.strict, field: o.field}
	if o.stream {
		return c.stream(&o, p)
	}
	var vals []float64
	var err error
	if len(nums) > 0 {
		vals, err = p.parseLine(strings.Join(nums, " "))
	} else {
		vals, err = p.readAll(c.stdin)
	}
	if err != nil {
		return c.fail(err)
	}
	if len(vals) == 0 {
		return c.fail(errors.New("no numbers in input"))
	}

	if _, err := fmt.Fprintln(c.stdout, o.line(vals)); err != nil {
		return c.fail(err)
	}
	return 0
}

// validate checks flag combinations that the flag package can't.
func (o *options) validate(fs *flag.FlagSet) error {
	given := map[string]bool{}
	fs.Visit(func(f *flag.Flag) { given[f.Name] = true })

	if o.lo.set && o.hi.set && o.lo.v >= o.hi.v {
		return errors.New("--min must be less than --max")
	}
	if (given["w"] || given["width"]) && o.width < 1 {
		return errors.New("--width must be at least 1")
	}
	if o.width > maxWidth {
		return fmt.Errorf("--width must be at most %d", maxWidth)
	}
	if given["field"] && o.field < 1 {
		return errors.New("--field must be at least 1")
	}
	o.aggFn = aggregates[o.agg]
	if o.aggFn == nil {
		return fmt.Errorf("--agg must be one of %s, not %q", aggregateNames(), o.agg)
	}
	return nil
}

// line draws vals as a single line of output, resampled to --width.
func (o *options) line(vals []float64) string {
	shown := vals
	if o.width > 0 {
		shown = resample(vals, o.width, o.aggFn)
	}
	return o.draw(shown, vals)
}

// draw renders shown as a sparkline, followed by stats for raw if asked.
func (o *options) draw(shown, raw []float64) string {
	ticks := blockTicks
	if o.ascii {
		ticks = asciiTicks
	}
	out := render(shown, autoScale(shown).pin(o.lo, o.hi), ticks, o.color)
	if o.stats {
		out += "  " + summarize(raw).String()
	}
	return out
}

// splitArgs separates flags from numbers so the two can be mixed freely
// and negative numbers aren't mistaken for flags. Everything after "--"
// is taken as a number.
func splitArgs(fs *flag.FlagSet, args []string) (flags, nums []string) {
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--":
			return flags, append(nums, args[i+1:]...)
		case len(a) < 2 || a[0] != '-' || isNumeric(a):
			nums = append(nums, a)
		default:
			flags = append(flags, a)
			name := strings.TrimLeft(a, "-")
			if !strings.Contains(name, "=") && takesValue(fs, name) && i+1 < len(args) {
				i++
				flags = append(flags, args[i])
			}
		}
	}
	return flags, nums
}

// isNumeric reports whether every token in s is a number, as in "-3" or
// "-1,-2".
func isNumeric(s string) bool {
	toks := fields(s)
	for _, t := range toks {
		if _, ok := parseNumber(t); !ok {
			return false
		}
	}
	return len(toks) > 0
}

// takesValue reports whether the named flag consumes the next argument.
func takesValue(fs *flag.FlagSet, name string) bool {
	f := fs.Lookup(name)
	if f == nil {
		return false
	}
	if b, ok := f.Value.(interface{ IsBoolFlag() bool }); ok && b.IsBoolFlag() {
		return false
	}
	return true
}

func (c *cli) env(key string) string {
	if c.getenv == nil {
		return ""
	}
	return c.getenv(key)
}

func (c *cli) fail(err error) int {
	fmt.Fprintf(c.stderr, "spark: %v\n", err)
	return 1
}

func (c *cli) usageError(err error) int {
	fmt.Fprintf(c.stderr, "spark: %v\nRun 'spark --help' for usage.\n", err)
	return 2
}

func versionString() string {
	if version != "" {
		return version
	}
	if bi, ok := debug.ReadBuildInfo(); ok && bi.Main.Version != "" && bi.Main.Version != "(devel)" {
		return bi.Main.Version
	}
	return "dev"
}
