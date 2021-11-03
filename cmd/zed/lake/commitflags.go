package lake

import (
	"flag"
	"os"
	"os/user"

	"github.com/brimdata/zed/api"
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
	User    string
	Message string
	Meta    string
}

func (c *CommitFlags) SetFlags(f *flag.FlagSet) {
	f.StringVar(&c.User, "user", username(), "user name for commit message")
	f.StringVar(&c.Message, "message", "", "commit message")
	f.StringVar(&c.Meta, "meta", "", "application metadata")
}

func (c *CommitFlags) CommitMessage() api.CommitMessage {
	return api.CommitMessage{
		Author: c.User,
		Body:   c.Message,
		Meta: 	c.Meta,
	}
}
