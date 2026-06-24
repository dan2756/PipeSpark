package main

import (
	"math"
	"strings"
)

// Tick sets for drawing, lowest level first.
var (
	blockTicks = []rune("▁▂▃▄▅▆▇█")
	asciiTicks = []rune("_.-~=^*#")
)

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

// pin replaces either end of the scale with a fixed bound. When only one
// end is pinned and the data lies entirely beyond it, the scale collapses
// onto the pinned value so that everything is clamped to one end.
func (s scale) pin(lo, hi bound) scale {
	if lo.set {
		s.lo = lo.v
	}
	if hi.set {
		s.hi = hi.v
	}
	if s.hi < s.lo {
		if lo.set {
			s.hi = s.lo
		} else {
			s.lo = s.hi
		}
	}
	return s
}

// level returns which of n ticks v falls on. The range is cut into n
// equal-width bands, so lo always gets the lowest tick and hi the highest.
// Values outside the range are clamped, and a flat range puts everything
// on the middle tick.
func (s scale) level(v float64, n int) int {
	switch {
	case v < s.lo:
		return 0
	case v > s.hi:
		return n - 1
	case s.hi <= s.lo:
		return (n - 1) / 2
	}
	span, off := s.hi-s.lo, v-s.lo
	if math.IsInf(span, 0) {
		// The range is wider than a float64 can hold, as from -1e308 to
		// 1e308; halving everything first keeps the arithmetic finite.
		span, off = s.hi/2-s.lo/2, v/2-s.lo/2
	}
	// Dividing before multiplying by n keeps values near the top of a huge
	// range from overflowing. The nudge keeps values that belong exactly on
	// a band edge, such as 0.15 between 0.1 and 0.2, from rounding down
	// into the band below.
	i := int(off/span*float64(n) + 1e-9)
	return min(i, n-1)
}

// render draws one tick per value. With color set, ticks are colored by
// level and the color is reset at the end.
func render(vals []float64, s scale, ticks []rune, color bool) string {
	var b strings.Builder
	prev := -1
	for _, v := range vals {
		l := s.level(v, len(ticks))
		if color && l != prev {
			b.WriteString(colorFor(l, len(ticks)))
			prev = l
		}
		b.WriteRune(ticks[l])
	}
	if prev >= 0 {
		b.WriteString(colorReset)
	}
	return b.String()
}
