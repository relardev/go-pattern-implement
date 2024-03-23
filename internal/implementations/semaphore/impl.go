package semaphore

import "context"

type Segment struct {
	// some fieldss
}

type SingleMaterializer interface {
	Materialize(context.Context, segment Segment) error
}

type Semaphore struct {
	sm SingleMaterializer
	c  chan struct{}
}

func New(sm SingleMaterializer, allowedParallelExecutions int) (*Semaphore, fun {
	return &Semaphore{
		sm: sm,
		c:  make(chan struct{}, allowedParallelExecutions),
	}
}

func (s *Semaphore) Materialize(ctx context.Context, segment Segment) error {
	select {
	case s.c <- struct{}{}:
		defer func() { <-s.c }()
		return s.sm.Materialize(segment)
	case <-ctx.Done():
		return ctx.Err()
	}
}
