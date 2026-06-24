package main

import (
	"math"
	"testing"
)

func TestLevelEndpoints(t *testing.T) {
	s := scale{lo: 0, hi: 1}
	tests := []struct {
		v    float64
		want int
	}{
		{0, 0},
		{1, 7},
		{0.124, 0},
		{0.125, 1},
		{0.5, 4},
		{0.499, 3},
		{0.875, 7},
		{0.874, 6},
	}
	for _, tt := range tests {
		if got := s.level(tt.v, 8); got != tt.want {
			t.Errorf("level(%v) = %d, want %d", tt.v, got, tt.want)
		}
	}
}

func TestRender(t *testing.T) {
	tests := []struct {
		name string
		vals []float64
		want string
	}{
		{"seq", []float64{1, 2, 3, 4, 5, 6, 7, 8}, "▁▂▃▄▅▆▇█"},
		{"mixed", []float64{1, 5, 22, 13, 53}, "▁▁▄▂█"},
		{"constant", []float64{3, 3, 3}, "▄▄▄"},
		{"single", []float64{42}, "▄"},
		{"negative", []float64{-3, -2, -1, 0}, "▁▃▆█"},
		{"floats", []float64{0.1, 0.2, 0.16}, "▁█▅"},
		{"band edge", []float64{0.1, 0.2, 0.15}, "▁█▅"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := render(tt.vals, autoScale(tt.vals), blockTicks, false); got != tt.want {
				t.Errorf("render(%v) = %q, want %q", tt.vals, got, tt.want)
			}
		})
	}
}

func TestLevelBandEdges(t *testing.T) {
	// A value exactly k/8 of the way up the range starts band k, however
	// awkward the range is to represent in binary.
	ranges := []scale{{0, 1}, {0.1, 0.2}, {0, 0.3}, {-1.1, 2.2}, {1e-6, 3e-6}, {100, 1100}, {-0.7, -0.1}}
	for _, s := range ranges {
		for k := 0; k <= 8; k++ {
			v := s.lo + (s.hi-s.lo)*float64(k)/8
			if got, want := s.level(v, 8), min(k, 7); got != want {
				t.Errorf("scale %v: level(%v) = %d, want %d", s, v, got, want)
			}
		}
	}
}

func TestLevelHugeRange(t *testing.T) {
	// Ranges near the limits of a float64 must not overflow into an
	// out-of-range tick.
	for _, s := range []scale{{0, 1e308}, {-1e308, 1e308}, {-math.MaxFloat64, math.MaxFloat64}} {
		if got := s.level(s.lo, 8); got != 0 {
			t.Errorf("scale %v: level(lo) = %d, want 0", s, got)
		}
		if got := s.level(s.hi, 8); got != 7 {
			t.Errorf("scale %v: level(hi) = %d, want 7", s, got)
		}
		if mid := s.lo/2 + s.hi/2; s.level(mid, 8) != 4 {
			t.Errorf("scale %v: level(%v) = %d, want 4", s, mid, s.level(mid, 8))
		}
	}
}

func TestLevelClamps(t *testing.T) {
	s := scale{lo: 0, hi: 10}
	if got := s.level(-100, 8); got != 0 {
		t.Errorf("below range: level = %d, want 0", got)
	}
	if got := s.level(1e9, 8); got != 7 {
		t.Errorf("above range: level = %d, want 7", got)
	}

	flat := scale{lo: 5, hi: 5}
	for v, want := range map[float64]int{4: 0, 5: 3, 6: 7} {
		if got := flat.level(v, 8); got != want {
			t.Errorf("flat scale: level(%v) = %d, want %d", v, got, want)
		}
	}
}

func TestPin(t *testing.T) {
	data := scale{lo: 10, hi: 20}
	tests := []struct {
		name   string
		lo, hi bound
		want   scale
	}{
		{"none", bound{}, bound{}, scale{10, 20}},
		{"both", bound{0, true}, bound{100, true}, scale{0, 100}},
		{"min", bound{0, true}, bound{}, scale{0, 20}},
		{"max", bound{}, bound{50, true}, scale{10, 50}},
		{"min above data", bound{30, true}, bound{}, scale{30, 30}},
		{"max below data", bound{}, bound{5, true}, scale{5, 5}},
	}
	for _, tt := range tests {
		if got := data.pin(tt.lo, tt.hi); got != tt.want {
			t.Errorf("%s: pin = %+v, want %+v", tt.name, got, tt.want)
		}
	}
}
