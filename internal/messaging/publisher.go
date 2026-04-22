package messaging

import "context"

type Publisher interface {
	Publish(context.Context, string, []byte) error
}

type InMemoryPublisher struct{}

func NewPublisher() *InMemoryPublisher {
	return &InMemoryPublisher{}
}

func (p *InMemoryPublisher) Publish(context.Context, string, []byte) error {
	return nil
}
