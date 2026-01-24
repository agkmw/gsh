package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
)

func runExternalCommand(stdin *os.File, stdout, stderr io.Writer, command string, args []string) {
	exe, found := findExecutable(command)
	if !found {
		fmt.Fprintf(stderr, "%s: command not found\n", command)
		return
	}

	c := exec.Command(path.Base(exe), args...)
	c.Env = os.Environ()
	c.Stdin = stdin
	c.Stdout = stdout
	c.Stderr = stderr
	c.Start()
	c.Wait()
}

func findExecutable(command string) (string, bool) {
	paths := strings.Split(os.Getenv("PATH"), ":")
	for _, p := range paths {
		fullPath := filepath.Join(p, command)
		info, err := os.Stat(fullPath)
		if err != nil {
			continue
		}
		if info.Mode().IsRegular() && info.Mode()&0o111 != 0 {
			return fullPath, true
		}
	}

	return "", false
}
