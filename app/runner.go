package main

import (
	"fmt"
	"os"
	"sync"
)

type process struct {
	stdin  *os.File
	stdout *os.File
	stderr *os.File
	args   []string // this is a combination of cmd and its args;
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

func buildPipeline(commands [][]string) []process {
	processes := make([]process, 0)

	var pipeErr error
	var nextStdin *os.File

	for i, segment := range commands {
		var stdin, stdout, stderr *os.File
		stdin = nextStdin

		if len(commands)-1 == i {
			stdout = os.Stdout
			stderr = os.Stderr
		} else {
			if i == 0 {
				stdin = os.Stdin
			}
			r, w, err := os.Pipe()
			pipeErr = err
			nextStdin = r
			stdout = w
			stderr = w
		}

		process := process{
			args:   segment,
			stdin:  stdin,
			stdout: stdout,
			stderr: stderr,
		}

		processes = append(processes, process)
	}

	if pipeErr != nil {
		fmt.Println(pipeErr)
		return nil
	}

	return processes
}

func runPipeline(tokens []string, hist *historyStore) {
	commandGroups := splitPipeline(tokens)

	processes := buildPipeline(commandGroups)

	var wg sync.WaitGroup
	for i, proc := range processes {
		wg.Add(1)
		go func(proc process) {
			defer wg.Done()
			cmd := proc.args[0]
			if len(proc.args) >= 2 {
				proc.args = proc.args[1:]
			} else {
				proc.args = []string{}
			}

			// these are redirected writers; so piped writer must be copied from these
			stdout, stderr, commandArgs, err := setupRedirection(proc.stdout, proc.stderr, proc.args)
			if err != nil {
				fmt.Println("failed to prepare writers: ", err)
				return
			}

			switch cmd {
			case EXIT:
				if i == len(processes)-1 {
					os.Exit(0)
				}
				return
			case ECHO:
				echoCommand(stdout, commandArgs)
			case TYPE:
				typeCommand(stdout, stderr, commandArgs)
			case PWD:
				pwdCommand(stdout, stderr)
			case CD:
				cdCommand(stderr, commandArgs)
			case HISTORY:
				historyCommand(stdout, stderr, commandArgs, hist)
			default:
				runExternalCommand(proc.stdin, stdout, stderr, cmd, commandArgs)
			}

			if stdout != proc.stdout {
				if f, ok := stdout.(*os.File); ok {
					f.WriteTo(proc.stdout)
					f.Close()
				}
			}

			if stderr != proc.stderr {
				if f, ok := stderr.(*os.File); ok {
					f.WriteTo(proc.stderr)
					f.Close()
				}
			}

			if proc.stdout != os.Stdout {
				proc.stdout.Close()
			}

			if proc.stderr != os.Stderr {
				proc.stderr.Close()
			}
		}(proc)
	}
	wg.Wait()
}

func runSingleCommand(tokens []string, historyStore *historyStore) {
	command := tokens[0]
	if len(tokens) >= 2 {
		tokens = tokens[1:]
	} else {
		tokens = []string{}
	}

	stdout, stderr, commandArgs, err := setupRedirection(os.Stdout, os.Stderr, tokens)
	if err != nil {
		fmt.Println("failed to prepare writers: ", err)
		return
	}

	switch command {
	case EXIT:
		os.Exit(0)
	case ECHO:
		echoCommand(stdout, commandArgs)
	case TYPE:
		typeCommand(stdout, stderr, commandArgs)
	case PWD:
		pwdCommand(stdout, stderr)
	case CD:
		cdCommand(stderr, commandArgs)
	case HISTORY:
		historyCommand(stdout, stderr, commandArgs, historyStore)
	default:
		runExternalCommand(os.Stdin, stdout, stderr, command, commandArgs)
	}

	closeRedirection(stdout, stderr)
}
