package zeekio_test

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/mccanne/zq/zio"
	"github.com/mccanne/zq/zio/zeekio"
	"github.com/mccanne/zq/zio/zngio"
	"github.com/mccanne/zq/zng/resolver"
	"github.com/stretchr/testify/require"
)

func TestWriter(t *testing.T) {
	t.Run("replaces-array", func(t *testing.T) {
		zng := `#1:record[array:array[int]]
1:[[1;2;3;]]`
		zeek := zeekfile(
			[]string{"array"},
			[]string{"vector[int]"},
			[]string{"1,2,3"},
		)
		runcase(t, zng, zeek)
	})
	t.Run("replaces-bstring", func(t *testing.T) {
		zng := `#1:record[bstring:bstring]
1:[test;]`
		zeek := zeekfile(
			[]string{"bstring"},
			[]string{"string"},
			[]string{"test"},
		)
		runcase(t, zng, zeek)
	})
	t.Run("replaces-type-in-containers", func(t *testing.T) {
		zng := `#1:record[array:array[bstring],set:set[bstring],id:record[bstring:bstring]]
1:[[test1;test2;test3;][test1;test2;][test4;]]`
		zeek := zeekfile(
			[]string{"array", "set", "id.bstring"},
			[]string{"vector[string]", "set[string]", "string"},
			[]string{"test1,test2,test3", "test1,test2", "test4"},
		)
		runcase(t, zng, zeek)
	})
}

func runcase(t *testing.T, zng, expected string) {
	out := bytes.NewBuffer(nil)
	r := zngio.NewReader(strings.NewReader(zng), resolver.NewContext())
	rec, err := r.Read()
	require.NoError(t, err)
	w := zeekio.NewWriter(out, zio.Flags{})
	require.NoError(t, w.Write(rec))
	require.Equal(t, expected, out.String())
}

func zeekfile(fields, types []string, rows ...[]string) string {
	z := `#separator \x09
#set_separator	,
#empty_field	(empty)
#unset_field	-
`
	z += fmt.Sprintf("#fields\t%s\n", strings.Join(fields, "\t"))
	z += fmt.Sprintf("#types\t%s\n", strings.Join(types, "\t"))
	for _, r := range rows {
		z += fmt.Sprintf("%s\n", strings.Join(r, "\t"))
	}
	return z

}
