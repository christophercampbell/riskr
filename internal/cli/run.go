package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/christophercampbell/riskr/pkg/config"
	"github.com/christophercampbell/riskr/pkg/gateway"
	"github.com/christophercampbell/riskr/pkg/log"
	"github.com/christophercampbell/riskr/pkg/streamer"
)

func runCmd(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "run requires component: gateway|streamer|all")
		return 1
	}
	component := args[0]
	cfgPath, rest := parseConfigFlag(args[1:])
	_ = rest // unused

	cfg, err := config.Load(cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "config load error: %v\n", err)
		return 1
	}
	logger := log.New(cfg.LogLevel)

	ctx, cancel := signalContext()
	defer cancel()

	switch component {
	case "gateway":
		return runGateway(ctx, cfg, logger)
	case "streamer":
		return runStreamer(ctx, cfg, logger)
	case "all":
		// run both in same process (dev only)
		go runStreamer(ctx, cfg, logger)
		return runGateway(ctx, cfg, logger)
	default:
		fmt.Fprintf(os.Stderr, "unknown run component %q\n", component)
		return 1
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

func runGateway(ctx context.Context, cfg *config.Config, logger log.Logger) int {
	if err := gateway.Run(ctx, cfg, logger); err != nil {
		logger.Error("gateway exited", "err", err)
		return 1
	}
	return 0
}

func runStreamer(ctx context.Context, cfg *config.Config, logger log.Logger) int {
	if err := streamer.Run(ctx, cfg, logger); err != nil {
		logger.Error("streamer exited", "err", err)
		return 1
	}
	return 0
}
