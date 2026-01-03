package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
)

const (
	EXIT = "exit"
	ECHO = "echo"
	TYPE = "type"
)

var builtins = []string{EXIT, ECHO, TYPE}

func main() {
	for {
		fmt.Print("$ ")

		r := bufio.NewReader(os.Stdin)
		input, err := r.ReadString('\n')
		if err != nil {
			fmt.Println(err)
		}

		fields := strings.Fields(input)

		cmd := fields[0]
		args := fields[1:]

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

				found := false
				rawPath := os.Getenv("PATH")
				paths := strings.Split(rawPath, ":")
				for _, p := range paths {
					// fmt.Println("debug: path: ", p)
					f, err := handlePath(arg, p)
					if err != nil {
						fmt.Printf("failed to handle path: %s", err)
						continue
					}

					if f {
						fmt.Printf("%s is %s\n", arg, path.Join(p, arg))
						found = true
						break
					}
				}

				if !found {
					fmt.Printf("%s: not found\n", arg)
				}

				continue
			}
			continue

		default:
			fmt.Printf("%s: command not found\n", cmd)
		}
	}
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
