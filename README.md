# riskr

Compliance-aware streaming risk engine (crypto-only MVP) built on NATS JetStream.

**Components**
- Inline Risk Gateway: low-latency decision API (≤100ms p95).
- Streaming Risk Worker: rolling exposure windows, pattern rules, overrides.
- Policy Manager: signed policy apply broadcast.
- Simulator: deterministic test flows to exercise rules.

**Quick start**
```bash
docker compose up -d
make build
make policy-apply   # load policies
make run-streamer &
make run-gateway &
make sim            # send sample tx events
```

See `docs/` or project spec in side doc for architecture.

