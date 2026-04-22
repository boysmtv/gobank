package messaging

import (
	"context"
	"encoding/json"

	"github.com/nats-io/nats.go"
	"github.com/yourorg/gobank/internal/domain"
	"github.com/yourorg/gobank/internal/repository"
	"go.uber.org/zap"
)

type AuditConsumer struct {
	js        nats.JetStreamContext
	auditRepo repository.AuditRepository
	log       *zap.Logger
}

func NewAuditConsumer(nc *nats.Conn, auditRepo repository.AuditRepository, log *zap.Logger) (*AuditConsumer, error) {
	js, err := nc.JetStream()
	if err != nil {
		return nil, err
	}
	return &AuditConsumer{js: js, auditRepo: auditRepo, log: log}, nil
}

func (c *AuditConsumer) Start(ctx context.Context) error {
	sub, err := c.js.Subscribe(SubjectAudit, func(msg *nats.Msg) {
		var event domain.AuditEvent
		if err := json.Unmarshal(msg.Data, &event); err != nil {
			c.log.Error("failed to unmarshal audit event", zap.Error(err))
			_ = msg.Nak()
			return
		}

		if err := c.auditRepo.Create(ctx, &event.AuditLog); err != nil {
			c.log.Error("failed to persist audit log",
				zap.String("action", string(event.Action)),
				zap.Error(err),
			)
			_ = msg.Nak()
			return
		}

		_ = msg.Ack()
	}, nats.Durable("audit-consumer"), nats.AckExplicit())

	if err != nil {
		return err
	}

	go func() {
		<-ctx.Done()
		_ = sub.Unsubscribe()
	}()

	return nil
}
