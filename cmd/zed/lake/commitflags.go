package lake

import (
	"flag"
	"os"
)

// CommitFlags implements flags used by all "zed lake" commands that need commit info.
type CommitFlags struct {
	Date    Date
	User    string
	Message string
}

func (c *CommitFlags) SetFlags(f *flag.FlagSet) {
	c.Date = DefaultDate()
	f.Var(&c.Date, "date", "date string for commit message")
	f.StringVar(&c.User, "user", os.Getenv("ZED_LAKE_USER"), "user name for commit message")
	f.StringVar(&c.Message, "message", "", "commit message")
}
