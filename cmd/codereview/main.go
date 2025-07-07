package main

import (
	"fmt"
	"os"

	"github.com/sammcj/gollama/codereview/cli"
)

func main() {
	c := cli.NewCLI()

	if err := c.Run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
