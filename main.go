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
	"os"
	"runtime/debug"
	"strings"
)

// version is set at build time with -ldflags "-X main.version=v1.2.3".
// When empty, the module version from the build info is used instead.
var version = ""

const usage = `Usage: spark [numbers...]
       command | spark

Turns a series of numbers into a sparkline. Numbers may be separated by
whitespace or commas; anything that isn't a number is skipped.

Flags:
  -h, --help      show this help
      --version   print the version

Examples:
  seq 1 8 | spark
  spark 1 5 22 13 53
  echo 1,2,3 | spark
`

// cli holds the streams the program talks to, so tests can swap them out.
type cli struct {
	stdin  io.Reader
	stdout io.Writer
	stderr io.Writer
}

func main() {
	c := &cli{stdin: os.Stdin, stdout: os.Stdout, stderr: os.Stderr}
	os.Exit(c.run(os.Args[1:]))
}

func (c *cli) run(args []string) int {
	fs := flag.NewFlagSet("spark", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	showVersion := fs.Bool("version", false, "")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fmt.Fprint(c.stdout, usage)
			return 0
		}
		return c.usageError(err)
	}
	if *showVersion {
		fmt.Fprintln(c.stdout, "spark", versionString())
		return 0
	}

	var vals []float64
	if fs.NArg() > 0 {
		vals = parseLine(strings.Join(fs.Args(), " "))
	} else {
		var err error
		if vals, err = readValues(c.stdin); err != nil {
			return c.fail(err)
		}
	}
	if len(vals) == 0 {
		return c.fail(errors.New("no numbers in input"))
	}

	fmt.Fprintln(c.stdout, render(vals, autoScale(vals), blockTicks))
	return 0
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
