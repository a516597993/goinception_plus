package protocol

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log/slog"
	"net"
	"strings"
	"sync"
	"time"

	gomysql "github.com/go-mysql-org/go-mysql/mysql"
	mysqlserver "github.com/go-mysql-org/go-mysql/server"

	"github.com/hanchuanchuan/goinception-plus/internal/audit"
	appconfig "github.com/hanchuanchuan/goinception-plus/internal/config"
	"github.com/hanchuanchuan/goinception-plus/internal/model"
	"github.com/hanchuanchuan/goinception-plus/internal/observability"
	requestadapter "github.com/hanchuanchuan/goinception-plus/internal/request"
	"github.com/hanchuanchuan/goinception-plus/internal/session"
)

type Server struct {
	cfg      appconfig.Config
	session  *session.Session
	password string
	runtime  *appconfig.Runtime
	registry *processRegistry
	listener net.Listener
	observer *observability.Service
	wg       sync.WaitGroup
}

func New(cfg appconfig.Config, s *session.Session, password string, runtime *appconfig.Runtime, observers ...*observability.Service) *Server {
	observer := observability.New(cfg.Log)
	if len(observers) > 0 && observers[0] != nil {
		observer = observers[0]
	}
	return &Server{cfg: cfg, session: s, password: password, runtime: runtime, registry: newProcessRegistry(), observer: observer}
}
func (s *Server) Run(ctx context.Context) error {
	address := net.JoinHostPort(s.cfg.Server.Host, fmt.Sprint(s.cfg.Server.Port))
	ln, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}
	s.listener = ln
	s.observer.SetReady(true)
	defer s.observer.SetReady(false)
	s.observer.Logger.Info("MySQL protocol listening", "address", address, "audit_only", s.cfg.Server.AuditOnly)
	sem := make(chan struct{}, s.cfg.Server.MaxConnections)
	go func() {
		<-ctx.Done()
		_ = ln.Close()
		// HandleCommand can be blocked on an idle socket. Closing every active
		// connection is required for bounded graceful shutdown.
		s.registry.closeAll()
	}()
	for {
		raw, acceptErr := ln.Accept()
		if acceptErr != nil {
			if ctx.Err() != nil {
				break
			}
			return acceptErr
		}
		select {
		case sem <- struct{}{}:
			s.wg.Add(1)
			go func() { defer s.wg.Done(); defer func() { <-sem }(); s.serveConn(ctx, raw) }()
		default:
			_ = raw.Close()
		}
	}
	done := make(chan struct{})
	go func() { s.wg.Wait(); close(done) }()
	select {
	case <-done:
		return nil
	case <-time.After(s.cfg.Server.ShutdownTimeout.Duration):
		return fmt.Errorf("shutdown timed out with active connections")
	}
}
func (s *Server) serveConn(ctx context.Context, raw net.Conn) {
	defer raw.Close()
	provider := mysqlserver.NewInMemoryProvider()
	provider.AddUser(s.cfg.Auth.Username, s.password)
	conf := mysqlserver.NewServer("5.7.25-goInception-Plus", gomysql.DEFAULT_COLLATION_ID, gomysql.AUTH_NATIVE_PASSWORD, nil, nil)
	handler := &handler{ctx: ctx, session: s.session, maxPacket: s.cfg.Server.MaxAllowedPacket, runtime: s.runtime, registry: s.registry, username: s.cfg.Auth.Username, auditTimeout: s.cfg.Server.AuditTimeout.Duration, auditOnly: s.cfg.Server.AuditOnly, observer: s.observer}
	conn, err := mysqlserver.NewCustomizedConn(raw, conf, provider, handler)
	if err != nil {
		return
	}
	handler.id = conn.ConnectionID()
	s.registry.add(&processInfo{ID: handler.id, User: conn.GetUser(), FromHost: remoteHost(raw.RemoteAddr()), Command: "Sleep", closeConn: func() { _ = raw.Close() }})
	defer s.registry.remove(handler.id)
	for {
		_ = raw.SetDeadline(time.Now().Add(s.cfg.Server.IdleTimeout.Duration))
		if err = conn.HandleCommand(); err != nil {
			return
		}
		if ctx.Err() != nil {
			return
		}
	}
}

type handler struct {
	ctx          context.Context
	session      *session.Session
	database     string
	maxPacket    int
	id           uint32
	runtime      *appconfig.Runtime
	registry     *processRegistry
	username     string
	warnings     [][]interface{}
	auditTimeout time.Duration
	auditOnly    bool
	observer     *observability.Service
}

func (h *handler) UseDB(name string) error {
	h.database = name
	h.registry.setDB(h.id, name)
	return nil
}
func (h *handler) HandleQuery(query string) (*gomysql.Result, error) {
	if len(query) > h.maxPacket {
		return nil, gomysql.NewError(gomysql.ER_NET_PACKET_TOO_LARGE, "Got a packet bigger than 'max_allowed_packet' bytes")
	}
	trimmed := strings.TrimSpace(query)
	lower := strings.ToLower(trimmed)
	var queryCtx context.Context
	var cancel context.CancelFunc
	if h.auditTimeout > 0 && strings.Contains(lower, "inception_magic_start") {
		queryCtx, cancel = context.WithTimeout(h.ctx, h.auditTimeout)
	} else {
		queryCtx, cancel = context.WithCancel(h.ctx)
	}
	h.registry.begin(h.id, "Query", trimmed, cancel)
	defer func() { cancel(); h.registry.end(h.id) }()
	if result, handled, err := h.management(query); handled {
		if err != nil {
			h.warnings = [][]interface{}{{"Error", int64(gomysql.ER_UNKNOWN_ERROR), err.Error()}}
		} else {
			h.warnings = nil
		}
		return result, err
	}
	switch {
	case strings.Contains(lower, "inception_magic_start"):
		envelope, parseErr := requestadapter.ParseLegacyEnvelope(query)
		if parseErr == nil {
			h.registry.setTarget(h.id, envelope.Options.Target.User, envelope.Options.Target.Host, envelope.Options.Target.Port)
		}
		requestID := envelope.Options.TraceID
		if requestID == "" {
			requestID = fmt.Sprintf("conn-%d-%d", h.id, time.Now().UnixNano())
		}
		hash := fmt.Sprintf("%x", sha256.Sum256([]byte(query)))
		started := time.Now()
		finishMetric := h.observer.Begin()
		failed := false
		defer func() { finishMetric(failed, time.Since(started)) }()
		logAttrs := []any{"request_id", requestID, "connection_id", h.id, "sql_sha256", hash}
		if parseErr == nil {
			logAttrs = append(logAttrs, "target_host", envelope.Options.Target.Host, "target_port", envelope.Options.Target.Port, "target_database", envelope.Options.Target.Database)
		}
		if h.auditOnly && parseErr == nil && (envelope.Options.Execute || envelope.Options.Backup) {
			failed = true
			record := model.AuditRecord{Sequence: 1, Stage: "CHECKED", ErrorLevel: model.SeverityError, StageStatus: "Check failed", ErrorMessage: "audit-only service rejects execute=1 or backup=1", SQL: envelope.SQL, Issues: []model.AuditIssue{{RuleID: audit.RuleAuditOnly, Level: model.SeverityError, Message: "audit-only service rejects execute=1 or backup=1", Sequence: 1}}}
			h.observer.Logger.Error("audit request rejected", append(logAttrs, "event", "audit_rejected", "rule_code", audit.RuleAuditOnly, "duration_ms", time.Since(started).Milliseconds())...)
			return LegacyResult([]model.AuditRecord{record})
		}
		if parseErr != nil {
			failed = true
			h.observer.Logger.Warn("audit request parse failed", append(logAttrs, "event", "audit_failed", "failure_stage", "request_parse", "error", parseErr.Error())...)
		}
		records, err := h.session.Audit(queryCtx, query)
		if err != nil {
			failed = true
			h.observer.Logger.Error("audit request failed", append(logAttrs, "event", "audit_failed", "failure_stage", failureStage(err, queryCtx), "error", err.Error(), "duration_ms", time.Since(started).Milliseconds())...)
			h.warnings = [][]interface{}{{"Error", int64(gomysql.ER_PARSE_ERROR), err.Error()}}
			return nil, gomysql.NewError(gomysql.ER_PARSE_ERROR, err.Error())
		}
		warnings, errors := 0, 0
		rules := make([]string, 0)
		for _, record := range records {
			if record.ErrorLevel == model.SeverityWarning {
				warnings++
			}
			if record.ErrorLevel == model.SeverityError {
				errors++
				failed = true
			}
			for _, issue := range record.Issues {
				rules = append(rules, issue.RuleID)
			}
		}
		h.observer.Logger.Log(h.ctx, slog.LevelInfo, "audit request completed", append(logAttrs, "event", "audit_completed", "statement_count", len(records), "warning_count", warnings, "error_count", errors, "rule_codes", strings.Join(rules, ","), "duration_ms", time.Since(started).Milliseconds())...)
		return LegacyResult(records)
	case strings.HasPrefix(lower, "set names ") || strings.HasPrefix(lower, "set character_set_") || strings.HasPrefix(lower, "set autocommit"):
		return &gomysql.Result{}, nil
	case strings.HasPrefix(lower, "use "):
		h.database = strings.Trim(strings.TrimSpace(trimmed[4:]), "` ;")
		h.registry.setDB(h.id, h.database)
		return &gomysql.Result{}, nil
	case lower == "select connection_id()" || lower == "select connection_id();":
		return simple([]string{"CONNECTION_ID()"}, [][]interface{}{{uint64(h.id)}})
	case lower == "select version()" || lower == "select version();" || lower == "select @@version" || lower == "select @@version;":
		return simple([]string{"VERSION()"}, [][]interface{}{{"5.7.25-goInception-Plus"}})
	case strings.HasPrefix(lower, "show variables"):
		values := [][]interface{}{{"version", "5.7.25-goInception-Plus"}, {"character_set_server", "utf8mb4"}, {"max_allowed_packet", fmt.Sprint(h.maxPacket)}}
		if index := strings.Index(lower, " like "); index >= 0 {
			wanted := strings.Trim(strings.TrimSpace(trimmed[index+6:]), "'\" ;")
			filtered := values[:0]
			for _, row := range values {
				if strings.EqualFold(row[0].(string), wanted) {
					filtered = append(filtered, row)
				}
			}
			values = filtered
		}
		return simple([]string{"Variable_name", "Value"}, values)
	default:
		message := "only inception audit requests and goInception management commands are supported"
		h.warnings = [][]interface{}{{"Error", int64(gomysql.ER_NOT_SUPPORTED_YET), message}}
		return nil, gomysql.NewError(gomysql.ER_NOT_SUPPORTED_YET, message)
	}
}

func failureStage(err error, ctx context.Context) string {
	if ctx.Err() != nil {
		return "timeout_or_cancel"
	}
	if strings.Contains(strings.ToLower(err.Error()), "parse") {
		return "sql_parse"
	}
	return "audit"
}
func (h *handler) HandleFieldList(string, string) ([]*gomysql.Field, error) {
	return nil, gomysql.NewError(gomysql.ER_NOT_SUPPORTED_YET, "COM_FIELD_LIST is not supported")
}
func (h *handler) HandleStmtPrepare(string) (int, int, interface{}, error) {
	return 0, 0, nil, gomysql.NewError(gomysql.ER_UNSUPPORTED_PS, "prepared statements are not supported")
}
func (h *handler) HandleStmtExecute(interface{}, string, []interface{}) (*gomysql.Result, error) {
	return nil, gomysql.NewError(gomysql.ER_UNSUPPORTED_PS, "prepared statements are not supported")
}
func (h *handler) HandleStmtClose(interface{}) error { return nil }
func (h *handler) HandleOtherCommand(cmd byte, _ []byte) error {
	return gomysql.NewError(gomysql.ER_UNKNOWN_COM_ERROR, fmt.Sprintf("unsupported command 0x%02x", cmd))
}
func simple(names []string, values [][]interface{}) (*gomysql.Result, error) {
	rs, err := gomysql.BuildSimpleTextResultset(names, values)
	if err != nil {
		return nil, err
	}
	return &gomysql.Result{Resultset: rs}, nil
}
