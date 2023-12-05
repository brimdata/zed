// Adapted from https://github.com/logrusorgru/grokky/blob/f28bfe018565ac1e90d93502eae1170006dd1f48/host_test.go

package grok

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	h := New()
	require.Len(t, h, 0)
	require.NotNil(t, h)
}

func TestHost_Add(t *testing.T) {
	h := New()
	require.ErrorIs(t, h.Add("", "expr"), ErrEmptyName)
	require.Len(t, h, 0)
	require.ErrorIs(t, h.Add("name", ""), ErrEmptyExpression)
	require.Len(t, h, 0)
	require.NoError(t, h.Add("DIGIT", `\d`))
	require.Len(t, h, 1)
	require.ErrorIs(t, h.Add("DIGIT", `[+-](0x)?\d`), ErrAlreadyExist)
	require.Len(t, h, 1)
	require.Error(t, h.Add("BAD", `(?![0-5])`))
	require.Len(t, h, 1)
	require.NoError(t, h.Add("TWODIG", `%{DIGIT}-%{DIGIT}`))
	require.Len(t, h, 2)
	require.Error(t, h.Add("THREE", `%{NOT}-%{EXIST}`))
	require.Len(t, h, 2)
	require.NoError(t, h.Add("FOUR", `%{DIGIT:one}-%{DIGIT:two}`))
	require.Len(t, h, 3)
	require.Error(t, h.Add("FIVE", `(?!\d)%{DIGIT}(?!\d)`))
	require.Len(t, h, 3)
	require.NoError(t, h.Add("SIX", `%{FOUR:four}-%{DIGIT:six}`))
	require.Len(t, h, 4)
}

func TestHost_Compile(t *testing.T) {
	h := New()
	_, err := h.Compile("")
	require.ErrorIs(t, err, ErrEmptyExpression)
	require.Len(t, h, 0)
	p, err := h.Compile(`\d+`)
	require.NoError(t, err)
	require.NotNil(t, p)
	require.Len(t, h, 0)
}

func TestHost_Get(t *testing.T) {
	h := New()
	require.NoError(t, h.Add("DIG", `\d`))
	p, err := h.Get("DIG")
	require.NoError(t, err)
	require.NotNil(t, p)
	p, err = h.Get("SEVEN")
	require.ErrorIs(t, err, ErrNotExist)
	require.Nil(t, p)
}

func TestHost_AddFromReader(t *testing.T) {
	s := `#
# for testing
#
ONE \d
TWO %{ONE:two}
THREE %{ONE:one}-%{TWO}-%{ONE:three}

#
# enough
#`
	h := New()
	require.NoError(t, h.AddFromReader(strings.NewReader(s)))
	require.Len(t, h, 3)
	_, err := h.Get("ONE")
	require.NoError(t, err)
	_, err = h.Get("TWO")
	require.NoError(t, err)
	_, err = h.Get("THREE")
	require.NoError(t, err)
}

func TestHost_AddFromReader_malformedPatterns(t *testing.T) {
	s := `
ONE \d
TWO %{THREE:two}`
	require.Error(t, New().AddFromReader(strings.NewReader(s)))
}

func TestHost_inject(t *testing.T) {
	h := New()
	h["TWO"] = `(?!\d)`
	require.Error(t, h.Add("ONE", `%{TWO:one}`))
}

func TestHost_addFromLine(t *testing.T) {
	h := New()
	require.Error(t, h.addFromLine("ONE (?!\\d)"))
}
