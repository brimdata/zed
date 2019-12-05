package resolver

import (
	"testing"

	"github.com/mccanne/zq/pkg/zeek"
	"github.com/stretchr/testify/require"
)

func TestTableAddColumns(t *testing.T) {
	tab := NewTable()
	d := tab.GetByColumns([]zeek.Column{{"s1", zeek.TypeString}})
	cols := []zeek.Column{{"ts", zeek.TypeTime}, {"s2", zeek.TypeString}}
	var err error
	_, err = tab.Extend(d, cols)
	require.NoError(t, err)
}
