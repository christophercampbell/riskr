package main

import (
	"github.com/christophercampbell/riskr/pkg/log"
	"github.com/christophercampbell/riskr/pkg/streamer"
	"github.com/urfave/cli/v2"
)

func runStreamer(cli *cli.Context) error {
	cfg, err := loadConfig(cli)
	if err != nil {
		return err
	}

	logger := log.New(cfg.LogLevel)

	if err = streamer.Run(cli.Context, cfg, logger); err != nil {
		logger.Error("streamer exited", "err", err)
	}
	return nil
}
