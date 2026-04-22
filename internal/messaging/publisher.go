package messaging

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

const (
	SubjectAudit       = "gobank.audit"
	SubjectTransaction = "gobank.transaction"
)

type Publisher interface {
	Publish(ctx context.Context, subject string, payload interface{}) error
}

type NATSPublisher struct {
	js  nats.JetStreamContext
	log *zap.Logger
}

func NewNATSPublisher(nc *nats.Conn, log *zap.Logger) (*NATSPublisher, error) {
	js, err := nc.JetStream()
	if err != nil {
		return nil, fmt.Errorf("getting jetstream context: %w", err)
	}

	// Create stream if not exists
	_, err = js.AddStream(&nats.StreamConfig{
		Name:     "GOBANK",
		Subjects: []string{"gobank.>"},
		Storage:  nats.FileStorage,
		Replicas: 1,
	})
	if err != nil && err != nats.ErrStreamNameAlreadyInUse {
		return nil, fmt.Errorf("creating stream: %w", err)
	}

	return &NATSPublisher{js: js, log: log}, nil
}

func (p *NATSPublisher) Publish(ctx context.Context, subject string, payload interface{}) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshaling payload: %w", err)
	}

	if _, err := p.js.Publish(subject, data); err != nil {
		p.log.Error("failed to publish message",
			zap.String("subject", subject),
			zap.Error(err),
		)
		return fmt.Errorf("publishing to %s: %w", subject, err)
	}
	return nil
}
