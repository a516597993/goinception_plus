package rules

import (
	"context"
	"testing"

	"github.com/hanchuanchuan/goinception-plus/internal/audit"
	"github.com/hanchuanchuan/goinception-plus/internal/model"
	parseradapter "github.com/hanchuanchuan/goinception-plus/internal/parser"
)

type dmlMetadata struct{ table audit.Table }

func (d dmlMetadata) LoadServerInfo(context.Context) (audit.ServerInfo, error) {
	return audit.ServerInfo{}, nil
}
func (d dmlMetadata) LoadSchema(context.Context, string) (audit.Schema, error) {
	return audit.Schema{}, nil
}
func (d dmlMetadata) LoadTable(context.Context, string, string) (audit.Table, error) {
	return d.table, nil
}

func TestDMLSafetyRequiresWhereAndBackupKey(t *testing.T) {
	ctx := &audit.Context{
		Request: model.RequestOptions{Backup: true}, Policy: audit.LegacyDefaults(),
		Metadata: dmlMetadata{table: audit.Table{
			Schema: "app", Name: "t", Columns: []audit.Column{{Name: "c"}},
		}},
	}
	stmt := model.Statement{
		Sequence: 1, Kind: model.StatementUpdate,
		DML: &model.DMLSpec{Tables: []model.TableRef{{Schema: "app", Name: "t"}}, Columns: []string{"c"}, FullyProjected: true},
	}
	issues := DMLSafety{}.Check(context.Background(), ctx, stmt)
	if len(issues) != 2 {
		t.Fatalf("got %+v", issues)
	}
}

func TestDMLSafetyAcceptsWhereAndPrimaryKey(t *testing.T) {
	ctx := &audit.Context{
		Request: model.RequestOptions{Backup: true}, Policy: audit.LegacyDefaults(),
		Metadata: dmlMetadata{table: audit.Table{
			Schema: "app", Name: "t", Columns: []audit.Column{{Name: "id"}, {Name: "c"}},
			Indexes: []audit.Index{{Name: "PRIMARY", Primary: true, Columns: []string{"id"}}},
		}},
	}
	stmt := model.Statement{
		Kind: model.StatementUpdate,
		DML:  &model.DMLSpec{Tables: []model.TableRef{{Schema: "app", Name: "t"}}, Columns: []string{"c"}, HasWhere: true, FullyProjected: true},
	}
	if issues := (DMLSafety{}).Check(context.Background(), ctx, stmt); len(issues) != 0 {
		t.Fatalf("got %+v", issues)
	}
}

func TestDMLLegacyExpressionRules(t *testing.T) {
	p := audit.LegacyDefaults()
	p.Legacy["check_implicit_type_conversion"] = "true"
	p.RuleLevels[audit.RuleDMLImplicitConversion] = model.SeverityError
	p.RuleLevels[audit.RuleDMLOrderByRand] = model.SeverityError
	p.RuleLevels[audit.RuleDMLWrongAnd] = model.SeverityWarning
	ctx := &audit.Context{Policy: p, Metadata: dmlMetadata{table: audit.Table{Columns: []audit.Column{{Name: "code", ColumnType: "varchar(20)"}}}}}
	stmt := model.Statement{Sequence: 1, Kind: model.StatementUpdate, DML: &model.DMLSpec{Tables: []model.TableRef{{Schema: "app", Name: "t"}}, HasWhere: true, OrderByRand: true, WrongAndExpr: true, Comparisons: []model.ComparisonSpec{{Column: model.ColumnRef{Name: "code"}, LiteralKind: model.LiteralSignedInteger}}}}
	ids := map[string]bool{}
	for _, issue := range (DMLSafety{}).Check(context.Background(), ctx, stmt) {
		ids[issue.RuleID] = true
	}
	for _, id := range []string{audit.RuleDMLImplicitConversion, audit.RuleDMLOrderByRand, audit.RuleDMLWrongAnd} {
		if !ids[id] {
			t.Errorf("missing %s in %+v", id, ids)
		}
	}
}

func TestImplicitConversionSemanticMatrix(t *testing.T) {
	tests := []struct {
		column string
		kind   model.LiteralKind
		want   bool
	}{
		{"bigint", model.LiteralString, true},
		{"decimal(10,2)", model.LiteralBinary, true},
		{"int", model.LiteralSignedInteger, false},
		{"double", model.LiteralDecimal, false},
		{"varchar(20)", model.LiteralSignedInteger, true},
		{"datetime", model.LiteralFloat, true},
		{"json", model.LiteralBoolean, true},
		{"varchar(20)", model.LiteralString, false},
		{"datetime", model.LiteralTemporal, false},
		{"int", model.LiteralNull, false},
		{"varchar(20)", model.LiteralUnknown, false},
	}
	for _, tc := range tests {
		if got := implicitConversion(tc.column, tc.kind); got != tc.want {
			t.Errorf("implicitConversion(%q, %q)=%v want=%v", tc.column, tc.kind, got, tc.want)
		}
	}
}

func TestImplicitConversionWithParsedAliasesAndReversedOperands(t *testing.T) {
	p := audit.LegacyDefaults()
	p.Legacy["check_implicit_type_conversion"] = "true"
	p.RuleLevels[audit.RuleDMLImplicitConversion] = model.SeverityError
	metadata := dmlMetadata{table: audit.Table{Columns: []audit.Column{{Name: "id", ColumnType: "bigint"}, {Name: "code", ColumnType: "varchar(20)"}}}}
	queries := []string{
		"UPDATE app.a AS a JOIN app.b AS b ON a.id=b.id SET a.id=a.id WHERE b.code=1",
		"UPDATE app.a AS a SET a.id=a.id WHERE '1'=a.id",
	}
	for _, sql := range queries {
		parsed, err := parseradapter.New().Parse(sql, "")
		if err != nil {
			t.Fatal(err)
		}
		ctx := &audit.Context{Policy: p, Metadata: metadata}
		issues := DMLSafety{}.Check(context.Background(), ctx, parsed.Statements[0])
		found := false
		for _, issue := range issues {
			if issue.RuleID == audit.RuleDMLImplicitConversion {
				found = true
			}
		}
		if !found {
			t.Errorf("parsed SQL did not trigger implicit conversion: %s; comparisons=%+v issues=%+v", sql, parsed.Statements[0].DML.Comparisons, issues)
		}
	}
}
