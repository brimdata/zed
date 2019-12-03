package tests

import (
	"fmt"
	"strings"
	"testing"

	"github.com/mccanne/zq/driver"
	"github.com/mccanne/zq/emitter"
	"github.com/mccanne/zq/pkg/zsio/detector"
	"github.com/mccanne/zq/pkg/zson"
	"github.com/mccanne/zq/pkg/zson/resolver"
	"github.com/mccanne/zq/scanner"
	"github.com/mccanne/zq/tests/suite"
	"github.com/mccanne/zq/tests/test"
	"github.com/mccanne/zq/zql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func cleanup(s string) string {
	s = strings.TrimSpace(s)
	return s + "\n"
}

func newStringReader(input string, r *resolver.Table) (zson.Reader, error) {
	return detector.NewReader(strings.NewReader(cleanup(input)), r)
}

func run(t test.Detail) (string, error) {
	program, err := zql.ParseProc(t.Query)
	if err != nil {
		return "", fmt.Errorf("parse error: %s (%s)", err, t.Query)
	}
	reader, err := newStringReader(t.Input, resolver.NewTable())
	if err != nil {
		return "", err
	}
	mux, err := driver.Compile(program, scanner.NewScanner(reader, nil))
	if err != nil {
		return "", err
	}
	// XXX text format to supported as its opions passed in as nil
	output, err := emitter.NewBytes(t.Format, nil)
	if err != nil {
		return "", err
	}
	d := driver.New(output)
	if err := d.Run(mux); err != nil {
		return "", err
	}
	return string(output.Bytes()), nil
}

func TestQueries(t *testing.T) {
	t.Parallel()
	for _, tst := range suite.Tests() {
		t.Run(tst.Name, func(t *testing.T) {
			results, err := run(tst)
			require.NoError(t, err)
			assert.Exactly(t, cleanup(tst.Expected), results, "Wrong query results")
		})
	}
}
