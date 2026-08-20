package session

import (
	"context"
	"testing"

	"github.com/hanchuanchuan/goinception-plus/internal/audit"
	"github.com/hanchuanchuan/goinception-plus/internal/audit/rules"
	"github.com/hanchuanchuan/goinception-plus/internal/execution"
	"github.com/hanchuanchuan/goinception-plus/internal/model"
	parseradapter "github.com/hanchuanchuan/goinception-plus/internal/parser"
)

type fakeExecutor struct{ called *bool }

func (f fakeExecutor) Execute(_ context.Context, _ model.RequestOptions, s []model.Statement) []model.ExecutionResult {
	*f.called = true
	return []model.ExecutionResult{{Sequence: s[0].Sequence, AffectedRows: 1}}
}
func TestExecuteOnlyAfterWholeBatchPasses(t *testing.T) {
	called := false
	factory := func(context.Context, model.TargetOptions) (execution.Engine, func() error, error) {
		return fakeExecutor{&called}, nil, nil
	}
	service := New(parseradapter.New(), audit.NewEngine(rules.KnownStatement{})).WithMetadataFactory(fakeOpenMetadata).WithExecutorFactory(factory)
	input := "/*--host=127.0.0.1;--user=test;--db=app;--execute=1;*/ inception_magic_start; create table t(id int); inception_magic_commit;"
	records, err := service.Audit(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if !called || records[0].Stage != "EXECUTED" {
		t.Fatalf("not executed: %+v", records)
	}
}
func TestAuditErrorBlocksWholeBatchExecution(t *testing.T) {
	called := false
	factory := func(context.Context, model.TargetOptions) (execution.Engine, func() error, error) {
		return fakeExecutor{&called}, nil, nil
	}
	service := New(parseradapter.New(), audit.NewEngine(rules.KnownStatement{}, rules.DMLSafety{})).WithMetadataFactory(fakeOpenMetadata).WithExecutorFactory(factory)
	input := "/*--host=127.0.0.1;--user=test;--execute=1;*/ inception_magic_start; update app.missing set c=1; inception_magic_commit;"
	_, err := service.Audit(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if called {
		t.Fatal("executor called after audit error")
	}
}

func TestIgnoreWarningsControlsExecutionGate(t *testing.T) {
	for _, tc := range []struct {
		name, directive string
		wantCalled      bool
	}{
		{"warnings block by default", "", false},
		{"legacy ignore-warnings allows execution", "--ignore-warnings=1;", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			called := false
			factory := func(context.Context, model.TargetOptions) (execution.Engine, func() error, error) {
				return fakeExecutor{&called}, nil, nil
			}
			policy := audit.LegacyDefaults()
			policy.CheckPrimaryKey = true
			service := New(parseradapter.New(), audit.NewEngine(rules.KnownStatement{}, rules.RequirePrimaryKey{})).WithPolicy(policy).WithMetadataFactory(fakeOpenMetadata).WithExecutorFactory(factory)
			input := "/*--host=127.0.0.1;--user=test;--db=app;--execute=1;" + tc.directive + "*/ inception_magic_start; create table warning_test(c int); inception_magic_commit;"
			if _, err := service.Audit(context.Background(), input); err != nil {
				t.Fatal(err)
			}
			if called != tc.wantCalled {
				t.Fatalf("executor called=%v, want %v", called, tc.wantCalled)
			}
		})
	}
}
