package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hanchuanchuan/goinception-plus/internal/audit"
	"github.com/hanchuanchuan/goinception-plus/internal/audit/rules"
	appconfig "github.com/hanchuanchuan/goinception-plus/internal/config"
	"github.com/hanchuanchuan/goinception-plus/internal/execution"
	executionmysql "github.com/hanchuanchuan/goinception-plus/internal/execution/mysql"
	metadatamysql "github.com/hanchuanchuan/goinception-plus/internal/metadata/mysql"
	"github.com/hanchuanchuan/goinception-plus/internal/observability"
	parseradapter "github.com/hanchuanchuan/goinception-plus/internal/parser"
	"github.com/hanchuanchuan/goinception-plus/internal/protocol"
	"github.com/hanchuanchuan/goinception-plus/internal/session"
)

var (
	version   = "0.7.0-dev"
	commit    = "unknown"
	buildTime = "unknown"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "audit" {
		runAudit(os.Args[2:])
		return
	}
	serveFlags := flag.NewFlagSet("goinception-plus", flag.ExitOnError)
	configPath := serveFlags.String("config", "", "TOML configuration file")
	showVersion := serveFlags.Bool("version", false, "print version")
	_ = serveFlags.Parse(os.Args[1:])
	if *showVersion {
		fmt.Printf("goinception-plus %s commit=%s built=%s\n", version, commit, buildTime)
		return
	}
	if *configPath == "" {
		exitError(fmt.Errorf("-config is required; use the audit subcommand for stdin JSON mode"))
	}
	cfg, err := appconfig.Load(*configPath)
	if err != nil {
		exitError(err)
	}
	password, err := cfg.AuthPassword()
	if err != nil {
		exitError(err)
	}
	backupOptions, err := cfg.BackupOptions()
	if err != nil {
		exitError(err)
	}
	runtimeConfig := appconfig.NewRuntime(cfg)
	observer := observability.New(cfg.Log)
	service := newSession(cfg.Policy(), executionmysql.OpenWithBackup(backupOptions)).WithPolicyProvider(runtimeConfig.Policy)
	server := protocol.New(cfg, service, password, runtimeConfig, observer)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if !cfg.Observability.Enabled {
		if err = server.Run(ctx); err != nil {
			exitError(err)
		}
		return
	}
	errs := make(chan error, 2)
	go func() { errs <- observer.Run(ctx, cfg.Observability) }()
	go func() { errs <- server.Run(ctx) }()
	if err = <-errs; err != nil {
		stop()
		time.Sleep(50 * time.Millisecond)
		exitError(err)
	}
}
func runAudit(args []string) {
	fs := flag.NewFlagSet("audit", flag.ExitOnError)
	checkPrimaryKey := fs.Bool("check-primary-key", false, "require primary key")
	checkTableComment := fs.Bool("check-table-comment", false, "require table comment")
	checkColumnComment := fs.Bool("check-column-comment", false, "require column comments")
	maxColumnCount := fs.Int("max-column-count", 0, "maximum columns; zero disables")
	_ = fs.Parse(args)
	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		exitError(err)
	}
	policy := audit.LegacyDefaults()
	policy.CheckPrimaryKey = *checkPrimaryKey
	policy.CheckTableComment = *checkTableComment
	policy.CheckColumnComment = *checkColumnComment
	policy.MaxColumnCount = *maxColumnCount
	records, err := newSession(policy, executionmysql.Open).Audit(context.Background(), string(input))
	if err != nil {
		exitError(err)
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err = encoder.Encode(records); err != nil {
		exitError(err)
	}
}
func newSession(policy audit.Policy, factory execution.Factory) *session.Session {
	return session.New(parseradapter.New(), audit.NewEngine(rules.KnownStatement{}, rules.DDLSafety{}, rules.LegacyDDL{}, rules.RequirePrimaryKey{}, rules.RequireTableComment{}, rules.RequireColumnComments{}, rules.LimitColumnCount{}, rules.DMLSafety{})).WithPolicy(policy).WithMetadataFactory(metadatamysql.OpenRequest).WithExecutorFactory(factory)
}
func exitError(err error) { _, _ = fmt.Fprintln(os.Stderr, err); os.Exit(1) }
