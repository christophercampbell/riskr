package gateway

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"github.com/nats-io/nats.go"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"github.com/christophercampbell/riskr/pkg/config"
	"github.com/christophercampbell/riskr/pkg/decision"
	"github.com/christophercampbell/riskr/pkg/events"
	"github.com/christophercampbell/riskr/pkg/log"
	"github.com/christophercampbell/riskr/pkg/natsjs"
	"github.com/christophercampbell/riskr/pkg/policy"
	"github.com/christophercampbell/riskr/pkg/rules"
)

type Server struct {
	cfg           *config.Config
	log           log.Logger
	nc            *nats.Conn
	rulez         []rules.Rule
	policyVersion string
}

func Run(ctx context.Context, cfg *config.Config, logger log.Logger) error {
	// connect nats
	nc, err := natsjs.Connect(ctx, cfg.NATS.URLs, nats.Name("riskr-policy-run"))
	if err != nil {
		return err
	}

	// load sanctions
	sanctions, err := loadSanctions(cfg.Sanctions.File)
	if err != nil {
		return err
	}

	// load policy
	p, err := policy.LoadFile(cfg.Policy.File)
	if err != nil {
		return err
	}

	s := &Server{cfg: cfg, log: logger, nc: nc, rulez: rules.BuildRules(p, sanctions, p.Params), policyVersion: p.Version}

	_, err = natsjs.SubscribeEphemeral(ctx, nc, natsjs.SubjPolicyBroadcast, func(m *nats.Msg) {
		var np policy.Policy
		if err := json.Unmarshal(m.Data, &np); err != nil {
			logger.Error("policy sub", "err", err)
			return
		}
		logger.Info("policy update", "ver", np.Version)
		s.rulez = rules.BuildRules(&np, sanctions, np.Params)
		s.policyVersion = np.Version
	})
	if err != nil {
		return err
	}

	return serveHTTP(ctx, cfg, logger, s)
}

// Request types for inline decision

type DecisionReq struct {
	Subject events.Subject `json:"subject"`
	Tx      struct {
		Type        string  `json:"type"` // withdraw|deposit
		Asset       string  `json:"asset"`
		Amount      string  `json:"amount"` // base units string (unused for now)
		USDValue    float64 `json:"usd_value"`
		DestAddress string  `json:"dest_address"`
	} `json:"tx"`
	Context map[string]any `json:"context"`
}

type DecisionResp struct {
	Decision      string            `json:"decision"`
	DecisionCode  string            `json:"decision_code"`
	PolicyVersion string            `json:"policy_version"`
	Evidence      []events.Evidence `json:"evidence"`
	ExpiresAt     *time.Time        `json:"expires_at,omitempty"`
}

func (s *Server) handleDecision(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	var req DecisionReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Build synthetic TxEvent for rule eval
	usd := decimal.NewFromFloat(req.Tx.USDValue)
	te := &events.TxEvent{
		SchemaVersion: events.SchemaVersion,
		EventID:       randID(),
		OccurredAt:    time.Now(),
		ObservedAt:    time.Now(),
		Subject:       req.Subject,
		Chain:         "INLINE", // not chain-specific, request-level
		TxHash:        "", Direction: "outbound",
		Asset:         req.Tx.Asset,
		Amount:        req.Tx.Amount,
		USDValue:      usd.String(),
		Confirmations: 0,
		MaxFinality:   0,
	}

	// Eval inline rules
	final := decision.Allow
	var evv []events.Evidence
	for _, rl := range s.rulez {
		if hit, dec, ev := rl.EvalInline(te); hit {
			final = decision.Max(final, dec)
			if ev.RuleID != "" {
				evv = append(evv, ev)
			}
		}
	}

	// publish provisional decision + synthetic tx event onto NATS for streamer
	if b, err := te.Marshal(); err == nil {
		_ = s.nc.Publish(natsjs.SubjTxEvent+".INLINE", b)
	}
	prov := events.DecisionEvent{SchemaVersion: events.SchemaVersion, DecisionID: randID(), EventID: te.EventID, IssuedAt: time.Now(), Stage: "provisional", Decision: final, DecisionCode: pickCode(final, evv), PolicyVersion: s.policyVersion, Evidence: evv}
	if b, err := prov.Marshal(); err == nil {
		_ = s.nc.Publish(natsjs.SubjDecisionProv, b)
	}

	resp := DecisionResp{Decision: final, DecisionCode: prov.DecisionCode, PolicyVersion: s.policyVersion, Evidence: evv}
	_ = json.NewEncoder(w).Encode(resp)
	dur := time.Since(start)
	if dur > time.Duration(s.cfg.LatencyBudgetMS)*time.Millisecond {
		s.log.Warn("decision latency over budget", "ms", dur.Milliseconds())
	}
}

func pickCode(dec string, ev []events.Evidence) string {
	if len(ev) == 0 {
		return "OK"
	}
	// pick highest severity first element
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
