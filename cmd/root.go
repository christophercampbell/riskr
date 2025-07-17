package main

import (
	"flag"
	"fmt"
	"os"
)

// Run is the entrypoint for the riskr CLI. args excludes program name.
func Run(args []string) int {
	if len(args) == 0 {
		usage()
		return 1
	}
	sub := args[0]
	switch sub {
	case "run":
		return runCmd(args[1:])
	case "sim":
		return simCmd(args[1:])
	case "policy":
		return policyCmd(args[1:])
	case "help", "-h", "--help":
		usage()
		return 0
	default:
		fmt.Fprintf(os.Stderr, "unknown subcommand %q\n", sub)
		usage()
		return 1
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, `riskr CLI
Usage:
  riskr run <gateway|streamer|all> [-c config]
  riskr sim [scenario] [-c config]
  riskr policy <apply|list|print> [-c config] [-f file]
`)
}

// parse common -c config flag
func parseConfigFlag(args []string) (cfgPath string, rest []string) {
	fs := flag.NewFlagSet("common", flag.ContinueOnError)
	fs.StringVar(&cfgPath, "c", "", "config file path")
	_ = fs.Parse(args) // ignoring error prints default usage; minimal
	return cfgPath, fs.Args()
}
