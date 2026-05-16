package store

import (
	"context"
	"errors"

	"github.com/cruxctl/crux/internal/domain"
)

var ErrNotFound = errors.New("not found")

type Store interface {
	RuntimeConfig(ctx context.Context) (domain.RuntimeConfig, error)
	UpdateRuntimeConfig(ctx context.Context, cfg domain.RuntimeConfig) error

	UpsertAgent(ctx context.Context, agent domain.Agent) error
	DeleteAgent(ctx context.Context, name string) error
	GetAgent(ctx context.Context, name string) (domain.Agent, error)
	ListAgents(ctx context.Context) ([]domain.Agent, error)

	CreateExecution(ctx context.Context, execution domain.Execution) error
	UpdateExecution(ctx context.Context, execution domain.Execution) error
	GetExecution(ctx context.Context, id string) (domain.Execution, error)
	ListExecutions(ctx context.Context) ([]domain.Execution, error)

	AppendEvent(ctx context.Context, event domain.Event) error
	ListEvents(ctx context.Context, executionID string) ([]domain.Event, error)
}
