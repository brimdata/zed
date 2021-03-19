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

const (
	Nanosecond  Duration = 1
	Microsecond          = 1000 * Nanosecond
	Millisecond          = 1000 * Microsecond
	Second               = 1000 * Millisecond
	Minute               = 60 * Second
	Hour                 = 60 * Minute
	Day                  = 24 * Hour
	Week                 = 7 * Day
	Year                 = 365 * Day
)

var units = []struct {
	name string
	size int64
}{
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
			if nunit > 0 {
				b.WriteString(strconv.FormatInt(nunit, 10))
				b.WriteString(u.name)
			}
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
	return json.Marshal(d.String())
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

func DurationFromParts(sec, ns int64) Duration {
	return Duration(sec)*Second + Duration(ns)
}

func DurationFromFloat(fsec float64) Duration {
	return Duration(fsec * 1e9)
}

func DurationFromFloat2(fsec float64) Duration {
	return Duration(fsec * 1e9)
}

var parseRE = regexp.MustCompile("([.0-9]+)(ns|us|ms|s|m|h|d|w|y)")
var syntaxRE = regexp.MustCompile("^-?([.0-9]+(ns|us|ms|s|m|h|d|w|y))+$")

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
		unit, ok := scale[m[2]]
		if !ok {
			return 0, fmt.Errorf("invalid duration: %q", s)
		}
		val, err := strconv.ParseInt(m[1], 10, 64)
		if err == nil {
			d += Duration(val) * unit
			continue
		}
		parts := strings.Split(m[1], ".")
		if len(parts) != 2 {
			return 0, fmt.Errorf("invalid duration: %q", s)
		}
		if len(parts[0]) > 0 {
			whole, err := strconv.ParseInt(parts[0], 10, 64)
			if err != nil {
				return 0, fmt.Errorf("invalid duration: %q", s)
			}
			d += Duration(whole) * unit
		}
		frac := strings.TrimRight(parts[1], "0")
		var extra Duration
		for _, digit := range []byte(frac) {
			extra += Duration(digit-'0') * unit
			unit /= 10
		}
		d += (extra + 5) / 10
	}
	if negative {
		d = -d
	}
	return d, nil
}
