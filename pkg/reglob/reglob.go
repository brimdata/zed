// The code in this source file is derived from:
// https://github.com/fitzgen/glob-to-regexp
// is covered by the copyright below.
// The changes are covered by the copyright and license in the
// LICENSE file in the root directory of this repository.

// Copyright (c) 2013, Nick Fitzgerald All rights reserved.
// See acknowledgments.txt for full license text from:
// https://github.com/fitzgen/glob-to-regexp#license

package reglob

import (
	"strings"
)

func Reglob(glob string) string {
	str := glob

	// The regexp we are building, as a string.
	reStr := ""

	// Whether we are matching so called "extended" globs (like bash) and should
	// support single character matching, matching ranges of characters, group
	// matching, etc.
	extended := false

	// When globstar is _false_ (default), '/foo/*' is translated a regexp like
	// '^\/foo\/.*$' which will match any string beginning with '/foo/'
	// When globstar is _true_, '/foo/*' is translated to regexp like
	// '^\/foo\/[^/]*$' which will match any string beginning with '/foo/' BUT
	// which does not have a '/' to the right of it.
	// E.g. with '/foo/*' these will match: "/foo/bar", "/foo/bar.txt" but
	// these will not "/foo/bar/baz", "/foo/bar/baz.txt"
	// Lastely, when globstar is _true_, "/foo/**" is equivelant to "/foo/*" when
	// globstar is _false_
	globstar := false

	// If we are doing extended matching, this boolean is true when we are inside
	// a group (eg {*.html,*.js}), and false otherwise.
	var inGroup = false

	// RegExp flags (eg "i" ) to pass in to RegExp constructor.
	flags := ""

	var c byte
	for i := 0; i < len(str); i++ {
		c = str[i]

		switch c {
		case '/', '$', '^', '+', '.', '(', ')', '=', '!', '|':
			reStr += "\\" + string(c)

		case '?':
			if extended {
				reStr += "."
			}

		case ']', '[':
			if extended {
				reStr += string(c)
			}

		case '{':
			if extended {
				inGroup = true
				reStr += "("
			}

		case '}':
			if extended {
				inGroup = false
				reStr += ")"
			}

		case ',':
			if inGroup {
				reStr += "|"
			}
			reStr += "\\" + string(c)

		case '\\':
			if len(str) > i+1 && str[i+1] == '*' {
				i++
				reStr += "\\*"
			} else {
				reStr += string(c)
			}

		case '*':
			// Move over all consecutive "*""s.
			// Also store the previous and next characters
			var prevChar byte
			if i-1 >= 0 {
				prevChar = str[i-1]
			}
			starCount := 1
			for len(str) > i+1 && str[i+1] == '*' {
				starCount++
				i++
			}

			var nextChar byte
			if i+1 < len(str) {
				nextChar = str[i+1]
			}

			if !globstar {
				// globstar is disabled, so treat any number of "*" as one
				reStr += ".*"
			} else {
				// globstar is enabled, so determine if this is a globstar segment
				isGlobstar := starCount > 1 && // multiple "*"'s
					(prevChar == '/' || prevChar == 0) && // from the start of the segment
					(nextChar == '/' || nextChar == 0) // to the end of the segment

				if isGlobstar {
					// it's a globstar, so match zero or more path segments
					reStr += "((?:[^/]*(?:/|$))*)"
					i++ // move over the "/"
				} else {
					// it's not a globstar, so only match one path segment
					reStr += "([^/]*)"
				}
			}

		default:
			reStr += string(c)
		}
	}

	// When regexp 'g' flag is specified don't
	// constrain the regular expression with ^ & $
	if flags == "" || !strings.Contains(flags, "g") {
		reStr = "^" + reStr + "$"
	}

	return reStr
}
