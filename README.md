# spark

Turns piped numbers into a sparkline.

```
$ seq 1 8 | spark
▁▂▃▄▅▆▇█
```

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
any whitespace or commas. Negative numbers, decimals and scientific notation
(`-4.2`, `.5`, `1e-3`) all work. Anything that isn't a number is skipped, so
you can usually pipe output straight in without cleaning it up first.

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
input.

## Flags

| Flag | Description |
| --- | --- |
| `--min N` | Pin the bottom of the scale. Lower values are clamped to the lowest bar. |
| `--max N` | Pin the top of the scale. Higher values are clamped to the highest bar. |
| `-w`, `--width N` | Resample the series to exactly N columns. |
| `--agg FUNC` | How `--width` combines values: `avg` (default), `max`, `min` or `sum`. |
| `--ascii` | Draw with `_.-~=^*#` instead of Unicode block characters. |
| `--stats` | Print `min`, `max`, `avg` and `n` after the sparkline. |
| `--strict` | Treat anything that isn't a number as an error instead of skipping it. |
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

## Exit status

| Code | Meaning |
| --- | --- |
| 0 | Sparkline printed. |
| 1 | No numbers in the input, or the input was invalid. |
| 2 | Bad flags. |

## License

MIT
