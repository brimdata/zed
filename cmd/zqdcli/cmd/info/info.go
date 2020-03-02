package info

import (
	"flag"
	"fmt"
	"reflect"

	"github.com/brimsec/zq/cmd/zqdcli/cmd"
	"github.com/brimsec/zq/cmd/zqdcli/format"
	"github.com/brimsec/zq/pkg/nano"
	"github.com/mccanne/charm"
)

var Info = &charm.Spec{
	Name:  "info",
	Usage: "info [spacename]",
	Short: "show information about a space",
	Long: `The info command displays the configuration settings and other information
about the currently selected space.`,
	New: New,
}

func init() {
	cmd.Cli.Add(Info)
	cmd.Cli.Add(Ls)
}

type Command struct {
	*cmd.Command
}

func New(parent charm.Command, flags *flag.FlagSet) (charm.Command, error) {
	return &Command{Command: parent.(*cmd.Command)}, nil
}

// Run lists all spaces in the current zqd host or if a parameter
// is provided (in glob style) lists the info about that space.
func (c *Command) Run(args []string) error {
	api, err := c.API()
	if err != nil {
		return err
	}
	if len(args) == 0 {
		return printInfo(api, c.Spacename)
	} else {
		matches, err := cmd.SpaceGlob(api, args)
		if err != nil {
			return err
		}
		return printInfoList(api, matches)
	}
}

func printInfoList(api *cmd.API, names []string) error {
	for _, name := range names {
		if err := printInfo(api, name); err != nil {
			return err
		}
	}
	return nil
}

func printInfo(api *cmd.API, name string) error {
	info, err := api.SpaceInfo(name)
	if err != nil {
		return err
	}
	fmt.Println(info.Name)
	infoVal := reflect.ValueOf(info).Elem()
	for i := 0; i < infoVal.NumField(); i++ {
		v := infoVal.Field(i)
		t := infoVal.Type().Field(i)
		name := cmd.JsonName(t)
		if v.Kind() == reflect.Ptr {
			v = v.Elem()
		}
		vi := v.Interface()
		switch t.Tag.Get("unit") {
		case "bytes":
			vi = format.Bytes(v.Int())
		case "":
			if v.Type() == reflect.TypeOf(nano.Ts(0)) {
				vi = nano.Ts(v.Int()).Time()
			}
		}
		fmt.Printf("  %s:\t%v\n", name, vi)
	}
	return nil
}
