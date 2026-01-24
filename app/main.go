package main

import (
	"fmt"
	"io"
	"os"
	"path"
	"slices"
	"strings"
	"sync"

	"github.com/chzyer/readline"
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

type Process struct {
	Stdin  *os.File
	Stdout *os.File
	Stderr *os.File
	Args   []string // this is a combination of cmd and its args;
}

func main() {
	historyStore, err := NewHistoryStore()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	defer historyStore.Close()

	knownCommands := make(map[string]bool)
	for _, b := range builtins {
		knownCommands[b] = true
	}
	pathDirs := strings.Split(os.Getenv("PATH"), ":")
	for _, p := range pathDirs {
		dirEntries, err := os.ReadDir(p)
		if err != nil {
			continue
		}
		for _, e := range dirEntries {
			if _, ok := knownCommands[e.Name()]; ok {
				continue
			}
			info, err := os.Stat(path.Join(p, e.Name()))
			if err != nil {
				continue
			}
			if info.Mode().IsRegular() && info.Mode()&0o111 != 0 {
				knownCommands[e.Name()] = true
				completer.SetChildren(append(completer.GetChildren(), readline.PcItem(e.Name())))
			}
		}
	}

	bellCompleter := BellAutoCompleter{
		prefixCompleter: completer,
		stdout:          os.Stdout,
	}

	cfg := readline.Config{
		Prompt:          "$ ",
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
		AutoComplete:    &bellCompleter,
	}

	l, err := readline.NewEx(&cfg)
	if err != nil {
		panic(err)
	}
	defer l.Close()

	l.CaptureExitSignal()

	bellCompleter.inputReader = l

	l.Config.FuncFilterInputRune = func(r rune) (rune, bool) {
		switch r {
		// p = 16; Ctrl+p moves up;
		case 16:
			l.Operation.SetBuffer(historyStore.Previous())
			return 0, false
		// n = 14; Ctrl+n moves down;
		case 14:
			l.Operation.SetBuffer(historyStore.Next())
			return 0, false
		default:
			historyStore.currentInput += string(r)
		}
		return r, true
	}

	for {
		line, err := l.Readline()
		if err == readline.ErrInterrupt {
			continue
		} else if err == io.EOF {
			break
		}

		historyStore.Append(os.Stderr, line)

		tokens := tokenizeInput(line)
		if len(tokens) < 1 {
			continue
		}

		isSingleCommand := true
		if slices.Contains(tokens, "|") {
			isSingleCommand = false
		}

		if isSingleCommand {
			runSingleCommand(tokens, historyStore)
			continue
		}

		runPipeline(tokens, historyStore)

		// reset history pointer
		historyStore.cursor = 1
	}
}

func buildPipeline(commands [][]string) []Process {
	processes := make([]Process, 0)

	var pipeErr error
	var nextStdin *os.File

	for i, segment := range commands {
		var stdin, stdout, stderr *os.File
		stdin = nextStdin

		if len(commands)-1 == i {
			stdout = os.Stdout
			stderr = os.Stderr
		} else if i == 0 {
			stdin = os.Stdin
			r, w, err := os.Pipe()
			pipeErr = err
			nextStdin = r
			stdout = w
			stderr = w
		} else {
			r, w, err := os.Pipe()
			pipeErr = err
			nextStdin = r
			stdout = w
			stderr = w
		}

		process := Process{
			Args:   segment,
			Stdin:  stdin,
			Stdout: stdout,
			Stderr: stderr,
		}

		processes = append(processes, process)
	}

	if pipeErr != nil {
		fmt.Println(pipeErr)
		return nil
	}

	return processes
}

func runPipeline(tokens []string, hist *HistoryStore) {
	commandGroups := splitPipeline(tokens)

	processes := buildPipeline(commandGroups)

	var wg sync.WaitGroup
	for i, proc := range processes {
		wg.Add(1)
		go func(proc Process) {
			defer wg.Done()
			cmd := proc.Args[0]
			if len(proc.Args) >= 2 {
				proc.Args = proc.Args[1:]
			} else {
				proc.Args = []string{}
			}

			// these are redirected writers; so piped writer must be copied from these
			outWriter, errWriter, parsedInput, err := setupRedirection(proc.Stdout, proc.Stderr, proc.Args)
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
				// return
			case ECHO:
				echo(outWriter, parsedInput)
			case TYPE:
				typeCommand(outWriter, errWriter, parsedInput)
			case PWD:
				pwd(outWriter, errWriter)
			case CD:
				cd(errWriter, parsedInput)
			case HISTORY:
				history(outWriter, errWriter, parsedInput, hist)
			default:
				runExternalCommand(proc.Stdin, outWriter, errWriter, cmd, parsedInput)
			}

			if outWriter != proc.Stdout {
				if f, ok := outWriter.(*os.File); ok {
					f.WriteTo(proc.Stdout)
					f.Close()
				}
			}

			if errWriter != proc.Stderr {
				if f, ok := errWriter.(*os.File); ok {
					f.WriteTo(proc.Stderr)
					f.Close()
				}
			}

			if proc.Stdout != os.Stdout {
				proc.Stdout.Close()
			}

			if proc.Stderr != os.Stderr {
				proc.Stderr.Close()
			}
		}(proc)
	}
	wg.Wait()
}

func runSingleCommand(parsedInput []string, hist *HistoryStore) {
	cmd := parsedInput[0]
	if len(parsedInput) >= 2 {
		parsedInput = parsedInput[1:]
	} else {
		parsedInput = []string{}
	}

	outWriter, errWriter, parsedInput, err := setupRedirection(os.Stdout, os.Stderr, parsedInput)
	if err != nil {
		fmt.Println("failed to prepare writers: ", err)
		return
	}

	switch cmd {
	case EXIT:
		os.Exit(0)
		// return
	case ECHO:
		echo(outWriter, parsedInput)
	case TYPE:
		typeCommand(outWriter, errWriter, parsedInput)
	case PWD:
		pwd(outWriter, errWriter)
	case CD:
		cd(errWriter, parsedInput)
	case HISTORY:
		history(outWriter, errWriter, parsedInput, hist)
	default:
		runExternalCommand(os.Stdin, outWriter, errWriter, cmd, parsedInput)
	}

	closeRedirection(outWriter, errWriter)
}
