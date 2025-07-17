package rules

import (
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"github.com/christophercampbell/riskr/pkg/decision"
	"github.com/christophercampbell/riskr/pkg/events"
	"github.com/christophercampbell/riskr/pkg/policy"
	"github.com/christophercampbell/riskr/pkg/state"
)

type Rule interface {
	ID() string
	EvalInline(e *events.TxEvent) (hit bool, dec string, ev events.Evidence)
	EvalStreaming(now time.Time, e *events.TxEvent, st state.View) (hit bool, dec string, ev events.Evidence)
}

// BuildRules constructs rule instances from policy defs + params + sanctions & thresholds.
func BuildRules(p *policy.Policy, sanctions map[string]struct{}, params map[string]any) []Rule {
	r := make([]Rule, 0, len(p.Rules))
	for _, rd := range p.Rules {
		switch rd.Type {
		case "ofac_addr":
			r = append(r, newOFACRule(rd, sanctions))
		case "jurisdiction_block":
			r = append(r, newJurisRule(rd))
		case "kyc_tier_tx_cap":
			r = append(r, newKYCTierCapRule(rd, params))
		case "daily_usd_volume":
			r = append(r, newDailyVolRule(rd, params))
		case "structuring_small_tx":
			r = append(r, newStructuringRule(rd, params))
		}
	}
	return r
}

// ------------------------ OFAC Rule ------------------------

type ofacRule struct {
	id      string
	action  string
	addrSet map[string]struct{}
}

func newOFACRule(rd policy.RuleDef, addrs map[string]struct{}) Rule {
	return &ofacRule{id: rd.ID, action: rd.Action, addrSet: addrs}
}

func (r *ofacRule) ID() string { return r.id }

func (r *ofacRule) EvalInline(e *events.TxEvent) (bool, string, events.Evidence) {
	// check any subject addr or tx direction address? For MVP we just check Subject.Addresses
	for _, a := range e.Subject.Addresses {
		if _, ok := r.addrSet[strings.ToLower(a)]; ok {
			return true, r.action, events.Evidence{RuleID: r.id, Key: "address", Value: a}
		}
	}
	return false, decision.Allow, events.Evidence{}
}

func (r *ofacRule) EvalStreaming(_ time.Time, e *events.TxEvent, _ state.View) (bool, string, events.Evidence) {
	return r.EvalInline(e) // same for now
}

// ------------------------ Jurisdiction Block Rule ------------------------

type jurisRule struct {
	id      string
	action  string
	blocked map[string]struct{}
}

func newJurisRule(rd policy.RuleDef) Rule {
	m := make(map[string]struct{}, len(rd.BlockedCountries))
	for _, c := range rd.BlockedCountries {
		m[strings.ToUpper(c)] = struct{}{}
	}
	return &jurisRule{id: rd.ID, action: rd.Action, blocked: m}
}

func (r *jurisRule) ID() string { return r.id }
func (r *jurisRule) EvalInline(e *events.TxEvent) (bool, string, events.Evidence) {
	if _, bad := r.blocked[strings.ToUpper(e.Subject.GeoISO)]; bad {
		return true, r.action, events.Evidence{RuleID: r.id, Key: "geo_iso", Value: e.Subject.GeoISO}
	}
	return false, decision.Allow, events.Evidence{}
}
func (r *jurisRule) EvalStreaming(_ time.Time, e *events.TxEvent, _ state.View) (bool, string, events.Evidence) {
	return r.EvalInline(e)
}

// ------------------------ KYC Tier Tx Cap Rule ------------------------

type kycTierCapRule struct {
	id     string
	action string
	caps   map[string]decimal.Decimal // tier->limit
}

func newKYCTierCapRule(rd policy.RuleDef, params map[string]any) Rule {
	caps := map[string]decimal.Decimal{}
	if p, ok := params["kyc_tier_caps_usd"].(map[string]any); ok {
		for tier, v := range p {
			caps[tier] = toDec(v)
		}
	}
	return &kycTierCapRule{id: rd.ID, action: rd.Action, caps: caps}
}

func (r *kycTierCapRule) ID() string { return r.id }
func (r *kycTierCapRule) EvalInline(e *events.TxEvent) (bool, string, events.Evidence) {
	lim := r.caps[e.Subject.KYCTier]
	usd := e.USDDecimal()
	if lim.GreaterThan(decimal.Zero) && usd.GreaterThan(lim) {
		return true, r.action, events.Evidence{RuleID: r.id, Key: "usd_value", Value: usd.String(), Limit: lim.String()}
	}
	return false, decision.Allow, events.Evidence{}
}
func (r *kycTierCapRule) EvalStreaming(_ time.Time, e *events.TxEvent, _ state.View) (bool, string, events.Evidence) {
	return r.EvalInline(e)
}

// ------------------------ Daily USD Volume Rule ------------------------

type dailyVolRule struct {
	id     string
	action string
	limit  decimal.Decimal
}

func newDailyVolRule(rd policy.RuleDef, params map[string]any) Rule {
	lim := toDec(params["daily_volume_limit_usd"])
	return &dailyVolRule{id: rd.ID, action: rd.Action, limit: lim}
}

func (r *dailyVolRule) ID() string { return r.id }
func (r *dailyVolRule) EvalInline(e *events.TxEvent) (bool, string, events.Evidence) {
	// Inline uses snapshotless; always Allow (streaming authoritative)
	return false, decision.Allow, events.Evidence{}
}
func (r *dailyVolRule) EvalStreaming(_ time.Time, e *events.TxEvent, st state.View) (bool, string, events.Evidence) {
	sum := st.RollingUSD24h(e.Subject.UserID)
	usd := e.USDDecimal()
	newSum := sum.Add(usd)
	if newSum.GreaterThan(r.limit) {
		return true, r.action, events.Evidence{RuleID: r.id, Key: "daily_usd", Value: newSum.String(), Limit: r.limit.String()}
	}
	return false, decision.Allow, events.Evidence{}
}

// ------------------------ Structuring Rule ------------------------

type structuringRule struct {
	id        string
	action    string
	amtThresh decimal.Decimal
	cntThresh int64
}

func newStructuringRule(rd policy.RuleDef, params map[string]any) Rule {
	amt := toDec(params["structuring_small_usd"])
	cnt := int64(5)
	if v, ok := params["structuring_small_count"]; ok {
		cnt = toInt(v)
	}
	return &structuringRule{id: rd.ID, action: rd.Action, amtThresh: amt, cntThresh: cnt}
}

func (r *structuringRule) ID() string { return r.id }
func (r *structuringRule) EvalInline(e *events.TxEvent) (bool, string, events.Evidence) {
	return false, decision.Allow, events.Evidence{}
}
func (r *structuringRule) EvalStreaming(_ time.Time, e *events.TxEvent, st state.View) (bool, string, events.Evidence) {
	// Count inbound < amtThresh in 24h
	cnt := st.RollingSmallCnt24h(e.Subject.UserID, r.amtThresh)
	if cnt+1 > r.cntThresh { // +1 includes current
		return true, r.action, events.Evidence{RuleID: r.id, Key: "small_cnt_24h", Value: cnt + 1, Limit: r.cntThresh}
	}
	return false, decision.Allow, events.Evidence{}
}

// ------------------------ helpers ------------------------

func toDec(v any) decimal.Decimal {
	switch t := v.(type) {
	case int:
		return decimal.NewFromInt(int64(t))
	case int64:
		return decimal.NewFromInt(t)
	case float64:
		return decimal.NewFromFloat(t)
	case string:
		d, _ := decimal.NewFromString(t)
		return d
	case nil:
		return decimal.Zero
	default:
		return decimal.Zero
	}
}

func toInt(v any) int64 {
	switch t := v.(type) {
	case int:
		return int64(t)
	case int64:
		return t
	case float64:
		return int64(t)
	case string:
		// ignore error
		return 0
	default:
		return 0
	}
}
