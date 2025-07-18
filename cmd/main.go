package main

import (
	"fmt"
	"github.com/christophercampbell/riskr/pkg/config"
	"github.com/christophercampbell/riskr/pkg/sim"
	"github.com/urfave/cli/v2"
	"os"
	"path"
	"path/filepath"
	"strings"
)

const (
	AppName = "riskr"
	Version = "0.1.0"
)

func main() {
	cliApp := cli.NewApp()
	cliApp.Name = AppName
	cliApp.Version = fmt.Sprintf("%v", Version)

	// Global flags
	cliApp.Flags = []cli.Flag{
		&cli.PathFlag{
			Name:     "config",
			Aliases:  []string{"c"},
			Usage:    "Configuration file path",
			Value:    "config.yaml",
			Required: false,
		},
	}

	cliApp.Commands = []*cli.Command{
		{
			Name:   "gateway",
			Usage:  "Start the gateway server",
			Action: runGateway,
		}, {
			Name:   "streamer",
			Usage:  "Start the streamer server",
			Action: runStreamer,
		}, {
			Name:  "policy",
			Usage: "Apply or list policies",
			Subcommands: []*cli.Command{
				{
					Name:   "apply",
					Usage:  "Apply policy",
					Action: policyApply,
					Flags: []cli.Flag{&cli.StringFlag{
						Name:     "file",
						Aliases:  []string{"f"},
						Usage:    "path of policy file to apply",
						Required: true,
					}},
				}, {
					Name:   "list",
					Usage:  "List policies",
					Action: policyList,
				}, {
					Name:   "print",
					Usage:  "Print policies",
					Action: policyPrint,
					Flags: []cli.Flag{&cli.StringFlag{
						Name:     "file",
						Aliases:  []string{"f"},
						Usage:    "path of policy file to print???",
						Required: true,
					}},
				},
			},
		}, {
			Name:   "sim",
			Usage:  "Run simulation scenario",
			Action: runSim,
			Flags: []cli.Flag{&cli.StringFlag{
				Name:     "scenario",
				Aliases:  []string{"s"},
				Usage:    fmt.Sprintf("Scenario name [%s]", strings.Join(sim.GetValidScenarios(), ",")),
				Required: true,
			}},
		},
	}

	err := cliApp.Run(os.Args)
	if err != nil {
		fmt.Printf("Error: %+v\n", err)
	}
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
