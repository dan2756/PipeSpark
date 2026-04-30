package main

import (
	"bufio"
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

// parseLine returns every finite number on line, skipping anything else.
func parseLine(line string) []float64 {
	var vals []float64
	for _, tok := range fields(line) {
		v, err := strconv.ParseFloat(tok, 64)
		if err != nil || math.IsNaN(v) || math.IsInf(v, 0) {
			continue
		}
		vals = append(vals, v)
	}
	return vals
}

// readValues reads r to the end and returns all the numbers in it.
func readValues(r io.Reader) ([]float64, error) {
	br := bufio.NewReader(r)
	var vals []float64
	for {
		line, err := br.ReadString('\n')
		vals = append(vals, parseLine(line)...)
		if err == io.EOF {
			return vals, nil
		}
		if err != nil {
			return nil, err
		}
	}
}
