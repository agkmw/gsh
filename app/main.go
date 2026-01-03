package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"strings"
)

const (
	EXIT = "exit"
	ECHO = "echo"
)

func main() {
	for {
		fmt.Print("$ ")

		r := bufio.NewReader(os.Stdin)
		input, err := r.ReadString('\n')
		if err != nil {
			fmt.Println(err)
		}

		fields := strings.Fields(input)

		cmd := fields[0]
		args := fields[1:]

		switch cmd {
		case EXIT:
			return

		case ECHO:
			var buf bytes.Buffer
			for i, arg := range args {
				buf.WriteString(arg)
				if i != len(args)-1 {
					buf.WriteString(" ")
				}
			}
			fmt.Println(buf.String())
			continue

		default:
			fmt.Printf("%s: command not found\n", cmd)
		}
	}
}
