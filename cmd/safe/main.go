package main

import (
	"fmt"
	"os"
)

var (
	args  []string
	flags map[string]string
)

func main() {
	args, flags = help.ParseFlags()
	if args[0] == "help" {
		if len(args) < 2 {
			help.Doc()
			return
		}
		help.Command(os.Args[2])
		return
	}
	exec, ok := help.Commands[args[0]]
	if !ok {
		fmt.Printf("unkown command: %v\n", args[0])
		help.Doc()
	}
	exec.Execute(help)
}
