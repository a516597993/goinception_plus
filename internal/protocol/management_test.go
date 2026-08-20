package protocol

import (
	"context"
	"testing"

	"github.com/hanchuanchuan/goinception-plus/internal/audit"
	appconfig "github.com/hanchuanchuan/goinception-plus/internal/config"
	"github.com/hanchuanchuan/goinception-plus/internal/model"
)

func managementHandler() *handler {
	r := newProcessRegistry()
	h := &handler{ctx: context.Background(), id: 10001, username: "archery", runtime: appconfig.NewRuntime(appconfig.Defaults()), registry: r, maxPacket: 1 << 20}
	r.add(&processInfo{ID: h.id, User: "archery", FromHost: "127.0.0.1", Command: "Sleep"})
	return h
}

func TestManagementCatalogAndConfig(t *testing.T) {
	h := managementHandler()
	result, err := h.HandleQuery("SHOW DATABASES")
	if err != nil {
		t.Fatal(err)
	}
	if len(result.RowDatas) != 1 || string(result.Fields[0].Name) != "Database" {
		t.Fatalf("unexpected SHOW DATABASES result")
	}

	result, err = h.HandleQuery("inception show variables like 'check_dml_where'")
	if err != nil {
		t.Fatal(err)
	}
	if len(result.RowDatas) != 1 || string(result.Fields[0].Name) != "Variable_name" {
		t.Fatalf("unexpected variable result")
	}
	if _, err = h.HandleQuery("inception set global check_dml_where=0"); err != nil {
		t.Fatal(err)
	}
	if h.runtime.Policy().RequireDMLWhere {
		t.Fatal("runtime set did not change policy")
	}
	if _, err = h.HandleQuery("inception set level GIP-DDL-CT-001=2"); err != nil {
		t.Fatal(err)
	}
	if h.runtime.Policy().Level(audit.RuleTableMustHavePK) != model.SeverityError {
		t.Fatal("protocol did not accept a stable GIP RuleID")
	}

	result, err = h.HandleQuery("inception show levels where value=2")
	if err != nil {
		t.Fatal(err)
	}
	if len(result.RowDatas) == 0 {
		t.Fatal("level WHERE filter returned no rows")
	}
}

func TestManagementProcesslistAndErrors(t *testing.T) {
	h := managementHandler()
	result, err := h.HandleQuery("inception get full processlist")
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Fields) != 10 || string(result.Fields[0].Name) != "Id" {
		t.Fatalf("unexpected processlist shape")
	}
	if _, err = h.HandleQuery("SHOW COLUMNS FROM missing"); err == nil {
		t.Fatal("missing virtual table was accepted")
	}
	if _, err = h.HandleQuery("inception get osc processlist"); err == nil {
		t.Fatal("OSC command should remain unsupported")
	}
}
