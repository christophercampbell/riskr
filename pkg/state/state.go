package state

import (
	"sync"
	"time"

	"github.com/shopspring/decimal"
)

// Very simple in-memory rolling 24h store for MVP.
// Production: time-bucketed durable state (RocksDB/Badger etc.).

type View interface {
	RollingUSD24h(user string) decimal.Decimal
	RollingSmallCnt24h(user string, amtThresh decimal.Decimal) int64
	AddTx(user string, at time.Time, usd decimal.Decimal)
}

type memView struct {
	mu sync.Mutex
	// per user list of entries (timestamp, usd)
	entries map[string][]entry
}

type entry struct {
	ts  time.Time
	usd decimal.Decimal
}

func NewMem() View { return &memView{entries: make(map[string][]entry)} }

func (m *memView) pruneLocked(u string, now time.Time) {
	cut := now.Add(-24 * time.Hour)
	s := m.entries[u]
	idx := 0
	for ; idx < len(s); idx++ {
		if s[idx].ts.After(cut) {
			break
		}
	}
	if idx > 0 {
		m.entries[u] = append([]entry(nil), s[idx:]...)
	}
}

func (m *memView) RollingUSD24h(u string) decimal.Decimal {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now()
	m.pruneLocked(u, now)
	s := m.entries[u]
	sum := decimal.Zero
	for _, e := range s {
		sum = sum.Add(e.usd)
	}
	return sum
}

func (m *memView) RollingSmallCnt24h(u string, amtThresh decimal.Decimal) int64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now()
	m.pruneLocked(u, now)
	cnt := int64(0)
	for _, e := range m.entries[u] {
		if e.usd.LessThan(amtThresh) {
			cnt++
		}
	}
	return cnt
}

func (m *memView) AddTx(u string, at time.Time, usd decimal.Decimal) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.entries[u] = append(m.entries[u], entry{ts: at, usd: usd})
}
