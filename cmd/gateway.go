package main

import (
	"github.com/christophercampbell/riskr/pkg/gateway"
	"github.com/urfave/cli/v2"
)

func runGateway(cli *cli.Context) error {
	cfg, logger, err := load(cli)
	if err != nil {
		return err
	}
	if err = gateway.Run(cli.Context, cfg, logger); err != nil {
		logger.Error("gateway exited", "err", err)
		return err
	}
	return nil
}
