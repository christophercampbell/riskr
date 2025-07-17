package gateway

import (
	"context"
	"net/http"
	"time"

	"github.com/christophercampbell/riskr/pkg/config"
	"github.com/christophercampbell/riskr/pkg/log"
)

func serveHTTP(ctx context.Context, cfg *config.Config, logger log.Logger, srv *Server) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/decision/check", srv.handleDecision)

	httpSrv := &http.Server{
		Addr:         cfg.HTTP.ListenAddr,
		Handler:      mux,
		ReadTimeout:  time.Duration(cfg.HTTP.ReadTimeoutMS) * time.Millisecond,
		WriteTimeout: time.Duration(cfg.HTTP.WriteTimeoutMS) * time.Millisecond,
	}

	errCh := make(chan error, 1)
	go func() {
		logger.Info("gateway listen", "addr", cfg.HTTP.ListenAddr)
		errCh <- httpSrv.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		ctx2, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = httpSrv.Shutdown(ctx2)
		return nil
	case err := <-errCh:
		return err
	}
}
