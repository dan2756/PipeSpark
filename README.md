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

## Exit status

| Code | Meaning |
| --- | --- |
| 0 | Sparkline printed. |
| 1 | No numbers in the input, or the input was invalid. |
| 2 | Bad flags. |

## License

MIT
