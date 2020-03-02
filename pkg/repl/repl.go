// Package repl is a simple read-eval-print loop.  It calls the Consumer
// to do all the eval work.
package repl

import (
	"io"

	"github.com/peterh/liner"
)

type Consumer interface {
	Consume(line string) bool
	Prompt() string
}

type REPL struct {
	consumer Consumer
}

func NewREPL(c Consumer) *REPL {
	return &REPL{consumer: c}
}

// Run executes the REPL.
func (r *REPL) Run() error {
	l := liner.NewLiner()
	defer l.Close() //nolint:errcheck
	l.SetMultiLineMode(true)
	for {
		line, e := l.Prompt(r.consumer.Prompt())
		if e == io.EOF {
			return io.EOF
		} else if e != nil {
			// Ignore this error; the prior is more interesting.
			return e
		}
		done := r.consumer.Consume(line)
		if done {
			return nil
		}
		l.AppendHistory(line)
	}
}

func Ask(prompt string) (string, error) {
	l := liner.NewLiner()
	defer l.Close() //nolint:errcheck
	answer, err := l.Prompt(prompt)
	if err != nil {
		return "", err
	}
	return answer, nil
}

func AskForPassword() (string, error) {
	l := liner.NewLiner()
	defer l.Close() //nolint:errcheck
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
