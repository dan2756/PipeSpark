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
spark [numbers...]
command | spark
```

Numbers can come from the command line or from standard input, separated by
any whitespace or commas. Anything that isn't a number is skipped, so you can
usually pipe output straight in without cleaning it up first.

```
$ spark 1 5 22 13 53
▁▁▄▂█
$ echo 1,2,3 | spark
▁▅█
```

The scale runs from the smallest value to the largest. A series where every
value is the same is drawn as a flat line of middle bars.

If no numbers are found, spark prints an error and exits with status 1.

## License

MIT
