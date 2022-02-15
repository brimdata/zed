package srverr

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

type errtyp1 struct{ s string }

func (e *errtyp1) Error() string { return e.s }

func TestE(t *testing.T) {
	e0 := E("bad")
	require.Equal(t, "bad", e0.Error())
	e1 := E("%v %x", "x1", 16)
	require.Equal(t, "x1 10", e1.Error())
	e2 := E(Invalid, "%v %x", "x1", 16)
	require.Equal(t, "invalid operation: x1 10", e2.Error())

	// verify Unwrap
	e3 := &errtyp1{s: "deep"}
	e4 := E("got: %w", e3)
	require.Equal(t, "got: deep", e4.Error())
	require.NotEqual(t, e4.(*Error).Err, e3)
	require.True(t, errors.Is(e4, e3))

	e5 := E(e3)
	require.Equal(t, "deep", e5.Error())
	require.Equal(t, e5.(*Error).Err, e3)
	require.True(t, errors.Is(e5, e3))

	// Message vs Error
	e6 := E(Invalid, "e6")
	require.Equal(t, "invalid operation: e6", e6.Error())
	require.Equal(t, "e6", e6.(*Error).Message())
	e7 := E(Invalid)
	require.Equal(t, "invalid operation", e7.Error())
	require.Equal(t, "invalid operation", e7.(*Error).Message())
	e8 := E("e8")
	require.Equal(t, "e8", e8.Error())
	require.Equal(t, "e8", e8.(*Error).Message())

	// E errors
	require.Panics(t, func() { E() })
	e9 := E(1)
	require.Regexp(t, "unknown type.*error_test", e9.Error())

	// Error formatting
	e10 := E("foo%d")
	require.Equal(t, "foo%d", e10.Error())
	e11 := E("foo%d", 10)
	require.Equal(t, "foo10", e11.Error())
}

func TestIs(t *testing.T) {
	require.True(t, errors.Is(E(Invalid), E(Invalid)))
	require.False(t, errors.Is(E(Invalid), E(Conflict)))

	e0 := errors.New("e0")
	require.True(t, errors.Is(E(e0), e0))
	require.True(t, errors.Is(E(Invalid, e0), e0))
}
