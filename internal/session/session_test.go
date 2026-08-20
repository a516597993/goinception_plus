package session

import (
	"context"
	"testing"

	"github.com/hanchuanchuan/goinception-plus/internal/audit"
	"github.com/hanchuanchuan/goinception-plus/internal/audit/rules"
	"github.com/hanchuanchuan/goinception-plus/internal/model"
	parseradapter "github.com/hanchuanchuan/goinception-plus/internal/parser"
)

type fakeMetadata struct{}

type modeCapturingParser struct{ mode string }

func (p *modeCapturingParser) Parse(_ string, mode string) (model.ParseResult, error) {
	p.mode = mode
	return model.ParseResult{Statements: []model.Statement{{Sequence: 1, Kind: model.StatementSelect, Supported: true, DML: &model.DMLSpec{FullyProjected: true}}}}, nil
}

func (fakeMetadata) LoadServerInfo(context.Context) (audit.ServerInfo, error) {
	return audit.ServerInfo{Version: "8.0.36"}, nil
}
func (fakeMetadata) LoadSchema(_ context.Context, name string) (audit.Schema, error) {
	return audit.Schema{Name: name}, nil
}
func (fakeMetadata) LoadTable(_ context.Context, schema, table string) (audit.Table, error) {
	return audit.Table{}, audit.ErrMetadataNotFound
}

func fakeOpenMetadata(context.Context, model.TargetOptions) (
	audit.MetadataProvider, audit.ServerInfo, func() error, error,
) {
	return fakeMetadata{}, audit.ServerInfo{Version: "8.0.36"}, func() error { return nil }, nil
}

func TestAuditLegacyRequest(t *testing.T) {
	service := New(parseradapter.New(), audit.NewEngine(rules.KnownStatement{})).
		WithMetadataFactory(fakeOpenMetadata)
	input := "/*--host=127.0.0.1;--port=3306;--user=test;--check=1;*/ inception_magic_start; use app; create table t(id bigint primary key); inception_magic_commit;"
	records, err := service.Audit(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 2 {
		t.Fatalf("got %d records, want 2", len(records))
	}
	for _, record := range records {
		if record.ErrorLevel != model.SeverityNone || record.StageStatus != "Checked" {
			t.Fatalf("unexpected issue: %+v", record)
		}
	}
}

func TestAuditReturnsStructuredParserError(t *testing.T) {
	service := New(parseradapter.New(), audit.NewEngine(rules.KnownStatement{})).
		WithMetadataFactory(fakeOpenMetadata)
	input := "/*--host=127.0.0.1;--user=test;*/ inception_magic_start; select from; inception_magic_commit;"
	records, err := service.Audit(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 || records[0].StageStatus != "CheckFailed" ||
		len(records[0].Issues) != 1 || records[0].Issues[0].RuleID != audit.RuleParserError {
		t.Fatalf("unexpected records: %+v", records)
	}
}

func TestUseUpdatesDatabaseAndMetadataIsNotNil(t *testing.T) {
	var observedDatabase string
	check := testRule{check: func(_ context.Context, ctx *audit.Context, statement model.Statement) []model.AuditIssue {
		if ctx.Metadata == nil {
			t.Fatal("metadata must be injected")
		}
		if statement.Kind == model.StatementCreateTable {
			observedDatabase = statement.Database
		}
		return nil
	}}
	service := New(parseradapter.New(), audit.NewEngine(check)).
		WithMetadataFactory(fakeOpenMetadata)
	input := "/*--host=127.0.0.1;--user=test;*/ inception_magic_start; use app; create table t(id int); inception_magic_commit;"
	if _, err := service.Audit(context.Background(), input); err != nil {
		t.Fatal(err)
	}
	if observedDatabase != "app" {
		t.Fatalf("got database %q, want app", observedDatabase)
	}
}

func TestRejectsUnsupportedTargetVersion(t *testing.T) {
	open := func(context.Context, model.TargetOptions) (audit.MetadataProvider, audit.ServerInfo, func() error, error) {
		return fakeMetadata{}, audit.ServerInfo{Version: "10.11.4-MariaDB"}, func() error { return nil }, nil
	}
	service := New(parseradapter.New(), audit.NewEngine(rules.KnownStatement{})).WithMetadataFactory(open)
	input := "/*--host=127.0.0.1;--user=test;*/ inception_magic_start; select 1; inception_magic_commit;"
	records, err := service.Audit(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 || records[0].Issues[0].RuleID != audit.RuleTargetVersion {
		t.Fatalf("unexpected records: %+v", records)
	}
}

func TestConfiguredSQLModeOverridesServerMode(t *testing.T) {
	p := &modeCapturingParser{}
	policy := audit.LegacyDefaults()
	policy.Legacy["sql_mode"] = "TRADITIONAL"
	open := func(context.Context, model.TargetOptions) (audit.MetadataProvider, audit.ServerInfo, func() error, error) {
		return fakeMetadata{}, audit.ServerInfo{Version: "8.0.36", SQLMode: "ANSI"}, func() error { return nil }, nil
	}
	service := New(p, audit.NewEngine(rules.KnownStatement{})).WithPolicy(policy).WithMetadataFactory(open)
	input := "/*--host=127.0.0.1;--user=test;*/ inception_magic_start; select 1; inception_magic_commit;"
	if _, err := service.Audit(context.Background(), input); err != nil {
		t.Fatal(err)
	}
	if p.mode != "TRADITIONAL" {
		t.Fatalf("parser mode=%q", p.mode)
	}
}

type testRule struct {
	check func(context.Context, *audit.Context, model.Statement) []model.AuditIssue
}

func (testRule) ID() string { return "TEST" }
func (r testRule) Check(ctx context.Context, auditCtx *audit.Context, statement model.Statement) []model.AuditIssue {
	return r.check(ctx, auditCtx, statement)
}
