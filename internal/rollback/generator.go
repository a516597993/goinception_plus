package rollback

import (
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/hanchuanchuan/goinception-plus/internal/audit"
	"github.com/hanchuanchuan/goinception-plus/internal/backup"
)

func Generate(change backup.RowChange) (string, error) {
	if len(change.Table.Columns) == 0 {
		return "", fmt.Errorf("table metadata has no columns")
	}
	switch change.Type {
	case backup.ChangeInsert:
		return deleteSQL(change.Table, change.After)
	case backup.ChangeDelete:
		return insertSQL(change.Table, change.Before)
	case backup.ChangeUpdate:
		return updateSQL(change.Table, change.Before, change.After)
	default:
		return "", fmt.Errorf("unsupported row change %q", change.Type)
	}
}
func insertSQL(t audit.Table, row []any) (string, error) {
	if len(row) != len(t.Columns) {
		return "", fmt.Errorf("row image column count mismatch")
	}
	cols := make([]string, len(t.Columns))
	vals := make([]string, len(row))
	for i, c := range t.Columns {
		cols[i] = quoteIdent(c.Name)
		vals[i] = literalColumn(c, row[i])
	}
	return fmt.Sprintf("INSERT INTO %s.%s (%s) VALUES (%s);", quoteIdent(t.Schema), quoteIdent(t.Name), strings.Join(cols, ","), strings.Join(vals, ",")), nil
}
func deleteSQL(t audit.Table, row []any) (string, error) {
	where, err := keyWhere(t, row)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("DELETE FROM %s.%s WHERE %s LIMIT 1;", quoteIdent(t.Schema), quoteIdent(t.Name), where), nil
}
func updateSQL(t audit.Table, before, after []any) (string, error) {
	if len(before) != len(t.Columns) || len(after) != len(t.Columns) {
		return "", fmt.Errorf("row image column count mismatch")
	}
	sets := make([]string, len(t.Columns))
	for i, c := range t.Columns {
		sets[i] = quoteIdent(c.Name) + "=" + literalColumn(c, before[i])
	}
	where, err := keyWhere(t, after)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("UPDATE %s.%s SET %s WHERE %s LIMIT 1;", quoteIdent(t.Schema), quoteIdent(t.Name), strings.Join(sets, ","), where), nil
}
func keyWhere(t audit.Table, row []any) (string, error) {
	var key *audit.Index
	for i := range t.Indexes {
		if t.Indexes[i].Primary {
			key = &t.Indexes[i]
			break
		}
	}
	if key == nil {
		for i := range t.Indexes {
			if t.Indexes[i].Unique && nonNullIndex(t, t.Indexes[i]) {
				key = &t.Indexes[i]
				break
			}
		}
	}
	if key == nil || len(key.Columns) == 0 {
		return "", fmt.Errorf("no reliable primary or unique key")
	}
	parts := make([]string, 0, len(key.Columns))
	for _, name := range key.Columns {
		pos := -1
		for i, c := range t.Columns {
			if strings.EqualFold(c.Name, name) {
				pos = i
				break
			}
		}
		if pos < 0 || pos >= len(row) {
			return "", fmt.Errorf("key column %s is absent from FULL row image", name)
		}
		if row[pos] == nil {
			parts = append(parts, quoteIdent(name)+" IS NULL")
		} else {
			parts = append(parts, quoteIdent(name)+"="+literalColumn(t.Columns[pos], row[pos]))
		}
	}
	return strings.Join(parts, " AND "), nil
}
func nonNullIndex(t audit.Table, index audit.Index) bool {
	if len(index.Columns) == 0 {
		return false
	}
	for _, name := range index.Columns {
		found := false
		for _, column := range t.Columns {
			if strings.EqualFold(column.Name, name) {
				found = true
				if column.Nullable {
					return false
				}
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}
func quoteIdent(v string) string { return "`" + strings.ReplaceAll(v, "`", "``") + "`" }
func literal(v any) string {
	switch x := v.(type) {
	case nil:
		return "NULL"
	case []byte:
		return "X'" + strings.ToUpper(hex.EncodeToString(x)) + "'"
	case string:
		return "'" + strings.ReplaceAll(strings.ReplaceAll(x, "\\", "\\\\"), "'", "''") + "'"
	case time.Time:
		return "'" + x.Format("2006-01-02 15:04:05.999999") + "'"
	case bool:
		if x {
			return "1"
		}
		return "0"
	case int:
		return strconv.Itoa(x)
	case int8:
		return strconv.FormatInt(int64(x), 10)
	case int16:
		return strconv.FormatInt(int64(x), 10)
	case int32:
		return strconv.FormatInt(int64(x), 10)
	case int64:
		return strconv.FormatInt(x, 10)
	case uint:
		return strconv.FormatUint(uint64(x), 10)
	case uint8:
		return strconv.FormatUint(uint64(x), 10)
	case uint16:
		return strconv.FormatUint(uint64(x), 10)
	case uint32:
		return strconv.FormatUint(uint64(x), 10)
	case uint64:
		return strconv.FormatUint(x, 10)
	case float32:
		return strconv.FormatFloat(float64(x), 'g', -1, 32)
	case float64:
		return strconv.FormatFloat(x, 'g', -1, 64)
	default:
		return "'" + strings.ReplaceAll(fmt.Sprint(x), "'", "''") + "'"
	}
}

func literalColumn(column audit.Column, v any) string {
	if v == nil {
		return "NULL"
	}
	kind := strings.ToLower(column.ColumnType)
	isBinary := strings.Contains(kind, "binary") || strings.Contains(kind, "blob") || strings.HasPrefix(kind, "bit")
	if b, ok := v.([]byte); ok {
		switch {
		case isBinary:
			return "X'" + strings.ToUpper(hex.EncodeToString(b)) + "'"
		case strings.Contains(kind, "decimal"), strings.Contains(kind, "numeric"), strings.Contains(kind, "float"), strings.Contains(kind, "double"):
			return string(b)
		default:
			return literal(string(b))
		}
	}
	// go-mysql may expose VARBINARY/BLOB row values as string depending on
	// table-map metadata. Preserve their bytes instead of treating them as
	// connection-charset text; arbitrary bytes are not necessarily valid UTF-8.
	if s, ok := v.(string); ok && isBinary {
		return "X'" + strings.ToUpper(hex.EncodeToString([]byte(s))) + "'"
	}
	if strings.Contains(kind, "decimal") || strings.Contains(kind, "numeric") {
		return fmt.Sprint(v)
	}
	return literal(v)
}
