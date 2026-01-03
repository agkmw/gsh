package main

import (
	"bufio"
	"fmt"
	"os"
)

const (
	EXIT = "exit"
)

func main() {
	for {
		fmt.Print("$ ")

		r := bufio.NewReader(os.Stdin)
		rawCmd, err := r.ReadString('\n')
		cmd := rawCmd[:len(rawCmd)-1]
		if err != nil {
			fmt.Println(err)
		}

		if cmd == EXIT {
			return
		}

		fmt.Printf("%s: command not found\n", cmd)
	}
}
