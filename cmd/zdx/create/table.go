package create

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/brimsec/zq/zdx"
)

// Table reads a TSV file with a key and value expressed as hex strings and
// implements the zdx.Reader interface to enumerate the key/value pairs found.
// If just a key without any integers is listed on a line, then an empty value
// is assumed.
type Table struct {
	table  map[string][]byte
	keys   []string
	offset int
}

func NewTable() *Table {
	return &Table{
		table: make(map[string][]byte),
	}
}

func (t *Table) Scan(f *os.File) error {
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		fmt.Println(scanner.Text())
		if err := t.parse(scanner.Text()); err != nil {
			return err
		}
	}
	return scanner.Err()
}

func (t *Table) parse(line string) error {
	keyval := strings.Split(line, ":")
	var value []byte
	switch len(keyval) {
	default:
		//XXX bad input... ignore for now
		return nil
	case 1:
		// value is empty
	case 2:
		var err error
		value, err = hex.DecodeString(keyval[1])
		if err != nil {
			return err
		}
	}
	key := keyval[0]
	t.table[key] = value
	return nil
}

func (t *Table) Read() (zdx.Pair, error) {
	off := t.offset
	if off >= len(t.keys) {
		return zdx.Pair{}, nil
	}
	key := t.keys[off]
	t.offset = off + 1
	// note value can be nil
	return zdx.Pair{[]byte(key), t.table[key]}, nil
}

func (t *Table) Open() error {
	n := len(t.table)
	if n == 0 {
		return nil
	}
	t.keys = make([]string, n)
	k := 0
	for key := range t.table {
		t.keys[k] = key
		k++
	}
	sort.Strings(t.keys)
	t.offset = 0
	return nil
}

func (t *Table) Close() error {
	t.keys = nil
	return nil
}
