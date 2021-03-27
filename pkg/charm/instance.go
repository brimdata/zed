package charm

import (
	"flag"
	"fmt"
)

// instance represents a command that has been created but not run.
// It's options and defaults may be queried with the options method and
// the command can be run with the run method.
type instance struct {
	spec    *Spec
	command Command
	flags   *flag.FlagSet
}

func newInstance(parent Command, spec *Spec) (*instance, error) {
	if spec.New == nil {
		return nil, fmt.Errorf("command '%s': New function is nil", spec.Name)
	}
	flags := flag.NewFlagSet(spec.Name, flag.ContinueOnError)
	cmd, err := spec.New(parent, flags)
	if err != nil {
		return nil, err
	}
	return &instance{spec, cmd, flags}, nil
}

// run runs this instance of the command.
func (i *instance) run(args []string) error {
	// set up the flags even if there aren't any args to parse
	// since we need to establish default state in the command objects
	rest, err := parseFlags(i.flags, args)
	if err != nil {
		return err
	}
	if len(rest) == 0 {
		empty := i.spec.Empty
		if empty == nil {
			// call current command with no args
			err = i.command.Run(rest)
			if err == ErrNoRun {
				err = fmt.Errorf("%s: no sub-command supplied", i.spec.Name)
			}
			return err
		}
		// Otherwise, this is an empty sub-command that has an empty spec,
		// so invoke the spec with the current command as the parent.
		return empty.Exec(i.command, rest)
	}
	// otherwise there are more stuff after the flags so
	// look for a subcommand
	child := i.spec.lookupSub(rest[0])
	if child == nil {
		// no command found so call the current command with the args
		err = i.command.Run(rest)
		if err == ErrNoRun {
			err = fmt.Errorf("%s: no such sub-command: %s", i.spec.Name, rest[0])
		}
		return err
	}
	// we found a subcommand so execute it recursively and
	// drop the cmd name from the args
	return child.Exec(i.command, rest[1:])
}

// options returns a formatted slice of strings ready for printing as
// help for this instance of a command.
func (i *instance) options(vflag bool) []string {
	hidden := flagMap(i.spec.HiddenFlags)
	redacted := flagMap(i.spec.RedactedFlags)
	var body []string
	i.flags.VisitAll(func(f *flag.Flag) {
		name := "-" + f.Name
		if hidden[f.Name] {
			if !vflag {
				return
			}
			name = "[" + name + "]"
		}
		line := name + " " + f.Usage
		if f.DefValue != "" && !redacted[f.Name] {
			line = fmt.Sprintf("%s (default \"%s\")", line, f.DefValue)
		}
		body = append(body, line)
	})
	return body
}
