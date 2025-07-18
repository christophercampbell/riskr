package main

import (
	"fmt"
	"github.com/christophercampbell/riskr/pkg/policy"
	"github.com/urfave/cli/v2"
)

func policyApply(cli *cli.Context) error {
	cfg, logger, err := load(cli)
	if err != nil {
		return err
	}
	file := cli.String("file")
	p, err := policy.LoadFile(file)
	if err != nil {
		return err
	}
	if err = policy.Apply(cli.Context, cfg, logger, p); err != nil {
		return err
	}
	logger.Info("policy applied", "version", p.Version)
	return nil
}

func policyList(cli *cli.Context) error {
	// TODO: implement me
	_, logger, err := load(cli)
	if err != nil {
		return err
	}
	logger.Warn("policy list: remote query not yet implemented")
	return nil
}

func policyPrint(cli *cli.Context) error {
	// TODO: implement something useful
	_, _, err := load(cli)
	if err != nil {
		return err
	}
	file := cli.String("file")
	p, err := policy.LoadFile(file)
	if err != nil {
		return err
	}
	fmt.Println(p.Pretty())
	return nil
}
