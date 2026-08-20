package config

import (
	"strconv"
	"strings"

	"github.com/hanchuanchuan/goinception-plus/internal/audit"
)

func levelBindings(v *IncLevel) map[string]*int {
	return map[string]*int{
		"er_alter_table_once": &v.AlterTableOnce, "er_auto_incr_id_warning": &v.AutoIncrIDWarning,
		"er_autoinc_unsigned": &v.AutoIncUnsigned, "er_blob_cant_have_default": &v.BlobCantHaveDefault,
		"er_cant_change_column": &v.CantChangeColumn, "er_cant_change_column_position": &v.CantChangeColumnPosition,
		"er_cant_set_charset": &v.CantSetCharset, "er_cant_set_collation": &v.CantSetCollation,
		"er_cant_set_engine": &v.CantSetEngine, "er_change_column_type": &v.ChangeColumnType,
		"er_change_too_much_rows": &v.ChangeTooMuchRows, "er_char_to_varchar_len": &v.CharToVarcharLen,
		"er_charset_on_column": &v.CharsetOnColumn, "er_column_have_no_comment": &v.ColumnMustHaveComment,
		"er_datetime_default": &v.DatetimeDefault, "er_foreign_key": &v.ForeignKey,
		"er_ident_use_keyword": &v.IdentifierKeyword, "er_implicit_type_conversion": &v.ImplicitTypeConversion,
		"er_inc_init_err": &v.AutoIncrementInitial, "er_incorrect_datetime_value": &v.IncorrectDatetimeValue,
		"er_index_name_idx_prefix": &v.IndexNamePrefix, "er_index_name_uniq_prefix": &v.UniqueIndexNamePrefix,
		"er_insert_too_much_rows": &v.InsertTooMuchRows, "er_invalid_data_type": &v.InvalidDataType,
		"er_invalid_ident": &v.InvalidIdentifier, "er_join_no_on_condition": &v.JoinNoOnCondition,
		"er_json_type_support": &v.JSONTypeSupport, "er_must_have_columns": &v.MustHaveColumns,
		"er_no_where_condition": &v.NoWhereCondition, "er_not_allowed_nullable": &v.NotAllowedNullable,
		"er_ordery_by_rand": &v.OrderByRand, "er_partition_not_allowed": &v.PartitionNotAllowed,
		"er_pk_cols_not_int": &v.PrimaryKeyColumnsNotInt, "er_pk_too_many_parts": &v.PrimaryKeyTooManyParts,
		"er_select_only_star": &v.SelectOnlyStar, "er_set_data_type_int_bigint": &v.AutoIncrementIntegerType,
		"er_table_charset_must_null": &v.TableCharsetMustNull, "er_table_charset_must_utf8": &v.TableCharsetMustUTF8,
		"er_table_must_have_comment": &v.TableMustHaveComment, "er_table_must_have_pk": &v.TableMustHavePK,
		"er_text_not_nullable_error": &v.TextNotNullable, "er_too_many_key_parts": &v.TooManyKeyParts,
		"er_too_many_keys": &v.TooManyKeys, "er_too_much_auto_timestamp_cols": &v.TooManyAutoTimestampCols,
		"er_use_enum": &v.UseEnum, "er_use_text_or_blob": &v.UseTextOrBlob,
		"er_view_support": &v.ViewSupport, "er_with_default_add_column": &v.WithDefaultAddColumn,
		"er_with_insert_field": &v.InsertField, "er_with_limit_condition": &v.WithLimitCondition,
		"er_with_orderby_condition": &v.WithOrderByCondition, "er_wrong_and_expr": &v.WrongAndExpression,
	}
}

var legacyLevelRule = map[string]string{
	"er_alter_table_once": audit.RuleDDLAlterOnce, "er_auto_incr_id_warning": audit.RuleDDLAutoName,
	"er_autoinc_unsigned": audit.RuleDDLAutoUnsigned, "er_blob_cant_have_default": audit.RuleDDLBlobDefault,
	"er_cant_change_column": audit.RuleDDLCantChange, "er_cant_change_column_position": audit.RuleDDLColumnPosition,
	"er_cant_set_charset": audit.RuleDDLCantSetCharset, "er_cant_set_collation": audit.RuleDDLCantSetCollation,
	"er_cant_set_engine": audit.RuleDDLCantSetEngine, "er_change_column_type": audit.RuleDDLColumnTypeChange,
	"er_change_too_much_rows": audit.RuleDMLMaxAffected, "er_char_to_varchar_len": audit.RuleDDLCharToVarchar,
	"er_charset_on_column": audit.RuleDDLColumnCharset, "er_column_have_no_comment": audit.RuleColumnMustComment,
	"er_datetime_default": audit.RuleDDLDatetimeDefault, "er_foreign_key": audit.RuleDDLForeignKey,
	"er_ident_use_keyword": audit.RuleDDLIdentifierKeyword, "er_implicit_type_conversion": audit.RuleDMLImplicitConversion,
	"er_inc_init_err": audit.RuleDDLAutoInitial, "er_incorrect_datetime_value": audit.RuleDDLIncorrectDatetime,
	"er_index_name_idx_prefix": audit.RuleDDLIndexPrefix, "er_index_name_uniq_prefix": audit.RuleDDLUniqueIndexPrefix,
	"er_insert_too_much_rows": audit.RuleDMLMaxInsert, "er_invalid_data_type": audit.RuleDDLInvalidType,
	"er_invalid_ident": audit.RuleDDLInvalidIdentifier, "er_join_no_on_condition": audit.RuleDMLJoinNoOn,
	"er_json_type_support": audit.RuleDDLJSONType, "er_must_have_columns": audit.RuleDDLMustColumns,
	"er_no_where_condition": audit.RuleDMLRequireWhere, "er_not_allowed_nullable": audit.RuleDDLNullable,
	"er_ordery_by_rand": audit.RuleDMLOrderByRand, "er_partition_not_allowed": audit.RuleDDLPartition,
	"er_pk_cols_not_int": audit.RuleDDLPKInteger, "er_pk_too_many_parts": audit.RuleDDLMaxPrimaryParts,
	"er_select_only_star": audit.RuleDMLSelectStar, "er_set_data_type_int_bigint": audit.RuleDDLAutoInteger,
	"er_table_charset_must_null": audit.RuleDDLTableCharsetNull, "er_table_charset_must_utf8": audit.RuleDDLTableCharsetAllow,
	"er_table_must_have_comment": audit.RuleTableMustComment, "er_table_must_have_pk": audit.RuleTableMustHavePK,
	"er_text_not_nullable_error": audit.RuleDDLTextNotNull, "er_too_many_key_parts": audit.RuleDDLMaxKeyParts,
	"er_too_many_keys": audit.RuleDDLMaxKeys, "er_too_much_auto_timestamp_cols": audit.RuleDDLTooManyTimestamps,
	"er_use_enum": audit.RuleDDLUseEnum, "er_use_text_or_blob": audit.RuleDDLUseTextBlob,
	"er_view_support": audit.RuleDDLView, "er_with_default_add_column": audit.RuleDDLAddDefault,
	"er_with_insert_field": audit.RuleDMLInsertField, "er_with_limit_condition": audit.RuleDMLLimit,
	"er_with_orderby_condition": audit.RuleDMLOrderBy, "er_wrong_and_expr": audit.RuleDMLWrongAnd,
}

func legacyVariables(v Inc) map[string]string {
	b := func(x bool) string { return strconv.FormatBool(x) }
	i := func(x int) string { return strconv.Itoa(x) }
	return map[string]string{
		"check_index_column_repeat": b(v.CheckIndexColumnRepeat), "check_identifier": b(v.CheckIdentifier),
		"check_identifier_lower": b(v.CheckIdentifierLower), "enable_identifer_keyword": b(v.EnableIdentifierKeyword),
		"enable_set_charset": b(v.EnableSetCharset), "enable_set_collation": b(v.EnableSetCollation), "enable_set_engine": b(v.EnableSetEngine),
		"check_primary_key": b(v.CheckPrimaryKey), "check_table_comment": b(v.CheckTableComment), "check_column_comment": b(v.CheckColumnComment),
		"enable_foreign_key": b(v.EnableForeignKey), "enable_partition_table": b(v.EnablePartition), "enable_drop_table": b(v.EnableDropTable),
		"enable_drop_database": b(v.EnableDropDatabase), "enable_truncate_table": b(v.EnableTruncate), "required_engine": strings.ToLower(v.RequiredEngine),
		"enable_column_charset": b(v.EnableColumnCharset), "enable_blob_type": b(v.EnableBlobType), "enable_json_type": b(v.EnableJSONType),
		"max_key_parts": i(v.MaxKeyParts), "max_primary_key_parts": i(v.MaxPrimaryKeyParts), "max_keys": i(v.MaxKeys),
		"check_autoincrement_datatype": b(v.CheckAutoIncrementDataType), "check_dml_where": b(v.CheckDMLWhere),
		"check_dml_limit": b(v.CheckDMLLimit), "check_dml_orderby": b(v.CheckDMLOrderBy),
		"check_implicit_type_conversion": b(v.CheckImplicitTypeConversion), "index_prefix": v.IndexPrefix,
		"max_column_count": i(v.MaxColumnCount), "max_update_rows": strconv.FormatInt(v.MaxUpdateRows, 10), "max_insert_rows": i(v.MaxInsertRows),
		"check_insert_field": b(v.CheckInsertField), "enable_select_star": b(v.EnableSelectStar),
		"uniq_index_prefix": v.UniqueIndexPrefix, "table_prefix": v.TablePrefix, "explain_rule": strings.ToLower(v.ExplainRule),
		"sql_mode": v.SQLMode, "sql_safe_updates": i(v.SQLSafeUpdates), "lock_wait_timeout": i(v.LockWaitTimeout),
		"skip_grant_table": b(v.SkipGrantTable), "support_charset": strings.ToLower(v.SupportCharset),
		"support_engine": strings.ToLower(v.SupportEngine), "lang": v.Language, "general_log": b(v.GeneralLog),
		"default_charset": strings.ToLower(v.DefaultCharset), "must_have_columns": v.MustHaveColumns,
		"custom_keywords": strings.Join(v.CustomKeywords, ","), "check_column_type_change": b(v.CheckColumnTypeChange),
		"check_index_prefix": b(v.CheckIndexPrefix), "enable_nullable": b(v.EnableNullable), "enable_enum_set_bit": b(v.EnableEnumSetBit),
		"enable_timestamp_type": b(v.EnableTimestampType), "enable_autoincrement_unsigned": b(v.EnableAutoIncrementUnsigned),
		"enable_pk_columns_only_int": b(v.EnablePKColumnsOnlyInt), "check_timestamp_count": b(v.CheckTimestampCount), "max_char_length": i(v.MaxCharLength),
	}
}
