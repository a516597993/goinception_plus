package rules

import (
	"context"
	"testing"

	"github.com/hanchuanchuan/goinception-plus/internal/audit"
	"github.com/hanchuanchuan/goinception-plus/internal/model"
	parseradapter "github.com/hanchuanchuan/goinception-plus/internal/parser"
)

func behaviorPolicy() audit.Policy {
	p := audit.LegacyDefaults()
	p.Legacy = map[string]string{
		"enable_column_charset":         "true",
		"enable_blob_type":              "true",
		"enable_json_type":              "true",
		"enable_enum_set_bit":           "true",
		"enable_nullable":               "true",
		"enable_timestamp_type":         "true",
		"enable_autoincrement_unsigned": "false",
		"check_autoincrement_datatype":  "true",
		"check_timestamp_count":         "false",
		"check_column_type_change":      "true",
		"max_char_length":               "0",
		"sql_mode":                      "TRADITIONAL",
	}
	return p
}

func legacyIssues(t *testing.T, sql string, policy audit.Policy, metadata audit.MetadataProvider) []model.AuditIssue {
	t.Helper()
	parsed, err := parseradapter.New().Parse(sql, policy.LegacyString("sql_mode"))
	if err != nil {
		t.Fatalf("parse %q: %v", sql, err)
	}
	ctx := &audit.Context{Policy: policy, Metadata: metadata, SeenAlterTables: map[string]int{}}
	engine := audit.NewEngine(LegacyDDL{})
	var issues []model.AuditIssue
	for _, statement := range parsed.Statements {
		issues = append(issues, engine.Check(context.Background(), ctx, statement)...)
	}
	return issues
}

func findIssue(issues []model.AuditIssue, rule string) (model.AuditIssue, bool) {
	for _, issue := range issues {
		if issue.RuleID == rule {
			return issue, true
		}
	}
	return model.AuditIssue{}, false
}

func assertLegacyRuleMatrix(t *testing.T, rule, triggerSQL, passSQL string, configure func(*audit.Policy), metadata audit.MetadataProvider) {
	t.Helper()
	for _, level := range []model.Severity{model.SeverityNone, model.SeverityWarning, model.SeverityError} {
		p := behaviorPolicy()
		configure(&p)
		p.RuleLevels[rule] = level
		issue, found := findIssue(legacyIssues(t, triggerSQL, p, metadata), rule)
		if level == model.SeverityNone {
			if found {
				t.Errorf("rule %s level=0 was not filtered: %+v", rule, issue)
			}
			continue
		}
		if !found || issue.Level != level {
			t.Errorf("rule %s level=%d issue=%+v found=%v", rule, level, issue, found)
		}
		if definition, ok := audit.RuleByCode(rule); ok && issue.LegacyKey != definition.LegacyKey {
			t.Errorf("rule %s LegacyKey=%q want=%q", rule, issue.LegacyKey, definition.LegacyKey)
		}
	}
	p := behaviorPolicy()
	configure(&p)
	p.RuleLevels[rule] = model.SeverityError
	if issue, found := findIssue(legacyIssues(t, passSQL, p, metadata), rule); found {
		t.Errorf("rule %s false positive for %q: %+v", rule, passSQL, issue)
	}
}

func TestLegacyDDLColumnRuleBehaviorMatrix(t *testing.T) {
	tests := []struct {
		name, rule, trigger, pass string
		configure                 func(*audit.Policy)
	}{
		{"char to varchar", audit.RuleDDLCharToVarchar, "CREATE TABLE t(c CHAR(20) DEFAULT '')", "CREATE TABLE t(c CHAR(10) DEFAULT '')", func(p *audit.Policy) { p.Legacy["max_char_length"] = "10" }},
		{"column charset", audit.RuleDDLColumnCharset, "CREATE TABLE t(c VARCHAR(10) CHARACTER SET utf8mb4 DEFAULT '')", "CREATE TABLE t(c VARCHAR(10) DEFAULT '')", func(p *audit.Policy) { p.Legacy["enable_column_charset"] = "false" }},
		{"nullable", audit.RuleDDLNullable, "CREATE TABLE t(c INT DEFAULT 0)", "CREATE TABLE t(c INT NOT NULL DEFAULT 0)", func(p *audit.Policy) { p.Legacy["enable_nullable"] = "false" }},
		{"missing default", audit.RuleDDLAddDefault, "CREATE TABLE t(c INT NOT NULL)", "CREATE TABLE t(c INT NOT NULL DEFAULT 0)", func(*audit.Policy) {}},
		{"datetime default", audit.RuleDDLDatetimeDefault, "CREATE TABLE t(c DATETIME NOT NULL)", "CREATE TABLE t(c DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP)", func(*audit.Policy) {}},
		{"incorrect datetime", audit.RuleDDLIncorrectDatetime, "CREATE TABLE t(c DATETIME NOT NULL DEFAULT '2024-02-30')", "CREATE TABLE t(c DATETIME NOT NULL DEFAULT '2024-02-29')", func(p *audit.Policy) { p.Legacy["sql_mode"] = "TRADITIONAL" }},
		{"blob default", audit.RuleDDLBlobDefault, "CREATE TABLE t(c TEXT DEFAULT 'x')", "CREATE TABLE t(c TEXT)", func(*audit.Policy) {}},
		{"json disabled", audit.RuleDDLJSONType, "CREATE TABLE t(c JSON)", "CREATE TABLE t(c VARCHAR(20) DEFAULT '')", func(p *audit.Policy) { p.Legacy["enable_json_type"] = "false" }},
		{"text disabled", audit.RuleDDLUseTextBlob, "CREATE TABLE t(c TEXT)", "CREATE TABLE t(c VARCHAR(20) DEFAULT '')", func(p *audit.Policy) { p.Legacy["enable_blob_type"] = "false" }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assertLegacyRuleMatrix(t, tc.rule, tc.trigger, tc.pass, tc.configure, nil)
		})
	}
}

func TestLegacyDDLAutoIncrementAndTimestampBehaviorMatrix(t *testing.T) {
	tests := []struct {
		name, rule, trigger, pass string
		configure                 func(*audit.Policy)
	}{
		{"auto increment type", audit.RuleDDLAutoInteger, "CREATE TABLE t(c VARCHAR(20) NOT NULL AUTO_INCREMENT, KEY idx_c(c))", "CREATE TABLE t(c BIGINT NOT NULL AUTO_INCREMENT, KEY idx_c(c))", func(*audit.Policy) {}},
		{"auto increment unsigned", audit.RuleDDLAutoUnsigned, "CREATE TABLE t(id BIGINT NOT NULL AUTO_INCREMENT, PRIMARY KEY(id))", "CREATE TABLE t(id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT, PRIMARY KEY(id))", func(p *audit.Policy) { p.Legacy["enable_autoincrement_unsigned"] = "true" }},
		{"auto increment name", audit.RuleDDLAutoName, "CREATE TABLE t(seq BIGINT NOT NULL AUTO_INCREMENT, PRIMARY KEY(seq))", "CREATE TABLE t(id BIGINT NOT NULL AUTO_INCREMENT, PRIMARY KEY(id))", func(*audit.Policy) {}},
		{"auto increment initial", audit.RuleDDLAutoInitial, "CREATE TABLE t(id BIGINT PRIMARY KEY) AUTO_INCREMENT=9", "CREATE TABLE t(id BIGINT PRIMARY KEY) AUTO_INCREMENT=1", func(*audit.Policy) {}},
		{"automatic timestamps", audit.RuleDDLTooManyTimestamps, "CREATE TABLE t(a TIMESTAMP DEFAULT CURRENT_TIMESTAMP, b TIMESTAMP DEFAULT CURRENT_TIMESTAMP)", "CREATE TABLE t(a TIMESTAMP DEFAULT CURRENT_TIMESTAMP, b TIMESTAMP DEFAULT '2024-01-01 00:00:00')", func(p *audit.Policy) { p.Legacy["check_timestamp_count"] = "true" }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assertLegacyRuleMatrix(t, tc.rule, tc.trigger, tc.pass, tc.configure, nil)
		})
	}
}

func TestLegacyDDLAlterAndViewBehaviorMatrix(t *testing.T) {
	metadata := dmlMetadata{table: audit.Table{Schema: "app", Name: "t", Columns: []audit.Column{{Name: "c", ColumnType: "varchar(50)"}}}}
	tests := []struct {
		name, rule, trigger, pass string
		metadata                  audit.MetadataProvider
	}{
		{"column position", audit.RuleDDLColumnPosition, "ALTER TABLE app.t ADD COLUMN n INT DEFAULT 0 FIRST", "ALTER TABLE app.t ADD COLUMN n INT DEFAULT 0", nil},
		{"change column", audit.RuleDDLCantChange, "ALTER TABLE app.t CHANGE COLUMN c n VARCHAR(50) DEFAULT ''", "ALTER TABLE app.t MODIFY COLUMN c VARCHAR(50) DEFAULT ''", metadata},
		{"unsafe type change", audit.RuleDDLColumnTypeChange, "ALTER TABLE app.t MODIFY COLUMN c VARCHAR(10) DEFAULT ''", "ALTER TABLE app.t MODIFY COLUMN c VARCHAR(100) DEFAULT ''", metadata},
		{"alter once", audit.RuleDDLAlterOnce, "ALTER TABLE app.t ADD COLUMN a INT DEFAULT 0; ALTER TABLE app.t ADD COLUMN b INT DEFAULT 0", "ALTER TABLE app.t ADD COLUMN a INT DEFAULT 0", nil},
		{"create view", audit.RuleDDLView, "CREATE VIEW v AS SELECT 1 AS c", "CREATE TABLE t(c INT DEFAULT 0)", nil},
		{"drop view", audit.RuleDDLView, "DROP VIEW v", "CREATE TABLE t(c INT DEFAULT 0)", nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assertLegacyRuleMatrix(t, tc.rule, tc.trigger, tc.pass, func(*audit.Policy) {}, tc.metadata)
		})
	}
}

func TestAlterViewReturnsExplicitParserUnsupportedError(t *testing.T) {
	if _, err := parseradapter.New().Parse("ALTER VIEW v AS SELECT 1 AS c", ""); err == nil {
		t.Fatal("TiDB 8.5 parser limitation must not silently admit ALTER VIEW")
	}
}

func TestEnumSetBitSwitchAndLevelMatrix(t *testing.T) {
	for _, columnType := range []string{"ENUM('a')", "SET('a')", "BIT(1)"} {
		t.Run(columnType, func(t *testing.T) {
			sql := "CREATE TABLE t(c " + columnType + " DEFAULT NULL)"
			for _, enabled := range []bool{false, true} {
				for _, level := range []model.Severity{model.SeverityNone, model.SeverityWarning, model.SeverityError} {
					p := behaviorPolicy()
					if enabled {
						p.Legacy["enable_enum_set_bit"] = "true"
					} else {
						p.Legacy["enable_enum_set_bit"] = "false"
					}
					p.RuleLevels[audit.RuleDDLUseEnum] = level
					issue, found := findIssue(legacyIssues(t, sql, p, nil), audit.RuleDDLUseEnum)
					want := !enabled && level != model.SeverityNone
					if found != want {
						t.Fatalf("enabled=%v level=%d issue=%+v found=%v want=%v", enabled, level, issue, found, want)
					}
					if found && issue.Level != level {
						t.Fatalf("enabled=%v level=%d issue=%+v", enabled, level, issue)
					}
				}
			}
		})
	}
}

func TestCTETargetVersionBehavior(t *testing.T) {
	tests := []string{
		"WITH c AS (SELECT id FROM app.t) SELECT id FROM c",
		"WITH c AS (SELECT id FROM app.t) UPDATE app.t SET id=id WHERE id IN (SELECT id FROM c)",
		"WITH c AS (SELECT id FROM app.t) DELETE FROM app.t WHERE id IN (SELECT id FROM c)",
	}
	metadata := dmlMetadata{table: audit.Table{Schema: "app", Name: "t", Columns: []audit.Column{{Name: "id", ColumnType: "bigint"}}}}
	for _, sql := range tests {
		parsed, err := parseradapter.New().Parse(sql, "")
		if err != nil {
			t.Fatalf("parse %q: %v", sql, err)
		}
		for _, version := range []string{"5.7.44", "8.0.39"} {
			ctx := &audit.Context{Policy: behaviorPolicy(), Metadata: metadata, Server: audit.ServerInfo{Version: version}}
			issues := audit.NewEngine(DMLSafety{}).Check(context.Background(), ctx, parsed.Statements[0])
			_, found := findIssue(issues, audit.RuleTargetVersion)
			if want := version == "5.7.44"; found != want {
				t.Errorf("version=%s sql=%q target-version issue=%v want=%v issues=%+v", version, sql, found, want, issues)
			}
		}
	}
}
