package main

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// summary describes a series for --stats.
type summary struct {
	min, max, sum float64
	n             int
}

func summarize(vals []float64) summary {
	if len(vals) == 0 {
		return summary{}
	}
	s := summary{min: vals[0], max: vals[0], n: len(vals)}
	for _, v := range vals {
		s.min = min(s.min, v)
		s.max = max(s.max, v)
		s.sum += v
	}
	return s
}

func (s summary) String() string {
	avg := 0.0
	if s.n > 0 {
		avg = s.sum / float64(s.n)
	}
	return fmt.Sprintf("min %s  max %s  avg %s  n %d",
		formatNum(s.min), formatNum(s.max), formatNum(avg), s.n)
}

// formatNum prints v compactly: whole numbers without a fraction, larger
// values to two decimal places and small ones to three significant
// digits, with trailing zeros dropped.
func formatNum(v float64) string {
	if v == 0 {
		return "0" // also catches -0
	}
	if v == math.Trunc(v) {
		return strconv.FormatFloat(v, 'f', 0, 64)
	}
	var s string
	if math.Abs(v) >= 1 {
		s = strconv.FormatFloat(v, 'f', 2, 64)
	} else {
		s = strconv.FormatFloat(v, 'g', 3, 64)
	}
	if strings.Contains(s, ".") && !strings.Contains(s, "e") {
		s = strings.TrimSuffix(strings.TrimRight(s, "0"), ".")
	}
	return s
}
