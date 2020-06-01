package format

import (
	"fmt"
	"math"
	"strconv"
)

const (
	Checkbox = "\u2713"
	Warning  = "\u26a0"

	KB = 1000
	MB = 1000 * KB
	GB = 1000 * MB
	TB = 1000 * GB

	NS  = 1
	US  = 1000 * NS
	MS  = 1000 * US
	SEC = 1000 * MS
	MIN = 60 * SEC
	HR  = 60 * MIN
	DAY = 24 * HR
)

func prec(m float64) int {
	switch {
	case m >= 1000:
		return 0
	case m >= 100:
		return 1
	default:
		return 2
	}
}

func abbrev(size int64) (string, string) {
	switch {
	case size < KB:
		return strconv.FormatInt(size, 10), "B"
	case size < MB:
		v := float64(size) / KB
		return strconv.FormatFloat(v, 'f', prec(v), 64), "KB"
	case size < GB:
		v := float64(size) / MB
		return strconv.FormatFloat(v, 'f', prec(v), 64), "MB"
	case size < TB:
		v := float64(size) / GB
		return strconv.FormatFloat(v, 'f', prec(v), 64), "GB"
	default:
		v := float64(size) / TB
		return strconv.FormatFloat(v, 'f', prec(v), 64), "TB"
	}
}

func Bytes(size int64) string {
	val, suffix := abbrev(size)
	if suffix == "bytes" {
		suffix = ""
	}
	return val + suffix
}

func Rate(size int64) string {
	val, suffix := abbrev(size)
	return fmt.Sprintf("%s %s/s", val, suffix)
}

func dur(ns int64) (string, string) {
	switch {
	case ns < US:
		return strconv.FormatInt(ns, 10), "ns"
	case ns < MS:
		v := float64(ns) / US
		return strconv.FormatFloat(v, 'f', prec(v), 64), "us"
	case ns < SEC:
		v := float64(ns) / MS
		return strconv.FormatFloat(v, 'f', prec(v), 64), "ms"
	case ns < MIN:
		v := float64(ns) / SEC
		return strconv.FormatFloat(v, 'f', prec(v), 64), "s"
	case ns < HR:
		v := float64(ns) / MIN
		return strconv.FormatFloat(v, 'f', prec(v), 64), "min"
	case ns < DAY:
		v := float64(ns) / HR
		return strconv.FormatFloat(v, 'f', prec(v), 64), "hr"
	default:
		v := float64(ns) / DAY
		return strconv.FormatFloat(v, 'f', prec(v), 64), "days"
	}
}

func Duration(ns int64) string {
	val, suffix := dur(ns)
	return val + suffix
}

func Percent(p float64) string {
	if p < 1 {
		return fmt.Sprintf("%.2f%%", p)
	}
	return fmt.Sprintf("%d%%", int(math.Round(p)))
}
