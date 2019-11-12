package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/mccanne/zq/zql"
	"github.com/peterh/liner"
)

func runGo(line string) error {
	got, err := zql.Parse("", []byte(line))
	if err != nil {
		return err
	}

	fmt.Printf("Go raw result:\n%s\n", got)

	js, _ := json.MarshalIndent(got, "", "    ")
	fmt.Printf("Go Result after json.Marshal:\n%s\n", js)
	return nil
}

func runJs(line string) error {
	cmd := exec.Command("node", "./main/main.js")
	cmd.Stdin = strings.NewReader(line)
	out, err := cmd.Output()
	if err != nil {
		return err
	}

	fmt.Printf("Js Result:\n%s", out)
	return nil
}

func parse(line string) {
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
