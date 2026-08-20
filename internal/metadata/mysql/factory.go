package mysql

import (
	"context"

	"github.com/hanchuanchuan/goinception-plus/internal/audit"
	"github.com/hanchuanchuan/goinception-plus/internal/model"
)

func OpenRequest(ctx context.Context, options model.TargetOptions) (
	audit.MetadataProvider, audit.ServerInfo, func() error, error,
) {
	db, err := Open(options)
	if err != nil {
		return nil, audit.ServerInfo{}, nil, err
	}
	closeFn := db.Close
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, audit.ServerInfo{}, nil, err
	}
	provider := New(db)
	server, err := provider.LoadServerInfo(ctx)
	if err != nil {
		_ = db.Close()
		return nil, audit.ServerInfo{}, nil, err
	}
	return provider, server, closeFn, nil
}

var _ func(context.Context, model.TargetOptions) (
	audit.MetadataProvider, audit.ServerInfo, func() error, error,
) = OpenRequest
