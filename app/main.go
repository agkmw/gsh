package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"slices"
	"strings"
)

const (
	EXIT = "exit"
	ECHO = "echo"
	TYPE = "type"
	PWD  = "pwd"
)

var builtins = []string{EXIT, ECHO, TYPE, PWD}

func main() {
	for {
		fmt.Print("$ ")

		r := bufio.NewReader(os.Stdin)
		input, err := r.ReadString('\n')
		if err != nil {
			fmt.Println(err)
		}

		// fields := strings.Fields(input)
		// cmd := fields[0]
		// args := fields[1:]
		// fmt.Println(args)

		args := make([]string, 0)
		var tmp bytes.Buffer
		inQuotes := false

		input = strings.TrimSpace(input)
		for i, c := range input {
			switch c {
			case '"':
				inQuotes = !inQuotes
			case ' ':
				if !inQuotes {
					if tmp.Len() > 0 {
						args = append(args, tmp.String())
						tmp.Reset()
					}
				} else {
					tmp.WriteRune(c)
				}
			default:
				tmp.WriteRune(c)
			}

			if i == len(input)-1 && tmp.Len() > 0 {
				args = append(args, tmp.String())
			}
		}

		// fmt.Println(args)

		cmd := args[0]
		args = args[1:]

		// fmt.Println(args)

		switch cmd {
		case EXIT:
			return

		case ECHO:
			var buf bytes.Buffer
			for i, arg := range args {
				buf.WriteString(arg)
				if i != len(args)-1 {
					buf.WriteString(" ")
				}
			}
			fmt.Println(buf.String())
			continue

		case TYPE:
			for _, arg := range args {
				if slices.Contains(builtins, arg) {
					fmt.Printf("%s is a shell builtin\n", arg)
					continue
				}

				p, found := searchInPATH(arg)
				if !found {
					fmt.Printf("%s: not found\n", arg)
				} else {
					fmt.Printf("%s is %s\n", arg, p)
				}

				continue
			}

		case PWD:
			dir, err := os.Getwd()
			if err != nil {
				fmt.Println(err)
				continue
			}
			fmt.Println(dir)

		default:
			p, found := searchInPATH(cmd)
			if !found {
				fmt.Printf("%s: command not found\n", cmd)
				continue
			}

			// fmt.Println("debug: ", cmd, args)
			c := exec.Command(path.Base(p), args...)
			// fmt.Printf("debug: %+v\n", c.Args)
			c.Stdin = os.Stdin
			c.Stdout = os.Stdout
			c.Stderr = os.Stderr
			if err := c.Run(); err != nil {
				fmt.Println(err)
				continue
			}
		}
	}
}

func searchInPATH(target string) (string, bool) {
	rawPath := os.Getenv("PATH")
	paths := strings.Split(rawPath, ":")
	for _, p := range paths {
		// fmt.Println("debug: path: ", p)
		found, err := handlePath(target, p)
		if err != nil {
			fmt.Printf("failed to handle path: %s", err)
			continue
		}

		if found {
			return path.Join(p, target), true
		}
	}

	return "", false
}

func handlePath(target string, path string) (bool, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("failed to read dir: %s\n", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			absPath, err := filepath.Abs(entry.Name())
			if err != nil {
				fmt.Printf("failed to get abs path: %s\n", entry.Name())
				continue
			}

			found, err := handlePath(target, absPath)
			if err != nil {
				fmt.Printf("failed to handle path: %s", err)
				continue
			}

			if found {
				return true, nil
			}

			continue
		}

		info, err := entry.Info()
		if err != nil {
			fmt.Printf("failed to get entry info: %s", err)
			continue
		}

		perm := info.Mode().Perm()
		if !strings.Contains(perm.String(), "x") {
			continue
		}

		// fmt.Printf("ENTRY: %s | PERM: %s\n", entry.Name(), perm)

		if target == entry.Name() {
			// fmt.Printf("found %s\n", target)
			return true, nil
		}
	}

	return false, nil
}
