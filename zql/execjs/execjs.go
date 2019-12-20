package execjs

import (
	"encoding/json"
	"os/exec"

	"github.com/mccanne/zq/ast"
)

type Runner string

// ParseProc executes the ZQL query using the javascript parser and returns the
// result as an unpacked ast.Proc.
func (r Runner) ParseProc(line string) (ast.Proc, error) {
	cmd := exec.Command("node", string(r), line)
	data, err := cmd.Output()
	if err != nil {
		var clierr cliError
		if err := json.Unmarshal(err.(*exec.ExitError).Stderr, &clierr); err == nil {
			return nil, clierr
		}
		return nil, err
	}
	return ast.UnpackProc(nil, data)
}

type cliError struct {
	Op      string `json:"op"`
	Message string `json:"error"`
}

func (c cliError) Error() string {
	return c.Message
}
