package rules

import (
	"context"
	"fmt"
	"strings"

	"github.com/hanchuanchuan/goinception-plus/internal/audit"
	"github.com/hanchuanchuan/goinception-plus/internal/model"
)

type RequirePrimaryKey struct{}

func (RequirePrimaryKey) ID() string { return audit.RuleTableMustHavePK }

func (r RequirePrimaryKey) Check(_ context.Context, ctx *audit.Context, stmt model.Statement) []model.AuditIssue {
	if !appliesToCreateTable(stmt) || !ctx.Policy.CheckPrimaryKey || stmt.DDL.HasPrimaryKey {
		return nil
	}
	return issue(ctx, stmt, r.ID(), fmt.Sprintf("Table '%s' must have a primary key.", stmt.DDL.Table))
}

type RequireTableComment struct{}

func (RequireTableComment) ID() string { return audit.RuleTableMustComment }

func (r RequireTableComment) Check(_ context.Context, ctx *audit.Context, stmt model.Statement) []model.AuditIssue {
	if !appliesToCreateTable(stmt) || !ctx.Policy.CheckTableComment || stmt.DDL.HasComment {
		return nil
	}
	return issue(ctx, stmt, r.ID(), fmt.Sprintf("Table '%s' must have a comment.", stmt.DDL.Table))
}

type RequireColumnComments struct{}

func (RequireColumnComments) ID() string { return audit.RuleColumnMustComment }

func (r RequireColumnComments) Check(_ context.Context, ctx *audit.Context, stmt model.Statement) []model.AuditIssue {
	if !appliesToCreateTable(stmt) || !ctx.Policy.CheckColumnComment {
		return nil
	}
	var missing []string
	for _, column := range stmt.DDL.Columns {
		if !column.HasComment {
			missing = append(missing, column.Name)
		}
	}
	if len(missing) == 0 {
		return nil
	}
	return issue(ctx, stmt, r.ID(), fmt.Sprintf("Columns must have comments: %s.", strings.Join(missing, ", ")))
}

type LimitColumnCount struct{}

func (LimitColumnCount) ID() string { return audit.RuleMaxColumnCount }

func (r LimitColumnCount) Check(_ context.Context, ctx *audit.Context, stmt model.Statement) []model.AuditIssue {
	limit := ctx.Policy.MaxColumnCount
	if !appliesToCreateTable(stmt) || limit <= 0 || len(stmt.DDL.Columns) <= limit {
		return nil
	}
	return issue(ctx, stmt, r.ID(), fmt.Sprintf(
		"Table '%s' has too many columns (limit %d, current %d).",
		stmt.DDL.Table, limit, len(stmt.DDL.Columns),
	))
}

func appliesToCreateTable(stmt model.Statement) bool {
	return stmt.Kind == model.StatementCreateTable && stmt.DDL != nil
}

func issue(ctx *audit.Context, stmt model.Statement, code, message string) []model.AuditIssue {
	return []model.AuditIssue{{
		RuleID: code, Level: ctx.Policy.Level(code),
		Message: message, Sequence: stmt.Sequence,
	}}
}
