package main

import (
	"slices"
	"strings"
)

// aggregate combines the values that end up in one column.
type aggregate func([]float64) float64

var aggregates = map[string]aggregate{
	"avg": func(v []float64) float64 { return sum(v) / float64(len(v)) },
	"max": func(v []float64) float64 { return slices.Max(v) },
	"min": func(v []float64) float64 { return slices.Min(v) },
	"sum": sum,
}

func sum(vals []float64) float64 {
	var t float64
	for _, v := range vals {
		t += v
	}
	return t
}

// aggregateNames lists the valid --agg values for messages.
func aggregateNames() string {
	names := make([]string, 0, len(aggregates))
	for name := range aggregates {
		names = append(names, name)
	}
	slices.Sort(names)
	return strings.Join(names, ", ")
}

// resample fits vals into exactly width columns. To shrink, the series is
// split into width consecutive buckets whose sizes differ by at most one,
// and each bucket is reduced with agg. To grow, each value is repeated
// across the columns it covers.
func resample(vals []float64, width int, agg aggregate) []float64 {
	n := len(vals)
	if width <= 0 || n == 0 || width == n {
		return vals
	}
	out := make([]float64, width)
	if width > n {
		for i := range out {
			out[i] = vals[i*n/width]
		}
		return out
	}
	for i := range out {
		lo, hi := i*n/width, (i+1)*n/width
		out[i] = agg(vals[lo:hi])
	}
	return out
}
