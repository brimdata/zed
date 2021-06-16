package units

// XXX this should be unified with cmd/zapi/format

import (
	"fmt"

	"github.com/alecthomas/units"
)

// Bytes implements flag.Value
type Bytes units.Base2Bytes

func String(bytes Bytes) string {
	return units.Base2Bytes(bytes).String()
}

func (b Bytes) String() string {
	return String(b)
}

func (b *Bytes) Set(s string) error {
	bytes, err := units.ParseStrictBytes(s)
	if err != nil {
		return err
	}
	*b = Bytes(bytes)
	return nil
}

func format(b units.MetricBytes, suffix string, unit units.MetricBytes) string {
	amt := b / unit
	if amt*unit == b {
		return fmt.Sprintf("%d%s", amt, suffix)
	}
	f := float64(b) / float64(unit)
	return fmt.Sprintf("%.2f%s", f, suffix)
}

func (b Bytes) Abbrev() string {
	v := units.MetricBytes(b)
	switch {
	case v >= units.PB:
		return format(v, "PB", units.PB)
	case v >= units.GB:
		return format(v, "GB", units.GB)
	case v >= units.MB:
		return format(v, "MB", units.MB)
	case v >= 10*units.KB:
		return format(v, "KB", units.KB)
	default:
		return format(v, "B", 1)
	}
}
