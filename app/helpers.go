package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

func tokenizeInput(input string) []string {
	tokens := make([]string, 0)
	var tokenBuffer bytes.Buffer
	inDoubleQuote := false
	inSingleQuote := false

	input = strings.TrimSpace(input)
	for i := 0; i < len(input); i++ {
		c := input[i]
		switch c {
		case '\\':
			if !inSingleQuote && !inDoubleQuote {
				i++
				tokenBuffer.WriteByte(input[i])
			} else if inDoubleQuote {
				// specialChars := []byte{'"', '\\', '$', '`', 'n'}
				specialChars := []byte{'"', '\\', '$', '`'}
				if slices.Contains(specialChars, input[i+1]) {
					i++
					// if input[i] == 'n' {
					// 	tmp.WriteByte('\n')
					// 	continue
					// }
				}
				tokenBuffer.WriteByte(input[i])
			} else {
				tokenBuffer.WriteByte(c)
			}
			continue

		case '"':
			if inSingleQuote {
				tokenBuffer.WriteByte(c)
				continue
			}
			inDoubleQuote = !inDoubleQuote

		case '\'':
			if inDoubleQuote {
				tokenBuffer.WriteByte(c)
				continue
			}
			inSingleQuote = !inSingleQuote

		case ' ':
			if !inSingleQuote && !inDoubleQuote {
				if tokenBuffer.Len() > 0 {
					tokens = append(tokens, tokenBuffer.String())
					tokenBuffer.Reset()
				}
			} else {
				tokenBuffer.WriteByte(c)
			}

		default:
			tokenBuffer.WriteByte(c)
		}

		if i == len(input)-1 && tokenBuffer.Len() > 0 {
			tokens = append(tokens, tokenBuffer.String())
			tokenBuffer.Reset()
		}
	}

	if tokenBuffer.Len() > 0 {
		tokens = append(tokens, tokenBuffer.String())
		tokenBuffer.Reset()
	}

	return tokens
}

func splitPipeline(input []string) [][]string {
	cmds := make([][]string, 0)
	previousIndex := 0

	for i, item := range input {
		if item == "|" {
			proc := input[previousIndex:i]
			previousIndex = i + 1
			cmds = append(cmds, proc)
		}
	}

	cmds = append(cmds, input[previousIndex:])

	return cmds
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

func setupRedirection(
	defaultStdout,
	defaultStderr *os.File,
	tokens []string,
) (stdout, stderr io.Writer, remainingArgs []string, err error) {
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
		remainingArgs = tokens[:redirectionIndex]
	} else {
		remainingArgs = tokens[:]
	}

	stdout = defaultStdout
	if redirectStdout {
		stdout = stdoutFile
	}

	stderr = defaultStderr
	if redirectStderr {
		stderr = stderrFile
	}

	return stdout, stderr, remainingArgs, nil
}

func closeRedirection(outWriter, errWriter io.Writer) {
	if f, ok := outWriter.(*os.File); ok {
		if f != nil && f != os.Stdout {
			if err := f.Close(); err != nil {
				fmt.Printf("failed to close %s\n", f.Name())
			}
		} else if f == os.Stdout {
			outWriter = os.Stdout
		}
	}

	if f, ok := errWriter.(*os.File); ok {
		if f != nil && f != os.Stderr {
			if err := f.Close(); err != nil {
				fmt.Printf("failed to close %s\n", f.Name())
			}
		} else if f == os.Stderr {
			errWriter = os.Stderr
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

func openHistoryFile(name string) (*os.File, error) {
	err := os.MkdirAll(filepath.Dir(name), 0o750)
	if err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	f, err := os.OpenFile(name, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0o644)
	if err != nil {
		var pe *os.PathError
		if errors.As(err, &pe) {
			return nil, fmt.Errorf("failed to open or create file: %s\n", pe.Path)
		}
		return nil, err
	}

	return f, nil
}
