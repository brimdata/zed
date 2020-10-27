package inputflags

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"

	"github.com/brimsec/zq/pkg/iosrc"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zio"
	"github.com/brimsec/zq/zio/azngio"
	"github.com/brimsec/zq/zio/detector"
	"github.com/brimsec/zq/zio/ndjsonio"
	"github.com/brimsec/zq/zng/resolver"
)

type Flags struct {
	zio.ReaderOpts
	// The JSON type config is loaded from the types filie when Init is called.
	jsonTypesFile string
}

func (f *Flags) Options() zio.ReaderOpts {
	return f.ReaderOpts
}

func (f *Flags) SetFlags(fs *flag.FlagSet) {
	fs.StringVar(&f.Format, "i", "auto", "format of input data [auto,zng,zst,ndjson,zeek,zjson,tzng,parquet]")
	fs.BoolVar(&f.Zng.Validate, "validate", true, "validate the input format when reading ZNG streams")
	fs.StringVar(&f.jsonTypesFile, "j", "", "path to json types file")
	fs.StringVar(&f.JSON.PathRegexp, "pathregexp", ndjsonio.DefaultPathRegexp,
		"regexp for extracting _path from json log name (when -inferpath=true)")
}

// Init is called after flags have been parsed.
func (f *Flags) Init() error {
	// Catch errors early ... ?! XXX
	if _, err := regexp.Compile(f.JSON.PathRegexp); err != nil {
		return err
	}
	if f.jsonTypesFile != "" {
		c, err := LoadJSONConfig(f.jsonTypesFile)
		if err != nil {
			return err
		}
		f.JSON.TypeConfig = c
	}
	return nil
}

func LoadJSONConfig(path string) (*ndjsonio.TypeConfig, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var tc ndjsonio.TypeConfig
	if err := json.Unmarshal(data, &tc); err != nil {
		return nil, fmt.Errorf("%s: unmarshaling error: %s", path, err)
	}
	if err := tc.Validate(); err != nil {
		return nil, fmt.Errorf("%s: %s", path, err)
	}
	return &tc, nil
}

func (f *Flags) Open(zctx *resolver.Context, paths []string, stopOnErr bool) ([]zbuf.Reader, error) {
	var readers []zbuf.Reader
	var warned bool
	for _, path := range paths {
		if path == "-" {
			path = iosrc.Stdin
		}
		file, err := detector.OpenFile(zctx, path, f.ReaderOpts)
		if err != nil {
			err = fmt.Errorf("%s: %w", path, err)
			if stopOnErr {
				return nil, err
			}
			fmt.Fprintln(os.Stderr, err)
			continue
		}
		if _, ok := file.Reader.(*azngio.Reader); ok {
			warned = true
			fmt.Fprintf(os.Stderr, "warning: %s: converting from alpha zng to beta zng (slow)\n", path)
		}
		readers = append(readers, file)
	}
	if warned {
		fmt.Fprintln(os.Stderr, "warning: update zng by running 'zq -o new.zng old.zng'")
	}
	return readers, nil
}
