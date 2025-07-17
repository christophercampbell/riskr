package streamer

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"os"
	"strings"
	"time"

	nats "github.com/nats-io/nats.go"

	"github.com/christophercampbell/riskr/pkg/config"
	"github.com/christophercampbell/riskr/pkg/decision"
	"github.com/christophercampbell/riskr/pkg/events"
	"github.com/christophercampbell/riskr/pkg/log"
	"github.com/christophercampbell/riskr/pkg/natsjs"
	"github.com/christophercampbell/riskr/pkg/policy"
	"github.com/christophercampbell/riskr/pkg/rules"
	"github.com/christophercampbell/riskr/pkg/state"
)

type Worker struct {
	cfg           *config.Config
	log           log.Logger
	nc            *nats.Conn
	state         state.View
	rulez         []rules.Rule
	policyVersion string
}

func Run(ctx context.Context, cfg *config.Config, logger log.Logger) error {
	nc, err := natsjs.Connect(ctx, cfg.NATS.URLs, nats.Name("riskr-streamer"))
	if err != nil {
		return err
	}

	// load sanctions + policy (same as gateway for now)
	p, err := policy.LoadFile(cfg.Policy.File)
	if err != nil {
		return err
	}
	sanctions, _ := loadSanctions(cfg.Sanctions.File) // ignore err for now

	w := &Worker{cfg: cfg, log: logger, nc: nc, state: state.NewMem(), rulez: rules.BuildRules(p, sanctions, p.Params), policyVersion: p.Version}

	// subscribe to policy apply
	_, err = natsjs.SubscribeEphemeral(ctx, nc, natsjs.SubjPolicyApply, func(m *nats.Msg) {
		var np policy.Policy
		if err := json.Unmarshal(m.Data, &np); err != nil {
			logger.Error("policy sub", "err", err)
			return
		}
		logger.Info("policy update", "ver", np.Version)
		w.rulez = rules.BuildRules(&np, sanctions, np.Params)
		w.policyVersion = np.Version
	})
	if err != nil {
		return err
	}

	// subscribe to tx events
	_, err = natsjs.SubscribeEphemeral(ctx, nc, natsjs.SubjTxEvent, func(m *nats.Msg) {
		var te events.TxEvent
		if err := te.Unmarshal(m.Data); err != nil {
			logger.Error("tx unmarshal", "err", err)
			return
		}
		w.handleTx(&te)
	})
	if err != nil {
		return err
	}

	// subscribe to provisional decisions
	_, err = natsjs.SubscribeEphemeral(ctx, nc, natsjs.SubjDecisionProv, func(m *nats.Msg) {
		var de events.DecisionEvent
		if err := de.Unmarshal(m.Data); err != nil {
			logger.Error("prov unmarshal", "err", err)
			return
		}
		// currently unused; could cross-check
	})
	if err != nil {
		return err
	}

	<-ctx.Done()
	return nil
}

func (w *Worker) handleTx(te *events.TxEvent) {
	usd := te.USDDecimal()
	// update state for streaming rules
	w.state.AddTx(te.Subject.UserID, te.OccurredAt, usd)

	final := decision.Allow
	var evv []events.Evidence
	now := time.Now()
	for _, rl := range w.rulez {
		if hit, dec, ev := rl.EvalStreaming(now, te, w.state); hit {
			final = decision.Max(final, dec)
			if ev.RuleID != "" {
				evv = append(evv, ev)
			}
		}
	}

	if final != decision.Allow {
		de := events.DecisionEvent{SchemaVersion: events.SchemaVersion, DecisionID: randID(), EventID: te.EventID, IssuedAt: time.Now(), Stage: "override", Decision: final, DecisionCode: pickCode(final, evv), PolicyVersion: w.policyVersion, Evidence: evv}
		if b, err := de.Marshal(); err == nil {
			_ = w.nc.Publish(natsjs.SubjDecisionFinal, b)
		}
		w.log.Info("stream override", "user", te.Subject.UserID, "decision", final)
	}
}

// local copy of gateway helpers (could refactor common)
func pickCode(dec string, ev []events.Evidence) string {
	if len(ev) == 0 {
		return "OK"
	}
	return ev[0].RuleID
}

func randID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func loadSanctions(path string) (map[string]struct{}, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	m := make(map[string]struct{})
	lines := strings.Split(string(b), "\n")
	for _, ln := range lines {
		ln = strings.TrimSpace(ln)
		if ln == "" || strings.HasPrefix(ln, "#") {
			continue
		}
		m[strings.ToLower(ln)] = struct{}{}
	}
	return m, nil
}
