package schema

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/brimdata/zed/api"
	"github.com/brimdata/zed/pkg/iosrc"
	"github.com/brimdata/zed/ppl/zqd/auth"
	"github.com/segmentio/ksuid"
)

type SpaceRow struct {
	tableName struct{}          `pg:"space"` // This is needed so the postgres orm knows the correct table name
	ID        api.SpaceID       `json:"id"`
	DataURI   iosrc.URI         `json:"data_uri"`
	Name      string            `json:"name"`
	Storage   api.StorageConfig `json:"storage"`
	TenantID  auth.TenantID     `json:"tenant_id"`
}

func NewSpaceID() api.SpaceID {
	id := ksuid.New()
	return api.SpaceID(fmt.Sprintf("sp_%s", id.String()))
}

func invalidResourceNameRune(r rune) bool {
	return r == '/' || !unicode.IsPrint(r)
}

func ValidResourceName(s string) bool {
	return strings.IndexFunc(s, invalidResourceNameRune) == -1
}

// SafeSpaceName converts the proposed name to a name that adheres to the constraints
// placed on a space's name (i.e. follows the name regex).
func SafeSpaceName(proposed string) string {
	var sb strings.Builder
	for _, r := range proposed {
		if invalidResourceNameRune(r) {
			r = '_'
		}
		sb.WriteRune(r)
	}
	return sb.String()
}
