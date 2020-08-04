package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/brimsec/zq/zql"
	"github.com/peterh/liner"
)

var target = "start"

func runGo(line string) error {
	got, err := zql.Parse("", []byte(line), zql.Entrypoint(target))
	if err != nil {
		return err
	}

	js, _ := json.MarshalIndent(got, "", "    ")
	fmt.Printf("Go Result:\n%s\n", js)
	return nil
}

func runJs(line string) error {
	cmd := exec.Command("node", "./main/main.js", "-e", target)
	cmd.Stdin = strings.NewReader(line)
	out, err := cmd.Output()
	if err != nil {
		return err
	}

	fmt.Printf("Js Result:\n%s", out)
	return nil
}

const targetCmd = "_target "

func parse(line string) {
	if strings.HasPrefix(line, targetCmd) {
		target = line[len(targetCmd):]
		return
	}

	if err := runGo(line); err != nil {
		fmt.Println("go error:", err)
	}
	if err := runJs(line); err != nil {
		fmt.Println("js error:", err)
	}
}

func iteractive() {
	rl := liner.NewLiner()
	defer rl.Close()
	for {
		line, err := rl.Prompt("> ")
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}
		rl.AppendHistory(line)
		parse(line)
	}
}

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		iteractive()
		return
	}
	parse(args[0])
}
