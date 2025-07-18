package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/urfave/cli/v2"
	"os"

	"github.com/christophercampbell/riskr/pkg/config"
	"github.com/christophercampbell/riskr/pkg/log"
	"github.com/christophercampbell/riskr/pkg/policy"
)

func runPolicy(cli *cli.Context) error {
	return nil
}

func policyCmd(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "policy requires subcommand: apply|list|print")
		return 1
	}
	sub := args[0]
	fs := flag.NewFlagSet("policy", flag.ExitOnError)
	cfgPath := fs.String("c", "", "config path")
	file := fs.String("f", "", "policy file (for apply/print)")
	_ = fs.Parse(args[1:])

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "config load error: %v\n", err)
		return 1
	}
	logger := log.New(cfg.LogLevel)

	switch sub {
	case "apply":
		if *file == "" {
			fmt.Fprintln(os.Stderr, "policy apply requires -f file")
			return 1
		}
		p, err := policy.LoadFile(*file)
		if err != nil {
			logger.Error("policy load", "err", err)
			return 1
		}
		if err := policy.Apply(context.Background(), cfg, logger, p); err != nil {
			logger.Error("policy apply", "err", err)
			return 1
		}
		logger.Info("policy applied", "version", p.Version)
		return 0
	case "print":
		if *file == "" {
			*file = cfg.Policy.File
		}
		p, err := policy.LoadFile(*file)
		if err != nil {
			logger.Error("policy load", "err", err)
			return 1
		}
		fmt.Println(p.Pretty())
		return 0
	case "list":
		// MVP local only; remote listing TODO
		fmt.Println("policy list: remote query not yet implemented")
		return 0
	default:
		fmt.Fprintf(os.Stderr, "unknown policy subcommand %q\n", sub)
		return 1
	}
}
