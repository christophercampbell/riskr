package main

import (
	"github.com/christophercampbell/riskr/pkg/streamer"
	"github.com/urfave/cli/v2"
)

func runStreamer(cli *cli.Context) error {
	cfg, logger, err := load(cli)
	if err != nil {
		return err
	}
	if err = streamer.Run(cli.Context, cfg, logger); err != nil {
		logger.Error("streamer exited", "err", err)
	}
	return nil
}
