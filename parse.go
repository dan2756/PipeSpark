package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
	"unicode"
)

// fields splits a line into tokens on whitespace and commas.
func fields(line string) []string {
	return strings.FieldsFunc(line, func(r rune) bool {
		return r == ',' || unicode.IsSpace(r)
	})
}

// parseNumber reports whether tok is a number. Values too large for a
// float64 come back as ±Inf rather than failing, so the caller can tell
// them apart from words.
func parseNumber(tok string) (float64, bool) {
	v, err := strconv.ParseFloat(tok, 64)
	if err != nil && !errors.Is(err, strconv.ErrRange) {
		return 0, false
	}
	return v, true
}

// checkFinite rejects NaN and infinite values, which have no place on a
// scale.
func checkFinite(tok string, v float64) error {
	if !math.IsNaN(v) && !math.IsInf(v, 0) {
		return nil
	}
	switch strings.ToLower(strings.TrimLeft(tok, "+-")) {
	case "nan", "inf", "infinity":
		return fmt.Errorf("%q: NaN and Inf are not allowed", tok)
	}
	return fmt.Errorf("%q is out of range", tok)
}

// parser turns text into values.
type parser struct {
	strict bool // non-numeric tokens are an error instead of being skipped
}

// parseLine returns the numbers on a single line.
func (p parser) parseLine(line string) ([]float64, error) {
	var vals []float64
	for _, tok := range fields(line) {
		v, ok := parseNumber(tok)
		if !ok {
			if p.strict {
				return nil, fmt.Errorf("not a number: %q", tok)
			}
			continue
		}
		if err := checkFinite(tok, v); err != nil {
			return nil, err
		}
		vals = append(vals, v)
	}
	return vals, nil
}

// readAll reads r to the end and returns every number in it. Parse errors
// say which line they came from.
func (p parser) readAll(r io.Reader) ([]float64, error) {
	br := bufio.NewReader(r)
	var vals []float64
	for n := 1; ; n++ {
		line, err := br.ReadString('\n')
		if err != nil && err != io.EOF {
			return nil, err
		}
		lv, perr := p.parseLine(line)
		if perr != nil {
			return nil, fmt.Errorf("line %d: %w", n, perr)
		}
		vals = append(vals, lv...)
		if err == io.EOF {
			return vals, nil
		}
	}
}
