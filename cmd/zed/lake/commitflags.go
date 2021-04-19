package lake

import (
	"flag"
	"os"
	"os/user"
)

func username() string {
	if s := os.Getenv("ZED_LAKE_USER"); s != "" {
		return s
	}
	u, err := user.Current()
	if err != nil {
		return ""
	}
	s := u.Username
	if s == "" {
		s = u.Name
	}
	if s == "" {
		return ""
	}
	h, err := os.Hostname()
	if err == nil {
		s += "@" + h
	}
	return s
}

// CommitFlags implements flags used by all "zed lake" commands that need commit info.
type CommitFlags struct {
	Date    Date
	User    string
	Message string
}

func (c *CommitFlags) SetFlags(f *flag.FlagSet) {
	c.Date = DefaultDate()
	f.Var(&c.Date, "date", "date string for commit message")
	f.StringVar(&c.User, "user", username(), "user name for commit message")
	f.StringVar(&c.Message, "message", "", "commit message")
}
