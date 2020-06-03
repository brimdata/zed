package fs

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
)

// MarshalJSONFile writes the JSON encoding of v to a file named by filename.
// The file is created atomically. If the file does not exist, Marshal creates
// it with permissions perm.  Internally MarshalJSONFile uses json.Marshal.
func MarshalJSONFile(v interface{}, filename string, perm os.FileMode) error {
	tmppath := filename + ".tmp"
	f, err := OpenFile(tmppath, os.O_RDWR|os.O_CREATE|os.O_TRUNC|os.O_EXCL, perm)
	if err != nil {
		return err
	}
	if err := json.NewEncoder(f).Encode(v); err != nil {
		f.Close()
		os.Remove(tmppath)
		return err
	}
	if err := f.Close(); err != nil {
		os.Remove(tmppath)
		return err
	}
	return os.Rename(tmppath, filename)
}

// UnmarshalJSONFile parses JSON-encoded data from the file named by filename
// and stores the result in the value pointed to by v.  Internally, Unmarshal
// uses json.Unmarshal.
func UnmarshalJSONFile(filename string, v interface{}) error {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}

	if err = json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("%s: unmarshaling error: %s", filename, err)
	}
	return nil
}
