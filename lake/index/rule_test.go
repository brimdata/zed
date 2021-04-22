package index

import (
	"testing"

	"github.com/brimdata/zed/field"
	"github.com/brimdata/zed/zng"
	"github.com/brimdata/zed/zson"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func boomerang(t *testing.T, r1 Rule) (r2 Rule) {
	t.Helper()
	v, err := zson.MarshalZNG(r1)
	require.NoError(t, err)
	require.NoError(t, zson.UnmarshalZNG(v, &r2))
	return r2
}

func TestRuleTypeMarshal(t *testing.T) {
	r1 := NewTypeRule(zng.TypeIP)
	r2 := boomerang(t, r1)
	assert.Equal(t, r1, r2)
}

func TestRuleZqlMarshal(t *testing.T) {
	keys := []field.Static{field.Dotted("id.orig_h")}
	r1, err := NewZedRule("count() by id.orig_h", "id.orig_h.count", keys)
	require.NoError(t, err)
	r2 := boomerang(t, r1)
	assert.Equal(t, r1, r2)
}
