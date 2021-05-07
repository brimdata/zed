package csvio

import (
	"strings"
	"testing"

	"github.com/brimdata/zed/zson"
	"github.com/stretchr/testify/require"
)

func TestNewReaderUsesContextParameter(t *testing.T) {
	zctx := zson.NewContext()
	rec, err := NewReader(strings.NewReader("f\n1\n"), zctx).Read()
	require.NoError(t, err)
	typ, err := zctx.LookupType(rec.Type.ID())
	require.NoError(t, err)
	require.Exactly(t, rec.Type, typ)
}
