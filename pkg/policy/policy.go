package policy

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/nats-io/nats.go"
	"os"

	yaml "gopkg.in/yaml.v3"

	"github.com/christophercampbell/riskr/pkg/config"
	"github.com/christophercampbell/riskr/pkg/log"
	"github.com/christophercampbell/riskr/pkg/natsjs"
)

type Policy struct {
	Version string         `yaml:"policy_version" json:"policy_version"`
	Params  map[string]any `yaml:"params" json:"params"`
	Rules   []RuleDef      `yaml:"rules" json:"rules"`
	Sig     string         `yaml:"signature" json:"signature"`
	Hash    string         `yaml:"-" json:"hash"`
}

type RuleDef struct {
	ID               string   `yaml:"id" json:"id"`
	Type             string   `yaml:"type" json:"type"`
	Action           string   `yaml:"action" json:"action"`
	BlockedCountries []string `yaml:"blocked_countries" json:"blocked_countries,omitempty"`
}

func LoadFile(path string) (*Policy, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var p Policy
	if err := yaml.Unmarshal(b, &p); err != nil {
		return nil, err
	}
	p.computeHash()
	return &p, nil
}

func (p *Policy) computeHash() {
	b, _ := json.Marshal(p)
	h := sha256.Sum256(b)
	p.Hash = hex.EncodeToString(h[:])
}

func (p *Policy) Pretty() string {
	b, _ := json.MarshalIndent(p, "", "  ")
	return string(b)
}

// Apply publishes policy to NATS where components subscribe and update.
func Apply(ctx context.Context, cfg *config.Config, logger log.Logger, p *Policy) error {
	conn, err := natsjs.Connect(ctx, cfg.NATS.URLs, nats.Name("riskr-policy-apply"))
	if err != nil {
		return err
	}
	defer conn.Close()
	b, _ := json.Marshal(p)
	if err := conn.Publish(natsjs.SubjPolicyApply, b); err != nil {
		return err
	}
	return nil
}
