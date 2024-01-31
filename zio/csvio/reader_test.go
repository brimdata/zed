package csvio

import (
	"strings"
	"testing"

	"github.com/brimdata/zed"
	"github.com/stretchr/testify/require"
)

func TestNewReaderUsesContextParameter(t *testing.T) {
	arena := zed.NewArena(zed.NewContext())
	defer arena.Unref()
	rec, err := NewReader(arena.Zctx(), strings.NewReader("f\n1\n"), ReaderOpts{}).Read(arena)
	require.NoError(t, err)
	typ, err := arena.Zctx().LookupType(rec.Type().ID())
	require.NoError(t, err)
	require.Exactly(t, rec.Type(), typ)
}
