// Package colw lays out columns for display of a list when you don't know
// ahead of time how many columns should exist.  Kind of like what ls does.
package colw

import (
	"errors"
	"io"
)

// ErrDoesNotFit is returned when the items do not fit in the supplied width.
// The only way this happens is if there is at least one item larger than the
// width less padding.  Otherwise, you would get one column taking up all the width.
var ErrDoesNotFit = errors.New("items do not fit")

func slen(in []string, pad int) []int {
	out := make([]int, 0, len(in))
	for _, s := range in {
		out = append(out, len(s)+pad)
	}
	return out
}

func build(lens []int, ncol int) []int {
	widths := make([]int, ncol)
	for k, v := range lens {
		colno := k % ncol
		if v > widths[colno] {
			widths[colno] = v
		}
	}
	return widths
}

func fit(trial []int, budget int) bool {
	var n int
	for _, v := range trial {
		n += v
	}
	return n <= budget
}

// Layout takes a list of strings, a pad for each column, and a total
// width of each row and computes the layout with the largest number of
// columns that will fit in the given row width if the strings are displayed
// in left to right order a row a time.
func Layout(in []string, width, pad int) (widths []int) {
	lens := slen(in, pad)
	var result []int
	for ncol := 1; ncol < width; ncol++ {
		next := build(lens, ncol)
		if !fit(next, width) {
			return result
		}
		result = next
	}
	return nil
}

func makePadding(n int, padchar byte) []byte {
	b := make([]byte, n)
	for k := 0; k < n; k++ {
		b[k] = padchar
	}
	return b
}

var newline = []byte{'\n'}

// Write writes the list of strings to the given writer with padding and
// newlines according to Layout
func Write(w io.Writer, in []string, width, pad int) error {
	widths := Layout(in, width, pad)
	if widths == nil {
		return ErrDoesNotFit
	}
	ncol := len(widths)
	padding := makePadding(width, ' ')
	for k, s := range in {
		_, err := w.Write([]byte(s))
		if err != nil {
			return err
		}
		if k == len(in)-1 {
			_, err := w.Write(newline)
			return err
		}
		colno := k % ncol
		if colno == ncol-1 {
			_, err := w.Write(newline)
			if err != nil {
				return err
			}
		} else {
			width := widths[colno]
			pad := width - len(s)
			if pad > 0 {
				_, err := w.Write(padding[0:pad])
				if err != nil {
					return err
				}

			}
		}
	}
	return nil
}
