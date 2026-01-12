package main

import (
	"os"

	"github.com/thirteen37/amail/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
