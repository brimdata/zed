package space

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/brimsec/zq/api"
	"github.com/segmentio/ksuid"
)

func NewSpaceID() api.SpaceID {
	id := ksuid.New()
	return api.SpaceID(fmt.Sprintf("sp_%s", id.String()))
}

func invalidSpaceNameRune(r rune) bool {
	return r == '/' || !unicode.IsPrint(r)
}

func ValidSpaceName(s string) bool {
	return strings.IndexFunc(s, invalidSpaceNameRune) == -1
}

// SafeName converts the proposed name to a name that adheres to the constraints
// placed on a space's name (i.e. follows the name regex).
func SafeName(proposed string) string {
	var sb strings.Builder
	for _, r := range proposed {
		if invalidSpaceNameRune(r) {
			r = '_'
		}
		sb.WriteRune(r)
	}
	return sb.String()
}
