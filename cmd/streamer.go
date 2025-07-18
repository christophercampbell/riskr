package main

import (
	"fmt"
	"github.com/urfave/cli/v2"
)

func runStreamer(cli *cli.Context) error {
	/*
		if err := streamer.Run(ctx, cfg, logger); err != nil {
			logger.Error("streamer exited", "err", err)
			return 1
		}
	*/
	fmt.Println("streamer started")
	return nil
}
