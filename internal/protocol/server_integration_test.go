package protocol

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"strconv"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"

	appconfig "github.com/hanchuanchuan/goinception-plus/internal/config"
)

func freePort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	return ln.Addr().(*net.TCPAddr).Port
}

func TestServerManagementOverMySQLProtocol(t *testing.T) {
	cfg := appconfig.Defaults()
	cfg.Server.Host, cfg.Server.Port = "127.0.0.1", freePort(t)
	cfg.Auth.Username, cfg.Auth.Password = "archery", "secret"
	runtime := appconfig.NewRuntime(cfg)
	server := New(cfg, nil, "secret", runtime)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- server.Run(ctx) }()

	dsn := fmt.Sprintf("archery:secret@tcp(127.0.0.1:%d)/?timeout=1s", cfg.Server.Port)
	var db *sql.DB
	for i := 0; i < 30; i++ {
		db, _ = sql.Open("mysql", dsn)
		if err := db.Ping(); err == nil {
			break
		}
		_ = db.Close()
		time.Sleep(20 * time.Millisecond)
	}
	var database string
	if err := db.QueryRow("SHOW DATABASES").Scan(&database); err != nil || database != "information_schema" {
		t.Fatalf("SHOW DATABASES database=%q err=%v", database, err)
	}
	var id uint64
	if err := db.QueryRow("SELECT CONNECTION_ID()").Scan(&id); err != nil || id < 10000 {
		t.Fatalf("connection id=%d err=%v", id, err)
	}
	rows, err := db.Query("inception show processlist")
	if err != nil {
		t.Fatal(err)
	}
	cols, _ := rows.Columns()
	_ = rows.Close()
	if len(cols) != 10 || cols[0] != "Id" {
		t.Fatalf("processlist columns=%v", cols)
	}
	if _, err = db.Exec("inception set level er_no_where_condition=1"); err != nil {
		t.Fatal(err)
	}
	if got := runtime.Levels(); len(got) == 0 {
		t.Fatal("runtime levels disappeared")
	}
	if _, err = db.Exec("KILL QUERY " + strconv.FormatUint(id+99999, 10)); err == nil {
		t.Fatal("unknown thread kill succeeded")
	}
	secondDB, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal(err)
	}
	secondConn, err := secondDB.Conn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	var secondID uint64
	if err = secondConn.QueryRowContext(context.Background(), "SELECT CONNECTION_ID()").Scan(&secondID); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec("KILL CONNECTION " + strconv.FormatUint(secondID, 10)); err != nil {
		t.Fatal(err)
	}
	if err = secondConn.PingContext(context.Background()); err == nil {
		t.Fatal("killed connection remained usable")
	}
	_ = secondConn.Close()
	_ = secondDB.Close()

	duplicate := New(cfg, nil, "secret", appconfig.NewRuntime(cfg))
	if err = duplicate.Run(context.Background()); err == nil {
		t.Fatal("second listener unexpectedly bound the same port")
	}
	_ = db.Close()
	cancel()
	select {
	case err = <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("server did not shut down")
	}
}

func TestAuditOnlyRejectsExecutionWithLegacyResult(t *testing.T) {
	cfg := appconfig.Defaults()
	cfg.Server.Host, cfg.Server.Port = "127.0.0.1", freePort(t)
	cfg.Server.AuditOnly = true
	cfg.Auth.Username, cfg.Auth.Password = "archery", "secret"
	server := New(cfg, nil, "secret", appconfig.NewRuntime(cfg))
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- server.Run(ctx) }()
	db, err := sql.Open("mysql", fmt.Sprintf("archery:secret@tcp(127.0.0.1:%d)/?timeout=1s", cfg.Server.Port))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for i := 0; i < 30; i++ {
		if err = db.Ping(); err == nil {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	query := "/*--host=127.0.0.1;--user=root;--execute=1;--backup=0;*/ inception_magic_start;select 1;inception_magic_commit;"
	rows, err := db.Query(query)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	cols, _ := rows.Columns()
	if len(cols) != 12 {
		t.Fatalf("columns=%v", cols)
	}
	values := make([]any, len(cols))
	pointers := make([]any, len(cols))
	for i := range values {
		pointers[i] = &values[i]
	}
	if !rows.Next() {
		t.Fatal("missing rejection result")
	}
	if err = rows.Scan(pointers...); err != nil {
		t.Fatal(err)
	}
	if string(values[2].([]byte)) != "2" {
		t.Fatalf("error level=%q", values[2])
	}
	cancel()
	select {
	case err = <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("server did not stop")
	}
}
