package rules

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/hanchuanchuan/goinception-plus/internal/audit"
	"github.com/hanchuanchuan/goinception-plus/internal/model"
)

type DMLSafety struct{}

func (DMLSafety) ID() string { return audit.RuleDMLTableMetadata }
func (DMLSafety) Check(ctx context.Context, a *audit.Context, s model.Statement) []model.AuditIssue {
	if s.DML == nil {
		return nil
	}
	var out []model.AuditIssue
	if s.DML.UsesCTE && strings.HasPrefix(strings.TrimSpace(a.Server.Version), "5.7") {
		out = append(out, errorIssue(s, audit.RuleTargetVersion, "common table expressions require MySQL 8.0"))
	}
	if (s.Kind == model.StatementUpdate || s.Kind == model.StatementDelete) && a.Policy.RequireDMLWhere && !s.DML.HasWhere {
		out = append(out, levelIssue(s, audit.RuleDMLRequireWhere, a.Policy.Level(audit.RuleDMLRequireWhere), "UPDATE and DELETE must contain a WHERE condition"))
	}
	if a.Policy.MaxAffectedRows > 0 && s.DML.EstimatedRows > a.Policy.MaxAffectedRows {
		out = append(out, levelIssue(s, audit.RuleDMLMaxAffected, a.Policy.Level(audit.RuleDMLMaxAffected), fmt.Sprintf("estimated affected rows %d exceed limit %d", s.DML.EstimatedRows, a.Policy.MaxAffectedRows)))
	}
	if s.Kind == model.StatementInsert && a.Policy.MaxInsertRows > 0 && s.DML.ValueRows > a.Policy.MaxInsertRows {
		out = append(out, levelIssue(s, audit.RuleDMLMaxInsert, a.Policy.Level(audit.RuleDMLMaxInsert), fmt.Sprintf("INSERT rows %d exceed limit %d", s.DML.ValueRows, a.Policy.MaxInsertRows)))
	}
	if (s.Kind == model.StatementUpdate || s.Kind == model.StatementDelete) && a.Policy.CheckDMLLimit && s.DML.HasLimit {
		out = append(out, levelIssue(s, audit.RuleDMLLimit, a.Policy.Level(audit.RuleDMLLimit), "UPDATE and DELETE must not contain LIMIT"))
	}
	if (s.Kind == model.StatementUpdate || s.Kind == model.StatementDelete) && a.Policy.CheckDMLOrderBy && s.DML.HasOrderBy {
		out = append(out, levelIssue(s, audit.RuleDMLOrderBy, a.Policy.Level(audit.RuleDMLOrderBy), "UPDATE and DELETE must not contain ORDER BY"))
	}
	if s.DML.HasJoinWithoutOn {
		out = append(out, levelIssue(s, audit.RuleDMLJoinNoOn, a.Policy.Level(audit.RuleDMLJoinNoOn), "JOIN must contain an ON condition"))
	}
	if s.Kind == model.StatementSelect && !a.Policy.AllowSelectStar && s.DML.HasSelectStar {
		out = append(out, levelIssue(s, audit.RuleDMLSelectStar, a.Policy.Level(audit.RuleDMLSelectStar), "SELECT * is disabled"))
	}
	if s.Kind == model.StatementInsert && a.Policy.CheckInsertField && len(s.DML.Columns) == 0 {
		out = append(out, levelIssue(s, audit.RuleDMLInsertField, a.Policy.Level(audit.RuleDMLInsertField), "INSERT requires an explicit column list"))
	}
	if s.DML.OrderByRand {
		out = append(out, levelIssue(s, audit.RuleDMLOrderByRand, a.Policy.Level(audit.RuleDMLOrderByRand), "ORDER BY RAND() is disabled"))
	}
	if s.DML.WrongAndExpr {
		out = append(out, levelIssue(s, audit.RuleDMLWrongAnd, a.Policy.Level(audit.RuleDMLWrongAnd), "suspicious AND expression in assignment or predicate"))
	}
	tables := make(map[string]audit.Table)
	for pos, r := range s.DML.Tables {
		if r.Schema == "" {
			out = append(out, errorIssue(s, audit.RuleNoDatabase, "No database selected."))
			continue
		}
		t, err := a.Metadata.LoadTable(ctx, r.Schema, r.Name)
		if err != nil {
			msg := fmt.Sprintf("load table %s.%s: %v", r.Schema, r.Name, err)
			if errors.Is(err, audit.ErrMetadataNotFound) {
				msg = fmt.Sprintf("table %s.%s does not exist", r.Schema, r.Name)
			}
			out = append(out, errorIssue(s, audit.RuleDMLTableMetadata, msg))
			continue
		}
		tables[strings.ToLower(r.Name)] = t
		if r.Alias != "" {
			tables[strings.ToLower(r.Alias)] = t
		}
		if pos == 0 && len(s.DML.Assignments) == 0 {
			for _, c := range s.DML.Columns {
				if !hasColumn(t, c) {
					out = append(out, errorIssue(s, audit.RuleDMLTableMetadata, fmt.Sprintf("column %s does not exist in %s.%s", c, r.Schema, r.Name)))
				}
			}
		}
		for _, c := range s.DML.Assignments {
			if c.Table != "" && !strings.EqualFold(c.Table, r.Name) && !strings.EqualFold(c.Table, r.Alias) {
				continue
			}
			if c.Schema != "" && !strings.EqualFold(c.Schema, r.Schema) {
				continue
			}
			if !hasColumn(t, c.Name) {
				out = append(out, errorIssue(s, audit.RuleDMLTableMetadata, fmt.Sprintf("column %s does not exist in %s.%s", c.Name, r.Schema, r.Name)))
			}
		}
		needsKey := s.Kind == model.StatementUpdate || s.Kind == model.StatementDelete || (s.Kind == model.StatementInsert && pos == 0)
		if a.Request.Backup && a.Policy.RequireBackupKey && needsKey && !hasReliableKey(t) {
			out = append(out, errorIssue(s, audit.RuleDMLRequireKey, fmt.Sprintf("table %s.%s has no reliable primary or non-null unique key for ROW binlog rollback", r.Schema, r.Name)))
		}
	}
	if a.Policy.LegacyBool("check_implicit_type_conversion") {
		for _, comparison := range s.DML.Comparisons {
			if column, table, ok := resolveComparisonColumn(comparison.Column, s.DML.Tables, tables); ok && implicitConversion(column.ColumnType, comparison.LiteralKind) {
				out = append(out, levelIssue(s, audit.RuleDMLImplicitConversion, a.Policy.Level(audit.RuleDMLImplicitConversion), fmt.Sprintf("implicit type conversion is not allowed for %s.%s (%s)", table, column.Name, column.ColumnType)))
			}
		}
	}
	return out
}

func resolveComparisonColumn(ref model.ColumnRef, refs []model.TableRef, tables map[string]audit.Table) (audit.Column, string, bool) {
	if ref.Table != "" {
		t, ok := tables[strings.ToLower(ref.Table)]
		if !ok {
			return audit.Column{}, "", false
		}
		for _, c := range t.Columns {
			if strings.EqualFold(c.Name, ref.Name) {
				return c, ref.Table, true
			}
		}
		return audit.Column{}, "", false
	}
	var found audit.Column
	tableName := ""
	count := 0
	seen := map[string]bool{}
	for _, refTable := range refs {
		key := strings.ToLower(refTable.Name)
		if seen[key] {
			continue
		}
		seen[key] = true
		for _, c := range tables[key].Columns {
			if strings.EqualFold(c.Name, ref.Name) {
				found, tableName, count = c, refTable.Name, count+1
			}
		}
	}
	return found, tableName, count == 1
}

func implicitConversion(columnType string, literalKind model.LiteralKind) bool {
	base := baseType(columnType)
	if literalKind == model.LiteralUnknown || literalKind == model.LiteralNull {
		return false
	}
	stringLiteral := literalKind == model.LiteralString || literalKind == model.LiteralBinary || literalKind == model.LiteralTemporal || literalKind == model.LiteralDuration || literalKind == model.LiteralJSON
	numericLiteral := literalKind == model.LiteralSignedInteger || literalKind == model.LiteralUnsignedInteger || literalKind == model.LiteralFloat || literalKind == model.LiteralDecimal || literalKind == model.LiteralBoolean
	if isIntegerType(base) || base == "decimal" || base == "float" || base == "double" || base == "real" || base == "bit" {
		return stringLiteral
	}
	switch base {
	case "date", "time", "datetime", "timestamp", "char", "binary", "varchar", "varbinary", "enum", "set", "tinyblob", "tinytext", "blob", "text", "mediumblob", "mediumtext", "longblob", "longtext", "json", "geometry":
		return numericLiteral
	}
	return false
}
func errorIssue(s model.Statement, id, msg string) model.AuditIssue {
	return levelIssue(s, id, model.SeverityError, msg)
}
func levelIssue(s model.Statement, id string, level model.Severity, msg string) model.AuditIssue {
	return model.AuditIssue{RuleID: id, Level: level, Message: msg, Sequence: s.Sequence}
}
func hasColumn(t audit.Table, n string) bool {
	for _, c := range t.Columns {
		if strings.EqualFold(c.Name, n) {
			return true
		}
	}
	return false
}
func hasReliableKey(t audit.Table) bool {
	for _, i := range t.Indexes {
		if i.Primary && len(i.Columns) > 0 {
			return true
		}
		if !i.Unique || len(i.Columns) == 0 {
			continue
		}
		reliable := true
		for _, name := range i.Columns {
			found := false
			for _, column := range t.Columns {
				if strings.EqualFold(column.Name, name) {
					found = true
					if column.Nullable {
						reliable = false
					}
					break
				}
			}
			if !found {
				reliable = false
			}
		}
		if reliable {
			return true
		}
	}
	return false
}
