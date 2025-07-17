package events

import (
	"encoding/json"
	"time"

	"github.com/shopspring/decimal"
)

// SchemaVersion increments when incompatible changes happen.
const SchemaVersion = "v1"

// TxEvent represents an observed on-chain transfer relevant to a subject.
type TxEvent struct {
	SchemaVersion string    `json:"schema_version"`
	EventID       string    `json:"event_id"`
	OccurredAt    time.Time `json:"occurred_at"`
	ObservedAt    time.Time `json:"observed_at"`
	Subject       Subject   `json:"subject"`
	Chain         string    `json:"chain"`
	TxHash        string    `json:"tx_hash"`
	Direction     string    `json:"direction"` // inbound|outbound
	Asset         string    `json:"asset"`
	Amount        string    `json:"amount"`    // base units decimal string
	USDValue      string    `json:"usd_value"` // computed at obs time
	Confirmations int       `json:"confirmations"`
	MaxFinality   int       `json:"max_finality_depth"`
}

type Subject struct {
	UserID    string   `json:"user_id"`
	AccountID string   `json:"account_id"`
	Addresses []string `json:"addresses"`
	GeoISO    string   `json:"geo_iso"`
	KYCTier   string   `json:"kyc_level"`
}

func (e *TxEvent) Marshal() ([]byte, error) { return json.Marshal(e) }
func (e *TxEvent) Unmarshal(b []byte) error { return json.Unmarshal(b, e) }

func (e *TxEvent) USDDecimal() decimal.Decimal {
	d, _ := decimal.NewFromString(e.USDValue)
	return d
}

// DecisionEvent captures a (provisional or final) decision.
type DecisionEvent struct {
	SchemaVersion string     `json:"schema_version"`
	DecisionID    string     `json:"decision_id"`
	EventID       string     `json:"event_id"` // correlates to TxEvent.EventID
	IssuedAt      time.Time  `json:"issued_at"`
	Stage         string     `json:"stage"` // provisional|final|override
	Decision      string     `json:"decision"`
	DecisionCode  string     `json:"decision_code"`
	PolicyVersion string     `json:"policy_version"`
	Evidence      []Evidence `json:"evidence"`
}

type Evidence struct {
	RuleID string      `json:"rule_id"`
	Key    string      `json:"key"`
	Value  interface{} `json:"value"`
	Limit  interface{} `json:"limit,omitempty"`
}

func (d *DecisionEvent) Marshal() ([]byte, error) { return json.Marshal(d) }
func (d *DecisionEvent) Unmarshal(b []byte) error { return json.Unmarshal(b, d) }
