package rules

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/hanchuanchuan/goinception-plus/internal/audit"
	"github.com/hanchuanchuan/goinception-plus/internal/model"
)

// DDLSafety contains structural checks that do not require target I/O. Object
// existence and ALTER simulation remain the responsibility of Metadata/Snapshot.
type DDLSafety struct{}

func (DDLSafety) ID() string { return audit.RuleDDLMetadata }
func (DDLSafety) Check(_ context.Context, ctx *audit.Context, stmt model.Statement) []model.AuditIssue {
	d := stmt.DDL
	if d == nil {
		return nil
	}
	var out []model.AuditIssue
	add := func(id, message string) {
		out = append(out, model.AuditIssue{RuleID: id, Level: ctx.Policy.Level(id), Message: message, Sequence: stmt.Sequence})
	}
	switch d.Action {
	case model.DDLDropDatabase:
		if !ctx.Policy.AllowDropDatabase {
			add(audit.RuleDDLDropDatabase, "DROP DATABASE is disabled")
		}
	case model.DDLDropTable:
		if !ctx.Policy.AllowDropTable {
			add(audit.RuleDDLDropTable, "DROP TABLE is disabled")
		}
	case model.DDLTruncateTable:
		if !ctx.Policy.AllowTruncate {
			add(audit.RuleDDLTruncate, "TRUNCATE TABLE is disabled")
		}
	}
	if d.HasForeignKey && !ctx.Policy.AllowForeignKey {
		add(audit.RuleDDLForeignKey, "foreign keys are disabled")
	}
	if d.Partitioned && !ctx.Policy.AllowPartition {
		add(audit.RuleDDLPartition, "partition tables are disabled")
	}
	if versionLessThan(ctx.Server.Version, 8, 0, 0) {
		if strings.HasPrefix(strings.ToLower(d.Collation), "utf8mb4_0900_") {
			add(audit.RuleTargetVersion, "utf8mb4_0900 collations require MySQL 8.0")
		}
		for _, op := range d.AlterOperations {
			if op.Action == model.AlterRenameColumn {
				add(audit.RuleTargetVersion, "ALTER TABLE RENAME COLUMN requires MySQL 8.0")
			}
		}
	}
	if hasExpressionIndex(d) && versionLessThan(ctx.Server.Version, 8, 0, 13) {
		add(audit.RuleTargetVersion, "functional indexes require MySQL 8.0.13 or newer")
	}
	if d.Action != model.DDLCreateTable {
		return out
	}
	if required := strings.TrimSpace(ctx.Policy.RequiredEngine); required != "" && d.Engine != "" && !strings.EqualFold(required, d.Engine) {
		add(audit.RuleDDLEngine, fmt.Sprintf("table engine must be %s, got %s", required, d.Engine))
	}
	columns := make(map[string]struct{}, len(d.Columns))
	for _, c := range d.Columns {
		key := strings.ToLower(c.Name)
		if _, exists := columns[key]; exists {
			add(audit.RuleDDLDuplicateCol, fmt.Sprintf("duplicate column %s", c.Name))
		}
		columns[key] = struct{}{}
	}
	indexes := make(map[string]struct{}, len(d.Indexes))
	for _, idx := range d.Indexes {
		key := strings.ToLower(idx.Name)
		if _, exists := indexes[key]; exists {
			add(audit.RuleDDLDuplicateIdx, fmt.Sprintf("duplicate index %s", idx.Name))
		}
		indexes[key] = struct{}{}
		for _, name := range idx.Columns {
			if _, exists := columns[strings.ToLower(name)]; !exists {
				add(audit.RuleDDLIndexColumn, fmt.Sprintf("index %s references missing column %s", idx.Name, name))
			}
		}
	}
	return out
}

func hasExpressionIndex(d *model.DDLSpec) bool {
	for _, index := range d.Indexes {
		if len(index.Expressions) > 0 {
			return true
		}
	}
	for _, op := range d.AlterOperations {
		if op.Index != nil && len(op.Index.Expressions) > 0 {
			return true
		}
	}
	return false
}

func versionLessThan(value string, major, minor, patch int) bool {
	parts := strings.SplitN(value, ".", 4)
	if len(parts) < 2 {
		return false
	}
	values := [3]int{}
	for i := 0; i < len(values) && i < len(parts); i++ {
		digits := strings.TrimLeftFunc(parts[i], func(r rune) bool { return r < '0' || r > '9' })
		digits = strings.TrimRightFunc(digits, func(r rune) bool { return r < '0' || r > '9' })
		if n, err := strconv.Atoi(digits); err == nil {
			values[i] = n
		}
	}
	want := [3]int{major, minor, patch}
	for i := range values {
		if values[i] != want[i] {
			return values[i] < want[i]
		}
	}
	return false
}
