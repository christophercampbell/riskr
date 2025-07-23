package streamer

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
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

const (
	connName = "riskr-streamer"
)

func durableGroupName(task string) string {
	return fmt.Sprintf("%s-%s", connName, task)
}

type Worker struct {
	cfg           *config.Config
	log           log.Logger
	nc            *nats.Conn
	state         state.View
	rulez         []rules.Rule
	policyVersion string
}

func Run(ctx context.Context, cfg *config.Config, logger log.Logger) error {
	logger.Info("starting streamer")
	nc, err := natsjs.Connect(ctx, cfg.NATS.URLs, nats.Name(connName))
	if err != nil {
		return err
	}
	js, err := natsjs.JetStream(nc)
	if err != nil {
		return err
	}
	if err = natsjs.Bootstrap(js); err != nil {
		return err
	}

	// load sanctions + policy (same as gateway for now)
	p, err := policy.LoadFile(cfg.ResolvePolicyFile())
	if err != nil {
		return err
	}
	sanctions, _ := loadSanctions(cfg.Sanctions.File) // ignore err for now

	w := &Worker{
		cfg:           cfg,
		log:           logger,
		nc:            nc,
		state:         state.NewMem(),
		rulez:         rules.BuildRules(p, sanctions, p.Params),
		policyVersion: p.Version,
	}

	// subscribe to policy apply
	policyApplyGroup := durableGroupName("policy-apply")
	logger.Info("subscribing", "subject", natsjs.SubjPolicyApply, "group", policyApplyGroup)
	policyApplySub, err := natsjs.SubscribeDurable(ctx, js, natsjs.SubjPolicyApply, policyApplyGroup, true, func(m *nats.Msg) {
		var np policy.Policy
		if err = json.Unmarshal(m.Data, &np); err != nil {
			logger.Error("policy sub", "err", err)
			return
		}
		logger.Info("policy update", "ver", np.Version)
		w.rulez = rules.BuildRules(&np, sanctions, np.Params)
		w.policyVersion = np.Version
		// w.handlePolicyApply ...
	})
	if err != nil {
		return err
	}
	defer policyApplySub.Unsubscribe()

	// subscribe to tx events
	txGroup := durableGroupName("tx-process")
	logger.Info("subscribing", "subject", natsjs.SubjTxEvent, "group", txGroup)
	txSub, err := natsjs.SubscribeDurable(ctx, js, natsjs.SubjTxEvent, txGroup, true,
		func(m *nats.Msg) {
			var te events.TxEvent
			if err = te.Unmarshal(m.Data); err != nil {
				logger.Error("tx unmarshal", "err", err)
				return
			}
			w.handleTx(&te)
		})
	if err != nil {
		return err
	}
	defer txSub.Unsubscribe()

	// subscribe to provisional decisions
	provDecGroup := durableGroupName("prov-dec")
	logger.Info("subscribing", "subject", natsjs.SubjDecisionProv, "group", provDecGroup)
	decProvSub, err := natsjs.SubscribeDurable(ctx, js, natsjs.SubjDecisionProv, provDecGroup, true, func(m *nats.Msg) {
		var de events.DecisionEvent
		if err = de.Unmarshal(m.Data); err != nil {
			logger.Error("prov unmarshal", "err", err)
			return
		}
		// currently unused; could cross-check
	})
	if err != nil {
		return err
	}
	defer decProvSub.Unsubscribe()

	<-ctx.Done()
	return nil
}

func (w *Worker) handleTx(te *events.TxEvent) {
	if msg, err := te.Marshal(); err != nil {
		w.log.Error("failed to handle tx", "err", err)
		return
	} else {
		w.log.Info("handling transaction", "tx", string(msg))
	}
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
