package main

import "fmt"

// gradient runs from green through yellow to red in the xterm 256-color
// palette, low values first.
var gradient = []int{46, 82, 118, 154, 190, 226, 214, 196}

const colorReset = "\x1b[0m"

// colorFor returns the escape sequence that sets the foreground color for
// level out of n levels.
func colorFor(level, n int) string {
	return fmt.Sprintf("\x1b[38;5;%dm", gradient[level*len(gradient)/n])
}
