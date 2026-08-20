package audit

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestRuleCatalogIsValid(t *testing.T) {
	if err := ValidateRuleCatalog(); err != nil {
		t.Fatal(err)
	}
	catalog := RuleCatalog()
	if len(catalog) != 76 {
		t.Fatalf("catalog contains %d rules, want 76; update RULE_CATALOG.md and this assertion together", len(catalog))
	}
	for _, definition := range catalog {
		if got, ok := RuleByCode(definition.Code); !ok || got != definition {
			t.Errorf("rule %s cannot be resolved consistently: got=%+v ok=%v", definition.Code, got, ok)
		}
	}
}

func TestLegacyDefaultsRegistersEveryCatalogRule(t *testing.T) {
	policy := LegacyDefaults()
	for _, definition := range RuleCatalog() {
		level, ok := policy.RuleLevels[definition.Code]
		if !ok {
			t.Errorf("catalog rule %s is missing from Policy.RuleLevels", definition.Code)
			continue
		}
		if level != definition.DefaultLevel {
			t.Errorf("catalog rule %s level=%d want=%d", definition.Code, level, definition.DefaultLevel)
		}
	}
}

func TestRuleCatalogDocumentationIsInSync(t *testing.T) {
	_, source, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate catalog test source")
	}
	content, err := os.ReadFile(filepath.Join(filepath.Dir(source), "..", "..", "docs", "RULE_CATALOG.md"))
	if err != nil {
		t.Fatal(err)
	}
	documented := make(map[string]bool)
	for _, line := range strings.Split(string(content), "\n") {
		fields := strings.Split(line, "|")
		if len(fields) > 2 {
			code := strings.TrimSpace(fields[1])
			if strings.HasPrefix(code, "GIP-") {
				documented[code] = true
			}
		}
	}
	for _, definition := range RuleCatalog() {
		if !documented[definition.Code] {
			t.Errorf("RULE_CATALOG.md is missing %s", definition.Code)
		}
		delete(documented, definition.Code)
	}
	for code := range documented {
		t.Errorf("RULE_CATALOG.md contains unknown rule %s", code)
	}
}
