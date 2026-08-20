module github.com/hanchuanchuan/goinception-plus

go 1.22

require (
	github.com/BurntSushi/toml v1.3.2
	github.com/go-mysql-org/go-mysql v1.9.1
	github.com/go-sql-driver/mysql v1.7.1
	github.com/pingcap/tidb/pkg/parser v0.0.0
)

require (
	github.com/Masterminds/semver v1.5.0 // indirect
	github.com/goccy/go-json v0.10.2 // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/klauspost/compress v1.17.8 // indirect
	github.com/pingcap/errors v0.11.5-0.20240311024730-e056997136bb // indirect
	github.com/pingcap/failpoint v0.0.0-20240528011301-b51a646c7c86 // indirect
	github.com/pingcap/log v1.1.1-0.20230317032135-a0d097d16e22 // indirect
	github.com/shopspring/decimal v1.2.0 // indirect
	github.com/siddontang/go v0.0.0-20180604090527-bdc77568d726 // indirect
	github.com/siddontang/go-log v0.0.0-20180807004314-8d05993dda07 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.27.0 // indirect
	golang.org/x/text v0.19.0 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.2.1 // indirect
)

replace github.com/pingcap/tidb/pkg/parser => ./third_party/tidb/pkg/parser
