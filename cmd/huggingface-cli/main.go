package main

import (
	"fmt"
	"os"

	"github.com/envblex/huggingface-cli/pkg/cli"
)

func main() {
	cmd := cli.NewRootCmd()
	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
