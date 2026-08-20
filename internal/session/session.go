package session

import (
	"context"
	"fmt"
	"strings"

	"github.com/hanchuanchuan/goinception-plus/internal/audit"
	"github.com/hanchuanchuan/goinception-plus/internal/execution"
	"github.com/hanchuanchuan/goinception-plus/internal/metadata"
	"github.com/hanchuanchuan/goinception-plus/internal/model"
	parseradapter "github.com/hanchuanchuan/goinception-plus/internal/parser"
	"github.com/hanchuanchuan/goinception-plus/internal/request"
)

type OpenMetadataFunc func(context.Context, model.TargetOptions) (
	audit.MetadataProvider, audit.ServerInfo, func() error, error,
)

type Session struct {
	parser       parseradapter.SQLParser
	engine       *audit.Engine
	policy       audit.Policy
	policySource func() audit.Policy
	openMetadata OpenMetadataFunc
	openExecutor execution.Factory
}

func (s *Session) WithExecutorFactory(factory execution.Factory) *Session {
	s.openExecutor = factory
	return s
}

func New(p parseradapter.SQLParser, engine *audit.Engine) *Session {
	return &Session{parser: p, engine: engine, policy: audit.LegacyDefaults()}
}

func (s *Session) WithPolicy(policy audit.Policy) *Session {
	s.policy = policy
	s.policySource = nil
	return s
}

func (s *Session) WithPolicyProvider(provider func() audit.Policy) *Session {
	s.policySource = provider
	return s
}

func (s *Session) WithMetadataFactory(factory OpenMetadataFunc) *Session {
	s.openMetadata = factory
	return s
}

func (s *Session) Audit(ctx context.Context, input string) ([]model.AuditRecord, error) {
	envelope, err := request.ParseLegacyEnvelope(input)
	if err != nil {
		return nil, err
	}
	if s.openMetadata == nil {
		return []model.AuditRecord{failureRecord(
			1, "", audit.RuleMetadataConnect, "target MySQL metadata factory is not configured",
		)}, nil
	}
	source, server, closeFn, err := s.openMetadata(ctx, envelope.Options.Target)
	if err != nil {
		return []model.AuditRecord{failureRecord(
			1, "", audit.RuleMetadataConnect, fmt.Sprintf("connect target MySQL: %v", err),
		)}, nil
	}
	if closeFn != nil {
		defer closeFn()
	}
	envelope.Options.Target.Version = server.Version
	envelope.Options.Target.SQLMode = server.SQLMode
	cache := metadata.NewRequestCacheWithCaseMode(source, server.LowerCaseTableNames != 0)
	if !supportedMySQLVersion(server.Version) {
		return []model.AuditRecord{failureRecord(1, "", audit.RuleTargetVersion,
			fmt.Sprintf("unsupported target server version %q; only MySQL 5.7 and 8.0 are supported", server.Version))}, nil
	}
	policy := s.policy
	if s.policySource != nil {
		policy = s.policySource()
	}
	parseSQLMode := server.SQLMode
	if configured := strings.TrimSpace(policy.LegacyString("sql_mode")); configured != "" {
		parseSQLMode = configured
	}
	envelope.Options.Target.SQLMode = parseSQLMode
	envelope.Options.Target.SQLSafeUpdates = policy.LegacyInt("sql_safe_updates")
	envelope.Options.Target.LockWaitTimeout = policy.LegacyInt("lock_wait_timeout")
	envelope.Options.Target.DefaultCharset = policy.LegacyString("default_charset")

	parseResult, err := s.parser.Parse(envelope.SQL, parseSQLMode)
	if err != nil {
		return []model.AuditRecord{failureRecord(
			1, envelope.SQL, audit.RuleParserError, err.Error(),
		)}, nil
	}

	auditCtx := &audit.Context{
		Request: envelope.Options, Database: envelope.Options.Target.Database,
		Policy: policy, Metadata: cache, Server: server,
		SeenAlterTables: make(map[string]int),
	}
	records := make([]model.AuditRecord, 0, len(parseResult.Statements))
	checkedGrants := make(map[string]bool)
	for _, statement := range parseResult.Statements {
		var issues []model.AuditIssue
		if statement.Kind == model.StatementUse {
			if _, err := cache.LoadSchema(ctx, statement.Database); err != nil {
				issues = append(issues, newIssue(
					audit.RuleSchemaNotFound, statement.Sequence,
					fmt.Sprintf("database %q is unavailable: %v", statement.Database, err),
				))
			} else {
				auditCtx.Database = statement.Database
			}
		} else {
			resolveStatementDatabase(&statement, auditCtx.Database)
			if requiresDatabase(statement) && statement.Database == "" {
				issues = append(issues, newIssue(
					audit.RuleNoDatabase, statement.Sequence, "No database selected.",
				))
			}
		}
		if len(issues) == 0 && !policy.LegacyBool("skip_grant_table") && statement.Database != "" && !checkedGrants[strings.ToLower(statement.Database)] {
			if checker, ok := auditCtx.Metadata.(audit.GrantChecker); ok {
				if grantErr := checker.CheckGrants(ctx, statement.Database, envelope.Options.Execute); grantErr != nil {
					issues = append(issues, newIssue(audit.RuleMetadataConnect, statement.Sequence, grantErr.Error()))
				} else {
					checkedGrants[strings.ToLower(statement.Database)] = true
				}
			}
		}
		if len(issues) == 0 && statement.DML != nil {
			if statement.Kind == model.StatementInsert && statement.DML.ValueRows > 0 {
				statement.DML.EstimatedRows = int64(statement.DML.ValueRows)
				statement.DML.EstimateMethod = "values"
			} else if policy.MaxAffectedRows > 0 && (statement.Kind == model.StatementUpdate || statement.Kind == model.StatementDelete || statement.DML.InsertSelect) {
				if estimator, ok := auditCtx.Metadata.(audit.ImpactEstimator); ok {
					var estimate model.ImpactEstimate
					var estimateErr error
					if advanced, advancedOK := auditCtx.Metadata.(audit.ImpactEstimatorWithRule); advancedOK {
						estimate, estimateErr = advanced.EstimateImpactWithRule(ctx, auditCtx.Database, statement.Normalized, policy.LegacyString("explain_rule"))
					} else {
						estimate, estimateErr = estimator.EstimateImpact(ctx, auditCtx.Database, statement.Normalized)
					}
					if estimateErr != nil {
						issues = append(issues, newIssue(audit.RuleDMLImpactEstimate, statement.Sequence, estimateErr.Error()))
					} else {
						statement.DML.EstimatedRows, statement.DML.EstimateMethod = estimate.Rows, estimate.Method
					}
				}
			}
		}
		if len(issues) == 0 && statement.DDL != nil && statement.Supported {
			if err := cache.PrepareDDL(ctx, &statement); err != nil {
				issues = append(issues, newIssue(audit.RuleDDLMetadata, statement.Sequence, err.Error()))
			}
		}
		issues = append(issues, s.engine.Check(ctx, auditCtx, statement)...)
		if statement.Sequence == 1 {
			for _, warning := range parseResult.Warnings {
				issues = append(issues, model.AuditIssue{
					RuleID: audit.RuleParserWarning, Level: model.SeverityWarning,
					Message: warning.Message, Sequence: statement.Sequence,
				})
			}
		}
		record := makeRecord(statement, issues)
		if record.ErrorLevel != model.SeverityError && statement.DDL != nil && statement.Supported {
			if err := cache.ApplyDDL(ctx, statement.DDL); err != nil {
				issues = append(issues, newIssue(audit.RuleDDLMetadata, statement.Sequence, err.Error()))
				record = makeRecord(statement, issues)
			}
		}
		records = append(records, record)
	}
	if envelope.Options.Execute {
		for _, r := range records {
			if r.ErrorLevel == model.SeverityError ||
				(r.ErrorLevel == model.SeverityWarning && !envelope.Options.IgnoreWarnings) {
				return records, nil
			}
		}
		if s.openExecutor == nil {
			records[0] = appendExecutionError(records[0], audit.RuleExecutionFailed, "execution engine is not configured")
			return records, nil
		}
		executor, closeExec, openErr := s.openExecutor(ctx, envelope.Options.Target)
		if openErr != nil {
			records[0] = appendExecutionError(records[0], audit.RuleExecutionFailed, openErr.Error())
			return records, nil
		}
		if closeExec != nil {
			defer closeExec()
		}
		results := executor.Execute(ctx, envelope.Options, parseResult.Statements)
		for _, result := range results {
			if result.Sequence < 1 || result.Sequence > len(records) {
				continue
			}
			i := result.Sequence - 1
			records[i].ExecuteTime = result.Duration
			records[i].AffectedRows = result.AffectedRows
			records[i].BackupDatabase = result.BackupDatabase
			records[i].BackupCompleted = result.BackupCompleted
			records[i].BackupTime = result.BackupDuration
			records[i].OperationID = result.SequenceID
			if result.Error != "" {
				ruleID := result.RuleID
				if ruleID == "" {
					ruleID = audit.RuleExecutionFailed
				}
				if ruleID == audit.RuleBackupFailed && result.Executed {
					records[i] = appendBackupError(records[i], ruleID, result.Error)
				} else {
					records[i] = appendExecutionError(records[i], ruleID, result.Error)
				}
				break
			}
			records[i].Stage = "EXECUTED"
			if result.BackupCompleted {
				records[i].Stage = "BACKUP"
				records[i].StageStatus = "BackupSuccessfully"
			} else {
				records[i].StageStatus = "ExecuteSuccessfully"
			}
		}
	}
	return records, nil
}

func appendBackupError(r model.AuditRecord, ruleID, message string) model.AuditRecord {
	issue := newIssue(ruleID, r.Sequence, message)
	r.Issues = append(r.Issues, issue)
	r.ErrorLevel = model.SeverityError
	r.ErrorMessage = strings.TrimSpace(r.ErrorMessage + "\n" + message)
	r.Stage = "BACKUP"
	r.StageStatus = "BackupFailed"
	return r
}

func appendExecutionError(r model.AuditRecord, ruleID, message string) model.AuditRecord {
	issue := newIssue(ruleID, r.Sequence, message)
	r.Issues = append(r.Issues, issue)
	r.ErrorLevel = model.SeverityError
	r.ErrorMessage = strings.TrimSpace(r.ErrorMessage + "\n" + message)
	r.Stage = "EXECUTED"
	r.StageStatus = "ExecuteFailed"
	return r
}

func supportedMySQLVersion(version string) bool {
	v := strings.TrimSpace(strings.ToLower(version))
	if strings.Contains(v, "mariadb") || strings.Contains(v, "tidb") {
		return false
	}
	return strings.HasPrefix(v, "5.7.") || v == "5.7" || strings.HasPrefix(v, "8.0.") || v == "8.0"
}

func resolveStatementDatabase(statement *model.Statement, current string) {
	if statement.Database == "" {
		statement.Database = current
	}
	if statement.DDL != nil {
		if statement.DDL.Schema == "" {
			statement.DDL.Schema = statement.Database
		} else {
			statement.Database = statement.DDL.Schema
		}
		for i := range statement.DDL.Tables {
			if statement.DDL.Tables[i].Schema == "" {
				statement.DDL.Tables[i].Schema = statement.Database
			}
		}
		for i := range statement.DDL.RenamePairs {
			if statement.DDL.RenamePairs[i].From.Schema == "" {
				statement.DDL.RenamePairs[i].From.Schema = statement.Database
			}
			if statement.DDL.RenamePairs[i].To.Schema == "" {
				statement.DDL.RenamePairs[i].To.Schema = statement.Database
			}
		}
	}
	if statement.DML != nil {
		for i := range statement.DML.Tables {
			if statement.DML.Tables[i].Schema == "" {
				statement.DML.Tables[i].Schema = statement.Database
			}
		}
		if statement.Database == "" && len(statement.DML.Tables) > 0 {
			statement.Database = statement.DML.Tables[0].Schema
		}
	}
}

func requiresDatabase(statement model.Statement) bool {
	switch statement.Kind {
	case model.StatementCreateTable, model.StatementAlterTable,
		model.StatementDropTable, model.StatementRenameTable,
		model.StatementTruncateTable, model.StatementInsert,
		model.StatementUpdate, model.StatementDelete,
		model.StatementCreateIndex, model.StatementDropIndex,
		model.StatementCreateView, model.StatementAlterView, model.StatementDropView:
		return true
	default:
		return false
	}
}

func makeRecord(statement model.Statement, issues []model.AuditIssue) model.AuditRecord {
	record := model.AuditRecord{
		Sequence: statement.Sequence, Stage: "CHECKED",
		StageStatus: "Checked", SQL: statement.Original, Issues: issues,
	}
	if statement.DML != nil {
		record.AffectedRows = statement.DML.EstimatedRows
	}
	var messages []string
	for _, issue := range issues {
		if issue.Level > record.ErrorLevel {
			record.ErrorLevel = issue.Level
		}
		messages = append(messages, issue.Message)
	}
	record.ErrorMessage = strings.Join(messages, "\n")
	switch record.ErrorLevel {
	case model.SeverityError:
		record.StageStatus = "CheckFailed"
	case model.SeverityWarning:
		record.StageStatus = "CheckedWithWarnings"
	}
	return record
}

func failureRecord(sequence int, sqlText, code, message string) model.AuditRecord {
	issue := newIssue(code, sequence, message)
	return model.AuditRecord{
		Sequence: sequence, Stage: "CHECKED", ErrorLevel: model.SeverityError,
		StageStatus: "CheckFailed", ErrorMessage: message,
		SQL: sqlText, Issues: []model.AuditIssue{issue},
	}
}

func newIssue(code string, sequence int, message string) model.AuditIssue {
	issue := model.AuditIssue{
		RuleID: code, Level: model.SeverityError,
		Message: message, Sequence: sequence,
	}
	if definition, ok := audit.RuleByCode(code); ok {
		issue.LegacyKey = definition.LegacyKey
	}
	return issue
}
