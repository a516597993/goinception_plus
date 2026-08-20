package protocol

import (
	"context"
	"fmt"
	"net"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	gomysql "github.com/go-mysql-org/go-mysql/mysql"

	appconfig "github.com/hanchuanchuan/goinception-plus/internal/config"
)

var (
	reInceptionShow = regexp.MustCompile(`(?is)^inception\s+(?:show|get)\s+(full\s+)?(variables|levels|processlist)(?:\s+(like|where)\s+(.+))?$`)
	reInceptionSet  = regexp.MustCompile(`(?is)^inception\s+set\s+(?:(global|session)\s+)?(?:(level)\s+)?([a-zA-Z0-9_-]+)\s*=\s*(.+)$`)
	reKill          = regexp.MustCompile(`(?is)^kill\s+(?:(query|connection)\s+)?([0-9]+)$`)
	reShowDB        = regexp.MustCompile(`(?is)^show\s+databases(?:\s+like\s+(.+))?$`)
	reShowTables    = regexp.MustCompile(`(?is)^show\s+(full\s+)?tables(?:\s+(?:from|in)\s+(` + "`?" + `[a-zA-Z0-9_$]+` + "`?" + `))?(?:\s+like\s+(.+))?$`)
	reTableStatus   = regexp.MustCompile(`(?is)^show\s+table\s+status(?:\s+(?:from|in)\s+(` + "`?" + `[a-zA-Z0-9_$]+` + "`?" + `))?(?:\s+like\s+(.+))?$`)
	reColumns       = regexp.MustCompile(`(?is)^show\s+(?:full\s+)?(?:columns|fields)\s+from\s+(` + "`?" + `[a-zA-Z0-9_$]+` + "`?" + `)(?:\s+(?:from|in)\s+(` + "`?" + `[a-zA-Z0-9_$]+` + "`?" + `))?(?:\s+like\s+(.+))?$`)
	reShowGrants    = regexp.MustCompile(`(?is)^show\s+grants(?:\s+for\s+.+)?$`)
)

type processInfo struct {
	ID        uint32
	User      string
	FromHost  string
	DestUser  string
	DestHost  string
	DestPort  uint16
	Database  string
	Command   string
	State     string
	Info      string
	Started   time.Time
	cancel    context.CancelFunc
	closeConn func()
}

type processRegistry struct {
	mu    sync.RWMutex
	items map[uint32]*processInfo
}

func newProcessRegistry() *processRegistry {
	return &processRegistry{items: make(map[uint32]*processInfo)}
}
func (r *processRegistry) add(p *processInfo) { r.mu.Lock(); r.items[p.ID] = p; r.mu.Unlock() }
func (r *processRegistry) remove(id uint32)   { r.mu.Lock(); delete(r.items, id); r.mu.Unlock() }
func (r *processRegistry) setDB(id uint32, db string) {
	r.mu.Lock()
	if p := r.items[id]; p != nil {
		p.Database = db
	}
	r.mu.Unlock()
}
func (r *processRegistry) setTarget(id uint32, user, host string, port uint16) {
	r.mu.Lock()
	if p := r.items[id]; p != nil {
		p.DestUser, p.DestHost, p.DestPort = user, host, port
	}
	r.mu.Unlock()
}
func (r *processRegistry) begin(id uint32, command, info string, cancel context.CancelFunc) {
	r.mu.Lock()
	if p := r.items[id]; p != nil {
		p.Command, p.State, p.Info, p.Started, p.cancel = command, "executing", info, time.Now(), cancel
	}
	r.mu.Unlock()
}
func (r *processRegistry) end(id uint32) {
	r.mu.Lock()
	if p := r.items[id]; p != nil {
		p.Command, p.State, p.Info, p.cancel = "Sleep", "", "", nil
	}
	r.mu.Unlock()
}
func (r *processRegistry) snapshot() []processInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]processInfo, 0, len(r.items))
	for _, p := range r.items {
		out = append(out, *p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}
func (r *processRegistry) kill(id uint32, connection bool) bool {
	r.mu.RLock()
	p := r.items[id]
	if p == nil {
		r.mu.RUnlock()
		return false
	}
	cancel, closeConn := p.cancel, p.closeConn
	r.mu.RUnlock()
	if cancel != nil {
		cancel()
	}
	if connection && closeConn != nil {
		closeConn()
	}
	return true
}

func (r *processRegistry) closeAll() {
	r.mu.RLock()
	items := make([]processInfo, 0, len(r.items))
	for _, p := range r.items {
		items = append(items, *p)
	}
	r.mu.RUnlock()
	for _, p := range items {
		if p.cancel != nil {
			p.cancel()
		}
		if p.closeConn != nil {
			p.closeConn()
		}
	}
}

func (h *handler) management(query string) (*gomysql.Result, bool, error) {
	q := strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(query), ";"))
	lower := strings.ToLower(q)
	if m := reInceptionShow.FindStringSubmatch(q); m != nil {
		switch strings.ToLower(m[2]) {
		case "variables":
			rows := [][]interface{}{}
			for _, v := range h.runtime.Variables() {
				if m[3] == "" || (strings.EqualFold(m[3], "like") && like(v.Name, unquote(m[4]))) {
					rows = append(rows, []interface{}{v.Name, v.Value})
				}
			}
			return mustSimple([]string{"Variable_name", "Value"}, rows), true, nil
		case "levels":
			rows := [][]interface{}{}
			for _, v := range h.runtime.Levels() {
				if levelMatches(v, m[3], m[4]) {
					rows = append(rows, []interface{}{v.Name, int64(v.Value), v.Desc})
				}
			}
			return mustSimple([]string{"Name", "Value", "Desc"}, rows), true, nil
		case "processlist":
			return h.processList(strings.TrimSpace(m[1]) != ""), true, nil
		}
	}
	if m := reInceptionSet.FindStringSubmatch(q); m != nil {
		value := unquote(m[4])
		var err error
		if strings.EqualFold(m[2], "level") {
			var n int
			n, err = strconv.Atoi(value)
			if err == nil {
				err = h.runtime.SetLevel(m[3], n)
			}
		} else {
			err = h.runtime.SetVariable(m[3], value)
		}
		if err != nil {
			return nil, true, gomysql.NewError(gomysql.ER_WRONG_VALUE_FOR_VAR, err.Error())
		}
		return &gomysql.Result{}, true, nil
	}
	if strings.HasPrefix(lower, "inception ") {
		return nil, true, gomysql.NewError(gomysql.ER_NOT_SUPPORTED_YET, "unsupported inception management command")
	}
	if m := reKill.FindStringSubmatch(q); m != nil {
		id, _ := strconv.ParseUint(m[2], 10, 32)
		if !h.registry.kill(uint32(id), !strings.EqualFold(m[1], "query")) {
			return nil, true, gomysql.NewError(gomysql.ER_NO_SUCH_THREAD, fmt.Sprintf("Unknown thread id: %d", id))
		}
		return &gomysql.Result{}, true, nil
	}
	if m := reShowDB.FindStringSubmatch(q); m != nil {
		rows := [][]interface{}{}
		if m[1] == "" || like("information_schema", unquote(m[1])) {
			rows = append(rows, []interface{}{"information_schema"})
		}
		return mustSimple([]string{"Database"}, rows), true, nil
	}
	if m := reShowTables.FindStringSubmatch(q); m != nil {
		db := cleanIdent(m[2])
		if db == "" {
			db = h.database
		}
		name := "Tables_in_" + db
		if db == "" {
			name = "Tables_in_"
		}
		cols := []string{name}
		if strings.TrimSpace(m[1]) != "" {
			cols = append(cols, "Table_type")
		}
		return mustSimple(cols, nil), true, nil
	}
	if m := reTableStatus.FindStringSubmatch(q); m != nil {
		return mustSimple([]string{"Name", "Engine", "Version", "Row_format", "Rows", "Avg_row_length", "Data_length", "Max_data_length", "Index_length", "Data_free", "Auto_increment", "Create_time", "Update_time", "Check_time", "Collation", "Checksum", "Create_options", "Comment"}, nil), true, nil
	}
	if m := reColumns.FindStringSubmatch(q); m != nil {
		db := cleanIdent(m[2])
		if db == "" {
			db = h.database
		}
		if db == "" {
			db = "information_schema"
		}
		return nil, true, gomysql.NewError(gomysql.ER_NO_SUCH_TABLE, fmt.Sprintf("Table '%s.%s' doesn't exist", db, cleanIdent(m[1])))
	}
	if lower == "show warnings" {
		return mustSimple([]string{"Level", "Code", "Message"}, h.warnings), true, nil
	}
	if reShowGrants.MatchString(q) {
		col := fmt.Sprintf("Grants for %s@%%", h.username)
		return mustSimple([]string{col}, [][]interface{}{{fmt.Sprintf("GRANT USAGE ON *.* TO '%s'@'%%'", h.username)}}), true, nil
	}
	return nil, false, nil
}

func (h *handler) processList(full bool) *gomysql.Result {
	rows := [][]interface{}{}
	for _, p := range h.registry.snapshot() {
		info := p.Info
		if !full && len(info) > 100 {
			info = info[:100]
		}
		seconds := int64(0)
		if !p.Started.IsZero() {
			seconds = int64(time.Since(p.Started) / time.Second)
		}
		rows = append(rows, []interface{}{uint64(p.ID), p.DestUser, p.DestHost, int64(p.DestPort), p.User + "@" + p.FromHost, p.Command, p.State, seconds, info, nil})
	}
	return mustSimple([]string{"Id", "Dest_User", "Dest_Host", "Dest_Port", "From_Host", "Command", "STATE", "Time", "Info", "Percent"}, rows)
}

func levelMatches(v appconfig.Level, filter, expression string) bool {
	if filter == "" {
		return true
	}
	if strings.EqualFold(filter, "like") {
		return like(v.Name, unquote(expression))
	}
	m := regexp.MustCompile(`(?is)^\s*(name|value|desc)\s*(=|like)\s*(.+?)\s*$`).FindStringSubmatch(expression)
	if m == nil {
		return false
	}
	var actual string
	switch strings.ToLower(m[1]) {
	case "name":
		actual = v.Name
	case "value":
		actual = strconv.Itoa(v.Value)
	default:
		actual = v.Desc
	}
	if strings.EqualFold(m[2], "like") {
		return like(actual, unquote(m[3]))
	}
	return strings.EqualFold(actual, unquote(m[3]))
}
func like(value, pattern string) bool {
	var b strings.Builder
	b.WriteString("(?i)^")
	for _, r := range pattern {
		if r == '%' {
			b.WriteString(".*")
		} else if r == '_' {
			b.WriteByte('.')
		} else {
			b.WriteString(regexp.QuoteMeta(string(r)))
		}
	}
	b.WriteByte('$')
	ok, _ := regexp.MatchString(b.String(), value)
	return ok
}
func unquote(v string) string {
	v = strings.TrimSpace(v)
	if len(v) >= 2 && ((v[0] == '\'' && v[len(v)-1] == '\'') || (v[0] == '"' && v[len(v)-1] == '"')) {
		v = v[1 : len(v)-1]
	}
	return strings.ReplaceAll(v, "''", "'")
}
func cleanIdent(v string) string { return strings.Trim(strings.TrimSpace(v), "`") }
func mustSimple(names []string, rows [][]interface{}) *gomysql.Result {
	r, err := simple(names, rows)
	if err != nil {
		panic(err)
	}
	return r
}
func remoteHost(addr net.Addr) string {
	if addr == nil {
		return ""
	}
	host, _, err := net.SplitHostPort(addr.String())
	if err != nil {
		return addr.String()
	}
	return host
}
