package rollback

import (
	"strings"
	"testing"

	"github.com/hanchuanchuan/goinception-plus/internal/audit"
	"github.com/hanchuanchuan/goinception-plus/internal/backup"
)

func table() audit.Table {
	return audit.Table{Schema: "app", Name: "users", Columns: []audit.Column{{Name: "id"}, {Name: "payload"}}, Indexes: []audit.Index{{Name: "PRIMARY", Columns: []string{"id"}, Primary: true, Unique: true}}}
}
func TestGenerateRollback(t *testing.T) {
	cases := []struct {
		change backup.RowChange
		want   string
	}{{backup.RowChange{Type: backup.ChangeInsert, Table: table(), After: []any{int64(1), []byte{1, 2}}}, "DELETE FROM `app`.`users` WHERE `id`=1 LIMIT 1;"}, {backup.RowChange{Type: backup.ChangeDelete, Table: table(), Before: []any{int64(1), nil}}, "INSERT INTO `app`.`users` (`id`,`payload`) VALUES (1,NULL);"}, {backup.RowChange{Type: backup.ChangeUpdate, Table: table(), Before: []any{int64(1), "old"}, After: []any{int64(2), "new"}}, "UPDATE `app`.`users` SET `id`=1,`payload`='old' WHERE `id`=2 LIMIT 1;"}}
	for _, c := range cases {
		got, err := Generate(c.change)
		if err != nil {
			t.Fatal(err)
		}
		if got != c.want {
			t.Fatalf("got %s want %s", got, c.want)
		}
	}
}

func TestBinaryStringUsesHexLiteral(t *testing.T) {
	table := table()
	table.Columns[1].ColumnType = "varbinary(16)"
	got, err := Generate(backup.RowChange{
		Type: backup.ChangeUpdate, Table: table,
		Before: []any{int64(1), string([]byte{0xff, 0x00})},
		After:  []any{int64(1), "new"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if want := "`payload`=X'FF00'"; !strings.Contains(got, want) {
		t.Fatalf("got %s, want fragment %s", got, want)
	}
}
