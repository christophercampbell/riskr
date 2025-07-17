package main

import (
	"github.com/christophercampbell/riskr/cmd/cli"
	"os"
)

func main() {
	os.Exit(cli.Run(os.Args[1:]))
}
