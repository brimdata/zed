package nano

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"
)

type Ts int64

const (
	Day   = int64(86400) * 1000000000
	MinTs = Ts(0)
	MaxTs = Ts(math.MaxInt64)
)

type jsonTs struct {
	Sec int64 `json:"sec"`
	Ns  int64 `json:"ns"`
}

func access(m map[string]interface{}, field string) (int64, bool) {
	if v, ok := m[field]; ok {
		f, ok := v.(float64)
		if ok {
			return int64(f), true
		}
	}
	return 0, false
}

func (t *Ts) UnmarshalJSON(in []byte) error {
	var v interface{}
	if err := json.Unmarshal(in, &v); err != nil {
		return err
	}
	switch v := v.(type) {
	case string:
		var err error
		*t, err = ParseTs(v)
		return err

	case float64:
		*t = Ts(v * 1e9)
		return nil

	case map[string]interface{}:
		sec, ok := access(v, "sec")
		if ok {
			ns, ok := access(v, "ns")
			if !ok {
				ns = 0
			}
			*t = Ts(int64(sec)*1000000000 + int64(ns))
			return nil
		}
		return fmt.Errorf("time object is not of the form {sec:x, ns:y}")
	}
	return fmt.Errorf("unsupported time format: %T", v)
}

// Split returns the seconds and nanoseconds since epoch of the timestamp.
func (t Ts) Split() (int64, int64) {
	sec := int64(t / 1000000000)
	ns := int64(t) - sec*1000000000
	return sec, ns
}

func (t Ts) MarshalJSON() ([]byte, error) {
	sec, ns := t.Split()
	v := jsonTs{sec, ns}
	return json.Marshal(&v)
}

func (t Ts) Time() time.Time {
	sec, ns := t.Split()
	return time.Unix(int64(sec), int64(ns)).UTC()
}

func (t Ts) Trunc(bin int64) Ts {
	return Ts(int64(t) / bin * bin)
}

func (t Ts) Midnight() Ts {
	return t.Trunc(Day)
}

func (t Ts) DayOf() Span {
	return Span{t.Midnight(), Day}
}

func (t Ts) String() string {
	return t.Time().Format(time.RFC3339)
}

func (t Ts) Pretty() string {
	return t.Time().Format("01/02/2006@15:04:05")
}

func (t Ts) StringFloat() string {
	sec, ns := t.Split()
	return fmt.Sprintf("%d.%09d", sec, ns)
}

func (t Ts) Add(v int64) Ts {
	return Ts(int64(t) + v)
}

func (t Ts) Sub(v int64) Ts {
	return Ts(int64(t) - v)
}

// SubTs returns the duration t-u.
func (t Ts) SubTs(u Ts) int64 {
	return int64(t - u)
}

// convert a golang time to a nano Ts
func TimeToTs(t time.Time) Ts {
	return Ts(t.UnixNano())
}

func Date(year int, month time.Month, day, hour, min, sec, nsec int) Ts {
	t := time.Date(year, month, day, hour, min, sec, nsec, time.UTC)
	return TimeToTs(t)
}

func Now() Ts {
	return TimeToTs(time.Now())
}

func ParseTs(s string) (Ts, error) {
	return Parse([]byte(s))
}

func Parse(s []byte) (Ts, error) {
	i, err := parse(s)
	if err != nil {
		return 0, err
	}
	if i < 0 {
		return 0, errors.New("time cannot be negative")
	}
	return Ts(i), nil
}

func parse(s []byte) (int64, error) {
	var v, scale, sign int64
	sign = 1
	scale = 1000000000
	k := 0
	n := len(s)
	if s[0] == '-' {
		sign, k = -1, 1
	}
	for ; k < n; k++ {
		c := s[k]
		if c != '.' && (c < '0' || c > '9') {
			return 0, fmt.Errorf("invalid time format: %s", string(s))
		}
		if c == '.' {
			for k++; k < n; k++ {
				c = s[k]
				if c < '0' || c > '9' {
					return 0, fmt.Errorf("invalid time format: %s", string(s))
				}
				v = v*10 + int64(c-'0')
				scale /= 10
			}
			break
		}
		v = v*10 + int64(c-'0')
	}
	return sign * v * scale, nil
}

// ParseMillis parses an unsigned integer representing milliseconds since the
// Unix epoch.
func ParseMillis(s []byte) (Ts, error) {
	if len(s) == 0 {
		return 0, fmt.Errorf("invalid time format: %s", string(s))
	}
	var v int64
	for _, c := range s {
		d := c - '0'
		if d > 9 {
			return 0, fmt.Errorf("invalid time format: %s", string(s))
		}
		v = v*10 + int64(d)
	}
	return Ts(v * 1_000_000), nil
}

// ParseRFC3339Nano parses a byte according to the time.RFC3339Nano
// format into a Ts, returning an error if parsing failed.
func ParseRFC3339Nano(s []byte) (Ts, error) {
	t, err := time.Parse(time.RFC3339Nano, string(s))
	if err != nil {
		return 0, err
	}
	return TimeToTs(t), nil
}

func ParseDuration(s []byte) (int64, error) {
	return parse(s)
}

// Max compares and returns the largest Ts.
func Max(a, b Ts) Ts {
	if a > b {
		return a
	}
	return b
}

// Min compares and returns the smallest Ts.
func Min(a, b Ts) Ts {
	if a < b {
		return a
	}
	return b
}

// Unix returns a Ts corresponding to the given Unix time, sec seconds
// and nsec nanoseconds since January 1, 1970 UTC.
func Unix(sec, ns int64) Ts {
	return Ts(int64(sec)*1000000000 + int64(ns))
}

// Duration returns an int64 representation of the combined sec and ns passed
// through.
func Duration(sec, ns int64) int64 {
	return int64(sec)*1000000000 + int64(ns)
}

func DurationString(dur int64) string {
	sec := int64(dur / 1000000000)
	ns := int64(dur) - sec*1000000000
	return fmt.Sprintf("%d.%09d", sec, ns)
}
