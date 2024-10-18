// Package ztest runs formulaic tests ("ztests") that can be (1) run in-process
// with the compiled-ini zq code base, (2) run as a sub-process using the zq
// executable build artifact, or (3) run as a bash script running a sequence
// of arbitrary shell commands invoking any of the build artifacts.  The
// first two cases comprise the "Zed test style" and the last case
// comprises the "script test style".  Case (1) is easier to debug by
// simply running "go test" compared replicating the test using "go run".
// Script-style tests don't have this convenience.
//
// In the Zed style, ztest runs a Zed program on an input and checks
// for an expected output.
//
// A Zed-style test is defined in a YAML file.
//
//	zed: count()
//
//	input: |
//	  #0:record[i:int64]
//	  0:[1;]
//	  0:[2;]
//
//	output: |
//	  #0:record[count:uint64]
//	  0:[2;]
//
// Input format is detected automatically and can be anything recognized by
// "zq -i auto" (including optional gzip compression).  Output format defaults
// to tzng but can be set to anything accepted by "zq -f".
//
//	zed: count()
//
//	input: |
//	  #0:record[i:int64]
//	  0:[1;]
//	  0:[2;]
//
//	output-flags: -f table
//
//	output: |
//	  count
//	  2
//
// Alternatively, tests can be configured to run as shell scripts.
// In this style of test, arbitrary bash scripts can run chaining together
// any of the tools in cmd/ in addition to zq.  Scripts are executed by "bash -e
// -o pipefail", and a nonzero shell exit code causes a test failure, so any failed
// command generally results in a test failure.  Here, the yaml sets up a collection
// of input files and stdin, the script runs, and the test driver compares expected
// output files, stdout, and stderr with data in the yaml spec.  In this case,
// instead of specifying, "zed", "input", "output", you specify the yaml arrays
// "inputs" and "outputs" --- where each array element defines a file, stdin,
// stdout, or stderr --- and a "script" that specifies a multi-line yaml string
// defining the script, e.g.,
//
// inputs:
//   - name: in1.tzng
//     data: |
//     #0:record[i:int64]
//     0:[1;]
//   - name: stdin
//     data: |
//     #0:record[i:int64]
//     0:[2;]
//
// script: |
//
//	zq -o out.tzng in1.tzng -
//	zq -o count.tzng "count()" out.tzng
//
// outputs:
//   - name: out.tzng
//     data: |
//     #0:record[i:int64]
//     0:[1;]
//     0:[2;]
//   - name: count.tzng
//     data: |
//     #0:record[count:uint64]
//     0:[2;]
//
// Each input and output has a name.  For inputs, a file (source)
// or inline data (data) may be specified.
// If no data is specified, then a file of the same name as the
// name field is looked for in the same directory as the yaml file.
// The source spec is a file path relative to the directory of the
// yaml file.  For outputs, expected output is defined in the same
// fashion as the inputs though you can also specify a "regexp" string
// instead of expected data.  If an output is named "stdout" or "stderr"
// then the actual output is taken from the stdout or stderr of the
// the shell script.
//
// Ztest YAML files for a package should reside in a subdirectory named
// testdata/ztest.
//
//	pkg/
//	  pkg.go
//	  pkg_test.go
//	  testdata/
//	    ztest/
//	      test-1.yaml
//	      test-2.yaml
//	      ...
//
// Name YAML files descriptively since each ztest runs as a subtest
// named for the file that defines it.
//
// pkg_test.go should contain a Go test named TestZTest that calls Run.
//
//	func TestZTest(t *testing.T) { ztest.Run(t, "testdata/ztest") }
//
// If the ZTEST_PATH environment variable is unset or empty and the test
// is not a script test, Run runs ztests in the current process and skips
// the script tests.  Otherwise, Run runs each ztest in a separate process
// using the zq executable in the directories specified by ZTEST_PATH.
//
// Tests of either style can be skipped by setting the skip field to a non-empty
// string.  A message containing the string will be written to the test log.
package ztest

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"unicode/utf8"

	zed "github.com/brimdata/super"
	"github.com/brimdata/super/cli/inputflags"
	"github.com/brimdata/super/cli/outputflags"
	"github.com/brimdata/super/compiler"
	"github.com/brimdata/super/compiler/optimizer/demand"
	"github.com/brimdata/super/pkg/storage"
	"github.com/brimdata/super/runtime"
	"github.com/brimdata/super/runtime/vcache"
	"github.com/brimdata/super/zbuf"
	"github.com/brimdata/super/zio"
	"github.com/brimdata/super/zio/anyio"
	"github.com/brimdata/super/zio/vngio"
	"github.com/brimdata/super/zio/zsonio"
	"github.com/pmezard/go-difflib/difflib"
	"github.com/segmentio/ksuid"
	"gopkg.in/yaml.v3"
)

func ShellPath() string {
	return os.Getenv("ZTEST_PATH")
}

type Bundle struct {
	TestName string
	FileName string
	Test     *ZTest
	Error    error
}

func (b *Bundle) RunScript(shellPath, tempDir string) error {
	return b.Test.RunScript(shellPath, filepath.Dir(b.FileName), tempDir)
}

func Load(dirname string) ([]Bundle, error) {
	var bundles []Bundle
	fileinfos, err := os.ReadDir(dirname)
	if err != nil {
		return nil, err
	}
	for _, fi := range fileinfos {
		filename := fi.Name()
		const dotyaml = ".yaml"
		if !strings.HasSuffix(filename, dotyaml) {
			continue
		}
		testname := strings.TrimSuffix(filename, dotyaml)
		filename = filepath.Join(dirname, filename)
		zt, err := FromYAMLFile(filename)
		bundles = append(bundles, Bundle{testname, filename, zt, err})
	}
	return bundles, nil
}

// Run runs the ztests in the directory named dirname.  For each file f.yaml in
// the directory, Run calls FromYAMLFile to load a ztest and then runs it in
// subtest named f.  path is a command search path like the
// PATH environment variable.
func Run(t *testing.T, dirname string) {
	shellPath := ShellPath()
	bundles, err := Load(dirname)
	if err != nil {
		t.Fatal(err)
	}
	for _, bundle := range bundles {
		b := bundle
		t.Run(b.TestName, func(t *testing.T) {
			t.Parallel()
			if b.Error != nil {
				t.Fatalf("%s: %s", b.FileName, b.Error)
			}
			b.Test.Run(t, shellPath, b.FileName)
		})
	}
}

type File struct {
	// Name is the name of the file with respect to the directoy in which
	// the test script runs.  For inputs, if no data source is specified,
	// then name is also the name of a data file in the diectory containing
	// the yaml test file, which is copied to the test script directory.
	// Name can also be stdio (for inputs) or stdout or stderr (for outputs).
	Name string `yaml:"name"`
	// Data and Source represent the different ways file data can
	// be defined for this file.  Data is a string turned into the contents
	// of the file. Source is a string representing
	// the pathname of a file the repo that is read to comprise the data.
	Data   *string `yaml:"data,omitempty"`
	Source string  `yaml:"source,omitempty"`
	// Re is a regular expression describing the contents of the file,
	// which is only applicable to output files.
	Re string `yaml:"regexp,omitempty"`
}

func (f *File) check() error {
	cnt := 0
	if f.Data != nil {
		cnt++
	}
	if f.Source != "" {
		cnt++
	}
	if cnt > 1 {
		return fmt.Errorf("%s: must specify at most one of data or source", f.Name)
	}
	return nil
}

func (f *File) load(dir string) ([]byte, *regexp.Regexp, error) {
	if f.Data != nil {
		return []byte(*f.Data), nil, nil
	}
	if f.Source != "" {
		b, err := os.ReadFile(filepath.Join(dir, f.Source))
		return b, nil, err
	}
	if f.Re != "" {
		re, err := regexp.Compile(f.Re)
		return nil, re, err
	}
	b, err := os.ReadFile(filepath.Join(dir, f.Name))
	if err == nil {
		return b, nil, nil
	}
	if os.IsNotExist(err) {
		err = fmt.Errorf("%s: no data source", f.Name)
	}
	return nil, nil, err
}

// ZTest defines a ztest.
type ZTest struct {
	Skip string `yaml:"skip,omitempty"`
	Tag  string `yaml:"tag,omitempty"`

	// For Zed-style tests.
	Zed         string `yaml:"zed,omitempty"`
	Input       string `yaml:"input,omitempty"`
	InputFlags  string `yaml:"input-flags,omitempty"`
	Output      string `yaml:"output,omitempty"`
	OutputFlags string `yaml:"output-flags,omitempty"`
	ErrorRE     string `yaml:"errorRE,omitempty"`
	Vector      bool   `yaml:"vector"`
	errRegex    *regexp.Regexp

	// For script-style tests.
	Script  string   `yaml:"script,omitempty"`
	Inputs  []File   `yaml:"inputs,omitempty"`
	Outputs []File   `yaml:"outputs,omitempty"`
	Env     []string `yaml:"env,omitempty"`
}

func (z *ZTest) check() error {
	if z.Script != "" {
		if z.Outputs == nil {
			return errors.New("outputs field missing in a sh test")
		}
		for _, f := range z.Inputs {
			if err := f.check(); err != nil {
				return err
			}
			if f.Re != "" {
				return fmt.Errorf("%s: cannot use regexp in an input", f.Name)
			}
		}
		for _, f := range z.Outputs {
			if err := f.check(); err != nil {
				return err
			}
		}
	} else if z.Zed == "" {
		return errors.New("either a zed field or script field must be present")
	}
	if z.ErrorRE != "" {
		var err error
		z.errRegex, err = regexp.Compile(z.ErrorRE)
		return err
	}
	return nil
}

// FromYAMLFile loads a ZTest from the YAML file named filename.
func FromYAMLFile(filename string) (*ZTest, error) {
	buf, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	d := yaml.NewDecoder(bytes.NewReader(buf))
	d.KnownFields(true)
	var z ZTest
	if err := d.Decode(&z); err != nil {
		return nil, err
	}
	var v interface{}
	if d.Decode(&v) != io.EOF {
		return nil, errors.New("found multiple YAML documents or garbage after first document")
	}
	return &z, nil
}

func (z *ZTest) ShouldSkip(path string) string {
	switch {
	case z.Script != "" && path == "":
		return "script test on in-process run"
	case z.Skip != "":
		return z.Skip
	case z.Tag != "" && z.Tag != os.Getenv("ZTEST_TAG"):
		return fmt.Sprintf("tag %q does not match ZTEST_TAG=%q", z.Tag, os.Getenv("ZTEST_TAG"))
	}
	return ""
}

func (z *ZTest) RunScript(shellPath, testDir, tempDir string) error {
	if err := z.check(); err != nil {
		return fmt.Errorf("bad yaml format: %w", err)
	}
	return runsh(shellPath, testDir, tempDir, z)
}

func (z *ZTest) RunInternal(path string) error {
	if err := z.check(); err != nil {
		return fmt.Errorf("bad yaml format: %w", err)
	}
	outputFlags := append([]string{"-f", "zson", "-pretty=0"}, strings.Fields(z.OutputFlags)...)
	inputFlags := strings.Fields(z.InputFlags)
	if z.Vector {
		if z.InputFlags != "" {
			return errors.New("input-flags cannot be specified if vector test is enabled")
		}
		verr := z.diffInternal(runvec(z.Zed, z.Input, outputFlags))
		if verr != nil {
			verr = fmt.Errorf("=== vector ===\n%w", verr)
		}
		serr := z.diffInternal(runzq(path, z.Zed, z.Input, outputFlags, inputFlags))
		if serr != nil {
			serr = fmt.Errorf("=== sequence ===\n%w", serr)
		}
		return errors.Join(verr, serr)
	}
	return z.diffInternal(runzq(path, z.Zed, z.Input, outputFlags, inputFlags))
}

func (z *ZTest) diffInternal(out, errout string, err error) error {
	if err != nil {
		if z.errRegex != nil {
			if !z.errRegex.MatchString(errout) {
				return fmt.Errorf("error doesn't match expected error regex: %s %s", z.ErrorRE, errout)
			}
		} else {
			if out != "" {
				out = "\noutput:\n" + out
			}
			return fmt.Errorf("%w%s", err, out)
		}
	} else if z.errRegex != nil {
		return fmt.Errorf("no error when expecting error regex: %s", z.ErrorRE)
	}
	if z.Output != out {
		return diffErr("output", z.Output, out)
	}
	return nil
}

func (z *ZTest) Run(t *testing.T, path, filename string) {
	if msg := z.ShouldSkip(path); msg != "" {
		t.Skip("skipping test:", msg)
	}
	var err error
	if z.Script != "" {
		err = z.RunScript(path, filepath.Dir(filename), t.TempDir())
	} else {
		err = z.RunInternal(path)
	}
	if err != nil {
		t.Fatalf("%s: %s", filename, err)
	}
}

func diffErr(name, expected, actual string) error {
	if !utf8.ValidString(expected) {
		expected = hex.Dump([]byte(expected))
		actual = hex.Dump([]byte(actual))
	}
	diff, err := difflib.GetUnifiedDiffString(difflib.UnifiedDiff{
		A:        difflib.SplitLines(expected),
		FromFile: "expected",
		B:        difflib.SplitLines(actual),
		ToFile:   "actual",
		Context:  5,
	})
	if err != nil {
		panic("ztest: " + err.Error())
	}
	return fmt.Errorf("expected and actual %s differ:\n%s", name, diff)
}

func runsh(path, testDir, tempDir string, zt *ZTest) error {
	var stdin io.Reader
	for _, f := range zt.Inputs {
		b, _, err := f.load(testDir)
		if err != nil {
			return err
		}
		if f.Name == "stdin" {
			stdin = bytes.NewReader(b)
			continue
		}
		if err := os.WriteFile(filepath.Join(tempDir, f.Name), b, 0644); err != nil {
			return err
		}
	}
	stdout, stderr, err := RunShell(tempDir, path, zt.Script, stdin, zt.Env)
	if err != nil {
		return fmt.Errorf("script failed: %w\n=== stdout ===\n%s=== stderr ===\n%s",
			err, stdout, stderr)
	}
	for _, f := range zt.Outputs {
		var actual string
		switch f.Name {
		case "stdout":
			actual = stdout
		case "stderr":
			actual = stderr
		default:
			b, err := os.ReadFile(filepath.Join(tempDir, f.Name))
			if err != nil {
				return fmt.Errorf("%s: %w", f.Name, err)
			}
			actual = string(b)
		}
		expected, expectedRE, err := f.load(testDir)
		if err != nil {
			return err
		}
		if expected != nil && string(expected) != actual {
			return diffErr(f.Name, string(expected), actual)
		}
		if expectedRE != nil && !expectedRE.MatchString(actual) {
			return fmt.Errorf("%s: regexp %q does not match %q", f.Name, expectedRE, actual)
		}
	}
	return nil
}

// runzq runs zedProgram over input and returns the output.  input
// may be in any format recognized by "zq -i auto" and may be gzip-compressed.
// outputFlags may contain any flags accepted by cli/outputflags.Flags.  If path
// is empty, the program runs in the current process.  If path is not empty, it
// specifies a command search path used to find a zq executable to run the
// program.
func runzq(path, zedProgram, input string, outputFlags []string, inputFlags []string) (string, string, error) {
	var errbuf, outbuf bytes.Buffer
	if path != "" {
		super, err := lookupSuper(path)
		if err != nil {
			return "", "", err
		}
		flags := append(outputFlags, inputFlags...)
		args := append([]string{"query"}, flags...)
		args = append(args, []string{"-c", zedProgram, "-"}...)
		cmd := exec.Command(super, args...)
		cmd.Stdin = strings.NewReader(input)
		cmd.Stdout = &outbuf
		cmd.Stderr = &errbuf
		err = cmd.Run()
		return outbuf.String(), errbuf.String(), err
	}
	proc, sset, err := compiler.Parse(false, zedProgram)
	if err != nil {
		return "", err.Error(), err
	}
	var inflags inputflags.Flags
	var flags flag.FlagSet
	inflags.SetFlags(&flags, true)
	if err := flags.Parse(inputFlags); err != nil {
		return "", "", err
	}
	r, err := anyio.GzipReader(strings.NewReader(input))
	if err != nil {
		return "", err.Error(), err
	}
	zctx := zed.NewContext()
	zrc, err := anyio.NewReaderWithOpts(zctx, r, demand.All(), inflags.Options())
	if err != nil {
		return "", err.Error(), err
	}
	defer zrc.Close()
	var outflags outputflags.Flags
	flags = flag.FlagSet{}
	outflags.SetFlags(&flags)
	if err := flags.Parse(outputFlags); err != nil {
		return "", "", err
	}
	if err := outflags.Init(); err != nil {
		return "", "", err
	}
	zw, err := anyio.NewWriter(zio.NopCloser(&outbuf), outflags.Options())
	if err != nil {
		return "", "", err
	}
	q, err := runtime.CompileQuery(context.Background(), zctx, compiler.NewCompiler(), proc, sset, []zio.Reader{zrc})
	if err != nil {
		zw.Close()
		return "", err.Error(), err
	}
	defer q.Pull(true)
	err = zbuf.CopyPuller(zw, q)
	if err2 := zw.Close(); err == nil {
		err = err2
	}
	if err != nil {
		errbuf.WriteString(err.Error())
	}
	return outbuf.String(), errbuf.String(), err
}

func lookupSuper(path string) (string, error) {
	var super string
	var err error
	for _, dir := range filepath.SplitList(path) {
		super, err = exec.LookPath(filepath.Join(dir, "super"))
		if err == nil {
			return super, nil
		}
	}
	return "", err
}

func runvec(zedProgram string, input string, outputFlags []string) (string, string, error) {
	var errbuf, outbuf bytes.Buffer
	var flags flag.FlagSet
	var outflags outputflags.Flags
	outflags.SetFlags(&flags)
	if err := flags.Parse(outputFlags); err != nil {
		return "", "", err
	}
	zctx := zed.NewContext()
	local := storage.NewLocalEngine()
	cache := vcache.NewCache(local)
	uri, cleanup, err := writeVNGFile(input)
	if err != nil {
		return "", "", err
	}
	defer cleanup()
	object, err := cache.Fetch(context.Background(), uri, ksuid.Nil)
	if err != nil {
		return "", "", err
	}
	defer object.Close()
	rctx := runtime.NewContext(context.Background(), zctx)
	puller, err := compiler.VectorCompile(rctx, false, zedProgram, object)
	if err != nil {
		return "", err.Error(), err
	}
	zw, err := anyio.NewWriter(zio.NopCloser(&outbuf), outflags.Options())
	if err != nil {
		return "", "", err
	}
	err = zbuf.CopyPuller(zw, puller)
	if err2 := zw.Close(); err == nil {
		err = err2
	}
	if err != nil {
		errbuf.WriteString(err.Error())
	}
	return outbuf.String(), errbuf.String(), nil
}

func writeVNGFile(input string) (*storage.URI, func(), error) {
	f, err := os.CreateTemp("", "test.*.vng")
	if err != nil {
		return nil, nil, err
	}
	w := vngio.NewWriter(f)
	r := zsonio.NewReader(zed.NewContext(), strings.NewReader(input))
	if err = errors.Join(zio.Copy(w, r), w.Close()); err != nil {
		return nil, nil, err
	}
	u, err := storage.ParseURI(f.Name())
	return u, func() { os.Remove(f.Name()) }, err
}
