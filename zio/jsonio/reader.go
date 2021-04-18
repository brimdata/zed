package jsonio

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/brimdata/zed/zio/ndjsonio"
	"github.com/brimdata/zed/zng"
	"github.com/brimdata/zed/zson"
)

const MaxReadBuffer = 25 * 1024 * 1024

type Reader struct {
	zctx    *zson.Context
	reader  io.Reader
	parser  *ndjsonio.InferParser
	objects []interface{}
}

func NewReader(r io.Reader, zctx *zson.Context) (*Reader, error) {
	return &Reader{
		zctx:   zctx,
		reader: r,
		parser: ndjsonio.NewInferParser(zctx),
	}, nil
}

func (r *Reader) Read() (*zng.Record, error) {
	if r.objects == nil {
		b, err := ioutil.ReadAll(io.LimitReader(r.reader, MaxReadBuffer))
		if err != nil {
			if err == io.EOF {
				err = nil
			}
			return nil, err
		}
		if len(b) == MaxReadBuffer {
			return nil, fmt.Errorf("JSON input buffer size exceeded: %d", MaxReadBuffer)

		}
		var v interface{}
		if err := json.Unmarshal(b, &v); err != nil {
			return nil, err
		}
		if object, ok := v.(map[string]interface{}); ok {
			r.objects = make([]interface{}, 0)
			return r.parse(object)
		}
		a, ok := v.([]interface{})
		if !ok {
			fmt.Printf("%T\n", v)
			return nil, errors.New("JSON input is neither an object or array")
		}
		r.objects = a
	}
	if len(r.objects) == 0 {
		return nil, nil
	}
	object := r.objects[0]
	r.objects = r.objects[1:]
	return r.parse(object)
}

func (r *Reader) parse(v interface{}) (*zng.Record, error) {
	object, ok := v.(map[string]interface{})
	if !ok {
		object = make(map[string]interface{})
		object["value"] = v
	}
	b, err := json.Marshal(object)
	if err != nil {
		return nil, err
	}
	zv, err := r.parser.ParseObject(b)
	if err != nil {
		return nil, fmt.Errorf("cannot parse JSON object: %q", string(b))
	}
	outType, err := r.zctx.LookupTypeRecord(zv.Type.(*zng.TypeRecord).Columns)
	if err != nil {
		return nil, err
	}
	return zng.NewRecordCheck(outType, zv.Bytes)
}
