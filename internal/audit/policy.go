package audit

import (
	"strconv"

	"github.com/hanchuanchuan/goinception-plus/internal/model"
)

// Policy mirrors the first legacy DDL switches without using global state.
type Policy struct {
	CheckPrimaryKey    bool
	CheckTableComment  bool
	CheckColumnComment bool
	MaxColumnCount     int
	RequireDMLWhere    bool
	RequireBackupKey   bool
	MaxAffectedRows    int64
	MaxInsertRows      int
	CheckDMLLimit      bool
	CheckDMLOrderBy    bool
	CheckInsertField   bool
	AllowSelectStar    bool
	AllowForeignKey    bool
	AllowPartition     bool
	AllowDropTable     bool
	AllowDropDatabase  bool
	AllowTruncate      bool
	RequiredEngine     string
	RuleLevels         map[string]model.Severity
	Legacy             map[string]string
}

func LegacyDefaults() Policy {
	p := Policy{
		RequireDMLWhere: true, RequireBackupKey: true, AllowSelectStar: true, RequiredEngine: "innodb",
		Legacy: map[string]string{}, RuleLevels: map[string]model.Severity{},
	}
	// The catalog owns the default level for every registered rule. Building
	// the policy baseline from it prevents newly added system rules (which do
	// not necessarily have a legacy inc_level key) from silently falling back
	// to Warning.
	for _, definition := range RuleCatalog() {
		p.RuleLevels[definition.Code] = definition.DefaultLevel
	}
	return p
}

func (p Policy) LegacyBool(name string) bool {
	switch p.Legacy[name] {
	case "1", "true", "on":
		return true
	default:
		return false
	}
}

func (p Policy) LegacyInt(name string) int {
	v, _ := strconv.Atoi(p.Legacy[name])
	return v
}

func (p Policy) LegacyString(name string) string { return p.Legacy[name] }

func (p Policy) Level(ruleID string) model.Severity {
	if level, ok := p.RuleLevels[ruleID]; ok {
		return level
	}
	if definition, ok := RuleByCode(ruleID); ok {
		return definition.DefaultLevel
	}
	return model.SeverityWarning
}
