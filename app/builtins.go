package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"strconv"
)

const (
	EXIT    = "exit"
	ECHO    = "echo"
	TYPE    = "type"
	PWD     = "pwd"
	CD      = "cd"
	HISTORY = "history"
)

var builtins = []string{EXIT, ECHO, TYPE, PWD, CD, HISTORY}

func echoCommand(stdout io.Writer, args []string) {
	var buf bytes.Buffer
	for i, arg := range args {
		buf.WriteString(arg)
		if i != len(args)-1 {
			buf.WriteString(" ")
		}
	}
	fmt.Fprintln(stdout, buf.String())
}

func pwdCommand(stdout, stderr io.Writer) {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return
	}
	fmt.Fprintln(stdout, cwd)
}

func typeCommand(stdout, stderr io.Writer, args []string) {
	for _, arg := range args {
		if slices.Contains(builtins, arg) {
			fmt.Fprintf(stdout, "%s is a shell builtin\n", arg)
			continue
		}

		p, found := findExecutable(arg)
		if !found {
			fmt.Fprintf(stderr, "%s: not found\n", arg)
			continue
		}
		fmt.Fprintf(stderr, "%s is %s\n", arg, p)
	}
}

func cdCommand(stderr io.Writer, args []string) {
	if len(args) > 1 {
		fmt.Fprintln(stderr, "cd: can't accept more than one argument")
		return
	}

	dest := args[0]
	if dest == "~" {
		dest = os.Getenv("HOME")
	}

	err := os.Chdir(dest)
	if err != nil {
		var pe *os.PathError
		if errors.As(err, &pe) {
			fmt.Fprintf(stderr, "cd: %s: No such file or directory\n", pe.Path)
			return
		}
		fmt.Fprintln(stderr, "cd: ", err)
	}
}

func historyCommand(out, errOut io.Writer, args []string, historyStore *historyStore) {
	entries := historyStore.entries()

	if len(args) == 0 {
		for i, entry := range entries {
			if entry == "" {
				continue
			}
			fmt.Fprintf(out, "    %d  %s\n", i+1, entry)
		}
		return
	}

	switch args[0] {
	case "-r":
		if len(args) != 2 {
			fmt.Fprintln(errOut, "history: usage: -r <path_to_history_file>")
			return
		}

		historyStore.loadFromPath(errOut, args[1])
		return

	case "-w":
		if len(args) != 2 {
			fmt.Fprintln(errOut, "history: usage: -w <path_to_history_file>")
			return
		}

		historyStore.writeToPath(errOut, args[1])
		return

	case "-a":
		if len(args) != 2 {
			fmt.Fprintln(errOut, "history: usage: -a <path_to_history_file>")
			return
		}

		historyStore.appendToPath(errOut, args[1])
		return

	default:
		historyLimit, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			fmt.Fprintf(errOut, "history: invalid argument")
			return
		}

		startIndex := len(entries) - int(historyLimit)
		for i := startIndex; i < len(entries); i++ {
			entry := entries[i]
			if entry == "" {
				continue
			}
			fmt.Fprintf(out, "    %d  %s\n", i+1, entry)
		}
	}
}
