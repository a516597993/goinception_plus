package rules

import (
	"context"
	"testing"

	"github.com/hanchuanchuan/goinception-plus/internal/audit"
	"github.com/hanchuanchuan/goinception-plus/internal/model"
)

func TestDDLSafetyRejectsDangerousAndInvalidCreate(t *testing.T) {
	p := audit.LegacyDefaults()
	ctx := &audit.Context{Policy: p}
	cases := []struct {
		stmt model.Statement
		rule string
	}{
		{model.Statement{Sequence: 1, DDL: &model.DDLSpec{Action: model.DDLDropDatabase}}, audit.RuleDDLDropDatabase},
		{model.Statement{Sequence: 1, DDL: &model.DDLSpec{Action: model.DDLDropTable}}, audit.RuleDDLDropTable},
		{model.Statement{Sequence: 1, DDL: &model.DDLSpec{Action: model.DDLTruncateTable}}, audit.RuleDDLTruncate},
		{model.Statement{Sequence: 1, DDL: &model.DDLSpec{Action: model.DDLCreateTable, HasForeignKey: true}}, audit.RuleDDLForeignKey},
		{model.Statement{Sequence: 1, DDL: &model.DDLSpec{Action: model.DDLCreateTable, Partitioned: true}}, audit.RuleDDLPartition},
	}
	for _, tc := range cases {
		issues := (DDLSafety{}).Check(context.Background(), ctx, tc.stmt)
		if len(issues) == 0 || issues[0].RuleID != tc.rule || issues[0].Level != model.SeverityError {
			t.Fatalf("want %s error, got %+v", tc.rule, issues)
		}
	}
}

func TestDMLPolicyRules(t *testing.T) {
	p := audit.LegacyDefaults()
	p.MaxInsertRows, p.CheckDMLLimit, p.CheckDMLOrderBy, p.CheckInsertField, p.AllowSelectStar = 1, true, true, true, false
	ctx := &audit.Context{Policy: p, Metadata: dmlMetadata{table: audit.Table{Indexes: []audit.Index{{Name: "PRIMARY", Primary: true, Columns: []string{"id"}}}, Columns: []audit.Column{{Name: "id"}}}}}
	stmt := model.Statement{Sequence: 1, Kind: model.StatementInsert, DML: &model.DMLSpec{Tables: []model.TableRef{{Schema: "app", Name: "t"}}, ValueRows: 2}}
	issues := (DMLSafety{}).Check(context.Background(), ctx, stmt)
	want := map[string]bool{audit.RuleDMLMaxInsert: false, audit.RuleDMLInsertField: false}
	for _, issue := range issues {
		if _, ok := want[issue.RuleID]; ok {
			want[issue.RuleID] = true
		}
	}
	for id, found := range want {
		if !found {
			t.Errorf("missing %s in %+v", id, issues)
		}
	}
}

func TestDDLVersionMatrix(t *testing.T) {
	checks := []struct {
		name    string
		version string
		ddl     *model.DDLSpec
		want    bool
	}{
		{"57 rename column", "5.7.44", &model.DDLSpec{Action: model.DDLAlterTable, AlterOperations: []model.AlterOperation{{Action: model.AlterRenameColumn}}}, true},
		{"80 rename column", "8.0.39", &model.DDLSpec{Action: model.DDLAlterTable, AlterOperations: []model.AlterOperation{{Action: model.AlterRenameColumn}}}, false},
		{"57 0900 collation", "5.7.44", &model.DDLSpec{Action: model.DDLCreateTable, Collation: "utf8mb4_0900_ai_ci"}, true},
		{"80 0900 collation", "8.0.39", &model.DDLSpec{Action: model.DDLCreateTable, Collation: "utf8mb4_0900_ai_ci"}, false},
		{"57 expression index", "5.7.44", &model.DDLSpec{Action: model.DDLCreateTable, Indexes: []model.IndexSpec{{Name: "idx_expr", Expressions: []string{"lower(name)"}}}, Columns: []model.ColumnSpec{{Name: "name"}}}, true},
		{"8012 expression index", "8.0.12", &model.DDLSpec{Action: model.DDLCreateTable, Indexes: []model.IndexSpec{{Name: "idx_expr", Expressions: []string{"lower(name)"}}}, Columns: []model.ColumnSpec{{Name: "name"}}}, true},
		{"8013 expression index", "8.0.13", &model.DDLSpec{Action: model.DDLCreateTable, Indexes: []model.IndexSpec{{Name: "idx_expr", Expressions: []string{"lower(name)"}}}, Columns: []model.ColumnSpec{{Name: "name"}}}, false},
	}
	for _, tc := range checks {
		t.Run(tc.name, func(t *testing.T) {
			ctx := &audit.Context{Policy: audit.LegacyDefaults(), Server: audit.ServerInfo{Version: tc.version}}
			issues := (DDLSafety{}).Check(context.Background(), ctx, model.Statement{Sequence: 1, DDL: tc.ddl})
			found := false
			for _, issue := range issues {
				if issue.RuleID == audit.RuleTargetVersion {
					found = true
				}
			}
			if found != tc.want {
				t.Fatalf("RuleTargetVersion found=%v want=%v, issues=%+v", found, tc.want, issues)
			}
		})
	}
}
