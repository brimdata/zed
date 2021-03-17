package nano

import (
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Duration int64

type unit struct {
	name string
	size int64
}

var units = []unit{
	{"y", int64(Year)},
	{"d", int64(Day)},
	{"h", int64(Hour)},
	{"m", int64(time.Minute)},
}

const minDur = "-292y171d23h47m16.854775808s"

func (d Duration) String() string {
	if int64(d) == math.MinInt64 {
		return minDur
	}
	if d == 0 {
		return "0s"
	}
	var b strings.Builder
	ns := int64(d)
	if ns < 0 {
		ns = -ns
		b.WriteByte('-')
	}
	for _, u := range units {
		if ns >= u.size {
			nunit := ns / u.size
			ns -= nunit * u.size
			b.WriteString(strconv.FormatInt(nunit, 10))
			b.WriteString(u.name)
			if ns == 0 {
				return b.String()
			}
		}
	}
	switch {
	case ns%1_000_000_000 == 0:
		b.WriteString(strconv.FormatInt(ns/1_000_000_000, 10))
		b.WriteString("s")
	case ns > 1_000_000_000:
		writeFixedPoint(&b, ns, 1_000_000_000)
		b.WriteString("s")
	case ns%1_000_000 == 0:
		b.WriteString(strconv.FormatInt(ns/1_000_000, 10))
		b.WriteString("ms")
	case ns > 1_000_000:
		writeFixedPoint(&b, ns, 1_000_000)
		b.WriteString("ms")
	case ns%1_000 == 0:
		b.WriteString(strconv.FormatInt(ns/1_000, 10))
		b.WriteString("us")
	case ns > 1_000:
		writeFixedPoint(&b, ns, 1_000)
		b.WriteString("us")
	default:
		b.WriteString(strconv.FormatInt(ns, 10))
		b.WriteString("ns")
	}
	return b.String()
}

func writeFixedPoint(b *strings.Builder, ns, scale int64) {
	v := ns / scale
	ns -= v * scale
	b.WriteString(strconv.FormatInt(v, 10))
	b.WriteByte('.')
	scale /= 10
	for ns > 0 {
		digit := ns / scale
		b.WriteByte('0' + byte(digit))
		ns -= digit * scale
		scale /= 10
	}
}

func (d Duration) MarshalJSON() ([]byte, error) {
	s := d.String()
	return json.Marshal(&s)
}

func (d *Duration) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	v, err := ParseDuration(s)
	if err != nil {
		return err
	}
	*d = v
	return nil
}

var parseRE = regexp.MustCompile("([.0-9]+)(ns|us|ms|s|m|h|d|w|y)")
var syntaxRE = regexp.MustCompile("^([.0-9]+(ns|us|ms|s|m|h|d|w|y))+$")

var scale = map[string]Duration{
	"ns": Nanosecond,
	"us": Microsecond,
	"ms": Millisecond,
	"s":  Second,
	"m":  Minute,
	"h":  Hour,
	"d":  Day,
	"w":  Week,
	"y":  Year,
}

func ParseDuration(s string) (Duration, error) {
	if len(s) == 0 {
		return 0, nil
	}
	var negative bool
	if s[0] == '-' {
		negative = true
		s = s[1:]
	}
	if !syntaxRE.MatchString(s) {
		return 0, fmt.Errorf("invalid duration: %q", s)
	}
	matches := parseRE.FindAllStringSubmatch(s, -1)
	var d Duration
	for _, m := range matches {
		if len(m) != 3 {
			return 0, fmt.Errorf("invalid duration: %q", s)
		}
		factor, ok := scale[m[2]]
		if !ok {
			return 0, fmt.Errorf("invalid duration: %q", s)
		}
		val, err := strconv.ParseInt(m[1], 10, 64)
		if err == nil {
			d += Duration(val) * factor
			continue
		}
		f, err := strconv.ParseFloat(m[1], 64)
		if err != nil {
			return 0, fmt.Errorf("invalid duration: %q", s)
		}
		d += Duration(f * float64(factor))
	}
	if negative {
		d = -d
	}
	return d, nil
}

func DurationFromParts(sec, ns int64) Duration {
	return Duration(sec*1_000_000_000 + ns)
}

func FloatToDuration(fsec float64) Duration {
	rsec := math.Round(fsec)
	ns := fsec - rsec
	return DurationFromParts(int64(rsec), int64(ns*1e9))
}
