# gsh — A Minimal POSIX-Like Shell in Go

[![CodeCrafters Challenge](https://img.shields.io/badge/CodeCrafters-Build%20Your%20Own%20Shell-blue)](https://codecrafters.io)

gsh is a lightweight, functional shell built in Go as a learning project. It implements a Read-Eval-Print Loop (REPL) with command history, tab completion, I/O redirection, and Unix pipelines.

## Features

- **Built-in commands**: `cd`, `exit`, `echo`, `pwd`, `type`, `history`
- **I/O redirection**: `>`, `>>`, `1>`, `1>>`, `2>`, `2>>`
- **Unix pipelines**: `cmd1 | cmd2 | cmd3`
- **Tab completion**: Auto-completes commands from `$PATH` and built-ins
- **History navigation**: Ctrl+P (previous) / Ctrl+N (next)
- **Signal handling**: Ctrl+C (interrupt), Ctrl+D (EOF exit)
- **Persistent history**: File-backed via `$HISTFILE` environment variable
- **Quoting**: Single-quote (literal), double-quote (limited escapes), backslash escaping

## Project Structure

app/
├── main.go        — REPL loop: read, tokenize, dispatch
├── reader.go      — readline setup, history key bindings, tab completion
├── parser.go      — quote-aware tokenizer (single-pass byte state machine)
├── runner.go      — command dispatch, pipeline construction & orchestration
├── executor.go    — external binary execution and $PATH lookup
├── builtins.go    — built-in command implementations (cd, exit, echo, etc.)
├── redirection.go — I/O redirection parsing and file descriptor setup
├── history.go     — in-memory history ring with file persistence
└── completer.go   — tab completions with bell feedback on no/multiple matches

## Quick Start

```sh
go build -o gsh ./app
./gsh
```
Set a custom history file:

`HISTFILE=~/.gsh_history ./gsh`

Limitations
- No job control: Child processes inherit the shell's process group. Ctrl+C can terminate the shell itself.
> Basic signal trapping is implemented, but due to lack of job control, signals may propagate improperly to the parent shell process group. This will be fixed in future updates.
- Simple pipe synchronisation: Pipeline segments run concurrently in goroutines with no explicit ordering guarantees.
