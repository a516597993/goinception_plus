package parser

import (
	"strings"
	"testing"

	"github.com/pingcap/tidb/pkg/parser/ast"
	parsermysql "github.com/pingcap/tidb/pkg/parser/mysql"

	"github.com/hanchuanchuan/goinception-plus/internal/model"
)

func TestParseAndClassify(t *testing.T) {
	sql := "use app; create table t(id bigint primary key); insert into t values (1); update t set id=2 where id=1; delete from t where id=2;"
	result, err := New().Parse(sql, "STRICT_TRANS_TABLES")
	if err != nil {
		t.Fatal(err)
	}
	want := []model.StatementKind{model.StatementUse, model.StatementCreateTable, model.StatementInsert, model.StatementUpdate, model.StatementDelete}
	if len(result.Statements) != len(want) {
		t.Fatalf("got %d statements, want %d", len(result.Statements), len(want))
	}
	for i := range want {
		if result.Statements[i].Kind != want[i] {
			t.Errorf("statement %d: got %s, want %s", i, result.Statements[i].Kind, want[i])
		}
		if result.Statements[i].Normalized == "" {
			t.Errorf("statement %d has empty normalized SQL", i)
		}
	}
}

func TestProjectLiteralKindByMySQLSemanticType(t *testing.T) {
	tests := []struct {
		mysqlType byte
		want      model.LiteralKind
	}{
		{parsermysql.TypeDate, model.LiteralTemporal},
		{parsermysql.TypeDatetime, model.LiteralTemporal},
		{parsermysql.TypeTimestamp, model.LiteralTemporal},
		{parsermysql.TypeDuration, model.LiteralDuration},
		{parsermysql.TypeJSON, model.LiteralJSON},
		{parsermysql.TypeUnspecified, model.LiteralUnknown},
	}
	for _, tc := range tests {
		value := ast.NewValueExpr(nil, "utf8mb4", "utf8mb4_bin")
		value.GetType().SetType(tc.mysqlType)
		if got := projectLiteralKind(value); got != tc.want {
			t.Errorf("mysql type %d kind=%q want=%q", tc.mysqlType, got, tc.want)
		}
	}
}

func TestComparisonLiteralSemanticKinds(t *testing.T) {
	tests := []struct {
		name    string
		literal string
		want    model.LiteralKind
	}{
		{"signed integer", "-1", model.LiteralSignedInteger},
		{"unsigned integer", "18446744073709551615", model.LiteralUnsignedInteger},
		{"float", "1.25e2", model.LiteralFloat},
		{"decimal", "1.25", model.LiteralDecimal},
		{"string", "'1'", model.LiteralString},
		{"boolean", "TRUE", model.LiteralBoolean},
		{"null", "NULL", model.LiteralNull},
		{"hex binary", "x'01'", model.LiteralBinary},
		{"bit binary", "b'01'", model.LiteralBinary},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := New().Parse("UPDATE t SET v=1 WHERE c = "+tc.literal, "")
			if err != nil {
				t.Fatal(err)
			}
			comparisons := result.Statements[0].DML.Comparisons
			if len(comparisons) != 1 || comparisons[0].LiteralKind != tc.want {
				t.Fatalf("literal %s comparisons=%+v want=%s", tc.literal, comparisons, tc.want)
			}
			if strings.Contains(string(comparisons[0].LiteralKind), "int64") || strings.Contains(string(comparisons[0].LiteralKind), "uint8") {
				t.Fatalf("Go reflection type escaped parser boundary: %+v", comparisons[0])
			}
		})
	}
	result, err := New().Parse("UPDATE t SET v=1 WHERE 7 = c", "")
	if err != nil {
		t.Fatal(err)
	}
	if got := result.Statements[0].DML.Comparisons; len(got) != 1 || got[0].LiteralKind != model.LiteralSignedInteger {
		t.Fatalf("reversed comparison projection=%+v", got)
	}
}

func TestProjectCreateTable(t *testing.T) {
	result, err := New().Parse(
		"create table users(id bigint primary key comment 'id', name varchar(20)) comment='users';",
		"",
	)
	if err != nil {
		t.Fatal(err)
	}
	ddl := result.Statements[0].DDL
	if ddl == nil || ddl.Table != "users" || !ddl.HasPrimaryKey || !ddl.HasComment {
		t.Fatalf("unexpected DDL projection: %+v", ddl)
	}
	if len(ddl.Columns) != 2 || !ddl.Columns[0].HasComment || ddl.Columns[1].HasComment {
		t.Fatalf("unexpected columns: %+v", ddl.Columns)
	}
}

func TestInvalidSQLMode(t *testing.T) {
	if _, err := New().Parse("select 1", "NOT_A_REAL_SQL_MODE"); err == nil {
		t.Fatal("expected invalid sql_mode error")
	}
}

func TestProjectedAlterIsSupportedAndUnsafeAlterIsRejected(t *testing.T) {
	result, err := New().Parse("alter table t add column c int", "")
	if err != nil {
		t.Fatal(err)
	}
	if !result.Statements[0].Supported || len(result.Statements[0].DDL.AlterOperations) != 1 {
		t.Fatal("stable ALTER TABLE projection must be admitted")
	}
	result, err = New().Parse("alter table t drop partition p0", "")
	if err != nil {
		t.Fatal(err)
	}
	if result.Statements[0].Supported {
		t.Fatal("unprojected partition ALTER must be rejected")
	}
}

func TestLegacyRuleProjection(t *testing.T) {
	result, err := New().Parse("create table t(c char(30) character set utf8mb4 default 'x', ts timestamp default current_timestamp on update current_timestamp) engine=innodb auto_increment=9; update t set c=1 and ts=now() where c=1 order by rand(); create view v as select c from t;", "")
	if err != nil {
		t.Fatal(err)
	}
	ddl := result.Statements[0].DDL
	if !ddl.HasEngineOption || ddl.AutoIncrementValue != 9 || len(ddl.Columns) != 2 || !ddl.Columns[0].ExplicitCharset || ddl.Columns[0].Length != 30 || !ddl.Columns[1].OnUpdate {
		t.Fatalf("incomplete DDL projection: %+v", ddl)
	}
	dml := result.Statements[1].DML
	if !dml.WrongAndExpr || !dml.OrderByRand || len(dml.Comparisons) == 0 {
		t.Fatalf("incomplete DML projection: %+v", dml)
	}
	if result.Statements[2].Kind != model.StatementCreateView || !result.Statements[2].Supported {
		t.Fatalf("view projection: %+v", result.Statements[2])
	}
}
