package mysql

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/hanchuanchuan/goinception-plus/internal/model"
)

// Run with GIP_MYSQL57_DSN=host:port:user:password. The test owns only the
// uniquely named gip_it_* schema it creates.
func TestMySQL57Integration(t *testing.T) {
	raw := os.Getenv("GIP_MYSQL57_DSN")
	if raw == "" {
		t.Skip("GIP_MYSQL57_DSN is not set")
	}
	parts := strings.Split(raw, ":")
	if len(parts) != 4 {
		t.Fatal("GIP_MYSQL57_DSN must be host:port:user:password")
	}
	var port uint16
	if _, err := fmt.Sscan(parts[1], &port); err != nil {
		t.Fatal(err)
	}
	db, err := Open(model.TargetOptions{Host: parts[0], Port: port, User: parts[2], Password: parts[3]})
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	p := New(db)
	info, err := p.LoadServerInfo(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(info.Version, "5.7.") {
		t.Fatalf("expected MySQL 5.7, got %q", info.Version)
	}
	schema := fmt.Sprintf("gip_it_%d", time.Now().UnixNano())
	if _, err = db.ExecContext(ctx, "CREATE DATABASE "+QuoteIdentifier(schema)+" CHARACTER SET utf8mb4"); err != nil {
		t.Fatal(err)
	}
	defer db.ExecContext(context.Background(), "DROP DATABASE IF EXISTS "+QuoteIdentifier(schema))
	ddl := "CREATE TABLE " + QuoteIdentifier(schema) + ".t (id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT, n INT NOT NULL DEFAULT 0, payload JSON, g INT GENERATED ALWAYS AS (n + 1) STORED, PRIMARY KEY(id), KEY idx_g(g)) ENGINE=InnoDB COMMENT='integration'"
	if _, err = db.ExecContext(ctx, ddl); err != nil {
		t.Fatal(err)
	}
	if _, err = db.ExecContext(ctx, "CREATE VIEW "+QuoteIdentifier(schema)+".v AS SELECT id,g FROM "+QuoteIdentifier(schema)+".t"); err != nil {
		t.Fatal(err)
	}
	table, err := p.LoadTable(ctx, schema, "t")
	if err != nil {
		t.Fatal(err)
	}
	if table.Engine != "InnoDB" || len(table.Columns) != 4 || len(table.Indexes) != 2 {
		t.Fatalf("unexpected table metadata: %+v", table)
	}
	for _, index := range table.Indexes {
		if len(index.Expressions) != 0 {
			t.Fatalf("MySQL 5.7 ordinary index must not expose empty functional expressions: %+v", index)
		}
	}
	view, err := p.LoadTable(ctx, schema, "v")
	if err != nil || !view.IsView || view.CreateSQL == "" {
		t.Fatalf("unexpected view metadata: %+v, err=%v", view, err)
	}
	if _, err = db.ExecContext(ctx, "INSERT INTO "+QuoteIdentifier(schema)+".t(payload) VALUES ('{\"a\":1}'),('{\"a\":2}')"); err != nil {
		t.Fatal(err)
	}
	impactSQL := "UPDATE t SET payload = JSON_SET(payload, '$.ok', true) WHERE id > 0"
	estimate, err := p.EstimateImpact(ctx, schema, impactSQL)
	if err != nil {
		t.Fatal(err)
	}
	if estimate.Method != "explain:max" || estimate.Rows < 1 {
		t.Fatalf("unexpected estimate: %+v", estimate)
	}
	first, err := p.EstimateImpactWithRule(ctx, schema, impactSQL, "first")
	if err != nil {
		t.Fatal(err)
	}
	if first.Method != "explain:first" || first.Rows < 1 {
		t.Fatalf("unexpected first-row estimate: %+v", first)
	}
}
