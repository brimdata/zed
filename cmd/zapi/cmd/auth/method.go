package auth

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"

	"github.com/mccanne/charm"
)

var Method = &charm.Spec{
	Name:  "method",
	Usage: "auth method",
	Short: "display auth method supported by zqd service",
	Long:  ``,
	New:   NewMethod,
}

type MethodCommand struct {
	*Command
}

func NewMethod(parent charm.Command, f *flag.FlagSet) (charm.Command, error) {
	return &MethodCommand{Command: parent.(*Command)}, nil
}

func (c *MethodCommand) Run(args []string) error {
	defer c.Cleanup()
	if err := c.Init(); err != nil {
		return err
	}
	if len(args) > 0 {
		return errors.New("method command takes no arguments")
	}
	conn := c.Connection()
	res, err := conn.AuthMethod(c.Context())
	if err != nil {
		return err
	}
	b, err := json.MarshalIndent(res, "", "\t")
	if err != nil {
		return err
	}
	fmt.Println(string(b))
	return nil
}
