package mysqlstore

import (
	"testing"

	"github.com/hanchuanchuan/goinception-plus/internal/model"
)

func TestLegacyStatementType(t *testing.T) {
	tests := map[model.StatementKind]string{
		model.StatementInsert:         "INSERT",
		model.StatementUpdate:         "UPDATE",
		model.StatementDelete:         "DELETE",
		model.StatementCreateDatabase: "CREATEDB",
		model.StatementCreateTable:    "CREATETABLE",
		model.StatementAlterTable:     "ALTERTABLE",
		model.StatementDropTable:      "DROPTABLE",
		model.StatementRenameTable:    "RENAMETABLE",
		model.StatementCreateIndex:    "CREATEINDEX",
		model.StatementDropIndex:      "DROPINDEX",
		model.StatementSelect:         "UNKNOWN",
	}
	for kind, want := range tests {
		if got := legacyStatementType(kind); got != want {
			t.Errorf("legacyStatementType(%s) = %q, want %q", kind, got, want)
		}
	}
}
