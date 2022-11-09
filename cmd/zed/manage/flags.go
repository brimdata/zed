package manage

import (
	"flag"
	"os"
	"time"

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
		return yaml.Unmarshal(b, &f.Config)
	})
	fs.DurationVar(&f.Config.Compact.ColdThreshold, "coldthresh", time.Minute*5, "age at which objects are considered for compaction")
}
