package metadata

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/hanchuanchuan/goinception-plus/internal/audit"
	"github.com/hanchuanchuan/goinception-plus/internal/model"
)

type Provider struct {
	source    audit.MetadataProvider
	lowerCase bool
	mu        sync.Mutex
	server    *serverEntry
	schemas   map[string]schemaEntry
	tables    map[string]tableEntry
	flights   map[string]*flight
}

type serverEntry struct {
	value audit.ServerInfo
	err   error
}
type schemaEntry struct {
	value audit.Schema
	err   error
}
type tableEntry struct {
	value audit.Table
	err   error
}
type flight struct{ done chan struct{} }

func NewRequestCache(source audit.MetadataProvider) *Provider {
	return NewRequestCacheWithCaseMode(source, false)
}

func NewRequestCacheWithCaseMode(source audit.MetadataProvider, lowerCase bool) *Provider {
	return &Provider{
		source: source, lowerCase: lowerCase,
		schemas: make(map[string]schemaEntry), tables: make(map[string]tableEntry),
		flights: make(map[string]*flight),
	}
}

func (p *Provider) LoadServerInfo(ctx context.Context) (audit.ServerInfo, error) {
	p.mu.Lock()
	if p.server != nil {
		entry := *p.server
		p.mu.Unlock()
		return entry.value, entry.err
	}
	p.mu.Unlock()
	value, err := p.source.LoadServerInfo(ctx)
	p.mu.Lock()
	if err == nil {
		p.server = &serverEntry{value: value}
	}
	p.mu.Unlock()
	return value, err
}

func (p *Provider) LoadSchema(ctx context.Context, schema string) (audit.Schema, error) {
	key := "schema:" + p.normalize(schema)
	for {
		p.mu.Lock()
		if entry, ok := p.schemas[key]; ok {
			p.mu.Unlock()
			return entry.value, entry.err
		}
		if running, ok := p.flights[key]; ok {
			p.mu.Unlock()
			select {
			case <-ctx.Done():
				return audit.Schema{}, ctx.Err()
			case <-running.done:
				continue
			}
		}
		current := &flight{done: make(chan struct{})}
		p.flights[key] = current
		p.mu.Unlock()

		value, err := p.source.LoadSchema(ctx, schema)
		p.mu.Lock()
		if cacheable(err) {
			p.schemas[key] = schemaEntry{value: value, err: err}
		}
		delete(p.flights, key)
		close(current.done)
		p.mu.Unlock()
		return value, err
	}
}

func (p *Provider) LoadTable(ctx context.Context, schema, table string) (audit.Table, error) {
	key := "table:" + p.normalize(schema) + "\x00" + p.normalize(table)
	for {
		p.mu.Lock()
		if entry, ok := p.tables[key]; ok {
			p.mu.Unlock()
			return cloneTable(entry.value), entry.err
		}
		if running, ok := p.flights[key]; ok {
			p.mu.Unlock()
			select {
			case <-ctx.Done():
				return audit.Table{}, ctx.Err()
			case <-running.done:
				continue
			}
		}
		current := &flight{done: make(chan struct{})}
		p.flights[key] = current
		p.mu.Unlock()

		value, err := p.source.LoadTable(ctx, schema, table)
		p.mu.Lock()
		if cacheable(err) {
			p.tables[key] = tableEntry{value: cloneTable(value), err: err}
		}
		delete(p.flights, key)
		close(current.done)
		p.mu.Unlock()
		return cloneTable(value), err
	}
}

func (p *Provider) EstimateImpact(ctx context.Context, database, sql string) (model.ImpactEstimate, error) {
	estimator, ok := p.source.(audit.ImpactEstimator)
	if !ok {
		return model.ImpactEstimate{}, fmt.Errorf("metadata provider does not support impact estimation")
	}
	return estimator.EstimateImpact(ctx, database, sql)
}

func (p *Provider) PutTable(table audit.Table) {
	key := "table:" + p.normalize(table.Schema) + "\x00" + p.normalize(table.Name)
	p.mu.Lock()
	p.tables[key] = tableEntry{value: cloneTable(table)}
	p.mu.Unlock()
}

func (p *Provider) PutSchema(schema audit.Schema) {
	key := "schema:" + p.normalize(schema.Name)
	p.mu.Lock()
	p.schemas[key] = schemaEntry{value: schema}
	p.mu.Unlock()
}

func (p *Provider) DeleteSchema(schema string) {
	key := "schema:" + p.normalize(schema)
	p.mu.Lock()
	p.schemas[key] = schemaEntry{err: audit.ErrMetadataNotFound}
	p.mu.Unlock()
}

func (p *Provider) DeleteTable(schema, table string) {
	key := "table:" + p.normalize(schema) + "\x00" + p.normalize(table)
	p.mu.Lock()
	p.tables[key] = tableEntry{err: audit.ErrMetadataNotFound}
	p.mu.Unlock()
}

func (p *Provider) RenameTable(oldSchema, oldTable string, table audit.Table) {
	p.DeleteTable(oldSchema, oldTable)
	p.PutTable(table)
}

func (p *Provider) normalize(value string) string {
	if p.lowerCase {
		return strings.ToLower(value)
	}
	return value
}

func cacheable(err error) bool {
	return err == nil || errors.Is(err, audit.ErrMetadataNotFound)
}

func cloneTable(value audit.Table) audit.Table {
	result := value
	result.Columns = append([]audit.Column(nil), value.Columns...)
	result.Indexes = make([]audit.Index, len(value.Indexes))
	for i, index := range value.Indexes {
		result.Indexes[i] = index
		result.Indexes[i].Columns = append([]string(nil), index.Columns...)
		result.Indexes[i].PrefixLengths = append([]int(nil), index.PrefixLengths...)
		result.Indexes[i].Expressions = append([]string(nil), index.Expressions...)
		result.Indexes[i].Directions = append([]string(nil), index.Directions...)
	}
	return result
}
