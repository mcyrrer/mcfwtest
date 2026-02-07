package probe

import (
	"context"
	"time"
)

// Status represents the result of a probe
type Status string

const (
	StatusOpen   Status = "open"
	StatusClosed Status = "closed"
	StatusError  Status = "error"
)

// Result represents the outcome of a single probe
type Result struct {
	Status  Status
	Latency time.Duration
	Message string
}

// Prober is the interface for network probes
type Prober interface {
	Probe(ctx context.Context, host string, port int, timeout time.Duration) Result
}
