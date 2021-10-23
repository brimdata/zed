package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/brimdata/zed/expr/function"
)

func main() {
	args := os.Args[1:]
	if len(args) < 1 {
		check(errors.New("need 1 arg for where to write markdown file"))
	}
	f, err := os.Create(args[0])
	check(err)
	defer f.Close()
	all := function.All.Copy()
	sort.SliceStable(all, func(i, j int) bool {
		return all[i].Name < all[j].Name
	})
	tableOfContents(f, all)
	for _, fn := range all {
		document(f, fn)
	}
}

func tableOfContents(w io.Writer, all function.Funcs) {
	fmt.Fprint(w, "# Table of Contents\n\n")
	for _, f := range all {
		fmt.Fprint(w, "- ")
		link(w, f.Name)
		fmt.Fprint(w, "\n")
	}
	fmt.Fprintf(w, "\n")
}

func link(w io.Writer, name string) {
	lnk := strings.ToLower(name)
	lnk = strings.ReplaceAll(lnk, " ", "-")
	lnk = strings.ReplaceAll(lnk, "_", "-")
	fmt.Fprintf(w, "[%s](#%s)", name, lnk)
}

func document(w io.Writer, fn *function.Func) {
	fmt.Fprintf(w, "## %s\n\n", fn.Name)
	codeBlock(w)
	signature(w, fn)
	codeBlock(w)
	fmt.Fprint(w, "\n")
	description(w, fn)
	fmt.Fprint(w, "\n")
	examples(w, fn)
}

func signature(w io.Writer, fn *function.Func) {
	fmt.Fprintf(w, "%s(", fn.Name)
	s := fn.Signature
	for k, typ := range s.Args {
		if k != 0 {
			fmt.Fprint(w, ", ")
		}
		// fmt.Fprintf(w, "%s %s", string(byte(97+k)), typ.String())
		fmt.Fprintf(w, "%s", typ.String())
	}
	fmt.Fprint(w, ")")
	fmt.Fprintf(w, " -> %s\n", s.Return)
}

func description(w io.Writer, fn *function.Func) {
	if fn.Desc != "" {
		fmt.Fprintln(w, fn.Desc)
	}
}

func examples(w io.Writer, fn *function.Func) {
	for _, ex := range fn.Examples {
		fmt.Fprintf(w, "### Example:\n\n")
		codeBlock(w)
		fmt.Fprintln(w, ex.Zed)
		codeBlock(w)
		if ex.Input != "" {
			fmt.Fprint(w, "\n")
			fmt.Fprintln(w, "**Input:**")
			codeBlock(w)
			fmt.Fprintln(w, ex.Input)
			codeBlock(w)
		}
		if ex.Output != "" {
			fmt.Fprint(w, "\n")
			fmt.Fprintln(w, "**Output:**")
			codeBlock(w)
			fmt.Fprintln(w, ex.Output)
			codeBlock(w)
		}
	}
}

func codeBlock(w io.Writer, tags ...string) {
	fmt.Fprint(w, "```\n")
}

// func tableRow(w io.Writer, args ...string) {
// for k, arg := range args {
// if k == 0 {
// fmt.Fprint(w, "|")
// }
// fmt.Fprint(w, " "+arg+" |")
// }
// fmt.Fprint(w, "\n")
// for k := range args {
// if k == 0 {
// fmt.Fprintf
// }
// }
// }

func check(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
