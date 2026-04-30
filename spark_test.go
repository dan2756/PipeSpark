package main

import "testing"

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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := render(tt.vals, autoScale(tt.vals), blockTicks); got != tt.want {
				t.Errorf("render(%v) = %q, want %q", tt.vals, got, tt.want)
			}
		})
	}
}
