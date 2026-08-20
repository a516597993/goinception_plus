package metadata

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/hanchuanchuan/goinception-plus/internal/audit"
	"github.com/hanchuanchuan/goinception-plus/internal/model"
)

// PrepareDDL resolves source metadata required to audit a DDL statement. It
// never mutates the snapshot; ApplyDDL performs mutations after admission.
func (p *Provider) PrepareDDL(ctx context.Context, stmt *model.Statement) error {
	d := stmt.DDL
	if d == nil || !d.FullyProjected {
		return nil
	}
	if d.Action == model.DDLCreateTable || d.Action == model.DDLCreateView {
		if _, err := p.LoadTable(ctx, d.Schema, d.Table); err == nil {
			if d.IfNotExists {
				d.Noop = true
				return nil
			}
			return fmt.Errorf("table %s.%s already exists", d.Schema, d.Table)
		} else if !errors.Is(err, audit.ErrMetadataNotFound) {
			return err
		}
		if d.CreateLike {
			r := *d.Reference
			if r.Schema == "" {
				r.Schema = d.Schema
				d.Reference = &r
			}
			t, err := p.LoadTable(ctx, r.Schema, r.Name)
			if err != nil {
				return fmt.Errorf("source table %s.%s: %w", r.Schema, r.Name, err)
			}
			d.Columns = columnsFromTable(t)
			d.Indexes = indexesFromTable(t)
			deriveFlags(d, t)
		}
		return nil
	}
	if d.Action == model.DDLRenameTable {
		for _, pair := range d.RenamePairs {
			toSchema := defaultSchema(pair.To.Schema, defaultSchema(pair.From.Schema, d.Schema))
			if _, err := p.LoadTable(ctx, toSchema, pair.To.Name); err == nil {
				return fmt.Errorf("rename target %s.%s already exists", toSchema, pair.To.Name)
			} else if !errors.Is(err, audit.ErrMetadataNotFound) {
				return err
			}
		}
	}
	for _, r := range affectedTables(d) {
		if r.Schema == "" {
			r.Schema = d.Schema
		}
		if _, err := p.LoadTable(ctx, r.Schema, r.Name); err != nil {
			if d.IfExists && errors.Is(err, audit.ErrMetadataNotFound) {
				continue
			}
			return fmt.Errorf("table %s.%s: %w", r.Schema, r.Name, err)
		}
	}
	return nil
}

// ApplyDDL updates the request-local schema view in SQL order.
func (p *Provider) ApplyDDL(ctx context.Context, d *model.DDLSpec) error {
	if d == nil || !d.FullyProjected {
		return nil
	}
	if d.Noop {
		return nil
	}
	switch d.Action {
	case model.DDLCreateDatabase:
		p.PutSchema(audit.Schema{Name: d.Schema})
	case model.DDLDropDatabase:
		p.DeleteSchema(d.Schema)
	case model.DDLCreateTable:
		p.PutTable(tableFromSpec(d))
	case model.DDLCreateView:
		p.PutTable(audit.Table{Schema: d.Schema, Name: d.Table, IsView: true, TableType: "VIEW"})
	case model.DDLDropTable:
		for _, r := range d.Tables {
			p.DeleteTable(defaultSchema(r.Schema, d.Schema), r.Name)
		}
	case model.DDLDropView:
		for _, r := range d.Tables {
			p.DeleteTable(defaultSchema(r.Schema, d.Schema), r.Name)
		}
	case model.DDLRenameTable:
		for _, pair := range d.RenamePairs {
			if err := p.renameLoaded(ctx, pair.From, pair.To, d.Schema); err != nil {
				return err
			}
		}
	case model.DDLTruncateTable:
		// Structure is unchanged; retaining the cached table is intentional.
	case model.DDLCreateIndex, model.DDLDropIndex, model.DDLAlterTable:
		t, err := p.LoadTable(ctx, d.Schema, d.Table)
		if err != nil {
			return err
		}
		oldSchema, oldName := t.Schema, t.Name
		if err = applyTableDDL(&t, d); err != nil {
			return err
		}
		if oldSchema != t.Schema || oldName != t.Name {
			p.RenameTable(oldSchema, oldName, t)
		} else {
			p.PutTable(t)
		}
	}
	return nil
}

func (p *Provider) renameLoaded(ctx context.Context, from, to model.TableRef, fallback string) error {
	from.Schema = defaultSchema(from.Schema, fallback)
	to.Schema = defaultSchema(to.Schema, from.Schema)
	t, err := p.LoadTable(ctx, from.Schema, from.Name)
	if err != nil {
		return err
	}
	t.Schema = to.Schema
	t.Name = to.Name
	p.RenameTable(from.Schema, from.Name, t)
	return nil
}
func affectedTables(d *model.DDLSpec) []model.TableRef {
	switch d.Action {
	case model.DDLAlterTable, model.DDLTruncateTable, model.DDLCreateIndex, model.DDLDropIndex:
		return d.Tables
	case model.DDLDropTable, model.DDLDropView:
		return d.Tables
	case model.DDLRenameTable:
		r := make([]model.TableRef, 0, len(d.RenamePairs))
		for _, p := range d.RenamePairs {
			r = append(r, p.From)
		}
		return r
	}
	return nil
}
func defaultSchema(v, f string) string {
	if v == "" {
		return f
	}
	return v
}
func columnsFromTable(t audit.Table) []model.ColumnSpec {
	r := make([]model.ColumnSpec, 0, len(t.Columns))
	for _, c := range t.Columns {
		defaultExpression := ""
		hasDefault := c.Default != nil || c.Nullable
		if c.Default != nil {
			defaultExpression = *c.Default
		} else if c.Nullable {
			defaultExpression = "null"
		}
		r = append(r, model.ColumnSpec{Name: c.Name, Type: c.ColumnType, Nullable: c.Nullable, HasComment: c.Comment != "", AutoIncrement: c.AutoIncrement, Unsigned: c.Unsigned, HasDefault: hasDefault, DefaultExpression: defaultExpression, Generated: c.GenerationExpression != "", CharacterSet: c.CharacterSet})
	}
	return r
}
func indexesFromTable(t audit.Table) []model.IndexSpec {
	r := make([]model.IndexSpec, 0, len(t.Indexes))
	for _, i := range t.Indexes {
		r = append(r, model.IndexSpec{Name: i.Name, Columns: append([]string(nil), i.Columns...), Unique: i.Unique, Primary: i.Primary, PrefixLengths: append([]int(nil), i.PrefixLengths...), Expressions: append([]string(nil), i.Expressions...)})
	}
	return r
}
func deriveFlags(d *model.DDLSpec, t audit.Table) {
	d.HasComment = t.Comment != ""
	for _, i := range t.Indexes {
		if i.Primary {
			d.HasPrimaryKey = true
		}
	}
}
func tableFromSpec(d *model.DDLSpec) audit.Table {
	t := audit.Table{Schema: d.Schema, Name: d.Table, Engine: d.Engine, CharacterSet: d.CharacterSet, Collation: d.Collation}
	for _, c := range d.Columns {
		t.Columns = append(t.Columns, audit.Column{Name: c.Name, ColumnType: c.Type, Nullable: c.Nullable, AutoIncrement: c.AutoIncrement, Unsigned: c.Unsigned, CharacterSet: c.CharacterSet})
		if c.PrimaryKey && findIndex(&t, "PRIMARY") < 0 {
			t.Indexes = append(t.Indexes, audit.Index{Name: "PRIMARY", Columns: []string{c.Name}, Unique: true, Primary: true, Visible: true})
		}
	}
	for _, i := range d.Indexes {
		index := audit.Index{Name: i.Name, Columns: append([]string(nil), i.Columns...), Unique: i.Unique, Primary: i.Primary, PrefixLengths: append([]int(nil), i.PrefixLengths...), Expressions: append([]string(nil), i.Expressions...), Visible: true}
		if position := findIndex(&t, i.Name); position >= 0 {
			t.Indexes[position] = index
		} else {
			t.Indexes = append(t.Indexes, index)
		}
	}
	return t
}

func applyTableDDL(t *audit.Table, d *model.DDLSpec) error {
	if d.Action == model.DDLCreateIndex {
		for _, i := range d.Indexes {
			if findIndex(t, i.Name) >= 0 {
				return fmt.Errorf("index %s already exists", i.Name)
			}
			t.Indexes = append(t.Indexes, audit.Index{Name: i.Name, Columns: i.Columns, Unique: i.Unique, Visible: true})
		}
		return nil
	}
	if d.Action == model.DDLDropIndex {
		for _, i := range d.Indexes {
			n := findIndex(t, i.Name)
			if n < 0 {
				return fmt.Errorf("index %s does not exist", i.Name)
			}
			t.Indexes = append(t.Indexes[:n], t.Indexes[n+1:]...)
		}
		return nil
	}
	for _, op := range d.AlterOperations {
		switch op.Action {
		case model.AlterAddColumns:
			for _, c := range op.Columns {
				if findColumn(t, c.Name) >= 0 {
					return fmt.Errorf("column %s already exists", c.Name)
				}
				t.Columns = append(t.Columns, columnFromSpec(c))
			}
		case model.AlterDropColumn:
			n := findColumn(t, op.Name)
			if n < 0 {
				return fmt.Errorf("column %s does not exist", op.Name)
			}
			t.Columns = append(t.Columns[:n], t.Columns[n+1:]...)
			dropColumnFromIndexes(t, op.Name)
		case model.AlterModifyColumn:
			if len(op.Columns) != 1 {
				return fmt.Errorf("invalid MODIFY COLUMN projection")
			}
			n := findColumn(t, op.Columns[0].Name)
			if n < 0 {
				return fmt.Errorf("column %s does not exist", op.Columns[0].Name)
			}
			t.Columns[n] = mergeColumnSpec(t.Columns[n], op.Columns[0])
		case model.AlterChangeColumn:
			if len(op.Columns) != 1 {
				return fmt.Errorf("invalid CHANGE COLUMN projection")
			}
			n := findColumn(t, op.Name)
			if n < 0 {
				return fmt.Errorf("column %s does not exist", op.Name)
			}
			oldName := t.Columns[n].Name
			t.Columns[n] = mergeColumnSpec(t.Columns[n], op.Columns[0])
			renameColumnInIndexes(t, oldName, op.Columns[0].Name)
		case model.AlterRenameColumn:
			n := findColumn(t, op.Name)
			if n < 0 {
				return fmt.Errorf("column %s does not exist", op.Name)
			}
			if findColumn(t, op.NewName) >= 0 {
				return fmt.Errorf("column %s already exists", op.NewName)
			}
			t.Columns[n].Name = op.NewName
			renameColumnInIndexes(t, op.Name, op.NewName)
		case model.AlterAddIndex:
			if op.Index == nil {
				return fmt.Errorf("missing index projection")
			}
			if findIndex(t, op.Index.Name) >= 0 {
				return fmt.Errorf("index %s already exists", op.Index.Name)
			}
			t.Indexes = append(t.Indexes, audit.Index{Name: op.Index.Name, Columns: op.Index.Columns, Unique: op.Index.Unique, Primary: op.Index.Primary, Visible: true})
		case model.AlterDropIndex, model.AlterDropPrimaryKey:
			n := findIndex(t, op.Name)
			if n < 0 {
				return fmt.Errorf("index %s does not exist", op.Name)
			}
			t.Indexes = append(t.Indexes[:n], t.Indexes[n+1:]...)
		case model.AlterRenameIndex:
			n := findIndex(t, op.Name)
			if n < 0 {
				return fmt.Errorf("index %s does not exist", op.Name)
			}
			if findIndex(t, op.NewName) >= 0 {
				return fmt.Errorf("index %s already exists", op.NewName)
			}
			t.Indexes[n].Name = op.NewName
		case model.AlterRenameTable:
			if op.NewTable == nil {
				return fmt.Errorf("missing rename target")
			}
			t.Schema = defaultSchema(op.NewTable.Schema, t.Schema)
			t.Name = op.NewTable.Name
		case model.AlterTableOptions:
			if d.HasEngineOption {
				t.Engine = d.Engine
			}
			if d.HasCharsetOption {
				t.CharacterSet = d.CharacterSet
			}
			if d.HasCollationOption {
				t.Collation = d.Collation
			}
		}
	}
	return nil
}

func columnFromSpec(c model.ColumnSpec) audit.Column {
	column := audit.Column{
		Name:          c.Name,
		ColumnType:    c.Type,
		Nullable:      c.Nullable,
		AutoIncrement: c.AutoIncrement,
		Unsigned:      c.Unsigned,
		CharacterSet:  c.CharacterSet,
		Collation:     c.Collation,
	}
	if c.Generated {
		column.GenerationExpression = "generated"
	}
	if c.OnUpdate {
		column.Extra = "on update"
	}
	if c.HasComment {
		column.Comment = "comment"
	}
	if c.HasDefault {
		value := c.DefaultExpression
		column.Default = &value
	}
	return column
}

func mergeColumnSpec(current audit.Column, spec model.ColumnSpec) audit.Column {
	current.Name = spec.Name
	current.ColumnType = spec.Type
	current.Nullable = spec.Nullable
	current.AutoIncrement = spec.AutoIncrement
	current.Unsigned = spec.Unsigned
	if spec.HasComment {
		current.Comment = "comment"
	}
	if spec.HasDefault {
		value := spec.DefaultExpression
		current.Default = &value
	}
	if spec.CharacterSet != "" {
		current.CharacterSet = spec.CharacterSet
	}
	if spec.Collation != "" {
		current.Collation = spec.Collation
	}
	if spec.Generated {
		current.GenerationExpression = "generated"
	}
	if spec.OnUpdate {
		current.Extra = "on update"
	}
	return current
}

func renameColumnInIndexes(t *audit.Table, oldName, newName string) {
	for i := range t.Indexes {
		for j, name := range t.Indexes[i].Columns {
			if strings.EqualFold(name, oldName) {
				t.Indexes[i].Columns[j] = newName
			}
		}
	}
}

func dropColumnFromIndexes(t *audit.Table, column string) {
	indexes := t.Indexes[:0]
	for _, index := range t.Indexes {
		columns := index.Columns[:0]
		prefixLengths := index.PrefixLengths[:0]
		for position, name := range index.Columns {
			if !strings.EqualFold(name, column) {
				columns = append(columns, name)
				if position < len(index.PrefixLengths) {
					prefixLengths = append(prefixLengths, index.PrefixLengths[position])
				}
			}
		}
		index.Columns = columns
		index.PrefixLengths = prefixLengths
		if len(index.Columns) > 0 || len(index.Expressions) > 0 {
			indexes = append(indexes, index)
		}
	}
	t.Indexes = indexes
}
func findColumn(t *audit.Table, n string) int {
	for i, v := range t.Columns {
		if strings.EqualFold(v.Name, n) {
			return i
		}
	}
	return -1
}
func findIndex(t *audit.Table, n string) int {
	for i, v := range t.Indexes {
		if strings.EqualFold(v.Name, n) {
			return i
		}
	}
	return -1
}
