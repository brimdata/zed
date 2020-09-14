package zio

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"regexp"

	"github.com/brimsec/zq/zio/ndjsonio"
	"github.com/brimsec/zq/zio/zngio"
)

// Reader has the union of all the flags accepted by the different
// Reader implementations.
type ReaderFlags struct {
	ReaderOpts
	// The JSON type config is loaded from the types filie when Init is called.
	jsonTypesFile string
}

func (r *ReaderFlags) Options() ReaderOpts {
	return r.ReaderOpts
}

func (r *ReaderFlags) SetFlags(fs *flag.FlagSet) {
	fs.StringVar(&r.Format, "i", "auto", "format of input data [auto,zng,ndjson,zeek,zjson,tzng,parquet]")
	fs.BoolVar(&r.Zng.Validate, "validate", true, "validate the input format when reading ZNG streams")
	fs.StringVar(&r.jsonTypesFile, "j", "", "path to json types file")
	fs.StringVar(&r.JSON.PathRegexp, "pathregexp", ndjsonio.DefaultPathRegexp,
		"regexp for extracting _path from json log name (when -inferpath=true)")
}

// Init is called after flags have been parsed.
func (r *ReaderFlags) Init() error {
	// Catch errors early ... ?! XXX
	if _, err := regexp.Compile(r.JSON.PathRegexp); err != nil {
		return err
	}
	if r.jsonTypesFile != "" {
		c, err := LoadJSONConfig(r.jsonTypesFile)
		if err != nil {
			return err
		}
		r.JSON.TypeConfig = c
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

// Writer has the union of all the flags accepted by the different
// Writer implementations.
type WriterFlags struct {
	WriterOpts
}

func (w *WriterFlags) Options() WriterOpts {
	return w.WriterOpts
}

func (w *WriterFlags) SetFlags(fs *flag.FlagSet) {
	fs.StringVar(&w.Format, "f", "zng", "format for output data [zng,ndjson,table,text,types,zeek,zjson,tzng]")
	fs.BoolVar(&w.UTF8, "U", false, "display zeek strings as UTF-8")
	fs.BoolVar(&w.Text.ShowTypes, "T", false, "display field types in text output")
	fs.BoolVar(&w.Text.ShowFields, "F", false, "display field names in text output")
	fs.BoolVar(&w.EpochDates, "E", false, "display epoch timestamps in csv and text output")
	fs.IntVar(&w.Zng.StreamRecordsMax, "b", 0, "limit for number of records in each ZNG stream (0 for no limit)")
	fs.IntVar(&w.Zng.LZ4BlockSize, "znglz4blocksize", zngio.DefaultLZ4BlockSize,
		"LZ4 block size in bytes for ZNG compression (nonpositive to disable)")
}
