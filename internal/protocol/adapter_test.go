package protocol

import (
	"testing"
	"time"

	"github.com/hanchuanchuan/goinception-plus/internal/model"
)

func TestLegacyResultShapeAndNulls(t *testing.T) {
	result, err := LegacyResult([]model.AuditRecord{{Sequence: 1, Stage: "CHECKED", StageStatus: "Checked", SQL: "select 1"}, {Sequence: 2, Stage: "BACKUP", StageStatus: "BackupSuccessfully", SQL: "update t set c=1", AffectedRows: 1, OperationID: "op", BackupDatabase: "bak", ExecuteTime: 1500 * time.Millisecond, BackupTime: 250 * time.Millisecond}})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Fields) != 12 || string(result.Fields[0].Name) != "order_id" || string(result.Fields[11].Name) != "backup_time" {
		t.Fatalf("unexpected fields: %+v", result.Fields)
	}
	if len(result.RowDatas) != 2 {
		t.Fatalf("unexpected encoded row count: %d", len(result.RowDatas))
	}
	if legacyStatus("BackupSuccessfully") != "Execute Successfully,Backup successfully" || duration(1500*time.Millisecond) != "1.50" {
		t.Fatal("legacy status or duration mapping changed")
	}
}
