package config

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/hanchuanchuan/goinception-plus/internal/audit"
	"github.com/hanchuanchuan/goinception-plus/internal/model"
)

func write(t *testing.T, text string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(p, []byte(text), 0600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestLegacyAuditConfigurationContract(t *testing.T) {
	incKeys := []string{"backup_host", "backup_password", "backup_port", "backup_user", "check_autoincrement_datatype", "check_column_comment", "check_dml_limit", "check_dml_orderby", "check_dml_where", "check_identifier", "check_identifier_lower", "check_implicit_type_conversion", "check_index_column_repeat", "check_primary_key", "check_table_comment", "default_charset", "enable_blob_type", "enable_column_charset", "enable_drop_table", "enable_foreign_key", "enable_identifer_keyword", "enable_json_type", "enable_partition_table", "enable_set_charset", "enable_set_collation", "enable_set_engine", "explain_rule", "general_log", "index_prefix", "lang", "lock_wait_timeout", "max_allowed_packet", "max_key_parts", "max_keys", "max_primary_key_parts", "skip_grant_table", "sql_mode", "sql_safe_updates", "support_charset", "support_engine", "table_prefix"}
	tags := map[string]bool{}
	typeOf := reflect.TypeOf(Inc{})
	for i := 0; i < typeOf.NumField(); i++ {
		tags[typeOf.Field(i).Tag.Get("toml")] = true
	}
	for _, key := range incKeys {
		if !tags[key] {
			t.Errorf("legacy inc key %s is not modeled", key)
		}
	}
	if len(incKeys) != 41 {
		t.Fatalf("fixture has %d inc keys, want 41", len(incKeys))
	}
	levels := levelBindings(&IncLevel{})
	if len(levels) != 52 {
		t.Fatalf("modeled %d canonical levels, want 52", len(levels))
	}
	for key := range levels {
		if legacyLevelRule[key] == "" {
			t.Errorf("legacy level %s has no RuleID", key)
		}
	}
}

func TestLegacyAliasesMapToCanonicalLevels(t *testing.T) {
	p := write(t, "[server]\nhost='127.0.0.1'\nport=4000\n[auth]\nusername='a'\npassword='b'\n[inc_level]\ner_column_must_have_comment=2\ner_insert_field=2\n")
	c, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if c.IncLevel.ColumnMustHaveComment != 2 || c.IncLevel.InsertField != 2 {
		t.Fatalf("aliases not normalized: %+v", c.IncLevel)
	}
}
func TestLoadMinimalAndMapLegacyRules(t *testing.T) {
	t.Setenv("GIP_TEST_PASSWORD", "secret")
	p := write(t, "[server]\nhost='127.0.0.1'\nport=4000\n[auth]\nusername='archery'\npassword_env='GIP_TEST_PASSWORD'\n[inc]\ncheck_dml_where=true\n[inc_level]\ner_no_where_condition=1\n")
	c, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	password, err := c.AuthPassword()
	if err != nil || password != "secret" {
		t.Fatalf("password=%q err=%v", password, err)
	}
	if c.Policy().Level(audit.RuleDMLRequireWhere) != model.SeverityWarning {
		t.Fatal("legacy level not mapped")
	}
}

func TestDistributedTemplatesLoad(t *testing.T) {
	for _, name := range []string{"config.toml", "config.toml.default", "config.toml.gip", "config.minimal.toml"} {
		path := filepath.Join("..", "..", "config", name)
		if _, err := Load(path); err != nil {
			t.Fatalf("load %s: %v", name, err)
		}
	}
}
func TestDefaultTemplateMatchesRequestedLegacyPolicy(t *testing.T) {
	c, err := Load(filepath.Join("..", "..", "config", "config.toml.default"))
	if err != nil {
		t.Fatal(err)
	}
	p := c.Policy()
	if !c.Inc.CheckIdentifier || !c.Inc.EnableBlobType || !c.Inc.EnableJSONType || !c.Inc.EnablePartition || !c.Inc.EnableDropTable {
		t.Fatalf("legacy switches not aligned: %+v", c.Inc)
	}
	if c.Inc.EnableSetEngine || c.Inc.EnableSelectStar || p.Level(audit.RuleTableMustHavePK) != model.SeverityError || p.Level(audit.RuleColumnMustComment) != model.SeverityError {
		t.Fatalf("legacy policy not aligned: %+v", p)
	}
}

func TestTemplatesRegisterEveryCatalogRule(t *testing.T) {
	for _, name := range []string{"config.toml.default", "config.toml.gip"} {
		c, err := Load(filepath.Join("..", "..", "config", name))
		if err != nil {
			t.Fatal(err)
		}
		policy := c.Policy()
		for _, definition := range audit.RuleCatalog() {
			if _, ok := policy.RuleLevels[definition.Code]; !ok {
				t.Errorf("%s does not register %s", name, definition.Code)
			}
		}
	}
}

func TestDefaultTemplatePopulatesLegacyPolicyAndNewLevels(t *testing.T) {
	c, err := Load(filepath.Join("..", "..", "config", "config.toml.default"))
	if err != nil {
		t.Fatal(err)
	}
	policy := c.Policy()
	checks := map[string]string{
		"enable_set_charset":             "true",
		"enable_set_engine":              "false",
		"check_implicit_type_conversion": "true",
		"sql_mode":                       "TRADITIONAL",
		"max_key_parts":                  "5",
	}
	for name, want := range checks {
		if got := policy.LegacyString(name); got != want {
			t.Errorf("Policy.Legacy[%q]=%q want=%q", name, got, want)
		}
	}
	for code, want := range map[string]model.Severity{
		audit.RuleDDLCantSetCharset:     model.SeverityError,
		audit.RuleDDLCantSetCollation:   model.SeverityError,
		audit.RuleDDLCantSetEngine:      model.SeverityError,
		audit.RuleDMLImplicitConversion: model.SeverityError,
	} {
		if got := policy.Level(code); got != want {
			t.Errorf("Policy.Level(%s)=%d want=%d", code, got, want)
		}
	}
}

func TestTemplatesDefaultToRejectEnumSetBitAsError(t *testing.T) {
	for _, name := range []string{"config.toml.default", "config.toml.gip"} {
		c, err := Load(filepath.Join("..", "..", "config", name))
		if err != nil {
			t.Fatal(err)
		}
		policy := c.Policy()
		if policy.LegacyBool("enable_enum_set_bit") {
			t.Errorf("%s unexpectedly enables ENUM/SET/BIT", name)
		}
		if got := policy.Level(audit.RuleDDLUseEnum); got != model.SeverityError {
			t.Errorf("%s ENUM/SET/BIT level=%d want=%d", name, got, model.SeverityError)
		}
	}
}
func TestUnknownRulePreventsStartup(t *testing.T) {
	p := write(t, "[server]\nhost='127.0.0.1'\nport=4000\n[auth]\nusername='a'\npassword='b'\n[inc]\ncheck_not_implemented=true\n")
	_, err := Load(p)
	if err == nil || !strings.Contains(err.Error(), "check_not_implemented") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGIPRuleIDsOverrideLegacyLevels(t *testing.T) {
	p := write(t, "[server]\nhost='127.0.0.1'\nport=4000\n[auth]\nusername='a'\npassword='b'\n[inc_level]\ner_table_must_have_pk=1\n[rules]\n'GIP-DDL-CT-001'=2\n")
	c, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if got := c.Policy().Level(audit.RuleTableMustHavePK); got != model.SeverityError {
		t.Fatalf("GIP RuleID must override legacy level, got %d", got)
	}
}

func TestGIPRuleIDsRejectUnknownAndInvalidLevel(t *testing.T) {
	base := "[server]\nhost='127.0.0.1'\nport=4000\n[auth]\nusername='a'\npassword='b'\n[rules]\n"
	for _, body := range []string{"'GIP-NOT-REAL-001'=2\n", "'GIP-DDL-CT-001'=3\n"} {
		if _, err := Load(write(t, base+body)); err == nil {
			t.Fatalf("invalid rules entry accepted: %s", body)
		}
	}
}
