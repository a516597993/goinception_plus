package rules

import (
	"context"

	"github.com/hanchuanchuan/goinception-plus/internal/audit"
	"github.com/hanchuanchuan/goinception-plus/internal/model"
)

type KnownStatement struct{}

func (KnownStatement) ID() string { return audit.RuleUnknownStatement }

func (r KnownStatement) Check(_ context.Context, _ *audit.Context, stmt model.Statement) []model.AuditIssue {
	if stmt.Supported {
		return nil
	}
	return []model.AuditIssue{{
		RuleID: r.ID(), Level: model.SeverityError, Sequence: stmt.Sequence,
		Message: "the statement is not supported by the current goInception Plus audit projection",
	}}
}
