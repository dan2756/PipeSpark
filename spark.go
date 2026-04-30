package main

import "strings"

// blockTicks are the eight levels of a sparkline, lowest first.
var blockTicks = []rune("▁▂▃▄▅▆▇█")

// scale maps values in [lo, hi] onto tick levels.
type scale struct {
	lo, hi float64
}

// autoScale returns a scale spanning the smallest and largest value.
func autoScale(vals []float64) scale {
	if len(vals) == 0 {
		return scale{}
	}
	s := scale{lo: vals[0], hi: vals[0]}
	for _, v := range vals[1:] {
		s.lo = min(s.lo, v)
		s.hi = max(s.hi, v)
	}
	return s
}

// level returns which of n ticks v falls on. The range is cut into n
// equal-width bands, so lo always gets the lowest tick and hi the highest.
// A flat range puts everything on the middle tick.
func (s scale) level(v float64, n int) int {
	switch {
	case s.hi <= s.lo:
		return (n - 1) / 2
	case v <= s.lo:
		return 0
	case v >= s.hi:
		return n - 1
	}
	i := int((v - s.lo) * float64(n) / (s.hi - s.lo))
	return min(i, n-1)
}

// render draws one tick per value.
func render(vals []float64, s scale, ticks []rune) string {
	var b strings.Builder
	for _, v := range vals {
		b.WriteRune(ticks[s.level(v, len(ticks))])
	}
	return b.String()
}
