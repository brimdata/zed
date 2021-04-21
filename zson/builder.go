package zson

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"net"
	"strconv"
	"time"
	"unicode/utf8"

	"github.com/brimdata/zed/pkg/nano"
	"github.com/brimdata/zed/zcode"
	"github.com/brimdata/zed/zng"
)

func Build(b *zcode.Builder, val Value) (zng.Value, error) {
	b.Reset()
	if err := buildValue(b, val); err != nil {
		return zng.Value{}, err
	}
	it := b.Bytes().Iter()
	bytes, _, err := it.Next()
	if err != nil {
		return zng.Value{}, err
	}
	return zng.Value{val.TypeOf(), bytes}, nil
}

func buildValue(b *zcode.Builder, val Value) error {
	switch val := val.(type) {
	case *Primitive:
		return BuildPrimitive(b, *val)
	case *Record:
		return buildRecord(b, val)
	case *Array:
		return buildArray(b, val)
	case *Set:
		return buildSet(b, val)
	case *Union:
		return buildUnion(b, val)
	case *Map:
		return buildMap(b, val)
	case *Enum:
		return buildEnum(b, val)
	case *TypeValue:
		return buildTypeValue(b, val)
	case *Null:
		b.AppendNull()
		return nil
	}
	return fmt.Errorf("unknown ast type: %T", val)
}

func BuildPrimitive(b *zcode.Builder, val Primitive) error {
	switch zng.AliasOf(val.Type).(type) {
	case *zng.TypeOfUint8, *zng.TypeOfUint16, *zng.TypeOfUint32, *zng.TypeOfUint64:
		v, err := strconv.ParseUint(val.Text, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid unsigned integer: %s", val.Text)
		}
		b.AppendPrimitive(zng.EncodeUint(v))
		return nil
	case *zng.TypeOfInt8, *zng.TypeOfInt16, *zng.TypeOfInt32, *zng.TypeOfInt64:
		v, err := strconv.ParseInt(val.Text, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid integer: %s", val.Text)
		}
		b.AppendPrimitive(zng.EncodeInt(v))
		return nil
	case *zng.TypeOfDuration:
		d, err := nano.ParseDuration(val.Text)
		if err != nil {
			return fmt.Errorf("invalid duration: %s", val.Text)
		}
		b.AppendPrimitive(zng.EncodeDuration(d))
		return nil
	case *zng.TypeOfTime:
		t, err := time.Parse(time.RFC3339Nano, val.Text)
		if err != nil {
			return fmt.Errorf("invalid ISO time: %s", val.Text)
		}
		b.AppendPrimitive(zng.EncodeTime(nano.TimeToTs(t)))
		return nil
	case *zng.TypeOfFloat64:
		v, err := strconv.ParseFloat(val.Text, 64)
		if err != nil {
			return fmt.Errorf("invalid floating point: %s", val.Text)
		}
		b.AppendPrimitive(zng.EncodeFloat64(v))
		return nil
	case *zng.TypeOfBool:
		var v bool
		if val.Text == "true" {
			v = true
		} else if val.Text != "false" {
			return fmt.Errorf("invalid bool: %s", val.Text)
		}
		b.AppendPrimitive(zng.EncodeBool(v))
		return nil
	case *zng.TypeOfBytes:
		s := val.Text
		if len(s) < 2 || s[0:2] != "0x" {
			return fmt.Errorf("invalid bytes: %s", s)
		}
		var bytes []byte
		if len(s) == 2 {
			// '0x' is an empty byte string (not null byte string)
			bytes = make([]byte, 0, 0)
		} else {
			var err error
			bytes, err = hex.DecodeString(s[2:])
			if err != nil {
				return fmt.Errorf("invalid bytes: %s (%s)", s, err.Error())
			}
		}
		b.AppendPrimitive(zcode.Bytes(bytes))
		return nil
	case *zng.TypeOfString, *zng.TypeOfType, *zng.TypeOfError:
		body := zng.EncodeString(val.Text)
		if !utf8.Valid(body) {
			return fmt.Errorf("invalid utf8 string: %q", val.Text)
		}
		b.AppendPrimitive(body)
		return nil
	case *zng.TypeOfBstring:
		b.AppendPrimitive(unescapeHex([]byte(val.Text)))
		return nil
	case *zng.TypeOfIP:
		ip := net.ParseIP(val.Text)
		if ip == nil {
			return fmt.Errorf("invalid IP: %s", val.Text)
		}
		b.AppendPrimitive(zng.EncodeIP(ip))
		return nil
	case *zng.TypeOfNet:
		_, net, err := net.ParseCIDR(val.Text)
		if err != nil {
			return fmt.Errorf("invalid network: %s (%s)", val.Text, err.Error())
		}
		b.AppendPrimitive(zng.EncodeNet(net))
		return nil
	case *zng.TypeOfNull:
		if val.Text != "" {
			return fmt.Errorf("invalid text body of null value: %q", val.Text)
		}
		b.AppendPrimitive(nil)
		return nil
	}
	return fmt.Errorf("unknown primitive: %T", val.Type)
}

func unescapeHex(in []byte) []byte {
	if bytes.IndexByte(in, '\\') < 0 {
		return in
	}
	b := make([]byte, 0, len(in))
	i := 0
	for i < len(in) {
		c := in[i]
		if c == '\\' && len(in[i:]) >= 4 && in[i+1] == 'x' {
			v1 := zng.Unhex(in[i+2])
			v2 := zng.Unhex(in[i+3])
			// This is undefined behavior for non hex \x chars.
			c = v1<<4 | v2
			i += 4
		} else {
			i++
		}
		b = append(b, c)
	}
	return b
}

func buildRecord(b *zcode.Builder, val *Record) error {
	b.BeginContainer()
	for _, v := range val.Fields {
		if err := buildValue(b, v); err != nil {
			return err
		}
	}
	b.EndContainer()
	return nil
}

func buildArray(b *zcode.Builder, array *Array) error {
	b.BeginContainer()
	for _, v := range array.Elements {
		if err := buildValue(b, v); err != nil {
			return err
		}
	}
	b.EndContainer()
	return nil
}

func buildSet(b *zcode.Builder, set *Set) error {
	b.BeginContainer()
	for _, v := range set.Elements {
		if err := buildValue(b, v); err != nil {
			return err
		}
	}
	b.EndContainer()
	return nil
}

func buildMap(b *zcode.Builder, m *Map) error {
	b.BeginContainer()
	for _, entry := range m.Entries {
		if err := buildValue(b, entry.Key); err != nil {
			return err
		}
		if err := buildValue(b, entry.Value); err != nil {
			return err
		}
	}
	b.EndContainer()
	return nil
}

func buildUnion(b *zcode.Builder, union *Union) error {
	if selector := union.Selector; selector >= 0 {
		b.BeginContainer()
		b.AppendPrimitive(zng.EncodeInt(int64(union.Selector)))
		if err := buildValue(b, union.Value); err != nil {
			return err
		}
		b.EndContainer()
	} else {
		b.AppendNull()
	}
	return nil
}

func buildEnum(b *zcode.Builder, enum *Enum) error {
	b.AppendPrimitive(zng.EncodeUint(uint64(enum.Selector)))
	return nil
}

func buildTypeValue(b *zcode.Builder, tv *TypeValue) error {
	b.AppendPrimitive(zcode.Bytes(FormatType(tv.Value)))
	return nil
}
