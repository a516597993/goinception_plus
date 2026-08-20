package metadata

import (
	"context"
	"errors"
	"testing"

	"github.com/hanchuanchuan/goinception-plus/internal/audit"
	"github.com/hanchuanchuan/goinception-plus/internal/model"
)

type snapshotSource struct{ tables map[string]audit.Table }

func (s *snapshotSource) LoadServerInfo(context.Context) (audit.ServerInfo, error) {
	return audit.ServerInfo{}, nil
}
func (s *snapshotSource) LoadSchema(context.Context, string) (audit.Schema, error) {
	return audit.Schema{}, nil
}
func (s *snapshotSource) LoadTable(_ context.Context, schema, table string) (audit.Table, error) {
	v, ok := s.tables[schema+"."+table]
	if !ok {
		return audit.Table{}, audit.ErrMetadataNotFound
	}
	return v, nil
}

func TestSnapshotCreateAlterDrop(t *testing.T) {
	c := NewRequestCache(&snapshotSource{tables: map[string]audit.Table{}})
	ctx := context.Background()
	create := &model.DDLSpec{Action: model.DDLCreateTable, Schema: "app", Table: "t", FullyProjected: true, Columns: []model.ColumnSpec{{Name: "id", Type: "int"}}}
	if err := c.PrepareDDL(ctx, &model.Statement{DDL: create}); err != nil {
		t.Fatal(err)
	}
	if err := c.ApplyDDL(ctx, create); err != nil {
		t.Fatal(err)
	}
	alter := &model.DDLSpec{Action: model.DDLAlterTable, Schema: "app", Table: "t", Tables: []model.TableRef{{Schema: "app", Name: "t"}}, FullyProjected: true, AlterOperations: []model.AlterOperation{{Action: model.AlterAddColumns, Columns: []model.ColumnSpec{{Name: "name", Type: "varchar(20)"}}}, {Action: model.AlterRenameColumn, Name: "name", NewName: "display_name"}}}
	if err := c.PrepareDDL(ctx, &model.Statement{DDL: alter}); err != nil {
		t.Fatal(err)
	}
	if err := c.ApplyDDL(ctx, alter); err != nil {
		t.Fatal(err)
	}
	table, err := c.LoadTable(ctx, "app", "t")
	if err != nil {
		t.Fatal(err)
	}
	if len(table.Columns) != 2 || table.Columns[1].Name != "display_name" {
		t.Fatalf("unexpected snapshot: %+v", table)
	}
	drop := &model.DDLSpec{Action: model.DDLDropTable, Schema: "app", Tables: []model.TableRef{{Schema: "app", Name: "t"}}, FullyProjected: true}
	if err := c.ApplyDDL(ctx, drop); err != nil {
		t.Fatal(err)
	}
	if _, err = c.LoadTable(ctx, "app", "t"); !errors.Is(err, audit.ErrMetadataNotFound) {
		t.Fatalf("drop tombstone: %v", err)
	}
}

func TestSnapshotCreateLikeCopiesStructure(t *testing.T) {
	s := &snapshotSource{tables: map[string]audit.Table{"app.base": {Schema: "app", Name: "base", Comment: "base", Columns: []audit.Column{{Name: "id", ColumnType: "bigint", Comment: "id"}, {Name: "payload", ColumnType: "varbinary(32)", Nullable: true}}, Indexes: []audit.Index{{Name: "PRIMARY", Columns: []string{"id"}, Primary: true, Unique: true}}}}}
	c := NewRequestCache(s)
	d := &model.DDLSpec{Action: model.DDLCreateTable, Schema: "app", Table: "copy", CreateLike: true, Reference: &model.TableRef{Schema: "app", Name: "base"}, FullyProjected: true}
	if err := c.PrepareDDL(context.Background(), &model.Statement{DDL: d}); err != nil {
		t.Fatal(err)
	}
	if len(d.Columns) != 2 || !d.HasPrimaryKey || !d.HasComment {
		t.Fatalf("LIKE not resolved: %+v", d)
	}
	if !d.Columns[1].HasDefault {
		t.Fatalf("nullable LIKE column must preserve implicit DEFAULT NULL: %+v", d.Columns[1])
	}
	if d.Columns[1].DefaultExpression != "null" {
		t.Fatalf("nullable LIKE column default expression must be null: %+v", d.Columns[1])
	}
}

func TestSnapshotCreateTableDeduplicatesInlineAndProjectedPrimaryKey(t *testing.T) {
	c := NewRequestCache(&snapshotSource{tables: map[string]audit.Table{}})
	d := &model.DDLSpec{
		Action: model.DDLCreateTable, Schema: "app", Table: "dedupe", FullyProjected: true,
		Columns: []model.ColumnSpec{{Name: "id", Type: "bigint", PrimaryKey: true}},
		Indexes: []model.IndexSpec{{Name: "PRIMARY", Columns: []string{"id"}, Primary: true, Unique: true}},
	}
	if err := c.PrepareDDL(context.Background(), &model.Statement{DDL: d}); err != nil {
		t.Fatal(err)
	}
	if err := c.ApplyDDL(context.Background(), d); err != nil {
		t.Fatal(err)
	}
	table, err := c.LoadTable(context.Background(), "app", "dedupe")
	if err != nil {
		t.Fatal(err)
	}
	if len(table.Indexes) != 1 || !table.Indexes[0].Primary {
		t.Fatalf("PRIMARY must exist exactly once: %+v", table.Indexes)
	}
}

func TestSnapshotAlterOperationsEvolveColumnsIndexesAndTableName(t *testing.T) {
	source := &snapshotSource{tables: map[string]audit.Table{"app.t": {
		Schema: "app", Name: "t",
		Columns: []audit.Column{{Name: "id", ColumnType: "bigint", Nullable: false}, {Name: "legacy", ColumnType: "varchar(10)", Nullable: true, Comment: "keep"}},
		Indexes: []audit.Index{{Name: "PRIMARY", Columns: []string{"id"}, Primary: true, Unique: true}, {Name: "idx_legacy", Columns: []string{"legacy"}}},
	}}}
	c := NewRequestCache(source)
	d := &model.DDLSpec{
		Action: model.DDLAlterTable, Schema: "app", Table: "t", Tables: []model.TableRef{{Schema: "app", Name: "t"}}, FullyProjected: true,
		AlterOperations: []model.AlterOperation{
			{Action: model.AlterAddColumns, Columns: []model.ColumnSpec{{Name: "added", Type: "int unsigned", Unsigned: true}}},
			{Action: model.AlterModifyColumn, Columns: []model.ColumnSpec{{Name: "added", Type: "bigint unsigned", Unsigned: true}}},
			{Action: model.AlterChangeColumn, Name: "legacy", Columns: []model.ColumnSpec{{Name: "renamed", Type: "varchar(32)", Nullable: false}}},
			{Action: model.AlterAddIndex, Index: &model.IndexSpec{Name: "idx_added", Columns: []string{"added"}}},
			{Action: model.AlterRenameIndex, Name: "idx_added", NewName: "idx_added_v2"},
			{Action: model.AlterDropColumn, Name: "added"},
			{Action: model.AlterRenameTable, NewTable: &model.TableRef{Schema: "app", Name: "t_v2"}},
		},
	}
	if err := c.PrepareDDL(context.Background(), &model.Statement{DDL: d}); err != nil {
		t.Fatal(err)
	}
	if err := c.ApplyDDL(context.Background(), d); err != nil {
		t.Fatal(err)
	}
	if _, err := c.LoadTable(context.Background(), "app", "t"); !errors.Is(err, audit.ErrMetadataNotFound) {
		t.Fatalf("old table must be tombstoned, got %v", err)
	}
	table, err := c.LoadTable(context.Background(), "app", "t_v2")
	if err != nil {
		t.Fatal(err)
	}
	if len(table.Columns) != 2 || table.Columns[1].Name != "renamed" || table.Columns[1].ColumnType != "varchar(32)" {
		t.Fatalf("unexpected columns after ALTER: %+v", table.Columns)
	}
	if len(table.Indexes) != 2 || table.Indexes[1].Name != "idx_legacy" || len(table.Indexes[1].Columns) != 1 || table.Indexes[1].Columns[0] != "renamed" {
		t.Fatalf("unexpected indexes after ALTER: %+v", table.Indexes)
	}
}
