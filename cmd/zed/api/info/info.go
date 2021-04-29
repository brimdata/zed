package info

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"reflect"
	"text/tabwriter"

	"github.com/brimdata/zed/cmd/zapi/format"
	apicmd "github.com/brimdata/zed/cmd/zed/api"
	"github.com/brimdata/zed/pkg/charm"
	"github.com/brimdata/zed/pkg/nano"
	"github.com/segmentio/ksuid"
)

var Info = &charm.Spec{
	Name:  "info",
	Usage: "info [pool]",
	Short: "show information about a pool",
	Long: `The info command displays the configuration settings and other information
about the currently selected pool.`,
	New: New,
}

func init() {
	apicmd.Cmd.Add(Info)
	apicmd.Cmd.Add(Ls)
}

type Command struct {
	*apicmd.Command
}

func New(parent charm.Command, flags *flag.FlagSet) (charm.Command, error) {
	return &Command{Command: parent.(*apicmd.Command)}, nil
}

// Run lists all pools in the current zqd host or if a parameter
// is provided (in glob style) lists the info about that pool.
func (c *Command) Run(args []string) error {
	ctx, cleanup, err := c.Init()
	if err != nil {
		return err
	}
	defer cleanup()
	var pools []ksuid.KSUID
	if len(args) > 0 {
		matches, err := apicmd.PoolGlob(ctx, c.Conn, args...)
		if err != nil {
			return err
		}
		for _, m := range matches {
			pools = append(pools, m.ID)
		}
	} else {
		if c.PoolName == "" {
			return errors.New("no pool provided")
		}
		if err != nil {
			return err
		}
		pools = []ksuid.KSUID{c.PoolID}
	}
	for _, id := range pools {
		info, err := c.Conn.PoolInfo(ctx, id)
		if err != nil {
			return err
		}
		if err := printPool(info.Name, *info); err != nil {
			return err
		}
	}
	return nil
}

func printIface(w io.Writer, iface interface{}) {
	infoVal := reflect.ValueOf(iface)
	for i := 0; i < infoVal.NumField(); i++ {
		v := infoVal.Field(i)
		t := infoVal.Type().Field(i)
		name := apicmd.JSONName(t)
		if v.Kind() == reflect.Ptr && v.IsNil() {
			fmt.Fprintf(w, "  %s:\t%v\n", name, nil)
			continue
		}
		if v.Kind() == reflect.Struct && t.Anonymous {
			printIface(w, v.Interface())
			continue
		}
		v = reflect.Indirect(v)
		vi := v.Interface()
		switch t.Tag.Get("unit") {
		case "bytes":
			vi = format.Bytes(v.Int())
		case "":
			if v.Type() == reflect.TypeOf(nano.Ts(0)) {
				vi = nano.Ts(v.Int()).Time()
			}
		}
		fmt.Fprintf(w, "  %s:\t%v\n", name, vi)
	}
}

func printPool(name string, iface interface{}) error {
	fmt.Println(name)
	w := tabwriter.NewWriter(os.Stdout, 0, 2, 1, ' ', 0)
	printIface(w, iface)
	return w.Flush()
}
