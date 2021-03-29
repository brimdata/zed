package index

import (
	"testing"

	"github.com/brimdata/zq/field"
	"github.com/brimdata/zq/zng"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func boomerang(t *testing.T, r1 Rule) Rule {
	t.Helper()
	b, err := r1.Marshal()
	require.NoError(t, err)
	r2, err := UnmarshalRule(b)
	require.NoError(t, err)
	return r2
}

func TestRuleTypeMarshal(t *testing.T) {
	r1 := NewTypeRule(zng.TypeIP)
	r2 := boomerang(t, r1)
	assert.Equal(t, r1, r2)
}

func TestRuleZqlMarshal(t *testing.T) {
	keys := []field.Static{field.Dotted("id.orig_h")}
	r1, err := NewZqlRule("count() by id.orig_h", "id.orig_h.count", keys)
	require.NoError(t, err)
	r2 := boomerang(t, r1)
	assert.Equal(t, r1, r2)
}
