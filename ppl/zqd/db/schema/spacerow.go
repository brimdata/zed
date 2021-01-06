package schema

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/brimsec/zq/api"
	"github.com/brimsec/zq/pkg/iosrc"
	"github.com/segmentio/ksuid"
)

type SpaceRow struct {
	tableName struct{}          `pg:"space"` // This is needed so the postgres orm knows the correct table name
	ID        api.SpaceID       `json:"id"`
	DataURI   iosrc.URI         `json:"data_uri"`
	Name      string            `json:"name"`
	ParentID  api.SpaceID       `json:"parent_id"`
	Storage   api.StorageConfig `json:"storage"`
}

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

// SafeSpaceName converts the proposed name to a name that adheres to the constraints
// placed on a space's name (i.e. follows the name regex).
func SafeSpaceName(proposed string) string {
	var sb strings.Builder
	for _, r := range proposed {
		if invalidSpaceNameRune(r) {
			r = '_'
		}
		sb.WriteRune(r)
	}
	return sb.String()
}
