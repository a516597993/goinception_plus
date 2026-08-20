package backup

import (
	"context"

	"github.com/hanchuanchuan/goinception-plus/internal/audit"
	"github.com/hanchuanchuan/goinception-plus/internal/model"
)

type ChangeType string

const (
	ChangeInsert ChangeType = "INSERT"
	ChangeUpdate ChangeType = "UPDATE"
	ChangeDelete ChangeType = "DELETE"
)

type RowChange struct {
	Sequence      int
	Type          ChangeType
	Table         audit.Table
	Before, After []any
}
type Capture interface {
	Validate(context.Context) error
	CaptureAndExecute(context.Context, model.Statement, func(context.Context) (model.ExecutionResult, error)) ([]RowChange, model.ExecutionResult, error)
}
type Store interface {
	Save(context.Context, Operation) (string, error)
}

type TargetOptions struct {
	Host     string
	Port     uint16
	User     string
	Password string
}

type BinlogPosition struct {
	File     string
	Position uint32
}

type Operation struct {
	OPID      string
	Statement model.Statement
	Source    model.TargetOptions
	Start     BinlogPosition
	End       BinlogPosition
	ThreadID  uint32
	Schema    string
	Table     string
	Rollbacks []RollbackRecord
}

type RollbackRecord struct{ Schema, Table, SQL string }
