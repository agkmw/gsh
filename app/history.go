package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type historyStore struct {
	file           *os.File
	mutex          *sync.RWMutex
	inmemoryStore  []string // in-memory store
	cursor         int
	currentInput   string // this is current command that hasn't been entered yet
	pendingEntries []string
}

func newHistoryStore() (*historyStore, error) {
	historyStore := historyStore{
		mutex:         &sync.RWMutex{},
		cursor:        1, // len(slice) - offset
		inmemoryStore: make([]string, 0),
	}

	s := os.Getenv("HISTFILE")
	if s != "" {
		histFile, err := openHistoryFile(s)
		if err != nil {
			return nil, fmt.Errorf("unable to load history file: %w", err)
		}

		historyStore.file = histFile
		historyStore.loadFromFile(os.Stderr)
	}

	return &historyStore, nil
}

func (h *historyStore) close() {
	if h.file != nil {
		if err := h.file.Close(); err != nil {
			fmt.Fprintln(os.Stderr, "failed to close history file: ", err)
		}
	}
}

func (h *historyStore) loadFromFile(errOut io.Writer) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	b, err := io.ReadAll(h.file)
	if err != nil {
		fmt.Fprintln(errOut, "failed to sync history: ", err)
		return
	}

	lines := strings.Split(string(b), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		h.inmemoryStore = append(h.inmemoryStore, string(line))
	}
}

func (h *historyStore) loadFromPath(stderr io.Writer, name string) {
	f, err := openHistoryFile(name)
	if err != nil {
		fmt.Fprintf(stderr, "failed to read from history from %s", name)
		return
	}

	h.file = f
	h.loadFromFile(stderr)
}

func (h *historyStore) writeToPath(errOut io.Writer, name string) {
	f, err := openHistoryFile(name)
	if err != nil {
		fmt.Fprintln(errOut, "failed to open or create history file to write: ", err)
		return
	}

	for _, line := range h.inmemoryStore {
		f.WriteString(line + "\n")
	}
}

func (h *historyStore) appendToPath(stderr io.Writer, name string) {
	f, err := openHistoryFile(name)
	if err != nil {
		fmt.Fprintln(stderr, "failed to open or create history file to append: ", err)
		return
	}

	for _, line := range h.pendingEntries {
		f.WriteString(line + "\n")
	}

	h.pendingEntries = make([]string, 0)
}

func (h *historyStore) append(errOut io.Writer, cmd string) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	h.inmemoryStore = append(h.inmemoryStore, cmd)
	h.pendingEntries = append(h.pendingEntries, cmd)

	if h.file != nil {
		_, err := h.file.WriteString(cmd + "\n")
		if err != nil {
			errOut.Write([]byte(err.Error()))
		}
	}
}

func (h *historyStore) entries() []string {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	return h.inmemoryStore
}

func (h *historyStore) previous() string {
	if len(h.inmemoryStore) == 0 {
		return ""
	}

	if h.cursor == 0 || len(h.inmemoryStore)-h.cursor < 0 {
		h.cursor = 0
		return ""
	}

	s := h.inmemoryStore[len(h.inmemoryStore)-h.cursor]
	h.cursor++
	return s
}

func (h *historyStore) next() string {
	if len(h.inmemoryStore) == 0 {
		return ""
	}

	if h.cursor == 0 || len(h.inmemoryStore)-h.cursor < 0 {
		h.cursor = 0
		return h.currentInput
	}

	// we should decrement the offset 2 times cuz in PrevCmd we increment
	// the offset ahead after getting the cmd
	h.cursor--
	h.cursor--
	s := h.inmemoryStore[len(h.inmemoryStore)-h.cursor]
	return s
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
