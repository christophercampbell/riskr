package sim

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/nats-io/nats.go"
	"time"

	"github.com/shopspring/decimal"

	"github.com/christophercampbell/riskr/pkg/config"
	"github.com/christophercampbell/riskr/pkg/events"
	"github.com/christophercampbell/riskr/pkg/log"
	"github.com/christophercampbell/riskr/pkg/natsjs"
)

const (
	CLEAN       = "clean"
	OFAC        = "ofac"
	DAILY       = "daily"
	STRUCTURING = "structuring"
)

var (
	ValidScenarios = map[string]struct{}{
		CLEAN:       {},
		OFAC:        {},
		DAILY:       {},
		STRUCTURING: {},
	}
)

func GetValidScenarios() []string {
	var validScenarios []string
	for scenario := range ValidScenarios {
		validScenarios = append(validScenarios, scenario)
	}
	return validScenarios
}

func Run(ctx context.Context, cfg *config.Config, logger log.Logger, scenario string) error {
	if _, valid := ValidScenarios[scenario]; !valid {
		validScenarios := GetValidScenarios()
		msg := fmt.Sprintf("invalid scenario. got %v, expected one of %v", scenario, validScenarios)
		return errors.New(msg)
	}

	nc, err := natsjs.Connect(ctx, cfg.NATS.URLs, nats.Name("riskr-sim"))
	if err != nil {
		return err
	}
	defer nc.Close()

	switch scenario {
	case "clean":
		return simClean(ctx, nc, logger)
	case "ofac":
		return simOFAC(ctx, nc, logger)
	case "daily":
		return simDaily(ctx, nc, logger)
	case "structuring":
		return simStructuring(ctx, nc, logger)
	default:
		return fmt.Errorf("unknown scenario %s", scenario)
	}
}

func simClean(ctx context.Context, nc *nats.Conn, logger log.Logger) error {
	logger.Info("sim clean")
	return pubTx(nc, logger, "U1", "A1", []string{"0xClean"}, "USDC", "1000000", 1.00)
}

func simOFAC(ctx context.Context, nc *nats.Conn, logger log.Logger) error {
	logger.Info("sim ofac")
	return pubTx(nc, logger, "U2", "A2", []string{"0x000000000000000000000000000000000000dEaD"}, "USDC", "1000000", 1.00)
}

func simDaily(ctx context.Context, nc *nats.Conn, logger log.Logger) error {
	logger.Info("sim daily limit breach")
	for i := 0; i < 6; i++ { // 6 * 10k = 60k > 50k
		if err := pubTx(nc, logger, "U3", "A3", []string{fmt.Sprintf("0xU3%02d", i)}, "USDC", "10000000", 10000.00); err != nil {
			return err
		}
	}
	return nil
}

func simStructuring(ctx context.Context, nc *nats.Conn, logger log.Logger) error {
	logger.Info("sim structuring")
	for i := 0; i < 6; i++ { // 6 * $5k deposits triggers R5 cnt>5 (<10k threshold)
		if err := pubTx(nc, logger, "U4", "A4", []string{fmt.Sprintf("0xU4%02d", i)}, "USDC", "5000000", 5000.00); err != nil {
			return err
		}
	}
	return nil
}

func pubTx(nc *nats.Conn, logger log.Logger, user, acct string, addrs []string, asset, amount string, usd float64) error {
	te := events.TxEvent{
		SchemaVersion: events.SchemaVersion,
		EventID:       randID(),
		OccurredAt:    time.Now(),
		ObservedAt:    time.Now(),
		Subject:       events.Subject{UserID: user, AccountID: acct, Addresses: addrs, GeoISO: "US", KYCTier: "L2"},
		Chain:         "SIM",
		TxHash:        randID(),
		Direction:     "inbound",
		Asset:         asset,
		Amount:        amount,
		USDValue:      decimal.NewFromFloat(usd).String(),
		Confirmations: 3,
		MaxFinality:   12,
	}
	b, _ := json.Marshal(te)
	return nc.Publish(natsjs.SubjTxEvent+".SIM", b)
}

// local helpers
func randID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
