package rules

import (
	"context"
	"testing"

	"github.com/hanchuanchuan/goinception-plus/internal/audit"
	"github.com/hanchuanchuan/goinception-plus/internal/model"
)

func TestCreateTableRuleSet(t *testing.T) {
	policy := audit.LegacyDefaults()
	policy.CheckPrimaryKey = true
	policy.CheckTableComment = true
	policy.CheckColumnComment = true
	policy.MaxColumnCount = 1
	ctx := &audit.Context{Policy: policy}
	stmt := model.Statement{
		Sequence: 2, Kind: model.StatementCreateTable,
		DDL: &model.DDLSpec{
			Table:   "users",
			Columns: []model.ColumnSpec{{Name: "id"}, {Name: "name"}},
		},
	}
	ruleSet := []audit.Rule{
		RequirePrimaryKey{},
		RequireTableComment{},
		RequireColumnComments{},
		LimitColumnCount{},
	}
	for _, rule := range ruleSet {
		issues := rule.Check(context.Background(), ctx, stmt)
		if len(issues) != 1 {
			t.Fatalf("%s: got %d issues, want 1", rule.ID(), len(issues))
		}
		if issues[0].RuleID != rule.ID() ||
			issues[0].Level != model.SeverityWarning ||
			issues[0].Sequence != 2 {
			t.Fatalf("%s: unexpected issue: %+v", rule.ID(), issues[0])
		}
	}
}

func TestCreateTableRuleSetPasses(t *testing.T) {
	policy := audit.LegacyDefaults()
	policy.CheckPrimaryKey = true
	policy.CheckTableComment = true
	policy.CheckColumnComment = true
	policy.MaxColumnCount = 2
	ctx := &audit.Context{Policy: policy}
	stmt := model.Statement{
		Kind: model.StatementCreateTable,
		DDL: &model.DDLSpec{
			Table: "users", HasPrimaryKey: true, HasComment: true,
			Columns: []model.ColumnSpec{{Name: "id", HasComment: true}},
		},
	}
	for _, rule := range []audit.Rule{
		RequirePrimaryKey{},
		RequireTableComment{},
		RequireColumnComments{},
		LimitColumnCount{},
	} {
		if issues := rule.Check(context.Background(), ctx, stmt); len(issues) != 0 {
			t.Fatalf("%s: unexpected issues: %+v", rule.ID(), issues)
		}
	}
}
