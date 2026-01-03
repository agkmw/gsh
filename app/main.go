package main

import (
	"bufio"
	"fmt"
	"os"
)

func main() {
	for {
		fmt.Print("$ ")

		r := bufio.NewReader(os.Stdin)
		cmd, err := r.ReadString('\n')
		if err != nil {
			fmt.Println(err)
		}

		fmt.Printf("%s: command not found\n", cmd[:len(cmd)-1])
	}
}
