package main

import (
	"os"
	"path"
	"strings"

	"github.com/chzyer/readline"
)

func startInputReader(historyStore *historyStore) *readline.Instance {
	completer := readline.NewPrefixCompleter(
		readline.PcItem(exitCmd),
		readline.PcItem(echoCmd),
		readline.PcItem(typeCmd),
		readline.PcItem(pwdCmd),
		readline.PcItem(cdCmd),
		readline.PcItem(historyCmd),
	)

	knownCommands := make(map[string]struct{})
	for _, b := range builtins {
		knownCommands[b] = struct{}{}
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
				knownCommands[e.Name()] = struct{}{}
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

	l.CaptureExitSignal()

	bellCompleter.inputReader = l

	l.Config.FuncFilterInputRune = func(r rune) (rune, bool) {
		switch r {
		// p = 16; Ctrl+p moves up;
		case 16:
			l.Operation.SetBuffer(historyStore.previous())
			return 0, false
		// n = 14; Ctrl+n moves down;
		case 14:
			l.Operation.SetBuffer(historyStore.next())
			return 0, false
		default:
			historyStore.currentInput += string(r)
		}
		return r, true
	}

	return l
}
