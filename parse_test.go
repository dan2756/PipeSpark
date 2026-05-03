package main

import (
	"flag"
	"reflect"
	"strings"
	"testing"
)

func TestFields(t *testing.T) {
	got := fields("1,2, 3\t4\r\n,,5 ")
	want := []string{"1", "2", "3", "4", "5"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("fields = %q, want %q", got, want)
	}
}

func TestParseLine(t *testing.T) {
	tests := []struct {
		line string
		want []float64
	}{
		{"1 2 3", []float64{1, 2, 3}},
		{"-1.5,+2,.5", []float64{-1.5, 2, 0.5}},
		{"1e3 2.5E-1 -4e+2", []float64{1000, 0.25, -400}},
		{"load 0.42 0.36 0.30", []float64{0.42, 0.36, 0.30}},
		{"12ms 15 - 3", []float64{15, 3}},
		{"", nil},
	}
	for _, tt := range tests {
		got, err := parser{}.parseLine(tt.line)
		if err != nil {
			t.Errorf("parseLine(%q): %v", tt.line, err)
			continue
		}
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("parseLine(%q) = %v, want %v", tt.line, got, tt.want)
		}
	}
}

func TestParseLineStrict(t *testing.T) {
	p := parser{strict: true}
	if _, err := p.parseLine("1 2 3"); err != nil {
		t.Errorf("clean line: %v", err)
	}
	_, err := p.parseLine("1 two 3")
	if err == nil || !strings.Contains(err.Error(), `"two"`) {
		t.Errorf("got %v, want error naming \"two\"", err)
	}
}

func TestParseLineRejectsNonFinite(t *testing.T) {
	tests := map[string]string{
		"1 NaN":     "NaN and Inf",
		"inf":       "NaN and Inf",
		"-Infinity": "NaN and Inf",
		"+inf 2":    "NaN and Inf",
		"1e999":     "out of range",
		"-1e400":    "out of range",
	}
	for line, want := range tests {
		for _, p := range []parser{{}, {strict: true}} {
			_, err := p.parseLine(line)
			if err == nil || !strings.Contains(err.Error(), want) {
				t.Errorf("parseLine(%q) strict=%v: got %v, want %q", line, p.strict, err, want)
			}
		}
	}
}

func TestReadAll(t *testing.T) {
	got, err := parser{}.readAll(strings.NewReader("1\n2 3\n\nx 4,5"))
	if err != nil {
		t.Fatal(err)
	}
	if want := []float64{1, 2, 3, 4, 5}; !reflect.DeepEqual(got, want) {
		t.Errorf("readAll = %v, want %v", got, want)
	}

	_, err = parser{strict: true}.readAll(strings.NewReader("1\n2\nthree\n"))
	if err == nil || !strings.HasPrefix(err.Error(), "line 3:") {
		t.Errorf("got %v, want error on line 3", err)
	}
}

func TestReadAllLongLine(t *testing.T) {
	// One huge line, well past bufio.Scanner's default token limit.
	line := strings.Repeat("7 ", 100000)
	got, err := parser{}.readAll(strings.NewReader(line))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 100000 {
		t.Errorf("read %d values, want 100000", len(got))
	}
}

func TestSplitArgs(t *testing.T) {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	var b bound
	fs.Var(&b, "min", "")
	fs.Bool("strict", false, "")

	tests := []struct {
		args        []string
		flags, nums []string
	}{
		{
			args: []string{"1", "2", "3"},
			nums: []string{"1", "2", "3"},
		},
		{
			args:  []string{"-3", "--min", "-10", "4", "--strict", "-0.5"},
			flags: []string{"--min", "-10", "--strict"},
			nums:  []string{"-3", "4", "-0.5"},
		},
		{
			args:  []string{"--min=-1", "-1,-2"},
			flags: []string{"--min=-1"},
			nums:  []string{"-1,-2"},
		},
		{
			args:  []string{"--strict", "--", "--min", "5"},
			flags: []string{"--strict"},
			nums:  []string{"--min", "5"},
		},
		{
			args:  []string{"-x", "1"},
			flags: []string{"-x"},
			nums:  []string{"1"},
		},
	}
	for _, tt := range tests {
		flags, nums := splitArgs(fs, tt.args)
		if !reflect.DeepEqual(flags, tt.flags) || !reflect.DeepEqual(nums, tt.nums) {
			t.Errorf("splitArgs(%q) = %q, %q; want %q, %q", tt.args, flags, nums, tt.flags, tt.nums)
		}
	}
}
