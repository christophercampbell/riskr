package decision

// Decision enum
const (
	Allow       = "ALLOW"
	HoldAuto    = "HOLD_AUTO"
	Review      = "REVIEW"
	RejectFatal = "REJECT_FATAL"
	SoftDeny    = "SOFT_DENY_RETRY"
)

// Severity ordering for escalation rules.
var severityRank = map[string]int{
	Allow:       0,
	SoftDeny:    1,
	HoldAuto:    2,
	Review:      3,
	RejectFatal: 4,
}

// Max returns the more severe of two decisions.
func Max(a, b string) string {
	if severityRank[a] >= severityRank[b] {
		return a
	}
	return b
}
