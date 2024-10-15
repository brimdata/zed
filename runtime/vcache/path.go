package vcache

import (
	"fmt"

	"github.com/brimdata/super/pkg/field"
)

// A path is an array of string or Forks
type Path []any //XXX clean this up later
type Fork []Path

func NewProjection(paths []field.Path) Path {
	var out Path
	for _, path := range paths {
		out = insertPath(out, path)
	}
	return out
}

// XXX this is N*N in path lengths... fix?
func insertPath(existing Path, addition field.Path) Path {
	if len(addition) == 0 {
		return existing
	}
	if len(existing) == 0 {
		return convertFieldPath(addition)
	}
	switch elem := existing[0].(type) {
	case string:
		if elem == addition[0] {
			return append(Path{elem}, insertPath(existing[1:], addition[1:])...)
		}
		return Path{Fork{existing, convertFieldPath(addition)}}
	case Fork:
		return Path{addToFork(elem, addition)}
	default:
		panic(fmt.Sprintf("bad type encounted in insertPath: %T", elem))
	}
}

func addToFork(fork Fork, addition field.Path) Fork {
	// The first element of each path in a fork must be the key distinguishing
	// the different paths (so no embedded Fork as the first element of a fork)
	for k, path := range fork {
		if path[0].(string) == addition[0] {
			fork[k] = insertPath(path, addition)
			return fork
		}
	}
	// No common prefix so add the addition to the fork.
	return append(fork, convertFieldPath(addition))
}

func convertFieldPath(path field.Path) Path {
	var out []any
	for _, s := range path {
		out = append(out, s)
	}
	return out
}
