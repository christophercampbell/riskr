package main

import (
	"github.com/christophercampbell/riskr/pkg/config"
	"github.com/christophercampbell/riskr/pkg/log"
	"github.com/urfave/cli/v2"
	"path"
	"path/filepath"
)

func load(cli *cli.Context) (*config.Config, log.Logger, error) {
	c, err := loadConfig(cli)
	if err != nil {
		return nil, nil, err
	}
	return c, log.New(c.LogLevel), nil
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
