package zed_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/brimdata/super"
	"github.com/brimdata/super/compiler"
	"github.com/brimdata/super/compiler/optimizer/demand"
	"github.com/brimdata/super/runtime"
	"github.com/brimdata/super/zio"
	"github.com/brimdata/super/zio/anyio"
	"github.com/brimdata/super/zio/arrowio"
	"github.com/brimdata/super/zio/zngio"
	"github.com/brimdata/super/ztest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestZed(t *testing.T) {
	t.Parallel()

	dirs, err := findZTests()
	require.NoError(t, err)

	t.Run("boomerang", func(t *testing.T) {
		t.Parallel()
		data, err := loadZTestInputsAndOutputs(dirs)
		require.NoError(t, err)
		runAllBoomerangs(t, "arrows", data)
		runAllBoomerangs(t, "parquet", data)
		runAllBoomerangs(t, "vng", data)
		runAllBoomerangs(t, "zjson", data)
		runAllBoomerangs(t, "zson", data)
	})

	for d := range dirs {
		d := d
		t.Run(filepath.ToSlash(d), func(t *testing.T) {
			t.Parallel()
			ztest.Run(t, d)
		})
	}
}

func findZTests() (map[string]struct{}, error) {
	dirs := map[string]struct{}{}
	pattern := fmt.Sprintf(`.*ztests\%c.*\.yaml$`, filepath.Separator)
	re := regexp.MustCompile(pattern)
	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".yaml") && re.MatchString(path) {
			dirs[filepath.Dir(path)] = struct{}{}
		}
		return nil
	})
	return dirs, err
}

func loadZTestInputsAndOutputs(ztestDirs map[string]struct{}) (map[string]string, error) {
	out := map[string]string{}
	for dir := range ztestDirs {
		bundles, err := ztest.Load(dir)
		if err != nil {
			return nil, err
		}
		for _, b := range bundles {
			if i := b.Test.Input; isValid(i) {
				out[b.FileName+"/input"] = i
			}
			if o := b.Test.Output; isValid(o) {
				out[b.FileName+"/output"] = o
			}
			for _, i := range b.Test.Inputs {
				if i.Data != nil && isValid(*i.Data) {
					out[b.FileName+"/inputs/"+i.Name] = *i.Data
				}
			}
			for _, o := range b.Test.Outputs {
				if o.Data != nil && isValid(*o.Data) {
					out[b.FileName+"/outputs/"+o.Name] = *o.Data
				}
			}
		}
	}
	return out, nil
}

// isValid returns true if and only if s can be read fully without error by
// anyio and contains at least one value.
func isValid(s string) bool {
	zrc, err := anyio.NewReader(zed.NewContext(), strings.NewReader(s), demand.All())
	if err != nil {
		return false
	}
	defer zrc.Close()
	var foundValue bool
	for {
		val, err := zrc.Read()
		if err != nil {
			return false
		}
		if val == nil {
			return foundValue
		}
		foundValue = true
	}
}

func runAllBoomerangs(t *testing.T, format string, data map[string]string) {
	t.Run(format, func(t *testing.T) {
		t.Parallel()
		for name, data := range data {
			data := data
			t.Run(name, func(t *testing.T) {
				t.Parallel()
				runOneBoomerang(t, format, data)
			})
		}
	})
}

func runOneBoomerang(t *testing.T, format, data string) {
	// Create an auto-detecting reader for data.
	zctx := zed.NewContext()
	dataReadCloser, err := anyio.NewReader(zctx, strings.NewReader(data), demand.All())
	require.NoError(t, err)
	defer dataReadCloser.Close()

	dataReader := zio.Reader(dataReadCloser)
	if format == "parquet" {
		// Fuse for formats that require uniform values.
		proc, _, err := compiler.NewCompiler().Parse(false, "fuse")
		require.NoError(t, err)
		rctx := runtime.NewContext(context.Background(), zctx)
		q, err := compiler.NewCompiler().NewQuery(rctx, proc, []zio.Reader{dataReadCloser})
		require.NoError(t, err)
		defer q.Pull(true)
		dataReader = runtime.AsReader(q)
	}

	// Copy from dataReader to baseline as format.
	var baseline bytes.Buffer
	writerOpts := anyio.WriterOpts{Format: format}
	baselineWriter, err := anyio.NewWriter(zio.NopCloser(&baseline), writerOpts)
	if err == nil {
		err = zio.Copy(baselineWriter, dataReader)
		require.NoError(t, baselineWriter.Close())
	}
	if err != nil {
		if errors.Is(err, arrowio.ErrMultipleTypes) ||
			errors.Is(err, arrowio.ErrNotRecord) ||
			errors.Is(err, arrowio.ErrUnsupportedType) {
			t.Skipf("skipping due to expected error: %s", err)
		}
		t.Fatalf("unexpected error writing %s baseline: %s", format, err)
	}

	// Create a reader for baseline.
	baselineReader, err := anyio.NewReaderWithOpts(zed.NewContext(), bytes.NewReader(baseline.Bytes()), demand.All(), anyio.ReaderOpts{
		Format: format,
		ZNG: zngio.ReaderOpts{
			Validate: true,
		},
	})
	require.NoError(t, err)
	defer baselineReader.Close()

	// Copy from baselineReader to boomerang as format.
	var boomerang bytes.Buffer
	boomerangWriter, err := anyio.NewWriter(zio.NopCloser(&boomerang), writerOpts)
	require.NoError(t, err)
	assert.NoError(t, zio.Copy(boomerangWriter, baselineReader))
	require.NoError(t, boomerangWriter.Close())

	require.Equal(t, baseline.String(), boomerang.String(), "baseline and boomerang differ")
}
