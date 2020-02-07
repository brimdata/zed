package pcap

import (
	"fmt"
	"path/filepath"
	"strconv"
	"time"

	"github.com/brimsec/zq/pkg/nano"
)

// ParseFileName parses the given .pcap filename and returns the starting
// timestamp of the file.
func ParseFileName(path string) (nano.Ts, error) {
	fname := filepath.Base(path)
	var sec int64
	if _, err := fmt.Sscanf(fname, "%d.pcap", &sec); err != nil {
		return 0, err
	}

	return nano.Ts(sec * 1e9), nil
}

// GetFileName returns the filename of the pcap given a packet's timestamp.
func GetFileName(ts time.Time) string {
	str := strconv.Itoa(int(ts.Truncate(time.Hour).Unix()))
	return str + ".pcap"
}
