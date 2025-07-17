package main

import (
	"github.com/christophercampbell/riskr/internal/cli"
	"os"
)

func main() {
	os.Exit(cli.Run(os.Args[1:]))
}
