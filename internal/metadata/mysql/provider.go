package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	driver "github.com/go-sql-driver/mysql"

	"github.com/hanchuanchuan/goinception-plus/internal/audit"
	"github.com/hanchuanchuan/goinception-plus/internal/model"
)

type Provider struct{ db *sql.DB }

func New(db *sql.DB) *Provider { return &Provider{db: db} }

func Open(options model.TargetOptions) (*sql.DB, error) {
	if options.Host == "" || options.Port == 0 || options.User == "" {
		return nil, fmt.Errorf("target MySQL host, port, and user are required")
	}
	config := driver.NewConfig()
	config.Net = "tcp"
	config.Addr = net.JoinHostPort(options.Host, fmt.Sprint(options.Port))
	config.User = options.User
	config.Passwd = options.Password
	config.DBName = options.Database
	config.ParseTime = true
	config.Timeout = 5 * time.Second
	config.ReadTimeout = 15 * time.Second
	config.WriteTimeout = 15 * time.Second
	config.Params = map[string]string{"charset": "utf8mb4"}
	db, err := sql.Open("mysql", config.FormatDSN())
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(4)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(5 * time.Minute)
	return db, nil
}

func (p *Provider) LoadServerInfo(ctx context.Context) (audit.ServerInfo, error) {
	const query = "SELECT VERSION(), @@SESSION.sql_mode, @@lower_case_table_names, " +
		"@@character_set_server, @@collation_server, @@explicit_defaults_for_timestamp"
	var result audit.ServerInfo
	if err := p.db.QueryRowContext(ctx, query).Scan(
		&result.Version, &result.SQLMode, &result.LowerCaseTableNames,
		&result.CharacterSetServer, &result.CollationServer,
		&result.ExplicitDefaultsForTimestamp,
	); err != nil {
		return audit.ServerInfo{}, fmt.Errorf("load MySQL server information: %w", err)
	}
	return result, nil
}

func (p *Provider) LoadSchema(ctx context.Context, schema string) (audit.Schema, error) {
	const query = "SELECT DEFAULT_CHARACTER_SET_NAME, DEFAULT_COLLATION_NAME " +
		"FROM information_schema.SCHEMATA WHERE SCHEMA_NAME = ?"
	result := audit.Schema{Name: schema}
	if err := p.db.QueryRowContext(ctx, query, schema).Scan(&result.CharacterSet, &result.Collation); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return audit.Schema{}, fmt.Errorf("%w: schema %q", audit.ErrMetadataNotFound, schema)
		}
		return audit.Schema{}, fmt.Errorf("load schema %q: %w", schema, err)
	}
	return result, nil
}

func (p *Provider) LoadTable(ctx context.Context, schema, table string) (audit.Table, error) {
	const tableQuery = "SELECT TABLE_TYPE, COALESCE(TABLE_COLLATION, ''), " +
		"COALESCE(ENGINE, ''), COALESCE(TABLE_ROWS, 0), COALESCE(TABLE_COMMENT, '') " +
		"FROM information_schema.TABLES WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?"
	result := audit.Table{Schema: schema, Name: table}
	if err := p.db.QueryRowContext(ctx, tableQuery, schema, table).Scan(
		&result.TableType, &result.Collation, &result.Engine, &result.Rows, &result.Comment,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return audit.Table{}, fmt.Errorf("%w: table %q.%q", audit.ErrMetadataNotFound, schema, table)
		}
		return audit.Table{}, fmt.Errorf("load table %q.%q: %w", schema, table, err)
	}
	result.IsView = strings.EqualFold(result.TableType, "VIEW")
	if index := strings.IndexByte(result.Collation, '_'); index > 0 {
		result.CharacterSet = result.Collation[:index]
	}
	if err := p.loadColumns(ctx, &result); err != nil {
		return audit.Table{}, err
	}
	if err := p.loadIndexes(ctx, &result); err != nil {
		return audit.Table{}, err
	}
	createSQL, err := p.showCreate(ctx, schema, table, result.IsView)
	if err != nil {
		return audit.Table{}, err
	}
	result.CreateSQL = createSQL
	return result, nil
}

func (p *Provider) EstimateImpact(ctx context.Context, database, statement string) (model.ImpactEstimate, error) {
	return p.EstimateImpactWithRule(ctx, database, statement, "max")
}

func (p *Provider) CheckGrants(ctx context.Context, database string, execute bool) error {
	rows, err := p.db.QueryContext(ctx, "SHOW GRANTS FOR CURRENT_USER()")
	if err != nil {
		return fmt.Errorf("read target grants: %w", err)
	}
	defer rows.Close()
	want := "`" + strings.ToLower(strings.ReplaceAll(database, "`", "``")) + "`.*"
	for rows.Next() {
		var grant string
		if err = rows.Scan(&grant); err != nil {
			return err
		}
		lower := strings.ToLower(grant)
		if strings.Contains(lower, " on *.* ") || strings.Contains(lower, " on "+want+" ") {
			if !execute || strings.Contains(lower, "all privileges") || strings.Contains(lower, "insert") || strings.Contains(lower, "update") || strings.Contains(lower, "delete") || strings.Contains(lower, "alter") || strings.Contains(lower, "create") || strings.Contains(lower, "drop") {
				return nil
			}
		}
	}
	if err = rows.Err(); err != nil {
		return err
	}
	mode := "audit"
	if execute {
		mode = "execution"
	}
	return fmt.Errorf("target user has no visible %s grant for database %s", mode, database)
}

func (p *Provider) EstimateImpactWithRule(ctx context.Context, database, statement, rule string) (model.ImpactEstimate, error) {
	conn, err := p.db.Conn(ctx)
	if err != nil {
		return model.ImpactEstimate{}, err
	}
	defer conn.Close()
	if database != "" {
		if _, err = conn.ExecContext(ctx, "USE "+QuoteIdentifier(database)); err != nil {
			return model.ImpactEstimate{}, err
		}
	}
	rows, err := conn.QueryContext(ctx, "EXPLAIN "+strings.TrimSuffix(strings.TrimSpace(statement), ";"))
	if err != nil {
		return model.ImpactEstimate{}, fmt.Errorf("explain impact: %w", err)
	}
	defer rows.Close()
	columns, err := rows.Columns()
	if err != nil {
		return model.ImpactEstimate{}, err
	}
	rowIndex := -1
	for i, name := range columns {
		if strings.EqualFold(name, "rows") {
			rowIndex = i
			break
		}
	}
	if rowIndex < 0 {
		return model.ImpactEstimate{}, fmt.Errorf("EXPLAIN returned no rows estimate")
	}
	var maximum int64
	var first int64
	var count int
	for rows.Next() {
		values := make([]sql.RawBytes, len(columns))
		dest := make([]any, len(columns))
		for i := range values {
			dest[i] = &values[i]
		}
		if err = rows.Scan(dest...); err != nil {
			return model.ImpactEstimate{}, err
		}
		value, parseErr := strconv.ParseInt(string(values[rowIndex]), 10, 64)
		if parseErr == nil && count == 0 {
			first = value
		}
		count++
		if parseErr == nil && value > maximum {
			maximum = value
		}
	}
	result := maximum
	if strings.EqualFold(rule, "first") {
		result = first
	}
	return model.ImpactEstimate{Rows: result, Method: "explain:" + strings.ToLower(rule), Exact: false}, rows.Err()
}

func (p *Provider) showCreate(ctx context.Context, schema, table string, view bool) (string, error) {
	showKind := "TABLE"
	if view {
		showKind = "VIEW"
	}
	query := "SHOW CREATE " + showKind + " " + QuoteIdentifier(schema) + "." + QuoteIdentifier(table)
	rows, err := p.db.QueryContext(ctx, query)
	if err != nil {
		return "", fmt.Errorf("show create %q.%q: %w", schema, table, err)
	}
	defer rows.Close()
	columns, err := rows.Columns()
	if err != nil {
		return "", err
	}
	if len(columns) < 2 || !rows.Next() {
		return "", fmt.Errorf("show create %q.%q returned no definition", schema, table)
	}
	values := make([]sql.NullString, len(columns))
	dest := make([]any, len(columns))
	for i := range values {
		dest[i] = &values[i]
	}
	if err := rows.Scan(dest...); err != nil {
		return "", fmt.Errorf("scan show create %q.%q: %w", schema, table, err)
	}
	return values[1].String, nil
}

func (p *Provider) loadColumns(ctx context.Context, table *audit.Table) error {
	const query = "SELECT COLUMN_NAME, COLUMN_TYPE, IS_NULLABLE, COLUMN_DEFAULT, EXTRA, " +
		"COALESCE(CHARACTER_SET_NAME, ''), COALESCE(COLLATION_NAME, ''), " +
		"COALESCE(COLUMN_COMMENT, ''), COALESCE(GENERATION_EXPRESSION, '') " +
		"FROM information_schema.COLUMNS " +
		"WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ? ORDER BY ORDINAL_POSITION"
	rows, err := p.db.QueryContext(ctx, query, table.Schema, table.Name)
	if err != nil {
		return fmt.Errorf("load columns for %q.%q: %w", table.Schema, table.Name, err)
	}
	defer rows.Close()
	for rows.Next() {
		var item audit.Column
		var nullable string
		var defaultValue sql.NullString
		if err := rows.Scan(
			&item.Name, &item.ColumnType, &nullable, &defaultValue, &item.Extra,
			&item.CharacterSet, &item.Collation, &item.Comment, &item.GenerationExpression,
		); err != nil {
			return fmt.Errorf("scan column for %q.%q: %w", table.Schema, table.Name, err)
		}
		item.Nullable = strings.EqualFold(nullable, "YES")
		item.AutoIncrement = strings.Contains(strings.ToLower(item.Extra), "auto_increment")
		item.Unsigned = strings.Contains(strings.ToLower(item.ColumnType), "unsigned")
		if defaultValue.Valid {
			value := defaultValue.String
			item.Default = &value
		}
		table.Columns = append(table.Columns, item)
	}
	return rows.Err()
}

func (p *Provider) loadIndexes(ctx context.Context, table *audit.Table) error {
	server, err := p.LoadServerInfo(ctx)
	if err != nil {
		return err
	}
	version8 := mysqlVersionAtLeast(server.Version, 8, 0, 13)
	query := "SELECT INDEX_NAME, NON_UNIQUE, SEQ_IN_INDEX, COLUMN_NAME, " +
		"COALESCE(SUB_PART, 0), COALESCE(COLLATION, 'A'), INDEX_TYPE"
	if version8 {
		query += ", COALESCE(EXPRESSION, ''), COALESCE(IS_VISIBLE, 'YES')"
	}
	query += " FROM information_schema.STATISTICS WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ? " +
		"ORDER BY INDEX_NAME, SEQ_IN_INDEX"
	rows, err := p.db.QueryContext(ctx, query, table.Schema, table.Name)
	if err != nil {
		return fmt.Errorf("load indexes for %q.%q: %w", table.Schema, table.Name, err)
	}
	defer rows.Close()
	byName := make(map[string]int)
	for rows.Next() {
		var name, direction, indexType, expression, visible string
		var nonUnique, sequence, prefix int
		var column sql.NullString
		dest := []any{&name, &nonUnique, &sequence, &column, &prefix, &direction, &indexType}
		if version8 {
			dest = append(dest, &expression, &visible)
		} else {
			visible = "YES"
		}
		if err := rows.Scan(dest...); err != nil {
			return fmt.Errorf("scan index for %q.%q: %w", table.Schema, table.Name, err)
		}
		position, ok := byName[name]
		if !ok {
			position = len(table.Indexes)
			byName[name] = position
			table.Indexes = append(table.Indexes, audit.Index{
				Name: name, Unique: nonUnique == 0, Primary: strings.EqualFold(name, "PRIMARY"),
				IndexType: indexType, Visible: strings.EqualFold(visible, "YES"),
			})
		}
		columnName := ""
		if column.Valid {
			columnName = column.String
		}
		table.Indexes[position].Columns = append(table.Indexes[position].Columns, columnName)
		table.Indexes[position].PrefixLengths = append(table.Indexes[position].PrefixLengths, prefix)
		table.Indexes[position].Directions = append(table.Indexes[position].Directions, direction)
		if strings.TrimSpace(expression) != "" {
			table.Indexes[position].Expressions = append(table.Indexes[position].Expressions, expression)
		}
	}
	return rows.Err()
}

func mysqlVersionAtLeast(value string, major, minor, patch int) bool {
	core := strings.SplitN(value, "-", 2)[0]
	parts := strings.Split(core, ".")
	got := []int{0, 0, 0}
	for i := 0; i < len(parts) && i < 3; i++ {
		got[i], _ = strconv.Atoi(parts[i])
	}
	if got[0] != major {
		return got[0] > major
	}
	if got[1] != minor {
		return got[1] > minor
	}
	return got[2] >= patch
}

func QuoteIdentifier(value string) string {
	return "\x60" + strings.ReplaceAll(value, "\x60", "\x60\x60") + "\x60"
}
