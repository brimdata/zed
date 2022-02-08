package zed_test

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"inet.af/netaddr"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
	"github.com/brimdata/zed/zio/anyio"
	"github.com/brimdata/zed/zio/parquetio"
	"github.com/brimdata/zed/zio/zsonio"
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
	t.Run("ParquetBoomerang", func(t *testing.T) {
		runParquetBoomerangs(t, dirs)
	})
	t.Run("ZsonBoomerang", func(t *testing.T) {
		runZsonBoomerangs(t, dirs)
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

func runZsonBoomerangs(t *testing.T, dirs map[string]struct{}) {
	if testing.Short() {
		return
	}
	const script = `
exec 2>&1
zq -f zson - > baseline.zson &&
zq -i zson -f zson baseline.zson > boomerang.zson &&
diff baseline.zson boomerang.zson
`
	bundles, err := findInputs(t, dirs, script, isValidForZson)
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
	Zson     string
	FileName string
	Err      error
}

func (b *BoomerangError) Error() string {
	return fmt.Sprintf("%s\n=== with this input ===\n\n%s\n\n=== from file ===\n\n%s\n\n", b.Err, b.Zson, b.FileName)
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

func isValidForZson(input string) bool {
	r, err := anyio.NewReader(strings.NewReader(input), zed.NewContext())
	if err != nil {
		return false
	}
	for {
		rec, err := r.Read()
		if err != nil {
			return false
		}
		if rec == nil {
			return true
		}
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
					strings.Contains(s, "column has no name") {
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
	r, err := anyio.NewReader(strings.NewReader(input), zed.NewContext())
	if err != nil {
		return false
	}
	var found bool
	for {
		rec, err := r.Read()
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

func TestRecordAccessNamed(t *testing.T) {
	const input = `{foo:"hello" (=zfile),bar:true (=zbool)} (=0)`
	reader := zsonio.NewReader(strings.NewReader(input), zed.NewContext())
	rec, err := reader.Read()
	require.NoError(t, err)
	s := rec.Deref("foo").AsString()
	assert.Equal(t, s, "hello")
	b := rec.Deref("bar").AsBool()
	assert.Equal(t, b, true)
}

func TestNonRecordDeref(t *testing.T) {
	const input = `
1
192.168.1.1
null
[1,2,3]
|[1,2,3]|`
	reader := zsonio.NewReader(strings.NewReader(input), zed.NewContext())
	for {
		val, err := reader.Read()
		if val == nil {
			break
		}
		require.NoError(t, err)
		v := val.Deref("foo")
		require.Nil(t, v)
	}
}

func TestNormalizeSet(t *testing.T) {
	t.Run("duplicate-element", func(t *testing.T) {
		b := zcode.NewBuilder()
		b.BeginContainer()
		b.Append([]byte("dup"))
		b.Append([]byte("dup"))
		b.TransformContainer(zed.NormalizeSet)
		b.EndContainer()
		set := zcode.Append(nil, []byte("dup"))
		expected := zcode.Append(nil, set)
		require.Exactly(t, expected, b.Bytes())
	})
	t.Run("unsorted-elements", func(t *testing.T) {
		b := zcode.NewBuilder()
		b.BeginContainer()
		b.Append([]byte("z"))
		b.Append([]byte("a"))
		b.TransformContainer(zed.NormalizeSet)
		b.EndContainer()
		set := zcode.Append(nil, []byte("a"))
		set = zcode.Append(set, []byte("z"))
		expected := zcode.Append(nil, set)
		require.Exactly(t, expected, b.Bytes())
	})
	t.Run("unsorted-and-duplicate-elements", func(t *testing.T) {
		b := zcode.NewBuilder()
		big := bytes.Repeat([]byte("x"), 256)
		small := []byte("small")
		b.Append(big)
		b.BeginContainer()
		// Append duplicate elements in reverse of set-normal order.
		for i := 0; i < 3; i++ {
			b.Append(big)
			b.Append(big)
			b.Append(small)
			b.Append(small)
			b.Append(nil)
			b.Append(nil)
		}
		b.TransformContainer(zed.NormalizeSet)
		b.EndContainer()
		set := zcode.Append(nil, nil)
		set = zcode.Append(set, small)
		set = zcode.Append(set, big)
		expected := zcode.Append(nil, big)
		expected = zcode.Append(expected, set)
		require.Exactly(t, expected, b.Bytes())
	})
}

func TestDuplicates(t *testing.T) {
	ctx := zed.NewContext()
	setType := ctx.LookupTypeSet(zed.TypeInt32)
	typ1, err := ctx.LookupTypeRecord([]zed.Column{
		zed.NewColumn("a", zed.TypeString),
		zed.NewColumn("b", setType),
	})
	require.NoError(t, err)
	typ2, err := zson.ParseType(ctx, "{a:string,b:|[int32]|}")
	require.NoError(t, err)
	assert.EqualValues(t, typ1.ID(), typ2.ID())
	assert.EqualValues(t, setType.ID(), typ2.(*zed.TypeRecord).Columns[1].Type.ID())
	typ3, err := ctx.LookupByValue(zed.EncodeTypeValue(setType))
	require.NoError(t, err)
	assert.Equal(t, setType.ID(), typ3.ID())
}

func TestTranslateNamed(t *testing.T) {
	c1 := zed.NewContext()
	c2 := zed.NewContext()
	set1, err := zson.ParseType(c1, "|[int64]|")
	require.NoError(t, err)
	set2, err := zson.ParseType(c2, "|[int64]|")
	require.NoError(t, err)
	named1, err := c1.LookupTypeNamed("foo", set1)
	require.NoError(t, err)
	named2, err := c2.LookupTypeNamed("foo", set2)
	require.NoError(t, err)
	named3, err := c2.TranslateType(named1)
	require.NoError(t, err)
	assert.Equal(t, named2, named3)
}

func TestCopyMutateColumns(t *testing.T) {
	c := zed.NewContext()
	cols := []zed.Column{{"foo", zed.TypeString}, {"bar", zed.TypeInt64}}
	typ, err := c.LookupTypeRecord(cols)
	require.NoError(t, err)
	cols[0].Type = nil
	require.NotNil(t, typ.Columns[0].Type)
}

func TestBuilder(t *testing.T) {
	const input = `
{key:1.2.3.4}
{a:1,b:2,c:3}
{a:7,r:{x:3}}
{a:7,r:null({x:int64})}
`
	r := zsonio.NewReader(strings.NewReader(input), zed.NewContext())
	r0, err := r.Read()
	require.NoError(t, err)
	r1, err := r.Read()
	require.NoError(t, err)
	r2, err := r.Read()
	require.NoError(t, err)
	r3, err := r.Read()
	require.NoError(t, err)

	zctx := zed.NewContext()

	t0, err := zctx.LookupTypeRecord([]zed.Column{
		{"key", zed.TypeIP},
	})
	assert.NoError(t, err)
	b0 := zed.NewBuilder(t0)
	rec := b0.Build(zed.EncodeIP(netaddr.MustParseIP("1.2.3.4")))
	assert.Equal(t, r0.Bytes, rec.Bytes)

	t1, err := zctx.LookupTypeRecord([]zed.Column{
		{"a", zed.TypeInt64},
		{"b", zed.TypeInt64},
		{"c", zed.TypeInt64},
	})
	assert.NoError(t, err)
	b1 := zed.NewBuilder(t1)
	rec = b1.Build(zed.EncodeInt(1), zed.EncodeInt(2), zed.EncodeInt(3))
	assert.Equal(t, r1.Bytes, rec.Bytes)

	subrec, err := zctx.LookupTypeRecord([]zed.Column{{"x", zed.TypeInt64}})
	assert.NoError(t, err)
	t2, err := zctx.LookupTypeRecord([]zed.Column{
		{"a", zed.TypeInt64},
		{"r", subrec},
	})
	assert.NoError(t, err)
	b2 := zed.NewBuilder(t2)
	// XXX this is where this package needs work
	// the second column here is a container here and this is where it would
	// be nice for the builder to know this structure and wrap appropriately,
	// but for now we do the work outside of the builder, which is perfectly
	// fine if you are extracting a container value from an existing place...
	// you just grab the whole thing.  But if you just have the leaf vals
	// of the record and want to build it up, it would be nice to have some
	// easy way to do it all...
	var rb zcode.Builder
	rb.Append(zed.EncodeInt(3))
	rec = b2.Build(zed.EncodeInt(7), rb.Bytes())
	assert.Equal(t, r2.Bytes, rec.Bytes)

	//rec, err = b2.Parse("7", "3")
	//assert.NoError(t, err)
	//assert.Equal(t, r2.Bytes, rec.Bytes)

	//rec, err = b2.Parse("7")
	//assert.Equal(t, err, zed.ErrIncomplete)
	//assert.Equal(t, r3.Bytes, rec.Bytes)
	_ = r3
}
