package protocol

import (
	"fmt"
	"time"

	gomysql "github.com/go-mysql-org/go-mysql/mysql"
	"github.com/hanchuanchuan/goinception-plus/internal/model"
)

var legacyColumns = []string{"order_id", "stage", "error_level", "stage_status", "error_message", "sql", "affected_rows", "sequence", "backup_dbname", "execute_time", "sqlsha1", "backup_time"}

func LegacyResult(records []model.AuditRecord) (*gomysql.Result, error) {
	rows := make([][]interface{}, 0, len(records))
	for _, r := range records {
		var message, backupDB, sha any
		if r.ErrorMessage != "" {
			message = r.ErrorMessage
		}
		if r.BackupDatabase != "" {
			backupDB = r.BackupDatabase
		}
		if r.SQLSHA1 != "" {
			sha = r.SQLSHA1
		}
		opid := r.OperationID
		if opid == "" {
			opid = fmt.Sprintf("0_0_%08d", r.Sequence-1)
		}
		rows = append(rows, []interface{}{int64(r.Sequence), legacyStage(r.Stage), int64(r.ErrorLevel), legacyStatus(r.StageStatus), message, r.SQL, r.AffectedRows, opid, backupDB, duration(r.ExecuteTime), sha, duration(r.BackupTime)})
	}
	rs, err := gomysql.BuildSimpleTextResultset(legacyColumns, rows)
	if err != nil {
		return nil, err
	}
	if len(rs.Fields) == 12 {
		rs.Fields[0].Type = gomysql.MYSQL_TYPE_LONG
		rs.Fields[2].Type = gomysql.MYSQL_TYPE_SHORT
		rs.Fields[6].Type = gomysql.MYSQL_TYPE_LONGLONG
	}
	return &gomysql.Result{Resultset: rs}, nil
}
func legacyStage(v string) string {
	switch v {
	case "BACKUP":
		return "BACKUP"
	case "EXECUTED":
		return "EXECUTED"
	default:
		return "CHECKED"
	}
}
func legacyStatus(v string) string {
	switch v {
	case "ExecuteSuccessfully":
		return "Execute Successfully"
	case "ExecuteFailed":
		return "Execute failed"
	case "BackupSuccessfully":
		return "Execute Successfully,Backup successfully"
	case "BackupFailed":
		return "Execute Successfully,Backup failed"
	default:
		return "Audit completed"
	}
}
func duration(v time.Duration) string {
	if v <= 0 {
		return "0"
	}
	return fmt.Sprintf("%.2f", v.Seconds())
}
