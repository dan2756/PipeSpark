# spark

Turns piped numbers into a sparkline.

```
$ seq 1 8 | spark
▁▂▃▄▅▆▇█
```

spark reads numbers from arguments or a pipe and draws them as a one-line
chart. It skips whatever isn't a number, so most command output can go
straight in, and it can resample long series, pin the scale, follow live
input in place, and print a quick summary alongside.

## Install

```
go install github.com/dan2756/spark@latest
```

## Usage

```
spark [flags] [numbers...]
command | spark [flags]
```

Numbers can come from the command line or from standard input, separated by
any whitespace, commas or `=`. Negative numbers, decimals and scientific
notation (`-4.2`, `.5`, `1e-3`) all work. Anything that isn't a number is
skipped, so you can usually pipe output straight in without cleaning it up
first. Splitting on `=` means `key=value` pairs give up their value:
`time=12.3` counts as the number `12.3`.

```
$ spark 1 5 22 13 53
▁▁▄▂█
$ echo 1,2,3 | spark
▁▅█
$ spark -3 -1 0 2 5
▁▃▄▆█
```

Flags and numbers can be mixed in any order; a leading `-` on a number is not
mistaken for a flag. Use `--` if you want everything after it treated as
input. Run with no arguments and nothing piped in, spark prints its usage
rather than waiting for you to type numbers.

A few more:

```
$ du -sk */ | spark                        # directory sizes
▁▁█▂▁▃
$ git log --format=%ad --date=format:%u | sort | uniq -c | spark --field 1
▆█▇█▅▁▁
$ for i in $(seq 20); do curl -so /dev/null -w '%{time_total}\n' https://example.com; done | spark --stats
▂▁▁▃▁▁█▁▂▁▁▁▂▁▁▁▄▁▁▁  min 0.081  max 0.412  avg 0.12  n 20
```

## Flags

| Flag | Description |
| --- | --- |
| `--min N` | Pin the bottom of the scale. Lower values are clamped to the lowest bar. |
| `--max N` | Pin the top of the scale. Higher values are clamped to the highest bar. |
| `-w`, `--width N` | Resample the series to exactly N columns (at most 1,000,000). |
| `--agg FUNC` | How `--width` combines values: `avg` (default), `max`, `min` or `sum`. |
| `--ascii` | Draw with `_.-~=^*#` instead of Unicode block characters. |
| `--stats` | Print `min`, `max`, `avg` and `n` after the sparkline. |
| `--color` | Color bars on a green-to-red gradient. Ignored when `NO_COLOR` is set. |
| `--strict` | Treat anything that isn't a number as an error instead of skipping it. |
| `--field K` | Take only the K-th number from each line of input. |
| `--stream` | Keep reading and redraw the last `--width` values in place. Without `--width`, the window fits the terminal, or is 60 columns when stdout isn't one. |
| `-h`, `--help` | Show usage. |
| `--version` | Print the version. |

## Scaling

By default the scale runs from the smallest value to the largest, cut into
eight equal bands, one per bar. The minimum always gets `▁` and the maximum
always gets `█`. A series where every value is the same is drawn as a flat
line of middle bars (`▄`).

Pin the scale when you're comparing runs or the numbers have a natural range:

```
$ spark --min 0 --max 100 12 50 99 140
▁▅██
```

`NaN` and `Inf` can't be placed on a scale, so they are always rejected, even
without `--strict`.

## Resampling

Long series can be squeezed into a fixed width. `--width N` splits the
values into N consecutive buckets (their sizes differ by at most one) and
draws one bar per bucket. By default each bar is the bucket's average; use
`--agg max` to keep spikes visible, `--agg min` for dips, or `--agg sum` for
counts.

```
$ seq 1 100 | spark -w 20
▁▁▁▂▂▃▃▃▄▄▅▅▆▆▆▇▇███
$ spark -w 4 --agg max 1 9 2 8 3 7 4 6
█▆▃▁
```

If N is larger than the number of values, each value is repeated to fill the
width. The scale is computed from the resampled values, so the full range of
bars is always used.

## Stats

`--stats` appends a summary of the input (before any resampling):

```
$ spark --stats 1 5 22 13 53
▁▁▄▂█  min 1  max 53  avg 18.8  n 5
```

## ASCII output

For terminals or fonts without the block characters, `--ascii` draws the same
eight levels as `_.-~=^*#`:

```
$ seq 1 8 | spark --ascii
_.-~=^*#
```

## Picking a column

`--field K` takes the K-th number on each line and ignores the rest. Only
numbers are counted, so words, headers and blank lines don't shift the
columns, and lines with fewer than K numbers are skipped (with `--strict`
they are an error).

```
$ printf 'web1 200 0.31\nweb2 200 0.87\nweb3 500 0.12\n' | spark --field 2
▃█▁
```

Here the host names aren't numbers, so field 1 is the status code and field 2
is the response time.

## Live input

`--stream` keeps reading standard input and redraws the sparkline after
every line, so you can watch a command that prints a number at a time. The
window holds the last `--width` values. Without `--width` it is sized to fit
the terminal (leaving room for `--stats` if you asked for it, and narrowing
further if the stats need more room than that). When stdout isn't a terminal,
the window is 60 columns wide.

```
$ vmstat 1 | spark --stream --field 13 --min 0 --max 100 --stats
▁▁▂▁▁▃▇█▅▂▁▁▁  min 0  max 97  avg 18.4  n 13
$ ping 1.1.1.1 | spark --stream --field 4 --stats
▁▁▂▁█▁▁▂▁▁  min 11.8  max 35.1  avg 14.32  n 10
```

In ping's output (Linux and macOS alike) the round-trip time is the fourth
number on each reply line: `64 bytes from 1.1.1.1: icmp_seq=3 ttl=57
time=35.1 ms` holds 64, 3, 57 and 35.1. The header, timeouts and the summary
at the end have fewer numbers and are skipped.

On a terminal the line is redrawn in place with a carriage return, and
Ctrl-C finishes the line before exiting. When standard output isn't a
terminal each update is written as its own line instead, which makes the
output easy to log or test. Without `--field`, every number on a line is
added to the window. `--stats` describes the values currently in the
window, and `--agg` has no effect in stream mode.

## Color

`--color` colors each bar by its height, from green for the lowest to red for
the highest, using the 256-color palette. Following
[no-color.org](https://no-color.org), setting `NO_COLOR` to any non-empty
value turns it off, so it's safe to keep `--color` in an alias.

## Platform notes

spark builds and behaves the same on Linux and macOS. Stream mode needs a
terminal that understands carriage returns and the `ESC [ K` erase sequence,
which every common terminal does. `--color` uses the xterm 256-color
palette.

## Exit status

| Code | Meaning |
| --- | --- |
| 0 | Sparkline printed. |
| 1 | No numbers in the input, the input was invalid, or the output couldn't be written. |
| 2 | Bad flags, or no input given at a terminal. |
| 130 | Interrupted with Ctrl-C in `--stream` mode. |

## License

MIT
