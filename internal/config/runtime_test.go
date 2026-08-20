package config

import (
	"testing"

	"github.com/hanchuanchuan/goinception-plus/internal/audit"
	"github.com/hanchuanchuan/goinception-plus/internal/model"
)

func TestRuntimeSetAndPolicySnapshot(t *testing.T) {
	r := NewRuntime(Defaults())
	before := r.Policy()
	if err := r.SetVariable("check_primary_key", "ON"); err != nil {
		t.Fatal(err)
	}
	if err := r.SetLevel("er_table_must_have_pk", 2); err != nil {
		t.Fatal(err)
	}
	after := r.Policy()
	if before.CheckPrimaryKey || !after.CheckPrimaryKey {
		t.Fatal("runtime policy did not use immutable snapshots")
	}
	if after.Level(audit.RuleTableMustHavePK) != model.SeverityError {
		t.Fatal("runtime level was not applied")
	}
	if err := r.SetVariable("unknown_old_key", "1"); err == nil {
		t.Fatal("unknown variable was accepted")
	}
	if err := r.SetLevel("er_table_must_have_pk", 3); err == nil {
		t.Fatal("invalid level was accepted")
	}
}

func TestRuntimeMasksBackupPassword(t *testing.T) {
	cfg := Defaults()
	cfg.Backup.Password = "secret"
	for _, v := range NewRuntime(cfg).Variables() {
		if v.Name == "backup_password" && v.Value != "******" {
			t.Fatalf("password leaked: %q", v.Value)
		}
	}
}

func TestRuntimeExposesAndSetsAllLegacyLevels(t *testing.T) {
	r := NewRuntime(Defaults())
	if got := len(r.Levels()); got != 52 {
		t.Fatalf("levels=%d want=52", got)
	}
	for _, name := range []string{"er_column_have_no_comment", "er_column_must_have_comment", "er_with_insert_field", "er_insert_field", "er_no_where_condition", "er_sql_no_where", "er_implicit_type_conversion"} {
		if err := r.SetLevel(name, 2); err != nil {
			t.Fatalf("set %s: %v", name, err)
		}
	}
	if r.Policy().Level(audit.RuleDMLImplicitConversion) != model.SeverityError {
		t.Fatal("implicit conversion level not applied")
	}
}

func TestRuntimeSetLevelAcceptsGIPRuleIDAndKeepsLegacyViewEffective(t *testing.T) {
	r := NewRuntime(Defaults())
	if err := r.SetLevel("gip-ddl-ct-001", 2); err != nil {
		t.Fatal(err)
	}
	if r.Policy().Level(audit.RuleTableMustHavePK) != model.SeverityError {
		t.Fatal("GIP RuleID level was not applied")
	}
	if err := r.SetLevel("er_table_must_have_pk", 1); err != nil {
		t.Fatal(err)
	}
	if r.Policy().Level(audit.RuleTableMustHavePK) != model.SeverityWarning {
		t.Fatal("legacy update did not modify an existing GIP override")
	}
}
