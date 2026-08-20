package audit

import (
	"context"
	"errors"

	"github.com/hanchuanchuan/goinception-plus/internal/model"
)

var ErrMetadataNotFound = errors.New("metadata object not found")

// MetadataProvider is request-scoped. Implementations must not retain schema
// data across independent audit requests.
type MetadataProvider interface {
	LoadServerInfo(context.Context) (ServerInfo, error)
	LoadSchema(context.Context, string) (Schema, error)
	LoadTable(context.Context, string, string) (Table, error)
}

type ImpactEstimator interface {
	EstimateImpact(context.Context, string, string) (model.ImpactEstimate, error)
}
type ImpactEstimatorWithRule interface {
	EstimateImpactWithRule(context.Context, string, string, string) (model.ImpactEstimate, error)
}
type GrantChecker interface {
	CheckGrants(context.Context, string, bool) error
}

type ServerInfo struct {
	Version                      string
	SQLMode                      string
	LowerCaseTableNames          int
	CharacterSetServer           string
	CollationServer              string
	ExplicitDefaultsForTimestamp bool
}

type Schema struct {
	Name         string
	CharacterSet string
	Collation    string
}

type Table struct {
	Schema       string
	Name         string
	CreateSQL    string
	Columns      []Column
	Indexes      []Index
	IsView       bool
	CharacterSet string
	Collation    string
	Engine       string
	TableType    string
	Comment      string
	Rows         uint64
}

type Column struct {
	Name                 string
	ColumnType           string
	Nullable             bool
	Default              *string
	Extra                string
	CharacterSet         string
	Collation            string
	Comment              string
	GenerationExpression string
	AutoIncrement        bool
	Unsigned             bool
}

type Index struct {
	Name          string
	Columns       []string
	Unique        bool
	Primary       bool
	PrefixLengths []int
	Expressions   []string
	Directions    []string
	IndexType     string
	Visible       bool
}

type Context struct {
	Request         model.RequestOptions
	Database        string
	Metadata        MetadataProvider
	Policy          Policy
	Server          ServerInfo
	SeenAlterTables map[string]int
}

type Rule interface {
	ID() string
	Check(context.Context, *Context, model.Statement) []model.AuditIssue
}

type Engine struct{ rules []Rule }

func NewEngine(rules ...Rule) *Engine {
	return &Engine{rules: append([]Rule(nil), rules...)}
}

func (e *Engine) Check(ctx context.Context, auditCtx *Context, stmt model.Statement) []model.AuditIssue {
	var issues []model.AuditIssue
	for _, rule := range e.rules {
		issues = append(issues, rule.Check(ctx, auditCtx, stmt)...)
	}
	for i := range issues {
		if definition, ok := RuleByCode(issues[i].RuleID); ok {
			issues[i].LegacyKey = definition.LegacyKey
		}
	}
	filtered := issues[:0]
	for _, issue := range issues {
		if issue.Level != model.SeverityNone {
			filtered = append(filtered, issue)
		}
	}
	return filtered
}
