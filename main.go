package main

import (
	"fmt"
	"os"
)

const version = "0.2.0"

func main() {
	if err := newRootCommand(os.Stdin, os.Stdout, os.Stderr).Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "azenv:", err)
		os.Exit(1)
	}
}
