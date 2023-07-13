package util

import (
	"fmt"
	"os"
	"strings"
)

type CommandHelp struct {
	Usage       string
	Short       string
	Description string
	Execute     func(Help)
}

func (c CommandHelp) String() string {
	if c.Description == "" {
		return fmt.Sprintf("Usage:\n\n    %v\n\n", c.Usage)
	}
	return fmt.Sprintf("Usage:\n\n    %v\n\n%v", c.Usage, c.Description)
}

type Help struct {
	Executable string
	Short      string
	Commands   map[string]CommandHelp
	Flags      map[string]string
}

func (h Help) ParseFlags() ([]string, map[string]string) {
	args := make([]string, 0)
	flags := make(map[string]string)
	if len(os.Args) == 1 {
		h.Doc()
		os.Exit(0)
		return nil, nil
	}
	for n := 1; n < len(os.Args); n++ {
		arg := os.Args[n]
		if strip, ok := strings.CutPrefix(arg, "--"); ok {
			subargs := strings.Split(strip, "=")
			if len(subargs) == 1 {
				if value, ok := h.Flags[subargs[0]]; !ok || value != "" {
					fmt.Println("invalid flag")
					h.Doc()
				} else {
					flags[subargs[0]] = ""
				}
			} else if len(subargs) == 2 {
				if value, ok := h.Flags[subargs[0]]; !ok || value == "" {
					fmt.Println("invalid flag")
					h.Doc()
				} else {
					flags[subargs[0]] = subargs[1]
				}
			} else {
				fmt.Println("invalid flag")
				h.Doc()
			}
		} else {
			args = append(args, arg)
		}
	}
	return args, flags
}

func (h Help) Command(name string) {
	help, ok := h.Commands[name]
	if !ok {
		fmt.Printf("%v help %v: unknown help topic. Run '%v help'.\n", h.Executable, name, h.Executable)
		return
	}
	fmt.Printf("%s\n", help)
}

func (h Help) Doc() {
	var head string
	if len(h.Flags) > 0 {
		flags := make([]string, 0)
		for flag, variable := range h.Flags {
			var flagDoc string
			if variable == "" {
				flagDoc = fmt.Sprintf("[--%s]", flag)
			} else {
				flagDoc = fmt.Sprintf("[--%s=<%s>]", flag, variable)
			}
			flags = append(flags, flagDoc)
		}
		flagDoc := strings.Join(flags, " ")
		head = fmt.Sprintf("%v %v\n\nUsage:\n\n    %v %v <command> [arguments]\n\nThe commands are:", h.Executable, h.Short, h.Executable, flagDoc)
	} else {
		head = fmt.Sprintf("%v %v\n\nUsage:\n\n    %v <command> [arguments]\n\nThe commands are:", h.Executable, h.Short, h.Executable)
	}
	commandlist := make([]string, 0)
	largestCommand := 0
	//largestShort := 0
	for command, _ := range h.Commands {
		if len(command) >= largestCommand {
			largestCommand = len(command)
		}
	}

	for command, help := range h.Commands {
		template := fmt.Sprintf("     %%-%vs    %%s", largestCommand)
		commandlist = append(commandlist, fmt.Sprintf(template, command, help.Short))
	}
	commands := strings.Join(commandlist, "\n")
	tail := fmt.Sprintf(`Use "%v help <command> for more information about a command.`, h.Executable)
	fmt.Printf("%s\n\n%s\n\n%s\n", head, commands, tail)
	os.Exit(0)
}
