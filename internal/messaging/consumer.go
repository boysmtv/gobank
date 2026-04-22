package messaging

import "context"

type Consumer interface {
	Consume(context.Context, string, func([]byte) error) error
}

type InMemoryConsumer struct{}

func NewConsumer() *InMemoryConsumer {
	return &InMemoryConsumer{}
}

func (c *InMemoryConsumer) Consume(context.Context, string, func([]byte) error) error {
	return nil
}
