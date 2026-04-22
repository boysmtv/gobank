package tracer

import "context"

type Tracer struct{}

func New() *Tracer {
	return &Tracer{}
}

func (t *Tracer) Start(ctx context.Context, name string) (context.Context, func()) {
	return ctx, func() {}
}
