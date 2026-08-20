package audit

import (
	"fmt"
	"regexp"

	"github.com/hanchuanchuan/goinception-plus/internal/model"
)

const (
	RuleUnknownStatement      = "GIP-COR-ST-001"
	RuleParserWarning         = "GIP-COR-PS-001"
	RuleParserError           = "GIP-COR-PS-002"
	RuleMetadataConnect       = "GIP-META-CN-001"
	RuleTargetVersion         = "GIP-META-CN-002"
	RuleSchemaNotFound        = "GIP-META-DB-001"
	RuleNoDatabase            = "GIP-META-DB-002"
	RuleDDLMetadata           = "GIP-DDL-MD-001"
	RuleTableMustHavePK       = "GIP-DDL-CT-001"
	RuleTableMustComment      = "GIP-DDL-CT-002"
	RuleColumnMustComment     = "GIP-DDL-CT-003"
	RuleMaxColumnCount        = "GIP-DDL-CT-004"
	RuleDDLDropDatabase       = "GIP-DDL-SF-001"
	RuleDDLDropTable          = "GIP-DDL-SF-002"
	RuleDDLTruncate           = "GIP-DDL-SF-003"
	RuleDDLForeignKey         = "GIP-DDL-SF-004"
	RuleDDLPartition          = "GIP-DDL-SF-005"
	RuleDDLEngine             = "GIP-DDL-CT-005"
	RuleDDLDuplicateCol       = "GIP-DDL-CT-006"
	RuleDDLDuplicateIdx       = "GIP-DDL-CT-007"
	RuleDDLIndexColumn        = "GIP-DDL-CT-008"
	RuleDDLInvalidIdentifier  = "GIP-DDL-CT-009"
	RuleDDLIdentifierKeyword  = "GIP-DDL-CT-010"
	RuleDDLIndexPrefix        = "GIP-DDL-CT-011"
	RuleDDLUniqueIndexPrefix  = "GIP-DDL-CT-012"
	RuleDDLMaxKeyParts        = "GIP-DDL-CT-013"
	RuleDDLMaxPrimaryParts    = "GIP-DDL-CT-014"
	RuleDDLMaxKeys            = "GIP-DDL-CT-015"
	RuleDDLPKInteger          = "GIP-DDL-CT-016"
	RuleDDLAutoInteger        = "GIP-DDL-CT-017"
	RuleDDLAutoUnsigned       = "GIP-DDL-CT-018"
	RuleDDLAutoName           = "GIP-DDL-CT-019"
	RuleDDLAutoInitial        = "GIP-DDL-CT-020"
	RuleDDLMustColumns        = "GIP-DDL-CT-021"
	RuleDDLTableCharsetNull   = "GIP-DDL-CT-022"
	RuleDDLTableCharsetAllow  = "GIP-DDL-CT-023"
	RuleDDLColumnCharset      = "GIP-DDL-CT-024"
	RuleDDLBlobDefault        = "GIP-DDL-CT-025"
	RuleDDLInvalidType        = "GIP-DDL-CT-026"
	RuleDDLJSONType           = "GIP-DDL-CT-027"
	RuleDDLNullable           = "GIP-DDL-CT-028"
	RuleDDLTextNotNull        = "GIP-DDL-CT-029"
	RuleDDLAddDefault         = "GIP-DDL-CT-030"
	RuleDDLDatetimeDefault    = "GIP-DDL-CT-031"
	RuleDDLCharToVarchar      = "GIP-DDL-CT-032"
	RuleDDLTooManyTimestamps  = "GIP-DDL-CT-033"
	RuleDDLIndexRepeat        = "GIP-DDL-CT-034"
	RuleDDLIncorrectDatetime  = "GIP-DDL-CT-035"
	RuleDDLUseEnum            = "GIP-DDL-CT-036"
	RuleDDLUseTextBlob        = "GIP-DDL-CT-037"
	RuleDDLAlterOnce          = "GIP-DDL-AT-001"
	RuleDDLCantChange         = "GIP-DDL-AT-002"
	RuleDDLColumnPosition     = "GIP-DDL-AT-003"
	RuleDDLColumnTypeChange   = "GIP-DDL-AT-004"
	RuleDDLCantSetCharset     = "GIP-DDL-SF-006"
	RuleDDLCantSetCollation   = "GIP-DDL-SF-007"
	RuleDDLCantSetEngine      = "GIP-DDL-SF-008"
	RuleDDLView               = "GIP-DDL-SF-009"
	RuleDMLTableMetadata      = "GIP-DML-MD-001"
	RuleDMLRequireWhere       = "GIP-DML-SF-001"
	RuleDMLRequireKey         = "GIP-DML-SF-002"
	RuleDMLImpactEstimate     = "GIP-DML-IM-001"
	RuleDMLMaxAffected        = "GIP-DML-IM-002"
	RuleDMLMaxInsert          = "GIP-DML-IM-003"
	RuleDMLLimit              = "GIP-DML-SF-003"
	RuleDMLOrderBy            = "GIP-DML-SF-004"
	RuleDMLJoinNoOn           = "GIP-DML-SF-005"
	RuleDMLSelectStar         = "GIP-DML-SF-006"
	RuleDMLInsertField        = "GIP-DML-SF-007"
	RuleDMLOrderByRand        = "GIP-DML-SF-008"
	RuleDMLWrongAnd           = "GIP-DML-SF-009"
	RuleDMLImplicitConversion = "GIP-DML-SF-010"
	RuleExecutionFailed       = "GIP-EXE-RN-001"
	RuleAuditOnly             = "GIP-COR-SF-001"
	RuleBinlogPrereq          = "GIP-BAK-BL-001"
	RuleBackupFailed          = "GIP-BAK-RN-001"
)

type RuleDefinition struct {
	Code         string
	LegacyKey    string
	DefaultLevel model.Severity
	Phase        int
	Summary      string
}

var definitions = []RuleDefinition{
	{RuleUnknownStatement, "", model.SeverityError, 1, "Reject statement types not admitted by the audit kernel"},
	{RuleParserWarning, "", model.SeverityWarning, 1, "Expose a warning returned by the SQL parser"},
	{RuleParserError, "", model.SeverityError, 1, "Report SQL parser failure as an audit result"},
	{RuleMetadataConnect, "", model.SeverityError, 2, "Target MySQL metadata connection failed"},
	{RuleTargetVersion, "", model.SeverityError, 2, "Target server must be MySQL 5.7 or 8.0"},
	{RuleSchemaNotFound, "", model.SeverityError, 2, "Selected database does not exist or is unavailable"},
	{RuleNoDatabase, "", model.SeverityError, 2, "No database selected for an object statement"},
	{RuleDDLMetadata, "", model.SeverityError, 2, "DDL object or schema snapshot validation failed"},
	{RuleTableMustHavePK, "er_table_must_have_pk", model.SeverityWarning, 2, "Require a primary key on CREATE TABLE"},
	{RuleTableMustComment, "er_table_must_have_comment", model.SeverityWarning, 2, "Require a table comment on CREATE TABLE"},
	{RuleColumnMustComment, "er_column_have_no_comment", model.SeverityWarning, 2, "Require comments on CREATE TABLE columns"},
	{RuleMaxColumnCount, "er_max_column_count", model.SeverityWarning, 2, "Limit the number of columns on CREATE TABLE"},
	{RuleDDLDropDatabase, "er_cant_drop_database", model.SeverityError, 2, "DROP DATABASE is disabled"},
	{RuleDDLDropTable, "er_cant_drop_table", model.SeverityError, 2, "DROP TABLE is disabled"},
	{RuleDDLTruncate, "er_cant_truncate_table", model.SeverityError, 2, "TRUNCATE TABLE is disabled"},
	{RuleDDLForeignKey, "er_foreign_key", model.SeverityError, 2, "Foreign keys are disabled"},
	{RuleDDLPartition, "er_partition_not_allowed", model.SeverityError, 2, "Partition tables are disabled"},
	{RuleDDLEngine, "er_table_engine", model.SeverityError, 2, "Require the configured table engine"},
	{RuleDDLDuplicateCol, "er_column_existed", model.SeverityError, 2, "Reject duplicate columns"},
	{RuleDDLDuplicateIdx, "", model.SeverityError, 2, "Reject duplicate index names"},
	{RuleDDLIndexColumn, "er_column_not_existed", model.SeverityError, 2, "Every index column must exist"},
	{RuleDDLInvalidIdentifier, "er_invalid_ident", model.SeverityError, 2, "Reject invalid identifiers"},
	{RuleDDLIdentifierKeyword, "er_ident_use_keyword", model.SeverityError, 2, "Reject keyword identifiers"},
	{RuleDDLIndexPrefix, "er_index_name_idx_prefix", model.SeverityWarning, 2, "Require normal index name prefix"},
	{RuleDDLUniqueIndexPrefix, "er_index_name_uniq_prefix", model.SeverityWarning, 2, "Require unique index name prefix"},
	{RuleDDLMaxKeyParts, "er_too_many_key_parts", model.SeverityError, 2, "Limit index key parts"},
	{RuleDDLMaxPrimaryParts, "er_pk_too_many_parts", model.SeverityError, 2, "Limit primary key parts"},
	{RuleDDLMaxKeys, "er_too_many_keys", model.SeverityWarning, 2, "Limit indexes per table"},
	{RuleDDLPKInteger, "er_pk_cols_not_int", model.SeverityWarning, 2, "Require integer primary key columns"},
	{RuleDDLAutoInteger, "er_set_data_type_int_bigint", model.SeverityWarning, 2, "Require integer auto increment column"},
	{RuleDDLAutoUnsigned, "er_autoinc_unsigned", model.SeverityWarning, 2, "Recommend unsigned auto increment column"},
	{RuleDDLAutoName, "er_auto_incr_id_warning", model.SeverityWarning, 2, "Recommend ID auto increment column name"},
	{RuleDDLAutoInitial, "er_inc_init_err", model.SeverityWarning, 2, "Require AUTO_INCREMENT initial value one"},
	{RuleDDLMustColumns, "er_must_have_columns", model.SeverityError, 2, "Require configured table columns"},
	{RuleDDLTableCharsetNull, "er_table_charset_must_null", model.SeverityError, 2, "Reject explicit table charset"},
	{RuleDDLTableCharsetAllow, "er_table_charset_must_utf8", model.SeverityError, 2, "Require supported table charset"},
	{RuleDDLColumnCharset, "er_charset_on_column", model.SeverityError, 2, "Reject column charset or collation"},
	{RuleDDLBlobDefault, "er_blob_cant_have_default", model.SeverityWarning, 2, "Reject default on large object columns"},
	{RuleDDLInvalidType, "er_invalid_data_type", model.SeverityWarning, 2, "Reject disabled data types"},
	{RuleDDLJSONType, "er_json_type_support", model.SeverityWarning, 2, "Reject JSON when disabled"},
	{RuleDDLNullable, "er_not_allowed_nullable", model.SeverityWarning, 2, "Reject nullable columns"},
	{RuleDDLTextNotNull, "er_text_not_nullable_error", model.SeverityWarning, 2, "Reject NOT NULL text/blob columns"},
	{RuleDDLAddDefault, "er_with_default_add_column", model.SeverityError, 2, "Require defaults for added columns"},
	{RuleDDLDatetimeDefault, "er_datetime_default", model.SeverityWarning, 2, "Require DATETIME defaults"},
	{RuleDDLCharToVarchar, "er_char_to_varchar_len", model.SeverityWarning, 2, "Recommend VARCHAR for long CHAR"},
	{RuleDDLTooManyTimestamps, "er_too_much_auto_timestamp_cols", model.SeverityWarning, 2, "Limit automatic timestamp columns"},
	{RuleDDLIndexRepeat, "", model.SeverityWarning, 2, "Reject redundant index leading columns"},
	{RuleDDLIncorrectDatetime, "er_incorrect_datetime_value", model.SeverityError, 2, "Reject invalid datetime literals"},
	{RuleDDLUseEnum, "er_use_enum", model.SeverityError, 2, "Reject ENUM columns"},
	{RuleDDLUseTextBlob, "er_use_text_or_blob", model.SeverityWarning, 2, "Reject text/blob columns"},
	{RuleDDLAlterOnce, "er_alter_table_once", model.SeverityWarning, 2, "Merge ALTER statements for one table"},
	{RuleDDLCantChange, "er_cant_change_column", model.SeverityWarning, 2, "Reject CHANGE COLUMN"},
	{RuleDDLColumnPosition, "er_cant_change_column_position", model.SeverityWarning, 2, "Reject column position changes"},
	{RuleDDLColumnTypeChange, "er_change_column_type", model.SeverityWarning, 2, "Report unsafe column type changes"},
	{RuleDDLCantSetCharset, "er_cant_set_charset", model.SeverityError, 2, "Reject explicit charset"},
	{RuleDDLCantSetCollation, "er_cant_set_collation", model.SeverityError, 2, "Reject explicit collation"},
	{RuleDDLCantSetEngine, "er_cant_set_engine", model.SeverityError, 2, "Reject explicit engine"},
	{RuleDDLView, "er_view_support", model.SeverityError, 2, "Reject views"},
	{RuleDMLTableMetadata, "", model.SeverityError, 3, "DML table and referenced columns must exist"},
	{RuleDMLRequireWhere, "er_no_where_condition", model.SeverityError, 3, "UPDATE and DELETE require a WHERE condition"},
	{RuleDMLRequireKey, "", model.SeverityError, 3, "Backup DML requires a reliable primary or unique key"},
	{RuleDMLImpactEstimate, "", model.SeverityError, 3, "DML impact rows could not be estimated safely"},
	{RuleDMLMaxAffected, "er_change_too_much_rows", model.SeverityError, 3, "DML estimated rows exceed configured maximum"},
	{RuleDMLMaxInsert, "er_insert_too_much_rows", model.SeverityError, 3, "INSERT VALUES row count exceeds configured maximum"},
	{RuleDMLLimit, "er_with_limit_condition", model.SeverityWarning, 3, "UPDATE or DELETE contains LIMIT"},
	{RuleDMLOrderBy, "er_with_orderby_condition", model.SeverityWarning, 3, "UPDATE or DELETE contains ORDER BY"},
	{RuleDMLJoinNoOn, "er_join_no_on_condition", model.SeverityError, 3, "JOIN requires an ON condition"},
	{RuleDMLSelectStar, "er_select_only_star", model.SeverityWarning, 3, "SELECT star is disabled"},
	{RuleDMLInsertField, "er_with_insert_field", model.SeverityWarning, 3, "INSERT requires an explicit column list"},
	{RuleDMLOrderByRand, "er_ordery_by_rand", model.SeverityError, 3, "Reject ORDER BY RAND"},
	{RuleDMLWrongAnd, "er_wrong_and_expr", model.SeverityWarning, 3, "Reject suspicious AND expressions"},
	{RuleDMLImplicitConversion, "er_implicit_type_conversion", model.SeverityError, 3, "Reject implicit type conversions"},
	{RuleExecutionFailed, "", model.SeverityError, 4, "Target SQL execution failed"},
	{RuleAuditOnly, "", model.SeverityError, 7, "Audit-only service rejects execution and backup requests"},
	{RuleBinlogPrereq, "", model.SeverityError, 5, "ROW/FULL binlog backup prerequisite failed"},
	{RuleBackupFailed, "", model.SeverityError, 5, "ROW binlog capture, rollback generation, or backup storage failed"},
}

func RuleCatalog() []RuleDefinition {
	return append([]RuleDefinition(nil), definitions...)
}

func RuleByCode(code string) (RuleDefinition, bool) {
	for _, definition := range definitions {
		if definition.Code == code {
			return definition, true
		}
	}
	return RuleDefinition{}, false
}

var ruleCodePattern = regexp.MustCompile("^GIP-(COR|DDL|DML|META|EXE|BAK|PRO)-[A-Z]{2}-[0-9]{3}$")

func ValidateRuleCatalog() error {
	seenCodes := make(map[string]struct{}, len(definitions))
	seenLegacyKeys := make(map[string]string, len(definitions))
	for _, definition := range definitions {
		if !ruleCodePattern.MatchString(definition.Code) {
			return fmt.Errorf("invalid rule code %q", definition.Code)
		}
		if _, exists := seenCodes[definition.Code]; exists {
			return fmt.Errorf("duplicate rule code %q", definition.Code)
		}
		seenCodes[definition.Code] = struct{}{}
		if definition.LegacyKey == "" {
			continue
		}
		if previous, exists := seenLegacyKeys[definition.LegacyKey]; exists {
			return fmt.Errorf("legacy key %q is used by %s and %s", definition.LegacyKey, previous, definition.Code)
		}
		seenLegacyKeys[definition.LegacyKey] = definition.Code
	}
	return nil
}
