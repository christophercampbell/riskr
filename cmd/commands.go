package main

import (
	"fmt"
	"github.com/christophercampbell/riskr/pkg/sim"
	"github.com/urfave/cli/v2"
	"strings"
)

var commands = &cli.App{
	Name:                 AppName,
	Version:              fmt.Sprintf("%v", Version),
	EnableBashCompletion: true,
	Flags: []cli.Flag{
		&cli.PathFlag{
			Name:     "config",
			Aliases:  []string{"c"},
			Usage:    "Configuration file path",
			Value:    "config.yaml",
			Required: false,
		},
	},
	Commands: []*cli.Command{
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
						Usage:    "path to a policy file",
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
			Usage:  "Run a simulation scenario",
			Action: runSim,
			Flags: []cli.Flag{&cli.StringFlag{
				Name:     "scenario",
				Aliases:  []string{"s"},
				Usage:    fmt.Sprintf("Scenario name [%s]", strings.Join(sim.GetValidScenarios(), ",")),
				Required: true,
			}},
		},
	},
}
