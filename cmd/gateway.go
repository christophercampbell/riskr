package main

import (
	"github.com/christophercampbell/riskr/pkg/gateway"
	"github.com/christophercampbell/riskr/pkg/log"
	"github.com/urfave/cli/v2"
)

func runGateway(cli *cli.Context) error {

	cfg, err := loadConfig(cli)
	if err != nil {
		return err
	}

	logger := log.New(cfg.LogLevel)

	if err = gateway.Run(cli.Context, cfg, logger); err != nil {
		logger.Error("gateway exited", "err", err)
		return err
	}
	return nil
}
