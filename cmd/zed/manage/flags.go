package manage

import (
	"bytes"
	"flag"
	"os"

	"github.com/brimdata/zed/cmd/zed/manage/lakemanage"
	"gopkg.in/yaml.v3"
)

type Flags struct {
	Config lakemanage.Config
}

func (f *Flags) SetFlags(fs *flag.FlagSet) {
	fs.Func("config", "path of manage yaml config file", func(s string) error {
		b, err := os.ReadFile(s)
		if err != nil {
			return err
		}
		d := yaml.NewDecoder(bytes.NewReader(b))
		d.KnownFields(true) // returns error for unknown fields
		return d.Decode(&f.Config)
	})
}
