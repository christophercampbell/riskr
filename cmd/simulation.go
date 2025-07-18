package main

import (
	"github.com/christophercampbell/riskr/pkg/sim"
	"github.com/urfave/cli/v2"
)

func runSim(cli *cli.Context) error {
	cfg, logger, err := load(cli)
	if err != nil {
		return err
	}
	scenario := cli.String("scenario")
	if err = sim.Run(cli.Context, cfg, logger, scenario); err != nil {
		return err
	}
	return nil
}
