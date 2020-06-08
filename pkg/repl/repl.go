// Package repl is a simple read-eval-print loop.  It calls the Consumer
// to do all the eval work.
package repl

import (
	"github.com/peterh/liner"
)

type Consumer interface {
	Consume(line string) bool
	Prompt() string
}

// Run executes the REPL.
func Run(c Consumer) error {
	l := liner.NewLiner()
	defer l.Close()
	l.SetMultiLineMode(true)
	for {
		line, err := l.Prompt(c.Prompt())
		if err != nil {
			return err
		}
		if c.Consume(line) {
			return nil
		}
		l.AppendHistory(line)
	}
}

func Ask(prompt string) (string, error) {
	l := liner.NewLiner()
	defer l.Close()
	return l.Prompt(prompt)
}

func AskForPassword() (string, error) {
	l := liner.NewLiner()
	defer l.Close()
	return l.PasswordPrompt("Password:")
}

func AskForUserPass() (string, string, error) {
	user, err := Ask("User: ")
	if err != nil {
		return "", "", err
	}
	password, err := AskForPassword()
	if err != nil {
		return "", "", err
	}
	return user, password, nil
}
