package main

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
)

type HistoryStore struct {
	file           *os.File
	mutex          *sync.RWMutex
	inmemoryStore  []string // in-memory store
	cursor         int
	currentInput   string // this is current command that hasn't been entered yet
	pendingEntries []string
}

func NewHistoryStore() (*HistoryStore, error) {
	historyStore := HistoryStore{
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
		historyStore.LoadFromFile(os.Stderr)
	}

	return &historyStore, nil
}

func (h *HistoryStore) LoadFromPath(stderr io.Writer, name string) {
	f, err := openHistoryFile(name)
	if err != nil {
		fmt.Fprintf(stderr, "failed to read from history from %s", name)
		return
	}

	h.file = f
	h.LoadFromFile(stderr)
}

func (h *HistoryStore) Close() {
	if h.file != nil {
		if err := h.file.Close(); err != nil {
			fmt.Fprintln(os.Stderr, "failed to close history file: ", err)
		}
	}
}

func (h *HistoryStore) LoadFromFile(errOut io.Writer) {
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

func (h *HistoryStore) WriteToPath(errOut io.Writer, name string) {
	f, err := openHistoryFile(name)
	if err != nil {
		fmt.Fprintln(errOut, "failed to open or create history file to write: ", err)
		return
	}

	for _, line := range h.inmemoryStore {
		f.WriteString(line + "\n")
	}
}

func (h *HistoryStore) AppendToPath(stderr io.Writer, name string) {
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

func (h *HistoryStore) Append(errOut io.Writer, cmd string) {
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

func (h *HistoryStore) Entries() []string {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	return h.inmemoryStore
}

func (h *HistoryStore) Previous() string {
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

func (h *HistoryStore) Next() string {
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
