package observability

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	appconfig "github.com/hanchuanchuan/goinception-plus/internal/config"
)

type Service struct {
	Logger     *slog.Logger
	started    time.Time
	ready      atomic.Bool
	active     atomic.Int64
	total      atomic.Uint64
	failed     atomic.Uint64
	durationNS atomic.Uint64
}

func New(cfg appconfig.Log) *Service {
	level := slog.LevelInfo
	switch cfg.Level {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}
	opts := &slog.HandlerOptions{Level: level}
	var handler slog.Handler = slog.NewTextHandler(os.Stderr, opts)
	if cfg.Format == "json" {
		handler = slog.NewJSONHandler(os.Stderr, opts)
	}
	return &Service{Logger: slog.New(handler), started: time.Now()}
}

func (s *Service) SetReady(v bool) { s.ready.Store(v) }
func (s *Service) Begin() func(bool, time.Duration) {
	s.active.Add(1)
	start := time.Now()
	return func(failed bool, ignored time.Duration) {
		s.active.Add(-1)
		s.total.Add(1)
		if failed {
			s.failed.Add(1)
		}
		s.durationNS.Add(uint64(time.Since(start)))
	}
}

func (s *Service) Run(ctx context.Context, cfg appconfig.Observability) error {
	if !cfg.Enabled {
		return nil
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, _ *http.Request) {
		if !s.ready.Load() {
			http.Error(w, "not ready", http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte("ready\n"))
	})
	mux.HandleFunc("/metrics", s.metrics)
	server := &http.Server{Addr: net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port)), Handler: mux, ReadHeaderTimeout: 3 * time.Second}
	ln, err := net.Listen("tcp", server.Addr)
	if err != nil {
		return err
	}
	s.Logger.Info("observability listening", "address", server.Addr)
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdown)
	}()
	err = server.Serve(ln)
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}

func (s *Service) metrics(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	seconds := float64(s.durationNS.Load()) / float64(time.Second)
	ready := 0
	if s.ready.Load() {
		ready = 1
	}
	_, _ = fmt.Fprintf(w, strings.Join([]string{
		"# TYPE goinception_plus_ready gauge", "goinception_plus_ready %d",
		"# TYPE goinception_plus_audit_active gauge", "goinception_plus_audit_active %d",
		"# TYPE goinception_plus_audit_total counter", "goinception_plus_audit_total %d",
		"# TYPE goinception_plus_audit_failed_total counter", "goinception_plus_audit_failed_total %d",
		"# TYPE goinception_plus_audit_duration_seconds_total counter", "goinception_plus_audit_duration_seconds_total %.6f",
		"# TYPE goinception_plus_uptime_seconds gauge", "goinception_plus_uptime_seconds %.0f", "",
	}, "\n"), ready, s.active.Load(), s.total.Load(), s.failed.Load(), seconds, time.Since(s.started).Seconds())
}
