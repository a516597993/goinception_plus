package execution

import (
	"context"

	"github.com/hanchuanchuan/goinception-plus/internal/model"
)

type Engine interface {
	Execute(context.Context, model.RequestOptions, []model.Statement) []model.ExecutionResult
}

type Factory func(context.Context, model.TargetOptions) (Engine, func() error, error)
