// Package natsjs provides NATS JetStream bootstrap + helpers for riskr.
//
// This version preserves the earlier scaffold's exported API but fixes error
// handling so it compiles across nats.go versions (no ErrStreamAlreadyUsed).
package natsjs

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/nats-io/nats.go"
)

const (
	StreamEvents    = "EVENTS"
	StreamDecisions = "DECISIONS"
	StreamPolicy    = "POLICY"

	SubjTxEvent = "riskr.events.tx"

	SubjDecisionProv     = "riskr.decisions.provisional"
	SubjDecisionFinal    = "riskr.decisions.final"
	SubjDecisionOverride = "riskr.decisions.override"

	SubjPolicyApply     = "riskr.policies.apply"   // CLI publishes new signed policy versions
	SubjPolicyBroadcast = "riskr.policies.current" // streamer rebroadcasts active policy payload
)

// Connect dials NATS and returns an *nats.Conn* bound to ctx lifetime.
// The caller must not Close() the connection if ctx is still live unless shutting down.
func Connect(ctx context.Context, urls []string, opts ...nats.Option) (*nats.Conn, error) {

	// Always tag the connection so it's easy to see in monitoring.
	opts = append(opts, nats.Name("riskr"))

	// Add a default timeout if the caller didn't supply one.
	opts = append(opts, nats.Timeout(5*time.Second))

	nc, err := nats.Connect(strings.Join(urls, ","), opts...)
	if err != nil {
		return nil, err
	}

	// Close conn when context cancelled.
	go func() {
		<-ctx.Done()
		nc.Close()
	}()

	return nc, nil
}

// JetStream acquires a JetStreamContext from an established NATS connection.
func JetStream(nc *nats.Conn) (nats.JetStreamContext, error) {
	return nc.JetStream(nats.PublishAsyncMaxPending(256))
}

// Bootstrap ensures the core streams required by riskr exist (events + policy).
// Safe to call multiple times; streams are updated if they already exist.
func Bootstrap(js nats.JetStreamContext) error {
	eventsCfg := &nats.StreamConfig{
		Name:      StreamEvents,
		Subjects:  []string{SubjTxEvent},
		Retention: nats.LimitsPolicy,
		Storage:   nats.FileStorage,
		NoAck:     false,
		Replicas:  1,
	}
	if err := ensureStream(js, eventsCfg); err != nil {
		return err
	}

	decisionsCfg := &nats.StreamConfig{
		Name:      StreamDecisions,
		Subjects:  []string{SubjDecisionProv, SubjDecisionFinal, SubjDecisionOverride},
		Retention: nats.LimitsPolicy,
		Storage:   nats.FileStorage,
		NoAck:     false,
		Replicas:  1,
	}
	if err := ensureStream(js, decisionsCfg); err != nil {
		return err
	}

	policyCfg := &nats.StreamConfig{
		Name:      StreamPolicy,
		Subjects:  []string{SubjPolicyApply, SubjPolicyBroadcast},
		Retention: nats.LimitsPolicy,
		Storage:   nats.FileStorage,
		NoAck:     false,
		Replicas:  1,
	}
	if err := ensureStream(js, policyCfg); err != nil {
		return err
	}

	return nil
}

// SubscribeEphemeral wraps nc.Subscribe and cancels on ctx.Done().
// Use for core NATS (non-JetStream) subjects.
func SubscribeEphemeral(ctx context.Context, nc *nats.Conn, subj string, cb nats.MsgHandler) (*nats.Subscription, error) {
	sub, err := nc.Subscribe(subj, cb)
	if err != nil {
		return nil, err
	}
	// Ensure we process pending then close when ctx ends.
	go func() {
		<-ctx.Done()
		_ = sub.Drain() // Drain > Unsubscribe: processes in-flight msgs
	}()
	return sub, nil
}

// SubscribeDurable creates or attaches to a durable JetStream consumer and
// drains it when ctx cancels. Handler is called for each message; handler
// must Ack or Nak (we auto-Ack after handler returns if autoAck==true).
func SubscribeDurable(
	ctx context.Context,
	js nats.JetStreamContext,
	subj string,
	durableGroup string,
	autoAck bool,
	cb func(msg *nats.Msg),
	opts ...nats.SubOpt,
) (*nats.Subscription, error) {

	// Force durable + manual ack; caller can override via opts but we set sane defaults.
	opts = append(
		[]nats.SubOpt{
			nats.Durable(durableGroup),
			nats.ManualAck(),
			nats.DeliverNew(),  // start from new unless you want DeliverAll()
			nats.AckExplicit(), // explicit ack policy (JS)
		},
		opts...,
	)

	sub, err := js.Subscribe(subj, func(m *nats.Msg) {
		cb(m)
		if autoAck {
			_ = m.Ack()
		}
	}, opts...)
	if err != nil {
		return nil, err
	}

	go func() {
		<-ctx.Done()
		_ = sub.Drain()
	}()
	return sub, nil
}

// EnsureDurableConsumer (helper) creates or updates a durable pull consumer.
// Use from gateway/streamer services so each has its own durable cursor.
// TODO
func EnsureDurableConsumer(js nats.JetStreamContext, stream, durable, filter string, ackWait time.Duration, maxAckPending int) error {
	cfg := &nats.ConsumerConfig{
		Durable:           durable,
		AckPolicy:         nats.AckExplicitPolicy,
		AckWait:           ackWait,
		MaxAckPending:     maxAckPending,
		FilterSubject:     filter, // empty => all subjects in stream
		ReplayPolicy:      nats.ReplayInstantPolicy,
		DeliverPolicy:     nats.DeliverAllPolicy,
		InactiveThreshold: 0,
	}
	return ensureConsumer(js, stream, cfg)
}

// PublishJSON marshals v and publishes to subj via JetStream.
// TODO for using durable publishing
func PublishJSON(js nats.JetStreamContext, subj string, v any) (*nats.PubAck, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return js.Publish(subj, b)
}

// -----------------------------------------------------------------------------
// Internal helpers
// -----------------------------------------------------------------------------

// ensureStream idempotently creates or updates a JetStream stream.
// We avoid nats.go error constant sniffing for portability across versions.
func ensureStream(js nats.JetStreamContext, cfg *nats.StreamConfig) error {
	// Exists? Update + return.
	if _, err := js.StreamInfo(cfg.Name); err == nil {
		if _, uerr := js.UpdateStream(cfg); uerr != nil {
			return fmt.Errorf("update stream %s: %w", cfg.Name, uerr)
		}
		return nil
	}
	// Try create; if another process raced us, Update as fallback.
	if _, err := js.AddStream(cfg); err != nil {
		if _, uerr := js.UpdateStream(cfg); uerr != nil {
			return fmt.Errorf("ensure stream %s (add->update fallback): addErr=%v updateErr=%w", cfg.Name, err, uerr)
		}
	}
	return nil
}

// ensureConsumer idempotently creates or updates a durable consumer on stream.
func ensureConsumer(js nats.JetStreamContext, stream string, cfg *nats.ConsumerConfig) error {
	// Exists? Update + return.
	if _, err := js.ConsumerInfo(stream, cfg.Durable); err == nil {
		if _, uerr := js.UpdateConsumer(stream, cfg); uerr != nil {
			return fmt.Errorf("update consumer %s on %s: %w", cfg.Durable, stream, uerr)
		}
		return nil
	}
	// Try create; race fallback -> update.
	if _, err := js.AddConsumer(stream, cfg); err != nil {
		if _, uerr := js.UpdateConsumer(stream, cfg); uerr != nil {
			return fmt.Errorf("ensure consumer %s (add->update fallback) on %s: addErr=%v updateErr=%w", cfg.Durable, stream, err, uerr)
		}
	}
	return nil
}
