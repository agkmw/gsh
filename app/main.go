package main

import (
	"fmt"
	"io"
	"os"
	"slices"

	"github.com/chzyer/readline"
)

func main() {
	historyStore, err := newHistoryStore()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	if historyStore != nil {
		defer historyStore.close()
	}

	reader := startInputReader(historyStore)
	defer reader.Close()

	for {
		line, err := reader.Readline()
		if err == readline.ErrInterrupt {
			continue
		} else if err == io.EOF {
			break
		}

		historyStore.append(os.Stderr, line)

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
		} else {
			runPipeline(tokens, historyStore)
		}

		// reset history pointer
		historyStore.reverseOffset = 1
	}
}
