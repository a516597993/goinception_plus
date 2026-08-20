package rules

import (
	"context"
	"testing"

	"github.com/hanchuanchuan/goinception-plus/internal/audit"
	"github.com/hanchuanchuan/goinception-plus/internal/model"
)

func legacyPolicy() audit.Policy {
	p := audit.LegacyDefaults()
	p.Legacy = map[string]string{
		"check_identifier": "true", "enable_identifer_keyword": "true", "check_identifier_lower": "false",
		"enable_set_charset": "true", "enable_set_collation": "true", "enable_set_engine": "true",
		"enable_column_charset": "false", "enable_blob_type": "false", "enable_json_type": "false",
		"check_autoincrement_datatype": "true", "max_key_parts": "2", "max_primary_key_parts": "1", "max_keys": "2",
		"index_prefix": "idx_", "uniq_index_prefix": "uniq_", "support_charset": "utf8mb4", "support_engine": "innodb",
		"check_index_column_repeat": "true", "check_column_type_change": "true",
		"check_index_prefix": "true", "enable_nullable": "false", "enable_enum_set_bit": "false",
	}
	for _, id := range []string{audit.RuleDDLInvalidIdentifier, audit.RuleDDLIdentifierKeyword, audit.RuleDDLIndexPrefix, audit.RuleDDLUniqueIndexPrefix, audit.RuleDDLMaxKeyParts, audit.RuleDDLMaxPrimaryParts, audit.RuleDDLMaxKeys, audit.RuleDDLAutoInteger, audit.RuleDDLBlobDefault, audit.RuleDDLJSONType, audit.RuleDDLUseTextBlob, audit.RuleDDLUseEnum, audit.RuleDDLNullable, audit.RuleDDLAddDefault, audit.RuleDDLColumnCharset, audit.RuleDDLAlterOnce, audit.RuleDDLCantChange, audit.RuleDDLColumnPosition} {
		p.RuleLevels[id] = model.SeverityError
	}
	return p
}

func issueIDs(issues []model.AuditIssue) map[string]bool {
	out := map[string]bool{}
	for _, issue := range issues {
		out[issue.RuleID] = true
	}
	return out
}

func TestLegacyDDLFieldIdentifierAndIndexRules(t *testing.T) {
	p := legacyPolicy()
	ctx := &audit.Context{Policy: p, SeenAlterTables: map[string]int{}}
	stmt := model.Statement{Sequence: 1, DDL: &model.DDLSpec{Action: model.DDLCreateTable, Schema: "app", Table: "select", FullyProjected: true,
		Columns: []model.ColumnSpec{{Name: "bad-name", Type: "text", Nullable: true, HasDefault: true, DefaultExpression: "'x'"}, {Name: "payload", Type: "json", Nullable: true}},
		Indexes: []model.IndexSpec{{Name: "bad", Columns: []string{"bad-name", "payload", "other"}}, {Name: "uq_bad", Unique: true, Columns: []string{"payload"}}}}}
	got := issueIDs((LegacyDDL{}).Check(context.Background(), ctx, stmt))
	for _, id := range []string{audit.RuleDDLIdentifierKeyword, audit.RuleDDLInvalidIdentifier, audit.RuleDDLBlobDefault, audit.RuleDDLUseTextBlob, audit.RuleDDLJSONType, audit.RuleDDLNullable, audit.RuleDDLIndexPrefix, audit.RuleDDLUniqueIndexPrefix, audit.RuleDDLMaxKeyParts} {
		if !got[id] {
			t.Errorf("missing %s in %+v", id, got)
		}
	}
}

func TestLegacyDDLLevelZeroSuppressesIssue(t *testing.T) {
	p := legacyPolicy()
	p.RuleLevels[audit.RuleDDLUseEnum] = model.SeverityNone
	ctx := &audit.Context{Policy: p, SeenAlterTables: map[string]int{}}
	issues := (LegacyDDL{}).Check(context.Background(), ctx, model.Statement{Sequence: 1, DDL: &model.DDLSpec{Action: model.DDLCreateTable, Table: "t", Columns: []model.ColumnSpec{{Name: "kind", Type: "enum('a')", Nullable: false, HasDefault: true}}}})
	engine := audit.NewEngine(LegacyDDL{})
	filtered := engine.Check(context.Background(), ctx, model.Statement{Sequence: 1, DDL: &model.DDLSpec{Action: model.DDLCreateTable, Table: "t", Columns: []model.ColumnSpec{{Name: "kind", Type: "enum('a')", Nullable: false, HasDefault: true}}}})
	if !issueIDs(issues)[audit.RuleDDLUseEnum] {
		t.Fatal("raw rule should emit level-zero issue")
	}
	if issueIDs(filtered)[audit.RuleDDLUseEnum] {
		t.Fatal("engine must suppress level-zero issue")
	}
}

func TestLegacyDDLAlterOnceChangeAndPosition(t *testing.T) {
	p := legacyPolicy()
	ctx := &audit.Context{Policy: p, SeenAlterTables: map[string]int{}}
	stmt := model.Statement{Sequence: 1, DDL: &model.DDLSpec{Action: model.DDLAlterTable, Schema: "app", Table: "t", AlterOperations: []model.AlterOperation{{Action: model.AlterChangeColumn, Name: "a", Columns: []model.ColumnSpec{{Name: "b", Type: "int", PositionChanged: true}}}}}}
	first := issueIDs((LegacyDDL{}).Check(context.Background(), ctx, stmt))
	second := issueIDs((LegacyDDL{}).Check(context.Background(), ctx, stmt))
	if !first[audit.RuleDDLCantChange] || !first[audit.RuleDDLColumnPosition] {
		t.Fatalf("first ALTER issues=%+v", first)
	}
	if !second[audit.RuleDDLAlterOnce] {
		t.Fatalf("second ALTER issues=%+v", second)
	}
}

func TestLegacyDDLAlterIndexChecksExistingMetadata(t *testing.T) {
	p := legacyPolicy()
	p.Legacy["max_keys"] = "1"
	ctx := &audit.Context{Policy: p, SeenAlterTables: map[string]int{}, Metadata: dmlMetadata{table: audit.Table{Schema: "app", Name: "t", Columns: []audit.Column{{Name: "id", ColumnType: "bigint"}}, Indexes: []audit.Index{{Name: "PRIMARY", Columns: []string{"id"}, Primary: true, Unique: true}}}}}
	stmt := model.Statement{Sequence: 1, DDL: &model.DDLSpec{Action: model.DDLAlterTable, Schema: "app", Table: "t", AlterOperations: []model.AlterOperation{{Action: model.AlterAddIndex, Index: &model.IndexSpec{Name: "idx_id", Columns: []string{"id"}}}}}}
	ids := issueIDs((LegacyDDL{}).Check(context.Background(), ctx, stmt))
	if !ids[audit.RuleDDLMaxKeys] || !ids[audit.RuleDDLIndexRepeat] {
		t.Fatalf("issues=%+v", ids)
	}
}

func TestLegacyDDLPrimaryNameIsNotCheckedAsKeyword(t *testing.T) {
	p := legacyPolicy()
	ctx := &audit.Context{Policy: p, SeenAlterTables: map[string]int{}}
	stmt := model.Statement{Sequence: 1, DDL: &model.DDLSpec{
		Action: model.DDLCreateTable,
		Table:  "t",
		Indexes: []model.IndexSpec{{Name: "PRIMARY", Primary: true, Unique: true, Columns: []string{"id"}}},
	}}
	if ids := issueIDs((LegacyDDL{}).Check(context.Background(), ctx, stmt)); ids[audit.RuleDDLIdentifierKeyword] {
		t.Fatalf("PRIMARY KEY must not be treated as a user-defined keyword identifier: %+v", ids)
	}
}
