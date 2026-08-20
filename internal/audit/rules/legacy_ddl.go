package rules

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/hanchuanchuan/goinception-plus/internal/audit"
	"github.com/hanchuanchuan/goinception-plus/internal/model"
)

type LegacyDDL struct{}

func (LegacyDDL) ID() string { return audit.RuleDDLInvalidIdentifier }

var validIdentifier = regexp.MustCompile(`^[A-Za-z0-9_]+$`)
var mysqlKeywords = map[string]bool{
	"add": true, "alter": true, "and": true, "as": true, "by": true, "create": true,
	"delete": true, "desc": true, "drop": true, "from": true, "group": true, "index": true,
	"insert": true, "join": true, "key": true, "limit": true, "order": true, "primary": true,
	"select": true, "table": true, "tables": true, "update": true, "use": true, "where": true,
}

func (LegacyDDL) Check(ctx context.Context, a *audit.Context, stmt model.Statement) []model.AuditIssue {
	d := stmt.DDL
	if d == nil {
		return nil
	}
	var out []model.AuditIssue
	add := func(rule, message string) {
		out = append(out, model.AuditIssue{RuleID: rule, Level: a.Policy.Level(rule), Message: message, Sequence: stmt.Sequence})
	}
	if d.Action == model.DDLCreateView || d.Action == model.DDLAlterView || d.Action == model.DDLDropView {
		add(audit.RuleDDLView, "views are disabled")
		return out
	}
	checkName := func(kind, name string) {
		if name == "" {
			return
		}
		if a.Policy.LegacyBool("check_identifier") && !validIdentifier.MatchString(name) {
			add(audit.RuleDDLInvalidIdentifier, fmt.Sprintf("invalid %s identifier %q", kind, name))
		}
		if a.Policy.LegacyBool("check_identifier_lower") && name != strings.ToLower(name) {
			add(audit.RuleDDLInvalidIdentifier, fmt.Sprintf("%s identifier %q must be lowercase", kind, name))
		}
		if a.Policy.LegacyBool("enable_identifer_keyword") && isKeyword(name, a.Policy.LegacyString("custom_keywords")) {
			add(audit.RuleDDLIdentifierKeyword, fmt.Sprintf("%s identifier %q is a MySQL keyword", kind, name))
		}
	}
	checkName("schema", d.Schema)
	checkName("table", d.Table)
	for _, table := range d.Tables {
		checkName("table", table.Name)
	}

	if d.HasCharsetOption {
		if !a.Policy.LegacyBool("enable_set_charset") {
			rule := audit.RuleDDLTableCharsetNull
			if d.Action == model.DDLCreateDatabase || d.Action == model.DDLAlterDatabase {
				rule = audit.RuleDDLCantSetCharset
			}
			add(rule, "explicit charset is disabled")
		} else if !allowedCSV(d.CharacterSet, a.Policy.LegacyString("support_charset")) {
			add(audit.RuleDDLTableCharsetAllow, fmt.Sprintf("unsupported table charset %s", d.CharacterSet))
		}
	}
	if d.HasCollationOption && !a.Policy.LegacyBool("enable_set_collation") {
		add(audit.RuleDDLCantSetCollation, "explicit table collation is disabled")
	}
	if d.HasEngineOption {
		if !a.Policy.LegacyBool("enable_set_engine") {
			add(audit.RuleDDLCantSetEngine, "explicit table engine is disabled")
		} else if !allowedCSV(d.Engine, a.Policy.LegacyString("support_engine")) {
			add(audit.RuleDDLEngine, fmt.Sprintf("unsupported table engine %s", d.Engine))
		}
	}
	if d.AutoIncrementValue > 1 {
		add(audit.RuleDDLAutoInitial, "AUTO_INCREMENT initial value must be 1")
	}

	columns := append([]model.ColumnSpec(nil), d.Columns...)
	var existingTable *audit.Table
	if (d.Action == model.DDLCreateIndex || d.Action == model.DDLDropIndex) && a.Metadata != nil {
		if loaded, err := a.Metadata.LoadTable(ctx, d.Schema, d.Table); err == nil {
			existingTable = &loaded
		}
	}
	if d.Action == model.DDLAlterTable {
		key := strings.ToLower(d.Schema + "." + d.Table)
		a.SeenAlterTables[key]++
		if a.SeenAlterTables[key] > 1 {
			add(audit.RuleDDLAlterOnce, fmt.Sprintf("multiple ALTER statements for %s should be merged", key))
		}
		if a.Metadata != nil {
			if loaded, err := a.Metadata.LoadTable(ctx, d.Schema, d.Table); err == nil {
				existingTable = &loaded
			}
		}
		for _, op := range d.AlterOperations {
			if op.Action == model.AlterChangeColumn {
				add(audit.RuleDDLCantChange, "CHANGE COLUMN is disabled")
			}
			for _, column := range op.Columns {
				columns = append(columns, column)
				if column.PositionChanged {
					add(audit.RuleDDLColumnPosition, fmt.Sprintf("changing position of column %s is disabled", column.Name))
				}
			}
			if (op.Action == model.AlterModifyColumn || op.Action == model.AlterChangeColumn) && a.Policy.LegacyBool("check_column_type_change") && a.Metadata != nil {
				if existingTable != nil && len(op.Columns) == 1 {
					oldName := op.Name
					if oldName == "" {
						oldName = op.Columns[0].Name
					}
					for _, old := range existingTable.Columns {
						if strings.EqualFold(old.Name, oldName) && unsafeColumnTypeChange(old.ColumnType, op.Columns[0].Type) {
							add(audit.RuleDDLColumnTypeChange, fmt.Sprintf("column %s type changes from %s to %s", oldName, old.ColumnType, op.Columns[0].Type))
						}
					}
				}
			}
		}
	}

	for _, column := range columns {
		checkName("column", column.Name)
		checkColumn(a, d, column, add)
	}
	if a.Policy.LegacyBool("check_timestamp_count") {
		defaults, updates := 0, 0
		for _, column := range columns {
			if baseType(column.Type) != "timestamp" {
				continue
			}
			if strings.Contains(strings.ToLower(column.DefaultExpression), "current_timestamp") {
				defaults++
			}
			if column.OnUpdate {
				updates++
			}
		}
		if defaults > 1 || updates > 1 {
			add(audit.RuleDDLTooManyTimestamps, "too many automatic TIMESTAMP columns")
		}
	}
	checkIndexes(a, d, existingTable, checkName, add)
	checkRequiredColumns(a, d, add)
	return out
}

func checkColumn(a *audit.Context, d *model.DDLSpec, column model.ColumnSpec, add func(string, string)) {
	base := baseType(column.Type)
	large := strings.Contains(base, "text") || strings.Contains(base, "blob") || base == "geometry"
	if column.ExplicitCharset && !a.Policy.LegacyBool("enable_column_charset") {
		add(audit.RuleDDLColumnCharset, fmt.Sprintf("column %s must not specify charset or collation", column.Name))
	}
	if (large || base == "json") && column.HasDefault && strings.ToLower(column.DefaultExpression) != "null" {
		add(audit.RuleDDLBlobDefault, fmt.Sprintf("column %s cannot have a default", column.Name))
	}
	if base == "json" && !a.Policy.LegacyBool("enable_json_type") {
		add(audit.RuleDDLJSONType, fmt.Sprintf("JSON column %s is disabled", column.Name))
	}
	if large && !a.Policy.LegacyBool("enable_blob_type") {
		add(audit.RuleDDLUseTextBlob, fmt.Sprintf("text/blob column %s is disabled", column.Name))
	}
	if (base == "enum" || base == "set" || base == "bit") && !a.Policy.LegacyBool("enable_enum_set_bit") {
		add(audit.RuleDDLUseEnum, fmt.Sprintf("ENUM/SET/BIT column %s is disabled", column.Name))
	}
	if base == "timestamp" && !a.Policy.LegacyBool("enable_timestamp_type") {
		add(audit.RuleDDLInvalidType, fmt.Sprintf("TIMESTAMP column %s is disabled", column.Name))
	}
	if !a.Policy.LegacyBool("enable_nullable") && column.Nullable && !column.Generated {
		add(audit.RuleDDLNullable, fmt.Sprintf("column %s must be NOT NULL", column.Name))
	}
	if (large || base == "json") && !column.Nullable {
		add(audit.RuleDDLTextNotNull, fmt.Sprintf("text/blob/json column %s must not be NOT NULL", column.Name))
	}
	if !column.HasDefault && base != "timestamp" && !large && base != "json" && !column.AutoIncrement && !column.PrimaryKey && !column.Generated {
		add(audit.RuleDDLAddDefault, fmt.Sprintf("column %s must have a default", column.Name))
	}
	if base == "datetime" && !column.HasDefault {
		add(audit.RuleDDLDatetimeDefault, fmt.Sprintf("DATETIME column %s must have a default", column.Name))
	}
	if column.HasDefault && isTemporalType(base) && invalidTemporalDefault(column.DefaultExpression, a.Policy.LegacyString("sql_mode")) {
		add(audit.RuleDDLIncorrectDatetime, fmt.Sprintf("column %s has invalid datetime default %s", column.Name, column.DefaultExpression))
	}
	if max := a.Policy.LegacyInt("max_char_length"); base == "char" && max > 0 && column.Length > max {
		add(audit.RuleDDLCharToVarchar, fmt.Sprintf("column %s should use VARCHAR", column.Name))
	}
	if column.AutoIncrement {
		if a.Policy.LegacyBool("check_autoincrement_datatype") && !isIntegerType(base) {
			add(audit.RuleDDLAutoInteger, fmt.Sprintf("auto increment column %s must be int or bigint", column.Name))
		}
		if a.Policy.LegacyBool("enable_autoincrement_unsigned") && !column.Unsigned {
			add(audit.RuleDDLAutoUnsigned, fmt.Sprintf("auto increment column %s should be unsigned", column.Name))
		}
		if !strings.EqualFold(column.Name, "id") {
			add(audit.RuleDDLAutoName, fmt.Sprintf("auto increment column %s should be named ID", column.Name))
		}
	}
}

func checkIndexes(a *audit.Context, d *model.DDLSpec, existing *audit.Table, checkName func(string, string), add func(string, string)) {
	indexes := append([]model.IndexSpec(nil), d.Indexes...)
	columns := append([]model.ColumnSpec(nil), d.Columns...)
	for _, op := range d.AlterOperations {
		if op.Index != nil {
			indexes = append(indexes, *op.Index)
		}
	}
	newCount := len(indexes)
	if existing != nil {
		dropped := map[string]bool{}
		for _, op := range d.AlterOperations {
			if op.Action == model.AlterDropIndex || op.Action == model.AlterDropPrimaryKey {
				dropped[strings.ToLower(op.Name)] = true
			}
		}
		for _, index := range existing.Indexes {
			if !dropped[strings.ToLower(index.Name)] {
				indexes = append(indexes, model.IndexSpec{Name: index.Name, Columns: append([]string(nil), index.Columns...), Unique: index.Unique, Primary: index.Primary})
			}
		}
		for _, column := range existing.Columns {
			columns = append(columns, model.ColumnSpec{Name: column.Name, Type: column.ColumnType})
		}
	}
	if max := a.Policy.LegacyInt("max_keys"); max > 0 && len(indexes) > max {
		add(audit.RuleDDLMaxKeys, fmt.Sprintf("table has %d indexes, maximum is %d", len(indexes), max))
	}
	for position, index := range indexes {
		if position >= newCount {
			continue
		}
		// PRIMARY is the server-defined name of a primary-key constraint, not a
		// user identifier.  Treating it as an identifier produces a false keyword
		// violation for every inline/table-level PRIMARY KEY.
		if !index.Primary {
			checkName("index", index.Name)
		}
		if index.Primary {
			if max := a.Policy.LegacyInt("max_primary_key_parts"); max > 0 && len(index.Columns) > max {
				add(audit.RuleDDLMaxPrimaryParts, fmt.Sprintf("primary key has %d parts, maximum is %d", len(index.Columns), max))
			}
			if a.Policy.LegacyBool("enable_pk_columns_only_int") {
				for _, name := range index.Columns {
					for _, column := range columns {
						if strings.EqualFold(name, column.Name) && !isIntegerType(baseType(column.Type)) {
							add(audit.RuleDDLPKInteger, fmt.Sprintf("primary key column %s should use int or bigint", name))
						}
					}
				}
			}
		} else {
			prefix := a.Policy.LegacyString("index_prefix")
			rule := audit.RuleDDLIndexPrefix
			if index.Unique {
				prefix, rule = a.Policy.LegacyString("uniq_index_prefix"), audit.RuleDDLUniqueIndexPrefix
			}
			if a.Policy.LegacyBool("check_index_prefix") && prefix != "" && !hasAnyPrefix(index.Name, prefix) {
				add(rule, fmt.Sprintf("index %s must use prefix %s", index.Name, prefix))
			}
		}
		if max := a.Policy.LegacyInt("max_key_parts"); max > 0 && len(index.Columns) > max {
			add(audit.RuleDDLMaxKeyParts, fmt.Sprintf("index %s has %d parts, maximum is %d", index.Name, len(index.Columns), max))
		}
	}
	if a.Policy.LegacyBool("check_index_column_repeat") {
		for i := range indexes {
			for j := i + 1; j < len(indexes); j++ {
				if leadingColumnsOverlap(indexes[i].Columns, indexes[j].Columns) {
					add(audit.RuleDDLIndexRepeat, fmt.Sprintf("indexes %s and %s have redundant leading columns", indexes[i].Name, indexes[j].Name))
				}
			}
		}
	}
}

func checkRequiredColumns(a *audit.Context, d *model.DDLSpec, add func(string, string)) {
	required := strings.TrimSpace(a.Policy.LegacyString("must_have_columns"))
	if required == "" || d.Action != model.DDLCreateTable {
		return
	}
	found := map[string]bool{}
	for _, c := range d.Columns {
		found[strings.ToLower(c.Name)] = true
	}
	var missing []string
	for _, item := range strings.Split(required, ",") {
		name := strings.Fields(strings.TrimSpace(item))
		if len(name) > 0 && !found[strings.ToLower(name[0])] {
			missing = append(missing, name[0])
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		add(audit.RuleDDLMustColumns, "missing required columns: "+strings.Join(missing, ","))
	}
}

func baseType(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if i := strings.IndexAny(value, " ( "); i >= 0 {
		value = value[:i]
	}
	return value
}
func isIntegerType(v string) bool {
	switch v {
	case "tinyint", "smallint", "mediumint", "int", "integer", "bigint":
		return true
	}
	return false
}
func isTemporalType(v string) bool { return v == "date" || v == "datetime" || v == "timestamp" }
func invalidTemporalDefault(expression, sqlMode string) bool {
	v := strings.TrimSpace(strings.ToLower(expression))
	if v == "" || v == "null" || strings.Contains(v, "current_timestamp") {
		return false
	}
	if i := strings.Index(v, "'"); i >= 0 {
		v = strings.Trim(v[i:], "'")
	}
	strict := strings.Contains(strings.ToUpper(sqlMode), "TRADITIONAL") || strings.Contains(strings.ToUpper(sqlMode), "NO_ZERO_DATE") || strings.Contains(strings.ToUpper(sqlMode), "STRICT")
	if strings.HasPrefix(v, "0000-00-00") {
		return strict
	}
	formats := []string{"2006-01-02", "2006-01-02 15:04:05"}
	for _, format := range formats {
		if _, err := time.Parse(format, v); err == nil {
			return false
		}
	}
	return true
}
func sameColumnType(a, b string) bool {
	return strings.EqualFold(strings.Join(strings.Fields(a), " "), strings.Join(strings.Fields(b), " "))
}
func unsafeColumnTypeChange(oldType, newType string) bool {
	if sameColumnType(oldType, newType) {
		return false
	}
	oldBase, newBase := baseType(oldType), baseType(newType)
	ranks := map[string]int{"tinyint": 1, "smallint": 2, "mediumint": 3, "int": 4, "integer": 4, "bigint": 5}
	if oldRank, ok := ranks[oldBase]; ok {
		if newRank, newOK := ranks[newBase]; newOK {
			return newRank < oldRank
		}
	}
	if oldBase == "float" && newBase == "double" {
		return false
	}
	if (oldBase == "char" || oldBase == "varchar") && (newBase == "char" || newBase == "varchar") {
		return typeLength(newType) < typeLength(oldType) || oldBase == "varchar" && newBase == "char"
	}
	if oldBase == newBase {
		return false
	}
	return true
}
func typeLength(value string) int {
	start := strings.Index(value, "(")
	end := strings.Index(value, ")")
	if start < 0 || end <= start {
		return 0
	}
	n, _ := strconv.Atoi(strings.TrimSpace(strings.Split(value[start+1:end], ",")[0]))
	return n
}
func allowedCSV(value, allowed string) bool {
	if value == "" {
		return true
	}
	for _, v := range strings.Split(allowed, ",") {
		if strings.EqualFold(strings.TrimSpace(v), value) {
			return true
		}
	}
	return false
}
func hasAnyPrefix(value, prefixes string) bool {
	for _, p := range strings.Split(prefixes, ",") {
		if p = strings.TrimSpace(p); p != "" && strings.HasPrefix(strings.ToLower(value), strings.ToLower(p)) {
			return true
		}
	}
	return false
}
func leadingColumnsOverlap(a, b []string) bool {
	if len(a) == 0 || len(b) == 0 {
		return false
	}
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	for i := 0; i < n; i++ {
		if !strings.EqualFold(a[i], b[i]) {
			return false
		}
	}
	return true
}
func isKeyword(name, custom string) bool {
	n := strings.ToLower(name)
	if mysqlKeywords[n] {
		return true
	}
	for _, v := range strings.Split(custom, ",") {
		if strings.EqualFold(strings.TrimSpace(v), name) {
			return true
		}
	}
	return false
}
