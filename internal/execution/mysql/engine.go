package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/hanchuanchuan/goinception-plus/internal/audit"
	"github.com/hanchuanchuan/goinception-plus/internal/backup"
	"github.com/hanchuanchuan/goinception-plus/internal/backup/binlogmysql"
	"github.com/hanchuanchuan/goinception-plus/internal/backup/mysqlstore"
	"github.com/hanchuanchuan/goinception-plus/internal/execution"
	metadatamysql "github.com/hanchuanchuan/goinception-plus/internal/metadata/mysql"
	"github.com/hanchuanchuan/goinception-plus/internal/model"
	"github.com/hanchuanchuan/goinception-plus/internal/rollback"
)

type Engine struct {
	db     *sql.DB
	backup backup.TargetOptions
}

func Open(ctx context.Context, options model.TargetOptions) (execution.Engine, func() error, error) {
	return OpenWithBackup(backup.TargetOptions{})(ctx, options)
}
func OpenWithBackup(backupOptions backup.TargetOptions) execution.Factory {
	return func(ctx context.Context, options model.TargetOptions) (execution.Engine, func() error, error) {
		db, err := metadatamysql.Open(options)
		if err != nil {
			return nil, nil, err
		}
		if err = db.PingContext(ctx); err != nil {
			_ = db.Close()
			return nil, nil, err
		}
		return &Engine{db: db, backup: backupOptions}, db.Close, nil
	}
}

func (e *Engine) Execute(ctx context.Context, options model.RequestOptions, statements []model.Statement) []model.ExecutionResult {
	results := make([]model.ExecutionResult, 0, len(statements))
	conn, err := e.db.Conn(ctx)
	if err != nil {
		return []model.ExecutionResult{{Sequence: 1, Error: err.Error()}}
	}
	defer conn.Close()
	if err = configureSession(ctx, conn, options.Target); err != nil {
		return []model.ExecutionResult{{Sequence: 1, RuleID: audit.RuleExecutionFailed, Error: "configure target session: " + err.Error()}}
	}
	var capture *binlogmysql.Capture
	var store *mysqlstore.Store
	var threadID uint32
	if options.Backup {
		capture = binlogmysql.New(e.db, options.Target)
		if err = capture.Validate(ctx); err != nil {
			return []model.ExecutionResult{{Sequence: 1, RuleID: audit.RuleBinlogPrereq, Error: err.Error()}}
		}
		if e.backup.Host == "" || e.backup.Port == 0 || e.backup.User == "" {
			return []model.ExecutionResult{{Sequence: 1, RuleID: audit.RuleBinlogPrereq, Error: "backup MySQL host, port, and user are required"}}
		}
		store, err = mysqlstore.Open(e.backup)
		if err != nil {
			return []model.ExecutionResult{{Sequence: 1, RuleID: audit.RuleBinlogPrereq, Error: err.Error()}}
		}
		defer store.Close()
		if err = store.Ping(ctx); err != nil {
			return []model.ExecutionResult{{Sequence: 1, RuleID: audit.RuleBinlogPrereq, Error: "connect backup MySQL: " + err.Error()}}
		}
		var id uint64
		if err = conn.QueryRowContext(ctx, "SELECT CONNECTION_ID()").Scan(&id); err != nil {
			return []model.ExecutionResult{{Sequence: 1, RuleID: audit.RuleBinlogPrereq, Error: err.Error()}}
		}
		threadID = uint32(id)
	}
	for _, s := range statements {
		r := model.ExecutionResult{Sequence: s.Sequence}
		startTime := time.Now()
		if !options.Backup || s.Kind == model.StatementUse || s.Kind == model.StatementSelect {
			result, execErr := conn.ExecContext(ctx, s.Normalized)
			r.Duration = time.Since(startTime)
			if execErr != nil {
				r.Error = execErr.Error()
				results = append(results, r)
				break
			}
			if n, nerr := result.RowsAffected(); nerr == nil {
				r.AffectedRows = n
			}
			r.Executed = true
			results = append(results, r)
			if !executionDelay(ctx, options, len(results)-1) {
				break
			}
			continue
		}
		start, positionErr := binlogmysql.MasterPosition(ctx, conn)
		if positionErr != nil {
			r.RuleID = audit.RuleBinlogPrereq
			r.Error = positionErr.Error()
			results = append(results, r)
			break
		}
		ddlSQL, ddlErr := prepareDDLRollback(ctx, conn, s)
		if ddlErr != nil {
			r.RuleID = audit.RuleBackupFailed
			r.Error = ddlErr.Error()
			results = append(results, r)
			break
		}
		result, execErr := conn.ExecContext(ctx, s.Normalized)
		r.Duration = time.Since(startTime)
		if execErr != nil {
			r.Error = execErr.Error()
			results = append(results, r)
			break
		}
		if n, nerr := result.RowsAffected(); nerr == nil {
			r.AffectedRows = n
		}
		r.Executed = true
		end, positionErr := binlogmysql.MasterPosition(ctx, conn)
		if positionErr != nil {
			r.RuleID = audit.RuleBackupFailed
			r.Error = positionErr.Error()
			results = append(results, r)
			break
		}
		opid := fmt.Sprintf("%d_%d_%08d", startTime.Unix(), threadID, s.Sequence-1)
		rollbacks := make([]backup.RollbackRecord, 0)
		if s.DML != nil {
			changes, captureErr := capture.Collect(ctx, start, end, threadID, r.AffectedRows)
			if captureErr != nil {
				r.RuleID = audit.RuleBackupFailed
				r.Error = captureErr.Error()
				results = append(results, r)
				break
			}
			for _, change := range changes {
				sqlText, genErr := rollback.Generate(change)
				if genErr != nil {
					r.RuleID = audit.RuleBackupFailed
					r.Error = genErr.Error()
					break
				}
				rollbacks = append(rollbacks, backup.RollbackRecord{Schema: change.Table.Schema, Table: change.Table.Name, SQL: sqlText})
			}
			if r.Error != "" {
				results = append(results, r)
				break
			}
		} else {
			for _, item := range ddlSQL {
				rollbacks = append(rollbacks, item)
			}
		}
		schema, table := statementObject(s)
		backupStart := time.Now()
		dbName, saveErr := store.Save(ctx, backup.Operation{OPID: opid, Statement: s, Source: options.Target, Start: start, End: end, ThreadID: threadID, Schema: schema, Table: table, Rollbacks: rollbacks})
		r.BackupDuration = time.Since(backupStart)
		if saveErr != nil {
			r.RuleID = audit.RuleBackupFailed
			r.Error = saveErr.Error()
			results = append(results, r)
			break
		}
		r.SequenceID = opid
		r.BackupDatabase = dbName
		r.BackupCompleted = true
		results = append(results, r)
		if !executionDelay(ctx, options, len(results)-1) {
			break
		}
	}
	return results
}

func configureSession(ctx context.Context, conn *sql.Conn, options model.TargetOptions) error {
	if strings.TrimSpace(options.SQLMode) != "" {
		if _, err := conn.ExecContext(ctx, "SET SESSION sql_mode = ?", options.SQLMode); err != nil {
			return err
		}
	}
	if options.SQLSafeUpdates >= 0 {
		if _, err := conn.ExecContext(ctx, "SET SESSION sql_safe_updates = ?", options.SQLSafeUpdates); err != nil {
			return err
		}
	}
	if options.LockWaitTimeout >= 0 {
		if _, err := conn.ExecContext(ctx, "SET SESSION lock_wait_timeout = ?", options.LockWaitTimeout); err != nil {
			return err
		}
	}
	if charset := strings.TrimSpace(options.DefaultCharset); charset != "" {
		for _, r := range charset {
			if !(r == '_' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9') {
				return fmt.Errorf("invalid default charset %q", charset)
			}
		}
		if _, err := conn.ExecContext(ctx, "SET NAMES "+charset); err != nil {
			return err
		}
	}
	return nil
}

func executionDelay(ctx context.Context, options model.RequestOptions, zeroBasedIndex int) bool {
	if options.SleepMillis <= 0 || options.SleepRows <= 0 || zeroBasedIndex%options.SleepRows != 0 {
		return true
	}
	timer := time.NewTimer(time.Duration(options.SleepMillis) * time.Millisecond)
	defer timer.Stop()
	select {
	case <-timer.C:
		return true
	case <-ctx.Done():
		return false
	}
}

func prepareDDLRollback(ctx context.Context, conn *sql.Conn, s model.Statement) ([]backup.RollbackRecord, error) {
	if s.DDL == nil {
		return nil, nil
	}
	d := s.DDL
	switch d.Action {
	case model.DDLCreateDatabase:
		return []backup.RollbackRecord{{Schema: d.Schema, Table: "database", SQL: "DROP DATABASE " + quote(d.Schema) + ";"}}, nil
	case model.DDLDropDatabase:
		return nil, fmt.Errorf("DROP DATABASE cannot be recovered losslessly from ROW binlog")
	case model.DDLCreateTable:
		return []backup.RollbackRecord{{Schema: d.Schema, Table: d.Table, SQL: "DROP TABLE " + qualified(d.Schema, d.Table) + ";"}}, nil
	case model.DDLDropTable:
		var out []backup.RollbackRecord
		for _, t := range d.Tables {
			schema := t.Schema
			if schema == "" {
				schema = d.Schema
			}
			create, err := showCreate(ctx, conn, schema, t.Name)
			if err != nil {
				return nil, err
			}
			out = append(out, backup.RollbackRecord{Schema: schema, Table: t.Name, SQL: create + ";"})
		}
		return out, nil
	case model.DDLRenameTable:
		out := make([]backup.RollbackRecord, 0, len(d.RenamePairs))
		for _, p := range d.RenamePairs {
			out = append(out, backup.RollbackRecord{Schema: p.To.Schema, Table: p.To.Name, SQL: "RENAME TABLE " + qualified(p.To.Schema, p.To.Name) + " TO " + qualified(p.From.Schema, p.From.Name) + ";"})
		}
		return out, nil
	case model.DDLCreateIndex:
		return []backup.RollbackRecord{{Schema: d.Schema, Table: d.Table, SQL: "DROP INDEX " + quote(d.Indexes[0].Name) + " ON " + qualified(d.Schema, d.Table) + ";"}}, nil
	case model.DDLDropIndex:
		return nil, fmt.Errorf("DROP INDEX rollback requires full index definition and is not yet supported")
	case model.DDLAlterTable:
		var out []backup.RollbackRecord
		for i := len(d.AlterOperations) - 1; i >= 0; i-- {
			op := d.AlterOperations[i]
			switch op.Action {
			case model.AlterAddColumns:
				for j := len(op.Columns) - 1; j >= 0; j-- {
					out = append(out, backup.RollbackRecord{Schema: d.Schema, Table: d.Table, SQL: "ALTER TABLE " + qualified(d.Schema, d.Table) + " DROP COLUMN " + quote(op.Columns[j].Name) + ";"})
				}
			case model.AlterAddIndex:
				out = append(out, backup.RollbackRecord{Schema: d.Schema, Table: d.Table, SQL: "ALTER TABLE " + qualified(d.Schema, d.Table) + " DROP INDEX " + quote(op.Index.Name) + ";"})
			case model.AlterRenameColumn:
				out = append(out, backup.RollbackRecord{Schema: d.Schema, Table: d.Table, SQL: "ALTER TABLE " + qualified(d.Schema, d.Table) + " RENAME COLUMN " + quote(op.NewName) + " TO " + quote(op.Name) + ";"})
			default:
				return nil, fmt.Errorf("backup does not support lossless rollback for ALTER action %s", op.Action)
			}
		}
		return out, nil
	case model.DDLTruncateTable:
		return nil, fmt.Errorf("TRUNCATE cannot be recovered from ROW binlog")
	default:
		return nil, fmt.Errorf("backup rollback is not implemented for %s", d.Action)
	}
}
func showCreate(ctx context.Context, conn *sql.Conn, schema, table string) (string, error) {
	rows, err := conn.QueryContext(ctx, "SHOW CREATE TABLE "+qualified(schema, table))
	if err != nil {
		return "", err
	}
	defer rows.Close()
	if !rows.Next() {
		return "", sql.ErrNoRows
	}
	var name, create string
	if err = rows.Scan(&name, &create); err != nil {
		return "", err
	}
	return create, nil
}
func statementObject(s model.Statement) (string, string) {
	if s.DDL != nil {
		return s.DDL.Schema, s.DDL.Table
	}
	if s.DML != nil && len(s.DML.Tables) > 0 {
		return s.DML.Tables[0].Schema, s.DML.Tables[0].Name
	}
	return s.Database, "statement"
}
func qualified(schema, table string) string {
	if schema == "" {
		return quote(table)
	}
	return quote(schema) + "." + quote(table)
}
func quote(v string) string { return "`" + strings.ReplaceAll(v, "`", "``") + "`" }

func ValidateBinlogPrerequisites(ctx context.Context, db *sql.DB) error {
	return binlogmysql.New(db, model.TargetOptions{}).Validate(ctx)
}
