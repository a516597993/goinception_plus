package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/BurntSushi/toml"

	"github.com/hanchuanchuan/goinception-plus/internal/audit"
	"github.com/hanchuanchuan/goinception-plus/internal/backup"
	"github.com/hanchuanchuan/goinception-plus/internal/model"
)

type Duration struct{ time.Duration }

func (d *Duration) UnmarshalText(v []byte) error {
	x, err := time.ParseDuration(string(v))
	if err == nil {
		d.Duration = x
	}
	return err
}

type Config struct {
	Server        Server         `toml:"server"`
	Auth          Auth           `toml:"auth"`
	Log           Log            `toml:"log"`
	Inc           Inc            `toml:"inc"`
	IncLevel      IncLevel       `toml:"inc_level"`
	Rules         map[string]int `toml:"rules"`
	Backup        Backup         `toml:"backup"`
	Observability Observability  `toml:"observability"`
}
type Server struct {
	Host             string   `toml:"host"`
	Port             int      `toml:"port"`
	MaxConnections   int      `toml:"max_connections"`
	MaxAllowedPacket int      `toml:"max_allowed_packet"`
	ReadTimeout      Duration `toml:"read_timeout"`
	WriteTimeout     Duration `toml:"write_timeout"`
	IdleTimeout      Duration `toml:"idle_timeout"`
	ShutdownTimeout  Duration `toml:"shutdown_timeout"`
	AuditTimeout     Duration `toml:"audit_timeout"`
	AuditOnly        bool     `toml:"audit_only"`
}
type Observability struct {
	Enabled bool   `toml:"enabled"`
	Host    string `toml:"host"`
	Port    int    `toml:"port"`
}
type Auth struct {
	Username    string `toml:"username"`
	Password    string `toml:"password"`
	PasswordEnv string `toml:"password_env"`
}
type Log struct {
	Level  string `toml:"level"`
	Format string `toml:"format"`
}
type Inc struct {
	LegacyBackupHost            string   `toml:"backup_host"`
	LegacyBackupPort            int      `toml:"backup_port"`
	LegacyBackupUser            string   `toml:"backup_user"`
	LegacyBackupPassword        string   `toml:"backup_password"`
	CheckIndexColumnRepeat      bool     `toml:"check_index_column_repeat"`
	CheckIdentifier             bool     `toml:"check_identifier"`
	CheckIdentifierLower        bool     `toml:"check_identifier_lower"`
	EnableIdentifierKeyword     bool     `toml:"enable_identifer_keyword"`
	EnableSetCharset            bool     `toml:"enable_set_charset"`
	EnableSetCollation          bool     `toml:"enable_set_collation"`
	EnableSetEngine             bool     `toml:"enable_set_engine"`
	CheckPrimaryKey             bool     `toml:"check_primary_key"`
	CheckTableComment           bool     `toml:"check_table_comment"`
	CheckColumnComment          bool     `toml:"check_column_comment"`
	EnableColumnCharset         bool     `toml:"enable_column_charset"`
	EnableBlobType              bool     `toml:"enable_blob_type"`
	EnableJSONType              bool     `toml:"enable_json_type"`
	MaxKeyParts                 int      `toml:"max_key_parts"`
	MaxPrimaryKeyParts          int      `toml:"max_primary_key_parts"`
	MaxKeys                     int      `toml:"max_keys"`
	CheckAutoIncrementDataType  bool     `toml:"check_autoincrement_datatype"`
	CheckDMLWhere               bool     `toml:"check_dml_where"`
	MaxColumnCount              int      `toml:"max_column_count"`
	MaxUpdateRows               int64    `toml:"max_update_rows"`
	MaxInsertRows               int      `toml:"max_insert_rows"`
	CheckDMLLimit               bool     `toml:"check_dml_limit"`
	CheckDMLOrderBy             bool     `toml:"check_dml_orderby"`
	CheckInsertField            bool     `toml:"check_insert_field"`
	EnableSelectStar            bool     `toml:"enable_select_star"`
	EnableForeignKey            bool     `toml:"enable_foreign_key"`
	EnablePartition             bool     `toml:"enable_partition_table"`
	EnableDropTable             bool     `toml:"enable_drop_table"`
	EnableDropDatabase          bool     `toml:"enable_drop_database"`
	EnableTruncate              bool     `toml:"enable_truncate_table"`
	RequiredEngine              string   `toml:"required_engine"`
	CheckImplicitTypeConversion bool     `toml:"check_implicit_type_conversion"`
	IndexPrefix                 string   `toml:"index_prefix"`
	UniqueIndexPrefix           string   `toml:"uniq_index_prefix"`
	TablePrefix                 string   `toml:"table_prefix"`
	ExplainRule                 string   `toml:"explain_rule"`
	SQLMode                     string   `toml:"sql_mode"`
	SQLSafeUpdates              int      `toml:"sql_safe_updates"`
	LockWaitTimeout             int      `toml:"lock_wait_timeout"`
	SkipGrantTable              bool     `toml:"skip_grant_table"`
	SupportCharset              string   `toml:"support_charset"`
	SupportEngine               string   `toml:"support_engine"`
	Language                    string   `toml:"lang"`
	GeneralLog                  bool     `toml:"general_log"`
	DefaultCharset              string   `toml:"default_charset"`
	LegacyMaxAllowedPacket      int      `toml:"max_allowed_packet"`
	MustHaveColumns             string   `toml:"must_have_columns"`
	CustomKeywords              []string `toml:"custom_keywords"`
	CheckColumnTypeChange       bool     `toml:"check_column_type_change"`
	CheckIndexPrefix            bool     `toml:"check_index_prefix"`
	EnableNullable              bool     `toml:"enable_nullable"`
	EnableEnumSetBit            bool     `toml:"enable_enum_set_bit"`
	EnableTimestampType         bool     `toml:"enable_timestamp_type"`
	EnableAutoIncrementUnsigned bool     `toml:"enable_autoincrement_unsigned"`
	EnablePKColumnsOnlyInt      bool     `toml:"enable_pk_columns_only_int"`
	CheckTimestampCount         bool     `toml:"check_timestamp_count"`
	MaxCharLength               int      `toml:"max_char_length"`
}
type IncLevel struct {
	AlterTableOnce             int `toml:"er_alter_table_once"`
	AutoIncrIDWarning          int `toml:"er_auto_incr_id_warning"`
	AutoIncUnsigned            int `toml:"er_autoinc_unsigned"`
	BlobCantHaveDefault        int `toml:"er_blob_cant_have_default"`
	CantChangeColumn           int `toml:"er_cant_change_column"`
	CantChangeColumnPosition   int `toml:"er_cant_change_column_position"`
	CantSetCharset             int `toml:"er_cant_set_charset"`
	CantSetCollation           int `toml:"er_cant_set_collation"`
	CantSetEngine              int `toml:"er_cant_set_engine"`
	ChangeColumnType           int `toml:"er_change_column_type"`
	TableMustHavePK            int `toml:"er_table_must_have_pk"`
	TableMustHaveComment       int `toml:"er_table_must_have_comment"`
	ColumnMustHaveComment      int `toml:"er_column_have_no_comment"`
	ColumnMustHaveCommentAlias int `toml:"er_column_must_have_comment"`
	MaxColumnCount             int `toml:"er_max_column_count"`
	NoWhereCondition           int `toml:"er_no_where_condition"`
	ChangeTooMuchRows          int `toml:"er_change_too_much_rows"`
	InsertTooMuchRows          int `toml:"er_insert_too_much_rows"`
	WithLimitCondition         int `toml:"er_with_limit_condition"`
	WithOrderByCondition       int `toml:"er_with_orderby_condition"`
	JoinNoOnCondition          int `toml:"er_join_no_on_condition"`
	SelectOnlyStar             int `toml:"er_select_only_star"`
	InsertField                int `toml:"er_with_insert_field"`
	InsertFieldAlias           int `toml:"er_insert_field"`
	CharToVarcharLen           int `toml:"er_char_to_varchar_len"`
	CharsetOnColumn            int `toml:"er_charset_on_column"`
	DatetimeDefault            int `toml:"er_datetime_default"`
	ForeignKey                 int `toml:"er_foreign_key"`
	IdentifierKeyword          int `toml:"er_ident_use_keyword"`
	ImplicitTypeConversion     int `toml:"er_implicit_type_conversion"`
	AutoIncrementInitial       int `toml:"er_inc_init_err"`
	IncorrectDatetimeValue     int `toml:"er_incorrect_datetime_value"`
	IndexNamePrefix            int `toml:"er_index_name_idx_prefix"`
	UniqueIndexNamePrefix      int `toml:"er_index_name_uniq_prefix"`
	JSONTypeSupport            int `toml:"er_json_type_support"`
	MustHaveColumns            int `toml:"er_must_have_columns"`
	NotAllowedNullable         int `toml:"er_not_allowed_nullable"`
	OrderByRand                int `toml:"er_ordery_by_rand"`
	PartitionNotAllowed        int `toml:"er_partition_not_allowed"`
	PrimaryKeyColumnsNotInt    int `toml:"er_pk_cols_not_int"`
	PrimaryKeyTooManyParts     int `toml:"er_pk_too_many_parts"`
	AutoIncrementIntegerType   int `toml:"er_set_data_type_int_bigint"`
	TableCharsetMustNull       int `toml:"er_table_charset_must_null"`
	TableCharsetMustUTF8       int `toml:"er_table_charset_must_utf8"`
	TextNotNullable            int `toml:"er_text_not_nullable_error"`
	TooManyKeyParts            int `toml:"er_too_many_key_parts"`
	TooManyKeys                int `toml:"er_too_many_keys"`
	TooManyAutoTimestampCols   int `toml:"er_too_much_auto_timestamp_cols"`
	UseEnum                    int `toml:"er_use_enum"`
	UseTextOrBlob              int `toml:"er_use_text_or_blob"`
	ViewSupport                int `toml:"er_view_support"`
	WithDefaultAddColumn       int `toml:"er_with_default_add_column"`
	WrongAndExpression         int `toml:"er_wrong_and_expr"`
	InvalidDataType            int `toml:"er_invalid_data_type"`
	InvalidIdentifier          int `toml:"er_invalid_ident"`
}
type Backup struct {
	Host        string `toml:"host"`
	Port        int    `toml:"port"`
	User        string `toml:"user"`
	Password    string `toml:"password"`
	PasswordEnv string `toml:"password_env"`
}

func Defaults() Config {
	return Config{
		Server: Server{Host: "0.0.0.0", Port: 4000, MaxConnections: 200, MaxAllowedPacket: 16 << 20, ReadTimeout: Duration{30 * time.Second}, WriteTimeout: Duration{30 * time.Second}, IdleTimeout: Duration{10 * time.Minute}, ShutdownTimeout: Duration{15 * time.Second}, AuditTimeout: Duration{30 * time.Second}},
		Log:    Log{Level: "info", Format: "text"},
		Inc:    Inc{CheckDMLWhere: true, MaxColumnCount: 100, EnableSetEngine: true, RequiredEngine: "innodb", ExplainRule: "first", SQLSafeUpdates: -1, LockWaitTimeout: -1, SkipGrantTable: true, SupportCharset: "utf8,utf8mb4", SupportEngine: "innodb", DefaultCharset: "utf8mb4", MaxKeyParts: 5, MaxPrimaryKeyParts: 3, MaxKeys: 10, EnableNullable: true, EnableTimestampType: true, CheckTimestampCount: true},
		IncLevel: IncLevel{
			AlterTableOnce: 1, BlobCantHaveDefault: 1, CantChangeColumnPosition: 1,
			CantSetCharset: 2, CantSetCollation: 2, CantSetEngine: 2, ChangeColumnType: 1,
			ChangeTooMuchRows: 2, CharToVarcharLen: 1, CharsetOnColumn: 2, ColumnMustHaveComment: 2,
			DatetimeDefault: 1, ForeignKey: 2, IdentifierKeyword: 2, ImplicitTypeConversion: 2,
			AutoIncrementInitial: 1, IncorrectDatetimeValue: 2, IndexNamePrefix: 1, UniqueIndexNamePrefix: 1,
			InsertTooMuchRows: 1, InvalidDataType: 1, InvalidIdentifier: 2, JoinNoOnCondition: 2,
			JSONTypeSupport: 1, MustHaveColumns: 2, NoWhereCondition: 2, NotAllowedNullable: 1,
			OrderByRand: 2, PartitionNotAllowed: 1, PrimaryKeyColumnsNotInt: 1, PrimaryKeyTooManyParts: 2,
			SelectOnlyStar: 1, AutoIncrementIntegerType: 1, TableCharsetMustNull: 2, TableCharsetMustUTF8: 2,
			TableMustHaveComment: 1, TableMustHavePK: 2, TextNotNullable: 1, TooManyKeyParts: 2,
			TooManyKeys: 1, TooManyAutoTimestampCols: 1, UseEnum: 2, UseTextOrBlob: 1,
			ViewSupport: 2, WithDefaultAddColumn: 2, InsertField: 1, WithLimitCondition: 1,
			WithOrderByCondition: 2, WrongAndExpression: 1, MaxColumnCount: 1,
		},
		Rules: map[string]int{}, Backup: Backup{Port: 3306}, Observability: Observability{Host: "127.0.0.1", Port: 4001},
	}
}
func Load(path string) (Config, error) {
	c := Defaults()
	meta, err := toml.DecodeFile(path, &c)
	if err != nil {
		return Config{}, err
	}
	if err = normalizeLegacyConfig(&c, meta); err != nil {
		return Config{}, err
	}
	if keys := meta.Undecoded(); len(keys) > 0 {
		parts := make([]string, len(keys))
		for i, k := range keys {
			parts[i] = k.String()
		}
		return Config{}, fmt.Errorf("unknown or unsupported configuration keys: %s", strings.Join(parts, ", "))
	}
	if err = c.Validate(); err != nil {
		return Config{}, err
	}
	return c, nil
}

func normalizeLegacyConfig(c *Config, meta toml.MetaData) error {
	if meta.IsDefined("inc_level", "er_column_must_have_comment") {
		if meta.IsDefined("inc_level", "er_column_have_no_comment") && c.IncLevel.ColumnMustHaveComment != c.IncLevel.ColumnMustHaveCommentAlias {
			return fmt.Errorf("conflicting inc_level aliases er_column_have_no_comment and er_column_must_have_comment")
		}
		c.IncLevel.ColumnMustHaveComment = c.IncLevel.ColumnMustHaveCommentAlias
	}
	if meta.IsDefined("inc_level", "er_insert_field") {
		if meta.IsDefined("inc_level", "er_with_insert_field") && c.IncLevel.InsertField != c.IncLevel.InsertFieldAlias {
			return fmt.Errorf("conflicting inc_level aliases er_with_insert_field and er_insert_field")
		}
		c.IncLevel.InsertField = c.IncLevel.InsertFieldAlias
	}
	if c.Inc.LegacyBackupHost != "" {
		if c.Backup.Host != "" {
			return fmt.Errorf("legacy inc.backup_* conflicts with [backup]")
		}
		c.Backup.Host, c.Backup.Port, c.Backup.User, c.Backup.Password = c.Inc.LegacyBackupHost, c.Inc.LegacyBackupPort, c.Inc.LegacyBackupUser, c.Inc.LegacyBackupPassword
	}
	if meta.IsDefined("inc", "max_allowed_packet") && !meta.IsDefined("server", "max_allowed_packet") {
		c.Server.MaxAllowedPacket = c.Inc.LegacyMaxAllowedPacket
	}
	return nil
}
func (c *Config) Validate() error {
	var problems []string
	if strings.TrimSpace(c.Server.Host) == "" {
		problems = append(problems, "server.host is required")
	}
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		problems = append(problems, "server.port must be 1..65535")
	}
	if c.Server.MaxConnections < 1 {
		problems = append(problems, "server.max_connections must be positive")
	}
	if c.Server.MaxAllowedPacket < 1024 || c.Server.MaxAllowedPacket > 1<<30 {
		problems = append(problems, "server.max_allowed_packet must be 1KiB..1GiB")
	}
	if c.Server.ReadTimeout.Duration <= 0 || c.Server.WriteTimeout.Duration <= 0 || c.Server.IdleTimeout.Duration <= 0 || c.Server.ShutdownTimeout.Duration <= 0 {
		problems = append(problems, "all server timeouts must be positive")
	}
	if c.Server.AuditTimeout.Duration <= 0 {
		problems = append(problems, "server.audit_timeout must be positive")
	}
	if c.Log.Level != "debug" && c.Log.Level != "info" && c.Log.Level != "warn" && c.Log.Level != "error" {
		problems = append(problems, "log.level must be debug, info, warn, or error")
	}
	if c.Log.Format != "text" && c.Log.Format != "json" {
		problems = append(problems, "log.format must be text or json")
	}
	if c.Observability.Enabled {
		if strings.TrimSpace(c.Observability.Host) == "" || c.Observability.Port < 1 || c.Observability.Port > 65535 {
			problems = append(problems, "observability requires host and port 1..65535")
		}
		if c.Observability.Port == c.Server.Port && c.Observability.Host == c.Server.Host {
			problems = append(problems, "observability and MySQL protocol addresses must differ")
		}
	}
	if strings.TrimSpace(c.Auth.Username) == "" {
		problems = append(problems, "auth.username is required")
	}
	if (c.Auth.Password == "") == (c.Auth.PasswordEnv == "") {
		problems = append(problems, "exactly one of auth.password or auth.password_env is required")
	}
	for name, value := range levelBindings(&c.IncLevel) {
		if *value < 0 || *value > 2 {
			problems = append(problems, "inc_level."+name+" must be 0, 1, or 2")
		}
	}
	for code, value := range c.Rules {
		if _, ok := audit.RuleByCode(code); !ok {
			problems = append(problems, "rules."+code+" is not a registered GIP RuleID")
		} else if value < 0 || value > 2 {
			problems = append(problems, "rules."+code+" must be 0, 1, or 2")
		}
	}
	if c.Inc.ExplainRule != "first" && c.Inc.ExplainRule != "max" {
		problems = append(problems, "inc.explain_rule must be first or max")
	}
	if c.Inc.SQLSafeUpdates < -1 || c.Inc.SQLSafeUpdates > 1 {
		problems = append(problems, "inc.sql_safe_updates must be -1, 0, or 1")
	}
	if c.Inc.LockWaitTimeout < -1 {
		problems = append(problems, "inc.lock_wait_timeout must be -1 or non-negative")
	}
	if c.Backup.Host != "" {
		if c.Backup.Port < 1 || c.Backup.Port > 65535 || c.Backup.User == "" {
			problems = append(problems, "backup host requires valid port and user")
		}
		if (c.Backup.Password == "") == (c.Backup.PasswordEnv == "") {
			problems = append(problems, "backup requires exactly one password source")
		}
	}
	if len(problems) > 0 {
		return fmt.Errorf("configuration validation failed: %s", strings.Join(problems, "; "))
	}
	return nil
}
func (c Config) AuthPassword() (string, error) {
	return secret(c.Auth.Password, c.Auth.PasswordEnv, "auth")
}
func (c Config) BackupOptions() (backup.TargetOptions, error) {
	if c.Backup.Host == "" {
		return backup.TargetOptions{}, nil
	}
	p, err := secret(c.Backup.Password, c.Backup.PasswordEnv, "backup")
	return backup.TargetOptions{Host: c.Backup.Host, Port: uint16(c.Backup.Port), User: c.Backup.User, Password: p}, err
}
func secret(value, env, name string) (string, error) {
	if env == "" {
		return value, nil
	}
	v, ok := os.LookupEnv(env)
	if !ok || v == "" {
		return "", fmt.Errorf("%s password environment variable %s is empty", name, env)
	}
	return v, nil
}
func (c Config) Policy() audit.Policy {
	p := audit.LegacyDefaults()
	p.CheckPrimaryKey = c.Inc.CheckPrimaryKey
	p.CheckTableComment = c.Inc.CheckTableComment
	p.CheckColumnComment = c.Inc.CheckColumnComment
	p.RequireDMLWhere = c.Inc.CheckDMLWhere
	p.MaxColumnCount = c.Inc.MaxColumnCount
	p.MaxAffectedRows = c.Inc.MaxUpdateRows
	p.MaxInsertRows = c.Inc.MaxInsertRows
	p.CheckDMLLimit = c.Inc.CheckDMLLimit
	p.CheckDMLOrderBy = c.Inc.CheckDMLOrderBy
	p.CheckInsertField = c.Inc.CheckInsertField
	p.AllowSelectStar = c.Inc.EnableSelectStar
	p.AllowForeignKey = c.Inc.EnableForeignKey
	p.AllowPartition = c.Inc.EnablePartition
	p.AllowDropTable = c.Inc.EnableDropTable
	p.AllowDropDatabase = c.Inc.EnableDropDatabase
	p.AllowTruncate = c.Inc.EnableTruncate
	p.RequiredEngine = strings.ToLower(c.Inc.RequiredEngine)
	for name, value := range legacyVariables(c.Inc) {
		p.Legacy[name] = value
	}
	for name, value := range levelBindings(&c.IncLevel) {
		if code := legacyLevelRule[name]; code != "" {
			p.RuleLevels[code] = severity(*value)
		}
	}
	// Stable GIP RuleID configuration is authoritative when both formats are
	// present. This lets deployments migrate one rule at a time while legacy
	// Archery management commands continue to work.
	for code, value := range c.Rules {
		p.RuleLevels[code] = severity(value)
	}
	return p
}
func severity(v int) model.Severity { return model.Severity(v) }
