package binlogmysql

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/binary"
	"fmt"
	"strings"
	"time"

	gomysql "github.com/go-mysql-org/go-mysql/mysql"
	"github.com/go-mysql-org/go-mysql/replication"

	"github.com/hanchuanchuan/goinception-plus/internal/audit"
	"github.com/hanchuanchuan/goinception-plus/internal/backup"
	metadatamysql "github.com/hanchuanchuan/goinception-plus/internal/metadata/mysql"
	"github.com/hanchuanchuan/goinception-plus/internal/model"
)

type Capture struct {
	db       *sql.DB
	target   model.TargetOptions
	metadata *metadatamysql.Provider
	timeout  time.Duration
}

func New(db *sql.DB, target model.TargetOptions) *Capture {
	return &Capture{db: db, target: target, metadata: metadatamysql.New(db), timeout: 30 * time.Second}
}

func (c *Capture) Validate(ctx context.Context) error {
	var enabled, format, image string
	if err := c.db.QueryRowContext(ctx, "SELECT @@log_bin, UPPER(@@binlog_format), UPPER(@@binlog_row_image)").Scan(&enabled, &format, &image); err != nil {
		return fmt.Errorf("read binlog prerequisites: %w", err)
	}
	if enabled != "1" && !strings.EqualFold(enabled, "ON") {
		return fmt.Errorf("log_bin must be ON")
	}
	if format != "ROW" {
		return fmt.Errorf("binlog_format must be ROW")
	}
	if image != "FULL" {
		return fmt.Errorf("binlog_row_image must be FULL")
	}
	return nil
}

func MasterPosition(ctx context.Context, q interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}) (backup.BinlogPosition, error) {
	rows, err := q.QueryContext(ctx, "SHOW MASTER STATUS")
	if err != nil {
		return backup.BinlogPosition{}, err
	}
	defer rows.Close()
	cols, err := rows.Columns()
	if err != nil {
		return backup.BinlogPosition{}, err
	}
	if !rows.Next() {
		return backup.BinlogPosition{}, fmt.Errorf("SHOW MASTER STATUS returned no row")
	}
	values := make([]any, len(cols))
	holders := make([]any, len(cols))
	for i := range values {
		holders[i] = &values[i]
	}
	if err = rows.Scan(holders...); err != nil {
		return backup.BinlogPosition{}, err
	}
	if len(values) < 2 {
		return backup.BinlogPosition{}, fmt.Errorf("SHOW MASTER STATUS returned %d columns", len(values))
	}
	file := asString(values[0])
	var pos uint32
	switch v := values[1].(type) {
	case int64:
		pos = uint32(v)
	case uint64:
		pos = uint32(v)
	case []byte:
		_, err = fmt.Sscan(string(v), &pos)
	case string:
		_, err = fmt.Sscan(v, &pos)
	default:
		err = fmt.Errorf("unsupported binlog position type %T", v)
	}
	if err != nil {
		return backup.BinlogPosition{}, err
	}
	return backup.BinlogPosition{File: file, Position: pos}, nil
}

func (c *Capture) Collect(ctx context.Context, start, end backup.BinlogPosition, threadID uint32, expected int64) ([]backup.RowChange, error) {
	if start == end || expected == 0 {
		return nil, nil
	}
	serverIDBytes := make([]byte, 4)
	if _, err := rand.Read(serverIDBytes); err != nil {
		return nil, err
	}
	serverID := binary.LittleEndian.Uint32(serverIDBytes)
	if serverID < 100 {
		serverID += 100
	}
	syncer := replication.NewBinlogSyncer(replication.BinlogSyncerConfig{ServerID: serverID, Flavor: "mysql", Host: c.target.Host, Port: c.target.Port, User: c.target.User, Password: c.target.Password, UseDecimal: true, ParseTime: true, ReadTimeout: c.timeout})
	defer syncer.Close()
	streamer, err := syncer.StartSync(gomysql.Position{Name: start.File, Pos: start.Position})
	if err != nil {
		return nil, fmt.Errorf("start binlog sync: %w", err)
	}
	readCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	current := gomysql.Position{Name: start.File, Pos: start.Position}
	currentThread := uint32(0)
	changes := make([]backup.RowChange, 0, expected)
	for {
		e, err := streamer.GetEvent(readCtx)
		if err != nil {
			return nil, fmt.Errorf("read binlog event at %s: %w", current.String(), err)
		}
		if e.Header.LogPos > 0 {
			current.Pos = e.Header.LogPos
		}
		if e.Header.EventType == replication.ROTATE_EVENT {
			if rotate, ok := e.Event.(*replication.RotateEvent); ok {
				current = gomysql.Position{Name: string(rotate.NextLogName), Pos: uint32(rotate.Position)}
			}
		}
		switch e.Header.EventType {
		case replication.QUERY_EVENT:
			if q, ok := e.Event.(*replication.QueryEvent); ok {
				currentThread = q.SlaveProxyID
			}
		case replication.XID_EVENT:
			// A binlog stream is ordered, but events from other connections may
			// occur between the requested start and end positions. Never carry a
			// completed transaction's owner into the following transaction.
			currentThread = 0
		case replication.WRITE_ROWS_EVENTv1, replication.WRITE_ROWS_EVENTv2:
			if currentThread == threadID {
				if re, ok := e.Event.(*replication.RowsEvent); ok {
					v, er := c.rowChanges(readCtx, backup.ChangeInsert, re)
					if er != nil {
						return nil, er
					}
					changes = append(changes, v...)
				}
			}
		case replication.DELETE_ROWS_EVENTv1, replication.DELETE_ROWS_EVENTv2:
			if currentThread == threadID {
				if re, ok := e.Event.(*replication.RowsEvent); ok {
					v, er := c.rowChanges(readCtx, backup.ChangeDelete, re)
					if er != nil {
						return nil, er
					}
					changes = append(changes, v...)
				}
			}
		case replication.UPDATE_ROWS_EVENTv1, replication.UPDATE_ROWS_EVENTv2:
			if currentThread == threadID {
				if re, ok := e.Event.(*replication.RowsEvent); ok {
					v, er := c.rowChanges(readCtx, backup.ChangeUpdate, re)
					if er != nil {
						return nil, er
					}
					changes = append(changes, v...)
				}
			}
		}
		if current.Compare(gomysql.Position{Name: end.File, Pos: end.Position}) >= 0 {
			break
		}
	}
	if int64(len(changes)) != expected {
		return nil, fmt.Errorf("ROW binlog incomplete: captured_rows=%d affected_rows=%d", len(changes), expected)
	}
	return changes, nil
}

func (c *Capture) rowChanges(ctx context.Context, kind backup.ChangeType, e *replication.RowsEvent) ([]backup.RowChange, error) {
	if e.Table == nil {
		return nil, fmt.Errorf("row event has no TABLE_MAP")
	}
	schema, table := string(e.Table.Schema), string(e.Table.Table)
	meta, err := c.metadata.LoadTable(ctx, schema, table)
	if err != nil {
		return nil, fmt.Errorf("load row event metadata %s.%s: %w", schema, table, err)
	}
	if uint64(len(meta.Columns)) != e.ColumnCount {
		return nil, fmt.Errorf("FULL row image mismatch for %s.%s: metadata=%d binlog=%d", schema, table, len(meta.Columns), e.ColumnCount)
	}
	result := make([]backup.RowChange, 0, len(e.Rows))
	if kind == backup.ChangeUpdate {
		if len(e.Rows)%2 != 0 {
			return nil, fmt.Errorf("UPDATE row image has odd row count")
		}
		for i := 0; i < len(e.Rows); i += 2 {
			result = append(result, backup.RowChange{Type: kind, Table: meta, Before: e.Rows[i], After: e.Rows[i+1]})
		}
		return result, nil
	}
	for _, row := range e.Rows {
		change := backup.RowChange{Type: kind, Table: meta}
		if kind == backup.ChangeInsert {
			change.After = row
		} else {
			change.Before = row
		}
		result = append(result, change)
	}
	return result, nil
}
func asString(v any) string {
	switch x := v.(type) {
	case []byte:
		return string(x)
	case string:
		return x
	default:
		return fmt.Sprint(x)
	}
}

var _ audit.MetadataProvider = (*metadatamysql.Provider)(nil)
