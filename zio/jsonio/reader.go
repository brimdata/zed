package jsonio

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/brimdata/zed"
)

type Reader struct {
	builder builder
	decoder *json.Decoder
}

func NewReader(r io.Reader, zctx *zed.Context) *Reader {
	d := json.NewDecoder(r)
	d.UseNumber()
	return &Reader{
		builder: builder{zctx: zctx},
		decoder: d,
	}
}

func (r *Reader) Read() (*zed.Value, error) {
	token, err := r.decoder.Token()
	if err != nil {
		if err == io.EOF {
			return nil, nil
		}
		return nil, err
	}
	r.builder.reset()
	if err := r.handleToken("", token); err != nil {
		return nil, err
	}
	return r.builder.value(), nil
}

func (r *Reader) handleToken(fieldName string, token json.Token) error {
	switch token {
	case json.Delim('['):
		r.builder.beginContainer(fieldName)
		if err := r.readArray(); err != nil {
			return err
		}
		r.builder.endArray()
		return nil
	case json.Delim('{'):
		r.builder.beginContainer(fieldName)
		if err := r.readRecord(); err != nil {
			return err
		}
		r.builder.endRecord()
		return nil
	}
	if !r.builder.addPrimitive(fieldName, token) {
		return r.tokenError(token)
	}
	return nil
}

func (r *Reader) readArray() error {
	for {
		token, err := r.decoder.Token()
		if token == json.Delim(']') || err != nil {
			return err
		}
		if err := r.handleToken("", token); err != nil {
			return err
		}
	}
}

func (r *Reader) readRecord() error {
	for {
		token, err := r.decoder.Token()
		if token == json.Delim('}') || err != nil {
			return err
		}
		field, ok := token.(string)
		if !ok {
			return r.tokenError(token)
		}
		token, err = r.decoder.Token()
		if err != nil {
			return err
		}
		if err := r.handleToken(field, token); err != nil {
			return err
		}
	}
}

func (r *Reader) tokenError(token json.Token) error {
	return fmt.Errorf("unexpected JSON token '%v' at offset %d", token, r.decoder.InputOffset())
}
