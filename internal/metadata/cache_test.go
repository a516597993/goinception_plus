package metadata

import (
	"context"
	"testing"

	"github.com/hanchuanchuan/goinception-plus/internal/audit"
)

type countingProvider struct{ tableLoads int }

func (p *countingProvider) LoadServerInfo(_ context.Context) (audit.ServerInfo, error) {
	return audit.ServerInfo{Version: "8.0.36"}, nil
}

func (p *countingProvider) LoadSchema(_ context.Context, name string) (audit.Schema, error) {
	return audit.Schema{Name: name}, nil
}

func (p *countingProvider) LoadTable(_ context.Context, schema, table string) (audit.Table, error) {
	p.tableLoads++
	return audit.Table{Schema: schema, Name: table}, nil
}

func TestRequestCacheAndOverlay(t *testing.T) {
	source := &countingProvider{}
	cache := NewRequestCache(source)
	ctx := context.Background()
	for range 2 {
		if _, err := cache.LoadTable(ctx, "app", "users"); err != nil {
			t.Fatal(err)
		}
	}
	if source.tableLoads != 1 {
		t.Fatalf("metadata loaded %d times, want 1", source.tableLoads)
	}
	cache.PutTable(audit.Table{Schema: "app", Name: "users", CreateSQL: "changed"})
	got, err := cache.LoadTable(ctx, "app", "users")
	if err != nil {
		t.Fatal(err)
	}
	if got.CreateSQL != "changed" || source.tableLoads != 1 {
		t.Fatalf("overlay not used: %+v, loads=%d", got, source.tableLoads)
	}
}
