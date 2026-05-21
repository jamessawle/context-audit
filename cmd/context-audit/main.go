package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	startup := flag.Bool("startup", false, "Audit harness context at a fresh session start")
	flag.Parse()

	if !*startup {
		fmt.Fprintln(os.Stderr, "context-audit v0.1 requires --startup")
		os.Exit(2)
	}

	if err := runStartup(); err != nil {
		fmt.Fprintf(os.Stderr, "context-audit: %v\n", err)
		os.Exit(1)
	}
}

func runStartup() error {
	return fmt.Errorf("not implemented yet")
}
