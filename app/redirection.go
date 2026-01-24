package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func setupRedirection(
	defaultStdout,
	defaultStderr *os.File,
	tokens []string,
) (stdout, stderr io.Writer, commandArgs []string, err error) {
	var stderrFile *os.File
	var stdoutFile *os.File

	redirectStdout := false
	redirectStderr := false

	redirectionIndex := 0
	hasRedirection := false
	for i, a := range tokens {
		if hasRedirection {
			break
		}

		switch a {
		case ">", "1>":
			stdoutFile, err = openRedirectionFile(tokens[i+1], true)
			if err != nil {
				return nil, nil, nil, fmt.Errorf("failed to prepare writers: %w", err)
			}
			redirectStdout = true
			redirectionIndex = i
			hasRedirection = true

		case "2>":
			stderrFile, err = openRedirectionFile(tokens[i+1], true)
			if err != nil {
				return nil, nil, nil, fmt.Errorf("failed to prepare writers: %w", err)
			}
			redirectStderr = true
			redirectionIndex = i
			hasRedirection = true
		case ">>", "1>>":
			stdoutFile, err = openRedirectionFile(tokens[i+1], false)
			if err != nil {
				return nil, nil, nil, fmt.Errorf("failed to prepare writers: %w", err)
			}
			redirectStdout = true
			redirectionIndex = i
			hasRedirection = true
		case "2>>":
			stderrFile, err = openRedirectionFile(tokens[i+1], false)
			if err != nil {
				return nil, nil, nil, fmt.Errorf("failed to prepare writers: %w", err)
			}
			redirectStderr = true
			redirectionIndex = i
			hasRedirection = true
		}
	}

	if hasRedirection {
		commandArgs = tokens[:redirectionIndex]
	} else {
		commandArgs = tokens[:]
	}

	stdout = defaultStdout
	if redirectStdout {
		stdout = stdoutFile
	}

	stderr = defaultStderr
	if redirectStderr {
		stderr = stderrFile
	}

	return stdout, stderr, commandArgs, nil
}

func closeRedirection(stdout, stderr io.Writer) {
	if f, ok := stdout.(*os.File); ok {
		if f != nil && f != os.Stdout {
			if err := f.Close(); err != nil {
				fmt.Printf("failed to close %s\n", f.Name())
			}
		} else if f == os.Stdout {
			stdout = os.Stdout
		}
	}

	if f, ok := stderr.(*os.File); ok {
		if f != nil && f != os.Stderr {
			if err := f.Close(); err != nil {
				fmt.Printf("failed to close %s\n", f.Name())
			}
		} else if f == os.Stderr {
			stderr = os.Stderr
		}
	}
}

func openRedirectionFile(name string, overwrite bool) (*os.File, error) {
	err := os.MkdirAll(filepath.Dir(name), 0o750)
	if err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	var f *os.File
	if overwrite {
		f, err = os.OpenFile(name, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0o644)
	} else {
		f, err = os.OpenFile(name, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	}

	if err != nil {
		var pe *os.PathError
		if errors.As(err, &pe) {
			return nil, fmt.Errorf("failed to open or create file: %s\n", pe.Path)
		}
		return nil, err
	}

	return f, nil
}
