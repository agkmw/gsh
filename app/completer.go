package main

import (
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/chzyer/readline"
)

type BellAutoCompleter struct {
	prefixCompleter   readline.AutoCompleter
	stdout            io.Writer
	inputReader       *readline.Instance
	awaitingSecondTab bool
	previousLine      string
}

func (c *BellAutoCompleter) Do(line []rune, pos int) (newLine [][]rune, length int) {
	newLine, length = c.prefixCompleter.Do(line, pos)

	var suggestions []string
	for _, word := range newLine {
		s := string(line) + strings.TrimSpace(string(word))
		suggestions = append(suggestions, s)
	}
	slices.Sort(suggestions)

	// no matches
	if len(newLine) == 0 {
		c.stdout.Write([]byte("\x07"))
		c.awaitingSecondTab = false
		return nil, 0
	}

	// single match
	if len(newLine) == 1 {
		c.awaitingSecondTab = false
		return newLine, length
	}

	// multiple matches
	// first tab
	if !c.awaitingSecondTab {
		c.stdout.Write([]byte("\x07"))
		if c.previousLine != string(line) {
			prefix := newLine[0]
			for _, word := range newLine {
				for !strings.HasPrefix(string(word), string(prefix)) {
					prefix = prefix[:len(prefix)-1]
				}
			}
			if len(prefix) == 0 {
				goto nocommonprefix
			}
			return [][]rune{prefix}, 0
		}
	nocommonprefix:
		c.awaitingSecondTab = true
		c.inputReader.Refresh()
		return nil, 0
	}

	fmt.Println()
	fmt.Println(strings.Join(suggestions, "  "))

	c.awaitingSecondTab = false
	c.inputReader.Refresh()
	return nil, 0
}
