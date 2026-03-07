package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: tg-bot <command> [args]")
		os.Exit(1)
	}
	fmt.Fprintln(os.Stderr, "not implemented:", os.Args[1])
	os.Exit(1)
}
