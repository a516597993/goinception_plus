package model

import "time"

type StatementKind string

const (
	StatementUnknown        StatementKind = "UNKNOWN"
	StatementUse            StatementKind = "USE"
	StatementCreateDatabase StatementKind = "CREATE_DATABASE"
	StatementAlterDatabase  StatementKind = "ALTER_DATABASE"
	StatementDropDatabase   StatementKind = "DROP_DATABASE"
	StatementCreateTable    StatementKind = "CREATE_TABLE"
	StatementAlterTable     StatementKind = "ALTER_TABLE"
	StatementDropTable      StatementKind = "DROP_TABLE"
	StatementRenameTable    StatementKind = "RENAME_TABLE"
	StatementTruncateTable  StatementKind = "TRUNCATE_TABLE"
	StatementCreateIndex    StatementKind = "CREATE_INDEX"
	StatementDropIndex      StatementKind = "DROP_INDEX"
	StatementCreateView     StatementKind = "CREATE_VIEW"
	StatementAlterView      StatementKind = "ALTER_VIEW"
	StatementDropView       StatementKind = "DROP_VIEW"
	StatementInsert         StatementKind = "INSERT"
	StatementUpdate         StatementKind = "UPDATE"
	StatementDelete         StatementKind = "DELETE"
	StatementSelect         StatementKind = "SELECT"
)

// Statement is the parser-neutral representation consumed by audit rules.
type Statement struct {
	Sequence    int
	Original    string
	Normalized  string
	Kind        StatementKind
	Database    string
	StartOffset int
	EndOffset   int
	Supported   bool
	DDL         *DDLSpec
	DML         *DMLSpec
}

type DMLSpec struct {
	Tables           []TableRef
	Columns          []string
	Assignments      []ColumnRef
	HasWhere         bool
	HasLimit         bool
	HasOrderBy       bool
	MultiTable       bool
	InsertSelect     bool
	OnDuplicate      bool
	FullyProjected   bool
	ValueRows        int
	HasJoinWithoutOn bool
	HasSelectStar    bool
	EstimatedRows    int64
	EstimateMethod   string
	UsesCTE          bool
	OrderByRand      bool
	WrongAndExpr     bool
	Comparisons      []ComparisonSpec
}

type ComparisonSpec struct {
	Column      ColumnRef
	LiteralKind LiteralKind
}

// LiteralKind is the stable business meaning of a parser literal. It must not
// contain Go reflection names or TiDB parser-driver implementation types.
type LiteralKind string

const (
	LiteralUnknown         LiteralKind = "unknown"
	LiteralNull            LiteralKind = "null"
	LiteralBoolean         LiteralKind = "boolean"
	LiteralSignedInteger   LiteralKind = "signed_integer"
	LiteralUnsignedInteger LiteralKind = "unsigned_integer"
	LiteralFloat           LiteralKind = "float"
	LiteralDecimal         LiteralKind = "decimal"
	LiteralString          LiteralKind = "string"
	LiteralBinary          LiteralKind = "binary"
	LiteralTemporal        LiteralKind = "temporal"
	LiteralDuration        LiteralKind = "duration"
	LiteralJSON            LiteralKind = "json"
)

type ParseWarning struct {
	Message string
}

type ParseResult struct {
	Statements []Statement
	Warnings   []ParseWarning
}

type ImpactEstimate struct {
	Rows   int64
	Method string
	Exact  bool
}

// DDLSpec is the stable AST projection used by DDL rules.
type DDLAction string

const (
	DDLCreateDatabase DDLAction = "CREATE_DATABASE"
	DDLAlterDatabase  DDLAction = "ALTER_DATABASE"
	DDLDropDatabase   DDLAction = "DROP_DATABASE"
	DDLCreateTable    DDLAction = "CREATE_TABLE"
	DDLAlterTable     DDLAction = "ALTER_TABLE"
	DDLDropTable      DDLAction = "DROP_TABLE"
	DDLRenameTable    DDLAction = "RENAME_TABLE"
	DDLTruncateTable  DDLAction = "TRUNCATE_TABLE"
	DDLCreateIndex    DDLAction = "CREATE_INDEX"
	DDLDropIndex      DDLAction = "DROP_INDEX"
	DDLCreateView     DDLAction = "CREATE_VIEW"
	DDLAlterView      DDLAction = "ALTER_VIEW"
	DDLDropView       DDLAction = "DROP_VIEW"
)

type TableRef struct{ Schema, Name, Alias string }
type ColumnRef struct{ Schema, Table, Name string }
type RenamePair struct{ From, To TableRef }

type AlterAction string

const (
	AlterAddColumns     AlterAction = "ADD_COLUMNS"
	AlterDropColumn     AlterAction = "DROP_COLUMN"
	AlterModifyColumn   AlterAction = "MODIFY_COLUMN"
	AlterChangeColumn   AlterAction = "CHANGE_COLUMN"
	AlterRenameColumn   AlterAction = "RENAME_COLUMN"
	AlterAddIndex       AlterAction = "ADD_INDEX"
	AlterDropIndex      AlterAction = "DROP_INDEX"
	AlterDropPrimaryKey AlterAction = "DROP_PRIMARY_KEY"
	AlterRenameIndex    AlterAction = "RENAME_INDEX"
	AlterRenameTable    AlterAction = "RENAME_TABLE"
	AlterTableOptions   AlterAction = "TABLE_OPTIONS"
)

type AlterOperation struct {
	Action   AlterAction
	Name     string
	NewName  string
	Columns  []ColumnSpec
	Index    *IndexSpec
	NewTable *TableRef
}

type DDLSpec struct {
	Action             DDLAction
	Schema             string
	Table              string
	Tables             []TableRef
	Reference          *TableRef
	RenamePairs        []RenamePair
	AlterOperations    []AlterOperation
	Columns            []ColumnSpec
	Indexes            []IndexSpec
	HasPrimaryKey      bool
	HasComment         bool
	CreateLike         bool
	CreateSelect       bool
	FullyProjected     bool
	IfExists           bool
	IfNotExists        bool
	Noop               bool
	Engine             string
	CharacterSet       string
	Collation          string
	HasEngineOption    bool
	HasCharsetOption   bool
	HasCollationOption bool
	AutoIncrementValue uint64
	HasForeignKey      bool
	Partitioned        bool
}

type ColumnSpec struct {
	Name              string
	Type              string
	Nullable          bool
	HasComment        bool
	PrimaryKey        bool
	AutoIncrement     bool
	Unsigned          bool
	HasDefault        bool
	Generated         bool
	CharacterSet      string
	Collation         string
	DefaultExpression string
	OnUpdate          bool
	Length            int
	PositionChanged   bool
	ExplicitCharset   bool
}

type IndexSpec struct {
	Name          string
	Columns       []string
	Unique        bool
	Primary       bool
	PrefixLengths []int
	Expressions   []string
}

type Severity uint8

const (
	SeverityNone Severity = iota
	SeverityWarning
	SeverityError
)

type AuditIssue struct {
	RuleID    string
	LegacyKey string
	Code      int
	Level     Severity
	Message   string
	Sequence  int
}

type AuditRecord struct {
	Sequence        int
	Stage           string
	ErrorLevel      Severity
	StageStatus     string
	ErrorMessage    string
	SQL             string
	AffectedRows    int64
	BackupDatabase  string
	ExecuteTime     time.Duration
	SQLSHA1         string
	BackupCompleted bool
	BackupTime      time.Duration
	OperationID     string
	Issues          []AuditIssue
}

type TargetOptions struct {
	Host            string
	Port            uint16
	User            string
	Password        string
	Database        string
	SQLMode         string
	Version         string
	SQLSafeUpdates  int
	LockWaitTimeout int
	DefaultCharset  string
}

type RequestOptions struct {
	Target         TargetOptions
	Check          bool
	Execute        bool
	Backup         bool
	IgnoreWarnings bool
	SleepMillis    int
	SleepRows      int
	TraceID        string
}

type AuditRequest struct {
	Options    RequestOptions
	Statements []Statement
}

type ExecutionResult struct {
	Sequence        int
	AffectedRows    int64
	Duration        time.Duration
	Error           string
	RuleID          string
	BackupDatabase  string
	BackupCompleted bool
	BackupDuration  time.Duration
	SequenceID      string
	Executed        bool
}
