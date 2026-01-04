package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
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
	CD   = "cd"
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

		args := make([]string, 0)
		var tmp bytes.Buffer
		inDoubleQuotes := false
		inSingleQuotes := false

		input = strings.TrimSpace(input)
		for i := 0; i < len(input); i++ {
			c := input[i]
			switch c {
			case '\\':
				if !inSingleQuotes && !inDoubleQuotes {
					i++
					tmp.WriteByte(input[i])
				} else if inDoubleQuotes {
					// newline is not escaped as of right now
					// specialChars := []byte{'"', '\\', '$', '`', 'n'}
					specialChars := []byte{'"', '\\', '$', '`'}
					if slices.Contains(specialChars, input[i+1]) {
						i++
						// if input[i] == 'n' {
						// 	tmp.WriteByte('\n')
						// 	continue
						// }
					}
					tmp.WriteByte(input[i])
				} else {
					tmp.WriteByte(c)
				}
				continue
			case '"':
				if inSingleQuotes {
					tmp.WriteByte(c)
					continue
				}
				inDoubleQuotes = !inDoubleQuotes
			case '\'':
				if inDoubleQuotes {
					tmp.WriteByte(c)
					continue
				}
				inSingleQuotes = !inSingleQuotes
			case ' ':
				if !inSingleQuotes && !inDoubleQuotes {
					if tmp.Len() > 0 {
						args = append(args, tmp.String())
						tmp.Reset()
					}
				} else {
					tmp.WriteByte(c)
				}
			case '1':
				if input[i-1] != ' ' {
					tmp.WriteByte(c)
					continue
				}
				if !inDoubleQuotes && !inSingleQuotes {
					if input[i+1] == '>' {
						i++
						tmp.WriteByte('>')
					} else {
						tmp.WriteByte(c)
					}
					continue
				}

			default:
				tmp.WriteByte(c)
			}

			if i == len(input)-1 && tmp.Len() > 0 {
				args = append(args, tmp.String())
				tmp.Reset()
			}
		}

		if tmp.Len() > 0 {
			args = append(args, tmp.String())
			tmp.Reset()
		}

		if len(args) < 1 {
			continue
		}

		// fmt.Println(args)
		cmd := args[0]
		if len(args) >= 2 {
			args = args[1:]
		} else {
			args = []string{}
		}
		// fmt.Println(args)

		var f *os.File
		defer func() {
			if f != nil {
				if err := f.Close(); err != nil {
					fmt.Printf("failed to close %s\n", f.Name())
				}
			}
		}()
		redirecting := false
		if slices.Contains(args, ">") {
			redirecting = true
			roi := 0
			for i := 0; i < len(args); i++ {
				if args[i] == ">" {
					// we need to make the parent dir if it doesn't exist
					destFile := args[i+1]
					err := os.MkdirAll(filepath.Dir(destFile), 0o750)
					if err != nil {
						fmt.Println("failed to create directory: ", err)
					}
					f, err = os.OpenFile(destFile, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0o644)
					roi = i
					if err != nil {
						var pe *os.PathError
						if errors.As(err, &pe) {
							fmt.Printf("failed to open or create file: %s\n", pe.Path)
							continue
						}
					}
					break
				}
			}

			args = args[:roi]
		}

		var out io.Writer
		out = os.Stdout
		if redirecting {
			out = f
		}

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
			fmt.Fprintln(out, buf.String())
			continue

		case TYPE:
			for _, arg := range args {
				if slices.Contains(builtins, arg) {
					fmt.Fprintf(out, "%s is a shell builtin\n", arg)
					continue
				}

				p, found := searchInPATH(arg)
				if !found {
					fmt.Fprintf(out, "%s: not found\n", arg)
				} else {
					fmt.Fprintf(out, "%s is %s\n", arg, p)
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

		case CD:
			if len(args) > 1 {
				fmt.Fprintln(out, "cd: can't accept more than one argument")
				continue
			}
			dest := args[0]
			if dest == "~" {
				dest = os.Getenv("HOME")
			}
			err := os.Chdir(dest)
			if err != nil {
				var pe *os.PathError
				if errors.As(err, &pe) {
					fmt.Printf("cd: %s: No such file or directory\n", pe.Path)
					continue
				}
				fmt.Println("cd: ", err)
				continue
			}

		default:
			// fmt.Println("hit here")
			p, found := searchInPATH(cmd)
			if !found {
				fmt.Printf("%s: command not found\n", cmd)
				continue
			}

			// fmt.Println("debug: ", cmd, args)
			c := exec.Command(path.Base(p), args...)
			// fmt.Printf("debug: %+v\n", c.Args)
			c.Stdin = os.Stdin
			c.Stdout = out
			c.Stderr = os.Stderr
			if err := c.Run(); err != nil {
				// fmt.Println(err)
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
		// found, err := handlePath(target, p)
		// if err != nil {
		// 	fmt.Printf("failed to handle path: %s", err))
		// 	continue
		// }
		//
		// if found {
		// 	return path.Join(p, target), true
		// }
		targetPath := filepath.Join(p, target)
		info, err := os.Stat(targetPath)
		if err != nil {
			continue
		}
		if info.Mode().IsRegular() && info.Mode()&0o111 != 0 {
			return targetPath, true
		}
	}

	return "", false
}

func handlePath(outBuf *bytes.Buffer, target string, path string) (bool, error) {
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

			found, err := handlePath(outBuf, target, absPath)
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

		if target == entry.Name() {
			return true, nil
		}
	}

	return false, nil
}

// if c is 1 && not in quotes then
//
//	peek next char
//	if next char is > then
//		redirect stdout
//
// if c is > && not in quotes then
//
//	redirect stdout
