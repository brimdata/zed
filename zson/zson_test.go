package zson_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/brimsec/zq/ast"
	"github.com/brimsec/zq/pkg/fs"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/brimsec/zq/zson"
	"github.com/brimsec/zq/ztest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type Input struct {
	Path  string
	Which string
	Src   string
}

func isTzng(s string) bool {
	if len(s) < 2 {
		return false
	}
	return s[:2] == "#0"
}

const header = `
script: |
  zq -f zson in.tzng > baseline.zson
  zq -i zson -f zson baseline.zson > boomerang.zson
  diff baseline.zson boomerang.zson
  echo EOF

outputs:
  - name: stdout
    data: |
      EOF

inputs:
  - name: in.tzng
    data: |
`

func TestBuildZsonTests(t *testing.T) {
	t.Skip("comment out this skip directive to rebuild zson/ztests")
	m := make(map[string]int)
	inputs, _ := searchForTnzgs()
	dirpath := filepath.Join("ztests", "auto")
	os.Mkdir(dirpath, 0755)
	for _, input := range inputs {
		name := filepath.Base(input.Path)
		cnt := m[name]
		m[name] += 1
		name = countedName(name, cnt)
		path := filepath.Join(dirpath, name)
		if err := writeTestFile(path, input.Src); err != nil {
			require.NoError(t, err)
			return
		}
	}
}

func countedName(name string, cnt int) string {
	if cnt == 0 {
		return name
	}
	ext := filepath.Ext(name)
	base := filepath.Base(name)
	base = strings.TrimSuffix(base, ext)
	return fmt.Sprintf("%s-%d%s", base, cnt+1, ext)
}

func writeTestFile(filename, src string) error {
	lines := strings.Split(src, "\n")
	var b strings.Builder
	b.WriteString(header[1:])
	for _, line := range lines {
		b.WriteString("      ")
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return ioutil.WriteFile(filename, []byte(b.String()), 0644)
}

func searchForTnzgs() ([]Input, error) {
	var inputs []Input
	pattern := fmt.Sprintf(`.*ztests\%c.*\.yaml$`, filepath.Separator)
	re := regexp.MustCompile(pattern)
	err := filepath.Walk("..", func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() && strings.HasSuffix(path, ".yaml") && re.MatchString(path) {
			zt, err := ztest.FromYAMLFile(path)
			if err != nil {
				return err
			}
			for _, src := range zt.Input {
				if !isTzng(src) {
					continue
				}
				input := Input{
					Path: path,
					Src:  src,
				}
				inputs = append(inputs, input)
				return nil
			}
			for k, src := range zt.Inputs {
				if src.Data == nil || !isTzng(*src.Data) {
					continue
				}
				input := Input{
					Path:  path,
					Which: fmt.Sprintf("input[%d]", k),
					Src:   *src.Data,
				}
				inputs = append(inputs, input)
				return nil
			}
		}
		return err
	})
	return inputs, err
}

func parse(path string) (ast.Value, error) {
	file, err := fs.Open(path)
	if err != nil {
		return nil, err
	}
	parser, err := zson.NewParser(file)
	if err != nil {
		return nil, err
	}
	return parser.ParseValue()
}

func TestZsonParser(t *testing.T) {
	val, err := parse("test.zson")
	require.NoError(t, err)
	s, err := json.MarshalIndent(val, "", "    ")
	require.NoError(t, err)
	assert.NotEqual(t, s, "")
	//fmt.Println(string(s))
}

func analyze(zctx *resolver.Context, path string) (zson.Value, error) {
	val, err := parse(path)
	if err != nil {
		return nil, err
	}
	analyzer := zson.NewAnalyzer()
	return analyzer.ConvertValue(zctx, val)
}

func TestZsonAnalyzer(t *testing.T) {
	zctx := resolver.NewContext()
	val, err := analyze(zctx, "test.zson")
	require.NoError(t, err)
	assert.NotNil(t, val)
	//pretty.Println(val)
}

func TestZsonBuilder(t *testing.T) {
	zctx := resolver.NewContext()
	val, err := analyze(zctx, "test.zson")
	require.NoError(t, err)
	b := zson.NewBuilder()
	zv, err := b.Build(val)
	require.NoError(t, err)
	rec := zng.NewRecord(zv.Type.(*zng.TypeRecord), zv.Bytes)
	zv, err = rec.Access("a")
	assert.Equal(t, "array[string]: [(31)(32)(33)]", zv.String())
}
