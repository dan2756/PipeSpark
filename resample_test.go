package main

import (
	"math"
	"reflect"
	"testing"
)

func TestResample(t *testing.T) {
	ten := []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	tests := []struct {
		name  string
		vals  []float64
		width int
		agg   string
		want  []float64
	}{
		{"avg pairs", ten, 5, "avg", []float64{1.5, 3.5, 5.5, 7.5, 9.5}},
		{"max pairs", ten, 5, "max", []float64{2, 4, 6, 8, 10}},
		{"min pairs", ten, 5, "min", []float64{1, 3, 5, 7, 9}},
		{"sum pairs", ten, 5, "sum", []float64{3, 7, 11, 15, 19}},
		// 10 into 3: buckets of 3, 3 and 4.
		{"uneven avg", ten, 3, "avg", []float64{2, 5, 8.5}},
		{"uneven sum", ten, 3, "sum", []float64{6, 15, 34}},
		{"one column", ten, 1, "avg", []float64{5.5}},
		{"same width", ten, 10, "sum", ten},
		{"stretch", []float64{1, 2, 3}, 6, "avg", []float64{1, 1, 2, 2, 3, 3}},
		{"stretch uneven", []float64{1, 2, 3}, 8, "sum", []float64{1, 1, 1, 2, 2, 2, 3, 3}},
		{"stretch single", []float64{4}, 3, "avg", []float64{4, 4, 4}},
		{"negative", []float64{-4, -2, 2, 4}, 2, "avg", []float64{-3, 3}},
		// Totals beyond a float64 average correctly and sum to the limit.
		{"huge avg", []float64{1e308, 1e308, -1e308, -1e308}, 2, "avg", []float64{1e308, -1e308}},
		{"huge sum", []float64{1e308, 1e308, -1e308, -1e308}, 2, "sum", []float64{math.MaxFloat64, -math.MaxFloat64}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resample(tt.vals, tt.width, aggregates[tt.agg])
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("resample(%v, %d, %s) = %v, want %v", tt.vals, tt.width, tt.agg, got, tt.want)
			}
		})
	}
}

func TestResampleKeepsEveryValue(t *testing.T) {
	// Every input value must land in exactly one bucket, whatever the
	// width, so the bucket sums always add back up to the total.
	vals := make([]float64, 97)
	for i := range vals {
		vals[i] = float64(i + 1)
	}
	for w := 1; w <= len(vals); w++ {
		got := resample(vals, w, aggregates["sum"])
		if len(got) != w {
			t.Fatalf("width %d: got %d columns", w, len(got))
		}
		if s := sum(got); s != sum(vals) {
			t.Errorf("width %d: buckets sum to %v, want %v", w, s, sum(vals))
		}
	}
}

func TestSummary(t *testing.T) {
	got := summarize([]float64{1, 5, 22, 13, 53}).String()
	if want := "min 1  max 53  avg 18.8  n 5"; got != want {
		t.Errorf("summary = %q, want %q", got, want)
	}
	if got := summarize([]float64{1e308, 1e308}).avg; got != 1e308 {
		t.Errorf("avg of huge values = %v, want 1e308", got)
	}
}

func TestFormatNum(t *testing.T) {
	tests := map[float64]string{
		0:           "0",
		1:           "1",
		-7:          "-7",
		18.8:        "18.8",
		10.0 / 3:    "3.33",
		-2.5:        "-2.5",
		12.999:      "13",
		0.33333:     "0.333",
		0.000123:    "0.000123",
		1.5e-7:      "1.5e-07",
		1234567.891: "1234567.89",
		1e20:        "100000000000000000000",
	}
	for v, want := range tests {
		if got := formatNum(v); got != want {
			t.Errorf("formatNum(%v) = %q, want %q", v, got, want)
		}
	}
}
