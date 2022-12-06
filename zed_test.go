package zed_test

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zio/anyio"
	"github.com/brimdata/zed/zio/arrowio"
	"github.com/brimdata/zed/zio/parquetio"
	"github.com/brimdata/zed/zson"
	"github.com/brimdata/zed/ztest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestZed(t *testing.T) {
	t.Parallel()
	dirs, err := findZTests()
	require.NoError(t, err)
	for d := range dirs {
		d := d
		t.Run(d, func(t *testing.T) {
			t.Parallel()
			ztest.Run(t, d)
		})
	}
	t.Run("ArrowStreamBoomerang", func(t *testing.T) {
		runArrowStreamBoomerangs(t, dirs)
	})
	t.Run("ParquetBoomerang", func(t *testing.T) {
		runParquetBoomerangs(t, dirs)
	})
	t.Run("ZSONBoomerang", func(t *testing.T) {
		runZSONBoomerangs(t, dirs)
	})
}

func findZTests() (map[string]struct{}, error) {
	dirs := map[string]struct{}{}
	pattern := fmt.Sprintf(`.*ztests\%c.*\.yaml$`, filepath.Separator)
	re := regexp.MustCompile(pattern)
	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() && strings.HasSuffix(path, ".yaml") && re.MatchString(path) {
			dirs[filepath.Dir(path)] = struct{}{}
		}
		return err
	})
	return dirs, err
}

func runZSONBoomerangs(t *testing.T, dirs map[string]struct{}) {
	if testing.Short() {
		return
	}
	const script = `
exec 2>&1
zq -f zson - > baseline.zson &&
zq -i zson -f zson baseline.zson > boomerang.zson &&
diff baseline.zson boomerang.zson
`
	bundles, err := findInputs(t, dirs, script, isValidForZSON)
	if err != nil {
		t.Fatal(err)
	}
	shellPath := ztest.ShellPath()
	for _, b := range bundles {
		b := b
		t.Run(b.TestName, func(t *testing.T) {
			t.Parallel()
			err := b.RunScript(shellPath, t.TempDir())
			if err != nil {
				err = &BoomerangError{
					*b.Test.Inputs[0].Data,
					b.FileName,
					err,
				}
			}
			require.NoError(t, err)
		})
	}
}

type BoomerangError struct {
	ZSON     string
	FileName string
	Err      error
}

func (b *BoomerangError) Error() string {
	return fmt.Sprintf("%s\n=== with this input ===\n\n%s\n\n=== from file ===\n\n%s\n\n", b.Err, b.ZSON, b.FileName)
}

func boomerang(script, input string) *ztest.ZTest {
	var empty string
	return &ztest.ZTest{
		Script: script,
		Inputs: []ztest.File{
			{
				Name: "stdin",
				Data: &input,
			},
		},
		Outputs: []ztest.File{
			{
				Name: "stdout",
				Data: &empty,
			},
			{
				Name: "stderr",
				Data: &empty,
			},
		},
	}
}

func expectFailure(b ztest.Bundle) bool {
	if b.Test.ErrorRE != "" {
		return true
	}
	for _, f := range b.Test.Outputs {
		if f.Name == "stderr" {
			return true
		}
	}
	return false
}

func isValidForZSON(input string) bool {
	zrc, err := anyio.NewReader(zed.NewContext(), strings.NewReader(input))
	if err != nil {
		return false
	}
	defer zrc.Close()
	for {
		rec, err := zrc.Read()
		if err != nil {
			return false
		}
		if rec == nil {
			return true
		}
	}
}

func runArrowStreamBoomerangs(t *testing.T, dirs map[string]struct{}) {
	if testing.Short() {
		return
	}
	const script = `
exec 2>&1
zq -f arrows -o baseline.arrows fuse - &&
zq -i arrows -f arrows -o boomerang.arrows baseline.arrows &&
diff baseline.arrows boomerang.arrows
`
	bundles, err := findInputs(t, dirs, script, isValidForParquet)
	if err != nil {
		t.Fatal(err)
	}
	shellPath := ztest.ShellPath()
	for _, b := range bundles {
		b := b
		t.Run(b.TestName, func(t *testing.T) {
			t.Parallel()
			err := b.RunScript(shellPath, t.TempDir())
			if err != nil {
				if s := err.Error(); strings.Contains(s, arrowio.ErrMultipleTypes.Error()) ||
					strings.Contains(s, arrowio.ErrNotRecord.Error()) ||
					strings.Contains(s, arrowio.ErrUnsupportedType.Error()) ||
					strings.Contains(s, "cannot yet use maps in shaping functions") {
					t.Skip("skipping because the Arrow writer cannot handle an input type")
				}
				err = &BoomerangError{
					*b.Test.Inputs[0].Data,
					b.FileName,
					err,
				}
			}
			require.NoError(t, err)
		})
	}
}

func runParquetBoomerangs(t *testing.T, dirs map[string]struct{}) {
	if testing.Short() {
		return
	}
	const script = `
exec 2>&1
zq -f parquet -o baseline.parquet fuse - &&
zq -i parquet -f parquet -o boomerang.parquet baseline.parquet &&
diff baseline.parquet boomerang.parquet
`
	bundles, err := findInputs(t, dirs, script, isValidForParquet)
	if err != nil {
		t.Fatal(err)
	}
	shellPath := ztest.ShellPath()
	for _, b := range bundles {
		b := b
		t.Run(b.TestName, func(t *testing.T) {
			t.Parallel()
			err := b.RunScript(shellPath, t.TempDir())
			if err != nil {
				if s := err.Error(); strings.Contains(s, parquetio.ErrEmptyRecordType.Error()) ||
					strings.Contains(s, parquetio.ErrNullType.Error()) ||
					strings.Contains(s, parquetio.ErrUnionType.Error()) ||
					strings.Contains(s, "column has no name") ||
					strings.Contains(s, "cannot yet use maps in shaping functions") {
					t.Skip("skipping because the Parquet writer cannot handle an input type")
				}
				err = &BoomerangError{
					*b.Test.Inputs[0].Data,
					b.FileName,
					err,
				}
			}
			require.NoError(t, err)
		})
	}
}

func isValidForParquet(input string) bool {
	zrc, err := anyio.NewReader(zed.NewContext(), strings.NewReader(input))
	if err != nil {
		return false
	}
	defer zrc.Close()
	var found bool
	for {
		rec, err := zrc.Read()
		if err != nil {
			return false
		}
		if rec == nil {
			return found
		}
		if !zed.IsRecordType(rec.Type) {
			// zio/parquetio requires records at top level.
			return false
		}
		found = true
	}
}

func findInputs(t *testing.T, dirs map[string]struct{}, script string, isValidInput func(string) bool) ([]ztest.Bundle, error) {
	var out []ztest.Bundle
	for path := range dirs {
		bundles, err := ztest.Load(path)
		if err != nil {
			t.Log(err)
			continue
		}
		// Transform the bundles into boomerang tests by taking each
		// source and creating a new ztest.Bundle.
		for _, bundle := range bundles {
			if bundle.Error != nil || expectFailure(bundle) {
				continue
			}
			// Normalize the diffrent kinds of test inputs into
			// a single pattern.
			if input := bundle.Test.Input; isValidInput(input) {
				out = append(out, ztest.Bundle{
					TestName: bundle.TestName,
					FileName: bundle.FileName,
					Test:     boomerang(script, input),
				})
			}
			for _, input := range bundle.Test.Inputs {
				if input.Data != nil && isValidInput(*input.Data) {
					out = append(out, ztest.Bundle{
						TestName: bundle.TestName,
						FileName: bundle.FileName,
						Test:     boomerang(script, *input.Data),
					})
				}
			}
		}
	}
	return out, nil
}

func TestTranslateNameConflictUnion(t *testing.T) {
	// This test confirms that a union with complicated type renaming is properly
	// decoded.  There was a bug where child typedefs would override the
	// top level typedef in TranslateTypeValue so foo in the value below had
	// two of the same union type instead of the two it should have had.
	zctx := zed.NewContext()
	val := zson.MustParseValue(zctx, `[{x:{y:63}}(=foo),{x:{abcdef:{x:{y:127}}(foo)}}(=foo)]`)
	foreign := zed.NewContext()
	twin, err := foreign.TranslateType(val.Type)
	require.NoError(t, err)
	union := twin.(*zed.TypeArray).Type.(*zed.TypeUnion)
	assert.Equal(t, `foo={x:{abcdef:foo={x:{y:int64}}}}`, zson.String(union.Types[0]))
	assert.Equal(t, `foo={x:{y:int64}}`, zson.String(union.Types[1]))
}
