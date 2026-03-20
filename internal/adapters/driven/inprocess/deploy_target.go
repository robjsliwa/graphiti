package inprocess

import (
	"context"
	"errors"

	"graphiti/internal/domain"
	"graphiti/internal/ports/driven"
)

// DeployFunc is the signature the host provides for handling deploys.
// It receives the full workflow definition and returns a result.
type DeployFunc func(ctx context.Context, payload domain.DeployPayload) (*domain.DeployResult, error)

// DeployTarget implements the DeployTarget port by calling a function
// in the host process. No HTTP, no webhook, no HMAC — just a function call.
type DeployTarget struct {
	fn DeployFunc
}

// Compile-time interface check.
var _ driven.DeployTarget = (*DeployTarget)(nil)

// NewDeployTarget creates a new in-process deploy target that calls the
// given function when a workflow is deployed.
func NewDeployTarget(fn DeployFunc) *DeployTarget {
	return &DeployTarget{fn: fn}
}

// Deploy calls the host's deploy function with the given payload.
func (d *DeployTarget) Deploy(ctx context.Context, payload domain.DeployPayload) (*domain.DeployResult, error) {
	if d.fn == nil {
		return nil, errors.New("deploy function is nil")
	}
	return d.fn(ctx, payload)
}
