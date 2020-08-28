// mockbrim is a command for testing purposes only. It is designed to simulate
// the exact way brim launches then forks a separate zqd process. zqd must be
// in $PATH for this to work.

package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
)

func die(err error) {
	if err != nil {
		panic(err)
	}
}

var (
	pidfile  string
	portfile string
	zqddata  string
)

func init() {
	flag.StringVar(&portfile, "portfile", "", "location to write zqd port")
	flag.StringVar(&pidfile, "pidfile", "", "location to write zqd pid")
	flag.StringVar(&zqddata, "zqddata", "", "location to write zqd data")
	flag.Parse()
}

func main() {
	r, _, err := os.Pipe()
	die(err)

	if portfile == "" {
		fmt.Fprintln(os.Stderr, "must provide -portfile arg")
		os.Exit(1)
	}
	if pidfile == "" {
		fmt.Fprintln(os.Stderr, "must provide -pidfile arg")
		os.Exit(1)
	}
	args := []string{
		"listen",
		"-l=localhost:0",
		"-loglevel=warn",
		"-portfile=" + portfile,
		"-data=" + zqddata,
		fmt.Sprintf("-brimfd=%d", r.Fd()),
	}
	stderr := bytes.NewBuffer(nil)
	cmd := exec.Command("zqd", args...)
	cmd.Stderr = stderr
	cmd.ExtraFiles = []*os.File{r}

	err = cmd.Start()
	die(err)
	pid := fmt.Sprintf("%d", cmd.Process.Pid)
	err = ioutil.WriteFile(pidfile, []byte(pid), 0644)
	die(err)
	cmd.Wait()
}
