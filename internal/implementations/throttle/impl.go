package throttle

import (
	"errors"
	"sync"
	"time"
)

type Processor interface {
	Process(int) error
}

// RateLimitedProcessor implements Processor interface and adds rate limiting.
type RateLimitedProcessor struct {
	processor Processor    // The underlying processor to wrap
	ticker    *time.Ticker // Ticker to control the rate

	mu            sync.Mutex // Protects calls count
	alreadyCalled bool       // Number of calls made in the current interval
}

func NewRateLimitedProcessor(processor Processor, alowancePerSecond int) *RateLimitedProcessor {
	rlp := &RateLimitedProcessor{
		processor: processor,
		ticker:    time.NewTicker(time.Second / time.Duration(alowancePerSecond)),
	}

	go rlp.resetCounter()
	return rlp
}

func (p *RateLimitedProcessor) resetCounter() {
	for range p.ticker.C {
		p.mu.Lock()
		p.alreadyCalled = false
		p.mu.Unlock()
	}
}

func (p *RateLimitedProcessor) Process(value int) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.alreadyCalled {
		return errors.New("rate limit exceeded")
	}

	p.alreadyCalled = true
	return p.processor.Process(value)
}
