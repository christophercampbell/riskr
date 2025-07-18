package main

import (
	"github.com/christophercampbell/riskr/pkg/log"
	"github.com/christophercampbell/riskr/pkg/sim"
	"github.com/urfave/cli/v2"
)

func runSim(cli *cli.Context) error {
	cfg, err := loadConfig(cli)
	if err != nil {
		return err
	}
	logger := log.New(cfg.LogLevel)
	scenario := cli.String("scenario")
	if err = sim.Run(cli.Context, cfg, logger, scenario); err != nil {
		return err
	}
	return nil
}
