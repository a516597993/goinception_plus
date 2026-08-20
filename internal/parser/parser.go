// Package parser is the only package allowed to expose TiDB parser types.
package parser

import (
	"fmt"
	"strings"

	tidbparser "github.com/pingcap/tidb/pkg/parser"
	"github.com/pingcap/tidb/pkg/parser/ast"
	"github.com/pingcap/tidb/pkg/parser/format"
	parsermysql "github.com/pingcap/tidb/pkg/parser/mysql"
	"github.com/pingcap/tidb/pkg/parser/opcode"
	_ "github.com/pingcap/tidb/pkg/parser/test_driver"

	"github.com/hanchuanchuan/goinception-plus/internal/model"
)

type SQLParser interface {
	Parse(sql, sqlMode string) (model.ParseResult, error)
}

type TiDBParser struct{}

func New() *TiDBParser { return &TiDBParser{} }

func (p *TiDBParser) Parse(sql, sqlMode string) (model.ParseResult, error) {
	mode, err := parsermysql.GetSQLMode(sqlMode)
	if err != nil {
		return model.ParseResult{}, fmt.Errorf("invalid sql_mode %q: %w", sqlMode, err)
	}

	impl := tidbparser.New()
	impl.SetSQLMode(mode)
	nodes, warnings, err := impl.ParseSQL(sql)
	if err != nil {
		return model.ParseResult{}, fmt.Errorf("TiDB 8.5 parser does not support the SQL: %w", err)
	}

	result := make([]model.Statement, 0, len(nodes))
	for i, node := range nodes {
		start := node.OriginTextPosition()
		end := start + len(node.OriginalText())
		statement := model.Statement{
			Sequence:    i + 1,
			Original:    node.OriginalText(),
			Normalized:  restore(node),
			Kind:        classify(node),
			StartOffset: start,
			EndOffset:   end,
		}
		statement.DDL = projectDDL(node)
		statement.DML = projectDML(node)
		if use, ok := node.(*ast.UseStmt); ok {
			statement.Database = use.DBName
		}
		statement.Supported = isSupported(statement)
		result = append(result, statement)
	}
	parseWarnings := make([]model.ParseWarning, 0, len(warnings))
	for _, warning := range warnings {
		parseWarnings = append(parseWarnings, model.ParseWarning{Message: warning.Error()})
	}
	return model.ParseResult{Statements: result, Warnings: parseWarnings}, nil
}

func isSupported(statement model.Statement) bool {
	switch statement.Kind {
	case model.StatementUse, model.StatementInsert, model.StatementUpdate,
		model.StatementDelete, model.StatementSelect:
		return statement.Kind == model.StatementUse || (statement.DML != nil && statement.DML.FullyProjected)
	case model.StatementCreateDatabase, model.StatementAlterDatabase, model.StatementDropDatabase,
		model.StatementCreateTable, model.StatementAlterTable, model.StatementDropTable,
		model.StatementRenameTable, model.StatementTruncateTable, model.StatementCreateIndex,
		model.StatementDropIndex, model.StatementCreateView, model.StatementDropView:
		return statement.DDL != nil && statement.DDL.FullyProjected
	default:
		return false
	}
}

func projectDML(node ast.StmtNode) *model.DMLSpec {
	v := &tableCollector{cteNames: cteNames(node)}
	_, _ = node.Accept(v)
	s := &model.DMLSpec{Tables: v.tables, FullyProjected: true, HasJoinWithoutOn: v.joinWithoutOn, OrderByRand: v.orderByRand, WrongAndExpr: v.wrongAnd, Comparisons: v.comparisons}
	switch n := node.(type) {
	case *ast.InsertStmt:
		for _, c := range n.Columns {
			s.Columns = append(s.Columns, c.Name.O)
		}
		s.InsertSelect = n.Select != nil
		s.OnDuplicate = len(n.OnDuplicate) > 0
		s.ValueRows = len(n.Lists)
	case *ast.UpdateStmt:
		s.UsesCTE = n.With != nil
		s.HasWhere = n.Where != nil
		s.HasLimit = n.Limit != nil
		s.HasOrderBy = n.Order != nil
		s.MultiTable = n.MultipleTable
		for _, a := range n.List {
			s.Columns = append(s.Columns, a.Column.Name.O)
			s.Assignments = append(s.Assignments, model.ColumnRef{Schema: a.Column.Schema.O, Table: a.Column.Table.O, Name: a.Column.Name.O})
			if expression, ok := a.Expr.(*ast.BinaryOperationExpr); ok && strings.EqualFold(expression.Op.String(), "and") {
				s.WrongAndExpr = true
			}
		}
	case *ast.DeleteStmt:
		s.UsesCTE = n.With != nil
		s.HasWhere = n.Where != nil
		s.HasLimit = n.Limit != nil
		s.HasOrderBy = n.Order != nil
		s.MultiTable = n.IsMultiTable
	case *ast.SelectStmt:
		s.UsesCTE = n.With != nil
		s.HasWhere = n.Where != nil
		s.HasLimit = n.Limit != nil
		s.HasOrderBy = n.OrderBy != nil
		for _, f := range n.Fields.Fields {
			if f.WildCard != nil {
				s.HasSelectStar = true
			}
		}
	default:
		return nil
	}
	s.OrderByRand = strings.Contains(strings.ReplaceAll(strings.ToLower(restore(node)), " ", ""), "orderbyrand(")
	return s
}

type tableCollector struct {
	tables        []model.TableRef
	joinWithoutOn bool
	cteNames      map[string]bool
	orderByRand   bool
	wrongAnd      bool
	comparisons   []model.ComparisonSpec
}

func (v *tableCollector) Enter(n ast.Node) (ast.Node, bool) {
	if _, ok := n.(*ast.DeleteTableList); ok {
		return n, true
	}
	if source, ok := n.(*ast.TableSource); ok {
		if table, tableOK := source.Source.(*ast.TableName); tableOK {
			if !v.cteNames[strings.ToLower(table.Name.O)] {
				r := tableRef(table)
				r.Alias = source.AsName.O
				v.addTable(r)
			}
			return n, true
		}
	}
	if t, ok := n.(*ast.TableName); ok {
		if !v.cteNames[strings.ToLower(t.Name.O)] {
			v.addTable(tableRef(t))
		}
	}
	if j, ok := n.(*ast.Join); ok && j.Right != nil && j.On == nil && len(j.Using) == 0 && !j.NaturalJoin {
		v.joinWithoutOn = true
	}
	if b, ok := n.(*ast.BinaryOperationExpr); ok {
		op := strings.ToLower(b.Op.String())
		if op == "and" && (isConstantExpr(b.L) || isConstantExpr(b.R)) {
			v.wrongAnd = true
		}
		if op == "=" || op == "!=" || op == "<>" || op == "<" || op == ">" || op == "<=" || op == ">=" || op == "eq" || op == "ne" || op == "lt" || op == "gt" || op == "le" || op == "ge" {
			if comparison, ok := projectComparison(b.L, b.R); ok {
				v.comparisons = append(v.comparisons, comparison)
			} else if comparison, ok = projectComparison(b.R, b.L); ok {
				v.comparisons = append(v.comparisons, comparison)
			}
		}
	}
	return n, false
}
func (v *tableCollector) Leave(n ast.Node) (ast.Node, bool) { return n, true }

func (v *tableCollector) addTable(r model.TableRef) {
	for _, existing := range v.tables {
		if strings.EqualFold(existing.Schema, r.Schema) && strings.EqualFold(existing.Name, r.Name) && strings.EqualFold(existing.Alias, r.Alias) {
			return
		}
	}
	v.tables = append(v.tables, r)
}

func isConstantExpr(expr ast.ExprNode) bool {
	_, ok := expr.(ast.ValueExpr)
	return ok
}

func projectComparison(left, right ast.ExprNode) (model.ComparisonSpec, bool) {
	column, ok := left.(*ast.ColumnNameExpr)
	if !ok {
		return model.ComparisonSpec{}, false
	}
	kind, ok := projectLiteralExpr(right)
	if !ok {
		return model.ComparisonSpec{}, false
	}
	return model.ComparisonSpec{Column: model.ColumnRef{Schema: column.Name.Schema.O, Table: column.Name.Table.O, Name: column.Name.Name.O}, LiteralKind: kind}, true
}

func projectLiteralExpr(expr ast.ExprNode) (model.LiteralKind, bool) {
	if value, ok := expr.(ast.ValueExpr); ok {
		return projectLiteralKind(value), true
	}
	unary, ok := expr.(*ast.UnaryOperationExpr)
	if !ok || (unary.Op != opcode.Plus && unary.Op != opcode.Minus) {
		return model.LiteralUnknown, false
	}
	value, ok := unary.V.(ast.ValueExpr)
	if !ok {
		return model.LiteralUnknown, false
	}
	kind := projectLiteralKind(value)
	if unary.Op == opcode.Minus && kind == model.LiteralUnsignedInteger {
		kind = model.LiteralSignedInteger
	}
	return kind, true
}

func projectLiteralKind(value ast.ValueExpr) model.LiteralKind {
	typeInfo := value.GetType()
	if typeInfo == nil {
		return model.LiteralUnknown
	}
	flags := typeInfo.GetFlag()
	if parsermysql.HasIsBooleanFlag(flags) {
		return model.LiteralBoolean
	}
	switch typeInfo.GetType() {
	case parsermysql.TypeNull:
		return model.LiteralNull
	case parsermysql.TypeTiny, parsermysql.TypeShort, parsermysql.TypeInt24, parsermysql.TypeLong, parsermysql.TypeLonglong, parsermysql.TypeYear:
		if parsermysql.HasUnsignedFlag(flags) {
			return model.LiteralUnsignedInteger
		}
		return model.LiteralSignedInteger
	case parsermysql.TypeFloat, parsermysql.TypeDouble:
		return model.LiteralFloat
	case parsermysql.TypeNewDecimal:
		return model.LiteralDecimal
	case parsermysql.TypeDate, parsermysql.TypeNewDate, parsermysql.TypeDatetime, parsermysql.TypeTimestamp:
		return model.LiteralTemporal
	case parsermysql.TypeDuration:
		return model.LiteralDuration
	case parsermysql.TypeJSON:
		return model.LiteralJSON
	case parsermysql.TypeBit, parsermysql.TypeTinyBlob, parsermysql.TypeMediumBlob, parsermysql.TypeLongBlob, parsermysql.TypeBlob, parsermysql.TypeGeometry:
		return model.LiteralBinary
	case parsermysql.TypeVarchar, parsermysql.TypeVarString, parsermysql.TypeString, parsermysql.TypeEnum, parsermysql.TypeSet:
		if parsermysql.HasBinaryFlag(flags) {
			return model.LiteralBinary
		}
		return model.LiteralString
	default:
		return model.LiteralUnknown
	}
}

func cteNames(node ast.StmtNode) map[string]bool {
	result := map[string]bool{}
	var with *ast.WithClause
	switch n := node.(type) {
	case *ast.UpdateStmt:
		with = n.With
	case *ast.DeleteStmt:
		with = n.With
	case *ast.SelectStmt:
		with = n.With
	}
	if with != nil {
		for _, cte := range with.CTEs {
			result[strings.ToLower(cte.Name.O)] = true
		}
	}
	return result
}

func projectDDL(node ast.StmtNode) *model.DDLSpec {
	switch n := node.(type) {
	case *ast.CreateDatabaseStmt:
		d := &model.DDLSpec{Action: model.DDLCreateDatabase, Schema: n.Name.O, FullyProjected: true, IfNotExists: n.IfNotExists}
		projectDatabaseOptions(d, n.Options)
		return d
	case *ast.AlterDatabaseStmt:
		d := &model.DDLSpec{Action: model.DDLAlterDatabase, Schema: n.Name.O, FullyProjected: len(n.Options) > 0}
		projectDatabaseOptions(d, n.Options)
		return d
	case *ast.DropDatabaseStmt:
		return &model.DDLSpec{Action: model.DDLDropDatabase, Schema: n.Name.O, FullyProjected: true, IfExists: n.IfExists}
	case *ast.CreateTableStmt:
		return projectCreateTable(n)
	case *ast.CreateViewStmt:
		r := tableRef(n.ViewName)
		return &model.DDLSpec{Action: model.DDLCreateView, Schema: r.Schema, Table: r.Name, Tables: []model.TableRef{r}, FullyProjected: true}
	case *ast.AlterTableStmt:
		return projectAlterTable(n)
	case *ast.DropTableStmt:
		action := model.DDLDropTable
		if n.IsView {
			action = model.DDLDropView
		}
		s := &model.DDLSpec{Action: action, FullyProjected: true, IfExists: n.IfExists}
		for _, t := range n.Tables {
			s.Tables = append(s.Tables, tableRef(t))
		}
		return s
	case *ast.RenameTableStmt:
		s := &model.DDLSpec{Action: model.DDLRenameTable, FullyProjected: true}
		for _, p := range n.TableToTables {
			s.RenamePairs = append(s.RenamePairs, model.RenamePair{From: tableRef(p.OldTable), To: tableRef(p.NewTable)})
		}
		return s
	case *ast.TruncateTableStmt:
		r := tableRef(n.Table)
		return &model.DDLSpec{Action: model.DDLTruncateTable, Schema: r.Schema, Table: r.Name, Tables: []model.TableRef{r}, FullyProjected: true}
	case *ast.CreateIndexStmt:
		r := tableRef(n.Table)
		idx := projectIndex(n.IndexName, n.KeyType == ast.IndexKeyTypeUnique, n.IndexPartSpecifications)
		return &model.DDLSpec{Action: model.DDLCreateIndex, Schema: r.Schema, Table: r.Name, Tables: []model.TableRef{r}, Indexes: []model.IndexSpec{idx}, FullyProjected: n.KeyType != ast.IndexKeyTypeSpatial && n.KeyType != ast.IndexKeyTypeFullText && n.KeyType != ast.IndexKeyTypeVector}
	case *ast.DropIndexStmt:
		r := tableRef(n.Table)
		return &model.DDLSpec{Action: model.DDLDropIndex, Schema: r.Schema, Table: r.Name, Tables: []model.TableRef{r}, Indexes: []model.IndexSpec{{Name: n.IndexName}}, FullyProjected: !n.IsHypo}
	default:
		return nil
	}
}

func projectDatabaseOptions(d *model.DDLSpec, options []*ast.DatabaseOption) {
	for _, option := range options {
		switch option.Tp {
		case ast.DatabaseOptionCharset:
			d.CharacterSet, d.HasCharsetOption = option.Value, true
		case ast.DatabaseOptionCollate:
			d.Collation, d.HasCollationOption = option.Value, true
		}
	}
}

func projectCreateTable(create *ast.CreateTableStmt) *model.DDLSpec {
	spec := &model.DDLSpec{
		Action:         model.DDLCreateTable,
		Schema:         create.Table.Schema.O,
		Table:          create.Table.Name.O,
		Columns:        make([]model.ColumnSpec, 0, len(create.Cols)),
		FullyProjected: create.Select == nil,
		IfNotExists:    create.IfNotExists,
	}
	if create.ReferTable != nil {
		r := tableRef(create.ReferTable)
		spec.Reference = &r
		spec.CreateLike = true
	}
	if create.Select != nil {
		spec.CreateSelect = true
		spec.FullyProjected = false
	}
	for _, option := range create.Options {
		switch option.Tp {
		case ast.TableOptionComment:
			spec.HasComment = option.StrValue != ""
		case ast.TableOptionEngine:
			spec.Engine = option.StrValue
			spec.HasEngineOption = true
		case ast.TableOptionCharset:
			spec.CharacterSet = option.StrValue
			spec.HasCharsetOption = true
		case ast.TableOptionCollate:
			spec.Collation = option.StrValue
			spec.HasCollationOption = true
		case ast.TableOptionAutoIncrement:
			spec.AutoIncrementValue = option.UintValue
		}
	}
	spec.Partitioned = create.Partition != nil
	for _, column := range create.Cols {
		item := projectColumn(column)
		if item.PrimaryKey {
			spec.HasPrimaryKey = true
		}
		spec.Columns = append(spec.Columns, item)
	}
	for _, constraint := range create.Constraints {
		index := model.IndexSpec{Name: constraint.Name}
		switch constraint.Tp {
		case ast.ConstraintPrimaryKey:
			if index.Name == "" {
				index.Name = "PRIMARY"
			}
			index.Primary = true
			index.Unique = true
			spec.HasPrimaryKey = true
		case ast.ConstraintUniq, ast.ConstraintUniqKey, ast.ConstraintUniqIndex:
			index.Unique = true
		case ast.ConstraintKey, ast.ConstraintIndex:
		case ast.ConstraintForeignKey:
			spec.HasForeignKey = true
			continue
		default:
			continue
		}
		for _, key := range constraint.Keys {
			if key.Column != nil {
				index.Columns = append(index.Columns, key.Column.Name.O)
				index.PrefixLengths = append(index.PrefixLengths, key.Length)
			} else {
				index.Expressions = append(index.Expressions, restoreExpr(key.Expr))
			}
		}
		spec.Indexes = append(spec.Indexes, index)
	}
	primaryPresent := false
	for _, index := range spec.Indexes {
		if index.Primary {
			primaryPresent = true
			break
		}
	}
	if !primaryPresent {
		for _, column := range spec.Columns {
			if column.PrimaryKey {
				spec.Indexes = append(spec.Indexes, model.IndexSpec{Name: "PRIMARY", Columns: []string{column.Name}, Primary: true, Unique: true})
			}
		}
	}
	return spec
}

func projectAlterTable(n *ast.AlterTableStmt) *model.DDLSpec {
	r := tableRef(n.Table)
	s := &model.DDLSpec{Action: model.DDLAlterTable, Schema: r.Schema, Table: r.Name, Tables: []model.TableRef{r}, FullyProjected: true}
	for _, a := range n.Specs {
		op := model.AlterOperation{}
		switch a.Tp {
		case ast.AlterTableOption:
			op.Action = model.AlterTableOptions
			for _, option := range a.Options {
				switch option.Tp {
				case ast.TableOptionEngine:
					s.Engine, s.HasEngineOption = option.StrValue, true
				case ast.TableOptionCharset:
					s.CharacterSet, s.HasCharsetOption = option.StrValue, true
				case ast.TableOptionCollate:
					s.Collation, s.HasCollationOption = option.StrValue, true
				case ast.TableOptionAutoIncrement:
					s.AutoIncrementValue = option.UintValue
				}
			}
		case ast.AlterTableAddColumns:
			op.Action = model.AlterAddColumns
			for _, c := range a.NewColumns {
				column := projectColumn(c)
				column.PositionChanged = a.Position != nil && a.Position.Tp != ast.ColumnPositionNone
				op.Columns = append(op.Columns, column)
			}
		case ast.AlterTableDropColumn:
			op.Action = model.AlterDropColumn
			op.Name = a.Name
		case ast.AlterTableModifyColumn:
			op.Action = model.AlterModifyColumn
			for _, c := range a.NewColumns {
				column := projectColumn(c)
				column.PositionChanged = a.Position != nil && a.Position.Tp != ast.ColumnPositionNone
				op.Columns = append(op.Columns, column)
			}
		case ast.AlterTableChangeColumn:
			op.Action = model.AlterChangeColumn
			if a.OldColumnName != nil {
				op.Name = a.OldColumnName.Name.O
			}
			for _, c := range a.NewColumns {
				column := projectColumn(c)
				column.PositionChanged = a.Position != nil && a.Position.Tp != ast.ColumnPositionNone
				op.Columns = append(op.Columns, column)
			}
		case ast.AlterTableRenameColumn:
			op.Action = model.AlterRenameColumn
			if a.OldColumnName != nil {
				op.Name = a.OldColumnName.Name.O
			}
			if a.NewColumnName != nil {
				op.NewName = a.NewColumnName.Name.O
			}
		case ast.AlterTableAddConstraint:
			op.Action = model.AlterAddIndex
			if a.Constraint == nil {
				s.FullyProjected = false
				continue
			}
			idx, ok := projectConstraint(a.Constraint)
			if !ok {
				s.FullyProjected = false
				continue
			}
			op.Index = &idx
		case ast.AlterTableDropIndex:
			op.Action = model.AlterDropIndex
			op.Name = a.Name
		case ast.AlterTableDropPrimaryKey:
			op.Action = model.AlterDropPrimaryKey
			op.Name = "PRIMARY"
		case ast.AlterTableRenameIndex:
			op.Action = model.AlterRenameIndex
			op.Name = a.FromKey.O
			op.NewName = a.ToKey.O
		case ast.AlterTableRenameTable:
			op.Action = model.AlterRenameTable
			if a.NewTable == nil {
				s.FullyProjected = false
				continue
			}
			v := tableRef(a.NewTable)
			op.NewTable = &v
		default:
			s.FullyProjected = false
			continue
		}
		s.AlterOperations = append(s.AlterOperations, op)
	}
	return s
}

func tableRef(t *ast.TableName) model.TableRef {
	return model.TableRef{Schema: t.Schema.O, Name: t.Name.O}
}
func projectColumn(c *ast.ColumnDef) model.ColumnSpec {
	v := model.ColumnSpec{Name: c.Name.Name.O, Type: c.Tp.String(), Nullable: true, Length: c.Tp.GetFlen(), CharacterSet: c.Tp.GetCharset(), Collation: c.Tp.GetCollate()}
	columnText := strings.ToLower(c.Text())
	v.ExplicitCharset = c.Tp.GetCharset() != "" || c.Tp.GetCollate() != "" || strings.Contains(columnText, "character set") || strings.Contains(columnText, " collate ")
	for _, o := range c.Options {
		switch o.Tp {
		case ast.ColumnOptionPrimaryKey:
			v.PrimaryKey = true
		case ast.ColumnOptionNotNull:
			v.Nullable = false
		case ast.ColumnOptionComment:
			v.HasComment = true
		case ast.ColumnOptionAutoIncrement:
			v.AutoIncrement = true
		case ast.ColumnOptionDefaultValue:
			v.HasDefault = true
			v.DefaultExpression = restoreExpr(o.Expr)
		case ast.ColumnOptionOnUpdate:
			v.OnUpdate = true
		case ast.ColumnOptionGenerated:
			v.Generated = true
		}
	}
	v.Unsigned = strings.Contains(strings.ToLower(v.Type), "unsigned")
	return v
}
func projectConstraint(c *ast.Constraint) (model.IndexSpec, bool) {
	i := model.IndexSpec{Name: c.Name}
	switch c.Tp {
	case ast.ConstraintPrimaryKey:
		if i.Name == "" {
			i.Name = "PRIMARY"
		}
		i.Primary = true
		i.Unique = true
	case ast.ConstraintUniq, ast.ConstraintUniqKey, ast.ConstraintUniqIndex:
		i.Unique = true
	case ast.ConstraintKey, ast.ConstraintIndex:
	default:
		return i, false
	}
	for _, k := range c.Keys {
		if k.Column == nil {
			i.Expressions = append(i.Expressions, restoreExpr(k.Expr))
			continue
		}
		i.Columns = append(i.Columns, k.Column.Name.O)
		i.PrefixLengths = append(i.PrefixLengths, k.Length)
	}
	return i, true
}
func projectIndex(name string, unique bool, parts []*ast.IndexPartSpecification) model.IndexSpec {
	i := model.IndexSpec{Name: name, Unique: unique}
	for _, p := range parts {
		if p.Column != nil {
			i.Columns = append(i.Columns, p.Column.Name.O)
			i.PrefixLengths = append(i.PrefixLengths, p.Length)
		} else {
			i.Expressions = append(i.Expressions, restoreExpr(p.Expr))
		}
	}
	return i
}

func restoreExpr(node ast.ExprNode) string {
	if node == nil {
		return ""
	}
	var out strings.Builder
	ctx := format.NewRestoreCtx(format.DefaultRestoreFlags, &out)
	if err := node.Restore(ctx); err != nil {
		return node.Text()
	}
	return out.String()
}

func restore(node ast.StmtNode) string {
	var out strings.Builder
	ctx := format.NewRestoreCtx(format.DefaultRestoreFlags, &out)
	if err := node.Restore(ctx); err != nil {
		return node.Text()
	}
	return out.String()
}

func classify(node ast.StmtNode) model.StatementKind {
	switch node.(type) {
	case *ast.UseStmt:
		return model.StatementUse
	case *ast.CreateDatabaseStmt:
		return model.StatementCreateDatabase
	case *ast.AlterDatabaseStmt:
		return model.StatementAlterDatabase
	case *ast.DropDatabaseStmt:
		return model.StatementDropDatabase
	case *ast.CreateTableStmt:
		return model.StatementCreateTable
	case *ast.CreateViewStmt:
		return model.StatementCreateView
	case *ast.AlterTableStmt:
		return model.StatementAlterTable
	case *ast.DropTableStmt:
		if node.(*ast.DropTableStmt).IsView {
			return model.StatementDropView
		}
		return model.StatementDropTable
	case *ast.RenameTableStmt:
		return model.StatementRenameTable
	case *ast.TruncateTableStmt:
		return model.StatementTruncateTable
	case *ast.CreateIndexStmt:
		return model.StatementCreateIndex
	case *ast.DropIndexStmt:
		return model.StatementDropIndex
	case *ast.InsertStmt:
		return model.StatementInsert
	case *ast.UpdateStmt:
		return model.StatementUpdate
	case *ast.DeleteStmt:
		return model.StatementDelete
	case *ast.SelectStmt:
		return model.StatementSelect
	default:
		return model.StatementUnknown
	}
}
