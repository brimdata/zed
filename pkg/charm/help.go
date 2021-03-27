package charm

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/brimsec/zq/pkg/terminal"
	"github.com/kr/text"
)

var NeedHelp = errors.New("help")

var Help = &Spec{
	Name:  "help",
	Usage: "help [command]",
	Short: "display help for a command",
	Long: `
For help on the top-level command just type "help".
For help on a subcommand, type "help command" where command is the name of
the command.  For help on command nested further, type "help cmd1 cmd2" and
so forth.`,
	HiddenFlags: "v",
	New: func(parent Command, f *flag.FlagSet) (Command, error) {
		c := &HelpCommand{}
		f.BoolVar(&c.vflag, "v", false, "show hidden commands and flags")
		return c, nil
	},
}

type HelpCommand struct {
	vflag bool
}

// splitFlags is like strings.Split with a comma and also trims whitespace
func splitFlags(flags string) []string {
	var out []string
	for _, flag := range strings.Split(flags, ",") {
		out = append(out, strings.TrimSpace(flag))
	}
	return out
}

// flagMap creates a map that maps a name to a boolean based on the existence
// of that name in the comma-separated string of flags.  Whitespace is removed
// from each name in the flags list.  A map that contains no entries is returned
// for an empty string.
func flagMap(flags string) map[string]bool {
	hidden := make(map[string]bool)
	for _, flag := range splitFlags(flags) {
		hidden[flag] = true
	}
	return hidden
}

func (c *HelpCommand) search(args []string) ([]*instance, error) {
	//XXX parent of root can be custom...
	parent, err := newInstance(nil, Help.Root())
	if err != nil {
		return nil, err
	}
	sequence := ""
	var instances []*instance
	instances = append(instances, parent)
	for _, arg := range args {
		sequence = sequence + " " + arg
		subcmd := parent.spec.lookupSub(arg)
		if subcmd == nil {
			if len(args) == 1 {
				return nil, fmt.Errorf("no such command: %s", arg)
			} else {
				return nil, fmt.Errorf("no such command:%s", sequence)
			}
		}
		child, err := newInstance(parent.command, subcmd)
		if err != nil {
			return nil, err
		}
		instances = append(instances, child)
		parent = child
	}
	return instances, nil
}

func (c *HelpCommand) Prepare(f *flag.FlagSet) {}

func (c *HelpCommand) Run(args []string) error {
	inst, err := c.search(args)
	if err != nil {
		return err
	}
	c.help(inst)
	return nil
}

func formatParagraph(body, tab string, lineWidth int) string {
	paragraphs := strings.Split(body, "\n\n")
	var chunks []string
	for _, paragraph := range paragraphs {
		var chunk string
		if len(paragraph) < lineWidth {
			chunk = strings.TrimRight(paragraph, " \t\n")
		} else {
			paragraph = strings.TrimSpace(paragraph)
			paragraph = text.Wrap(paragraph, lineWidth)
			lines := strings.Split(paragraph, "\n")
			chunk = strings.Join(lines, "\n"+tab)
		}
		chunks = append(chunks, chunk)
	}
	body = strings.Join(chunks, "\n\n"+tab)
	body = strings.TrimRight(body, " \t\n")
	return tab + body + "\n\n"
}

const tab = "    "

func header(heading string) string {
	boldOn := "\033[1m"
	boldOff := "\033[0m"
	return boldOn + heading + boldOff
}

func helpItem(heading, body string) {
	hdr := header(heading)
	body = hdr + "\n" + tab + body + "\n\n"
	fmt.Fprint(os.Stderr, body)
}

func helpDesc(heading, body string) {
	hdr := header(heading)
	body = tab + body + "\n\n"
	w := terminal.Width()
	lineWidth := w - len(tab) - 5
	if len(body) > lineWidth {
		body = formatParagraph(body, tab, lineWidth)
	}
	fmt.Fprint(os.Stderr, hdr+"\n"+body)
}

func helpList(heading string, lines []string) {
	hdr := header(heading)
	body := strings.Join(lines, "\n"+tab)
	body = hdr + "\n" + tab + body + "\n\n"
	fmt.Fprint(os.Stderr, body)
}

func optionSection(body []string) []string {
	if len(body) == 0 {
		return []string{"no flags for this command"}
	}
	return body
}

func (c *HelpCommand) getCommands(target *Spec, vflag bool) []string {
	var lines []string
	for _, cmd := range target.children {
		name := cmd.Name
		if cmd.Hidden {
			if vflag {
				name = "[" + name + "]"
			} else {
				continue
			}
		}
		line := name + " - " + cmd.Short
		lines = append(lines, line)
	}
	return lines
}

func buildOptions(path []*instance, parentCmd string, vflag bool) []string {
	n := len(path)
	if n == 1 {
		options := path[0].options(vflag)
		if len(options) == 0 {
			options = []string{"no flags for this command"}
		}
		return options
	}
	pathCmd := path[0].spec.Name
	if parentCmd != "" {
		pathCmd = parentCmd + " " + pathCmd
	}
	childOptions := buildOptions(path[1:], parentCmd, vflag)
	options := path[0].options(vflag)
	if len(options) == 0 {
		return childOptions
	}
	// add a line separator then add the command path in parens as the header
	// for the next set of flags
	childOptions = append(childOptions, "")
	childOptions = append(childOptions, "["+pathCmd+" flags]")
	for _, option := range options {
		childOptions = append(childOptions, option)
	}
	return childOptions
}

func optionsSection(path []*instance, vflag bool) []string {
	return buildOptions(path, "", vflag)
}

func (c *HelpCommand) help(path []*instance) {
	spec := path[len(path)-1].spec
	name := spec.Name + " - " + spec.Short
	helpItem("NAME", name)
	helpDesc("USAGE", spec.Usage)
	helpList("OPTIONS", optionsSection(path, c.vflag))
	if len(spec.children) > 0 {
		helpList("COMMANDS", c.getCommands(spec, c.vflag))
	}
	helpDesc("DESCRIPTION", spec.Long)
}
