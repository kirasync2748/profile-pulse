// Package counter provides the atomic view-counting store used by ProfilePulse.
//
// The Store interface decouples the counting logic from its persistence
// backend so that tests can use a mock and the production deployment can use
// Upstash Redis while local development can use a standard Redis instance.
//
// The hot path performs exactly one atomic INCR per request and uses the
// returned value, avoiding a separate GET. This is intentional: the service
// targets the Upstash free tier and every Redis command counts against the
// allowance.
package counter

import (
	"context"
	"errors"
)

// ErrUnavailable indicates the counter backend could not be reached. Handlers
// translate this into a 5xx response and an error badge without leaking
// internal details.
var ErrUnavailable = errors.New("counter unavailable")

// Store is the abstraction over the counter backend. Incr must atomically
// increment the value stored at key and return the new value.
type Store interface {
	Incr(ctx context.Context, key string) (int64, error)
	Close() error
}

// NoopStore is a Store that always reports the backend as unavailable. It is
// used when no Redis backend is configured so the service can still start and
// serve health/dashboard endpoints.
type NoopStore struct{}

func (NoopStore) Incr(context.Context, string) (int64, error) { return 0, ErrUnavailable }
func (NoopStore) Close() error                                { return nil }
