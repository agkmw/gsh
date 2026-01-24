package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"slices"
	"strconv"
)

func echo(out io.Writer, args []string) {
	var buf bytes.Buffer
	for i, arg := range args {
		buf.WriteString(arg)
		if i != len(args)-1 {
			buf.WriteString(" ")
		}
	}
	fmt.Fprintln(out, buf.String())
}

func pwd(out, errout io.Writer) {
	dir, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(errout, err)
		return
	}
	fmt.Fprintln(out, dir)
}

func typeCommand(out, errout io.Writer, args []string) {
	for _, arg := range args {
		if slices.Contains(builtins, arg) {
			fmt.Fprintf(out, "%s is a shell builtin\n", arg)
			continue
		}

		p, found := findExecutable(arg)
		if !found {
			fmt.Fprintf(errout, "%s: not found\n", arg)
			continue
		}
		fmt.Fprintf(errout, "%s is %s\n", arg, p)
	}
}

func cd(errout io.Writer, args []string) {
	if len(args) > 1 {
		fmt.Fprintln(errout, "cd: can't accept more than one argument")
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
			fmt.Fprintf(errout, "cd: %s: No such file or directory\n", pe.Path)
			return
		}
		fmt.Fprintln(errout, "cd: ", err)
	}
}

func runExternalCommand(stdin *os.File, stdout, stderr io.Writer, command string, args []string) {
	p, found := findExecutable(command)
	if !found {
		fmt.Fprintf(stderr, "%s: command not found\n", command)
		return
	}

	c := exec.Command(path.Base(p), args...)
	c.Env = os.Environ()
	c.Stdin = stdin
	c.Stdout = stdout
	c.Stderr = stderr
	c.Start()
	c.Wait()
}

func history(out, errOut io.Writer, args []string, hist *HistoryStore) {
	lines := hist.Entries()

	if len(args) == 0 {
		for i, line := range lines {
			if line == "" {
				continue
			}
			fmt.Fprintf(out, "    %d  %s\n", i+1, line)
		}
		return
	}

	if args[0] == "-r" {
		if len(args) != 2 {
			fmt.Fprintln(errOut, "history: usage: -r <path_to_history_file>")
			return
		}

		hist.LoadFromPath(errOut, args[1])
		return
	}

	if args[0] == "-w" {
		if len(args) != 2 {
			fmt.Fprintln(errOut, "history: usage: -w <path_to_history_file>")
			return
		}

		hist.WriteToPath(errOut, args[1])
		return
	}

	if args[0] == "-a" {
		if len(args) != 2 {
			fmt.Fprintln(errOut, "history: usage: -a <path_to_history_file>")
			return
		}

		hist.AppendToPath(errOut, args[1])
		return
	}

	c, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		fmt.Fprintf(errOut, "history: invalid argument")
		return
	}

	target := int(c)
	start := len(lines) - target

	for i := start; i < len(lines); i++ {
		line := lines[i]
		if line == "" {
			continue
		}
		fmt.Fprintf(out, "    %d  %s\n", i+1, line)
	}
}
