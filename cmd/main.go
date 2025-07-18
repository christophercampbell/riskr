package main

import (
	"context"
	"fmt"
	"github.com/christophercampbell/riskr/pkg/config"
	"github.com/urfave/cli/v2"
	"log"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"syscall"
)

const (
	AppName = "riskr"
	Version = "0.1.0"
)

var (
	configFlag = cli.PathFlag{
		Name:     "config",
		Aliases:  []string{"c"},
		Usage:    "Configuration file path",
		Value:    "config.yaml",
		Required: false,
	}
)

func main() {
	cliApp := cli.NewApp()
	cliApp.Name = AppName
	cliApp.Version = fmt.Sprintf("%v", Version)

	cliApp.Commands = []*cli.Command{
		{
			Name:   "gateway",
			Usage:  "Start the gateway server",
			Action: runGateway,
			Flags:  []cli.Flag{&configFlag},
		}, {
			Name:   "streamer",
			Usage:  "Start the streamer server",
			Action: runStreamer,
			Flags:  []cli.Flag{&configFlag},
		}, {
			Name:   "policy",
			Usage:  "Apply or list policies",
			Action: runPolicy,
			Flags:  []cli.Flag{&configFlag},
		}, {
			Name:   "sim",
			Usage:  "Run simulation scenario",
			Action: runSim,
			Flags:  []cli.Flag{&configFlag},
		},
	}

	err := cliApp.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func signalContext() (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		cancel()
	}()
	return ctx, cancel
}

func loadConfig(cli *cli.Context) (*config.Config, error) {
	configFile := cli.Path("config")
	var err error
	if !path.IsAbs(configFile) {
		configFile, err = filepath.Abs(configFile)
	}
	if err != nil {
		return nil, err
	}
	config, err := config.Load(configFile)
	if err != nil {
		return nil, err
	}
	return config, nil
}
