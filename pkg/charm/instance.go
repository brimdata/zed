package charm

import (
	"flag"
	"fmt"
	"sort"
	"strings"
)

var (
	HelpFlag   = "h"
	HiddenFlag = "hidden"
)

// instance represents a command that has been created but not run.
// It's options and defaults may be queried with the options method and
// the command can be run with the run method.
type instance struct {
	spec    *Spec
	command Command
	flags   map[string]*flag.Flag
}

// options returns a formatted slice of strings ready for printing as
// help for this instance of a command.
func (i *instance) options(showHidden bool) []string {
	hidden := flagMap(i.spec.HiddenFlags)
	redacted := flagMap(i.spec.RedactedFlags)
	var body []string
	for _, f := range i.flags {
		name := "-" + f.Name
		if hidden[f.Name] {
			if !showHidden {
				continue
			}
			name = "[" + name + "]"
		}
		line := name + " " + f.Usage
		if f.DefValue != "" && !redacted[f.Name] {
			line = fmt.Sprintf("%s (default \"%s\")", line, f.DefValue)
		}
		body = append(body, line)
	}
	sort.Slice(body, func(i, j int) bool {
		return strings.ToLower(body[i]) < strings.ToLower(body[j])
	})
	return body
}

func parse(spec *Spec, args []string, parent Command) (Path, []string, bool, error) {
	var path Path
	var help, hidden, usage bool
	flags := flag.NewFlagSet(spec.Name, flag.ContinueOnError)
	flags.BoolVar(&help, HelpFlag, false, "display help")
	flags.BoolVar(&hidden, HiddenFlag, false, "show hidden options")
	flags.Usage = func() {
		usage = true
	}
	for {
		cmd, err := spec.New(parent, flags)
		if err != nil {
			return nil, nil, false, err
		}
		component := &instance{
			spec:    spec,
			command: cmd,
		}
		path = append(path, component)
		parent = cmd
		if err := flags.Parse(args); err != nil {
			if usage {
				s := strings.Join(args, " ")
				err = fmt.Errorf("at flag: %q: %w", s, err)
			}
			return path, nil, false, err
		}
		if help {
			return path, nil, hidden, NeedHelp
		}
		rest := flags.Args()
		if len(rest) != 0 {
			spec = component.spec.lookupSub(rest[0])
			if spec != nil {
				// We found a subcommand, so continue building the chain.
				args = rest[1:]
				continue
			}
		}
		return path, rest, false, nil
	}
}

func diff(flags *flag.FlagSet, all map[string]*flag.Flag) map[string]*flag.Flag {
	difference := make(map[string]*flag.Flag)
	flags.VisitAll(func(f *flag.Flag) {
		if _, ok := all[f.Name]; !ok {
			all[f.Name] = f
			difference[f.Name] = f
		}
	})
	return difference
}

func parseHelp(spec *Spec, args []string) (Path, error) {
	var path Path
	flags := flag.NewFlagSet(spec.Name, flag.ContinueOnError)
	var b bool
	flags.BoolVar(&b, HelpFlag, false, "display help")
	flags.BoolVar(&b, HiddenFlag, false, "show hidden options")
	flags.Usage = func() {}
	var parent Command
	all := make(map[string]*flag.Flag)
	for {
		cmd, err := spec.New(parent, flags)
		if err != nil {
			return nil, err
		}
		component := &instance{
			spec:    spec,
			command: cmd,
			flags:   diff(flags, all),
		}
		path = append(path, component)
		parent = cmd
		if err := flags.Parse(args); err != nil {
			return nil, err
		}
		rest := flags.Args()
		if len(rest) != 0 {
			spec = component.spec.lookupSub(rest[0])
			if spec != nil {
				// We found a subcommand, so continue building the chain.
				args = rest[1:]
				continue
			}
		}
		return path, nil
	}
}
