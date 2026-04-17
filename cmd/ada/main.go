package main

import (
	"fmt"
	"os"

	"github.com/yesabhishek/ada/internal/cli"
)

func main() {
	root := cli.NewRootCommand()
	if err := root.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(cli.ExitCode(err))
	}
}
