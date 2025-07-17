package cli

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/christophercampbell/riskr/pkg/config"
	"github.com/christophercampbell/riskr/pkg/log"
	"github.com/christophercampbell/riskr/pkg/sim"
)

func simCmd(args []string) int {
	fs := flag.NewFlagSet("sim", flag.ExitOnError)
	cfgPath := fs.String("c", "", "config path")
	_ = fs.Parse(args)
	scenario := "clean"
	if a := fs.Arg(0); a != "" {
		scenario = a
	}

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "config load error: %v\n", err)
		return 1
	}
	logger := log.New(cfg.LogLevel)

	if err := sim.Run(context.Background(), cfg, logger, scenario); err != nil {
		logger.Error("sim failed", "err", err)
		return 1
	}
	return 0
}
