package request

import (
	"encoding/binary"
	"fmt"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf16"

	"github.com/hanchuanchuan/goinception-plus/internal/model"
)

const (
	MaxRequestBytes   = 16 << 20
	MaxStatementBytes = 1 << 20
	MaxStatements     = 10000
)

var allowedDirectives = map[string]struct{}{
	"host": {}, "port": {}, "user": {}, "password": {}, "db": {},
	"check": {}, "execute": {}, "backup": {}, "sql_mode": {},
	"mysql_version": {}, "ignore_warnings": {}, "sleep": {}, "sleep_rows": {},
	"trace_id": {}, "trace-id": {},
}

type LegacyEnvelope struct {
	Options model.RequestOptions
	SQL     string
}

func ParseLegacyEnvelope(input string) (LegacyEnvelope, error) {
	input = normalizeConsoleEncoding(input)
	if len(input) > MaxRequestBytes {
		return LegacyEnvelope{}, fmt.Errorf("inception request exceeds %d bytes", MaxRequestBytes)
	}
	parts, err := splitSQL(input)
	if err != nil {
		return LegacyEnvelope{}, err
	}
	if len(parts) < 3 {
		return LegacyEnvelope{}, fmt.Errorf("request must contain inception_magic_start, SQL, and inception_magic_commit")
	}

	directiveText, startText, err := splitLeadingDirectives(parts[0])
	if err != nil {
		return LegacyEnvelope{}, err
	}
	if !markerEquals(startText, "inception_magic_start") {
		return LegacyEnvelope{}, fmt.Errorf("first statement must be inception_magic_start")
	}
	if !markerEquals(parts[len(parts)-1], "inception_magic_commit") &&
		!markerEquals(parts[len(parts)-1], "inception_magic_end") {
		return LegacyEnvelope{}, fmt.Errorf("request must end with inception_magic_commit")
	}
	for _, part := range parts[1 : len(parts)-1] {
		if markerEquals(part, "inception_magic_start") ||
			markerEquals(part, "inception_magic_commit") ||
			markerEquals(part, "inception_magic_end") {
			return LegacyEnvelope{}, fmt.Errorf("duplicate or misplaced inception control marker")
		}
	}

	bodyParts := parts[1 : len(parts)-1]
	if len(bodyParts) > MaxStatements {
		return LegacyEnvelope{}, fmt.Errorf("statement count exceeds %d", MaxStatements)
	}
	for _, part := range bodyParts {
		if len(part) > MaxStatementBytes {
			return LegacyEnvelope{}, fmt.Errorf("SQL statement exceeds %d bytes", MaxStatementBytes)
		}
	}
	values, err := parseDirectives(directiveText)
	if err != nil {
		return LegacyEnvelope{}, err
	}
	port, err := parsePort(values["port"])
	if err != nil {
		return LegacyEnvelope{}, err
	}
	check, err := parseBool(values, "check", true)
	if err != nil {
		return LegacyEnvelope{}, err
	}
	execute, err := parseBool(values, "execute", false)
	if err != nil {
		return LegacyEnvelope{}, err
	}
	backup, err := parseBool(values, "backup", false)
	if err != nil {
		return LegacyEnvelope{}, err
	}
	if strings.TrimSpace(values["host"]) == "" {
		return LegacyEnvelope{}, fmt.Errorf("target host is required")
	}
	if strings.TrimSpace(values["user"]) == "" {
		return LegacyEnvelope{}, fmt.Errorf("target user is required")
	}
	if backup && !execute {
		return LegacyEnvelope{}, fmt.Errorf("backup requires execute=1")
	}
	ignoreWarnings, err := parseBool(values, "ignore_warnings", false)
	if err != nil {
		return LegacyEnvelope{}, err
	}
	sleepMillis, err := parseNonNegativeInt(values["sleep"], "sleep", 100000)
	if err != nil {
		return LegacyEnvelope{}, err
	}
	sleepRows, err := parseNonNegativeInt(values["sleep_rows"], "sleep_rows", int(^uint(0)>>1))
	if err != nil {
		return LegacyEnvelope{}, err
	}
	if sleepMillis > 0 && sleepRows == 0 {
		sleepRows = 1
	}
	if sleepMillis == 0 {
		sleepRows = 0
	}
	traceID := values["trace_id"]
	if traceID == "" {
		traceID = values["trace-id"]
	}
	if len(traceID) > 128 {
		return LegacyEnvelope{}, fmt.Errorf("trace-id exceeds 128 characters")
	}
	for _, r := range traceID {
		if !(unicode.IsLetter(r) || unicode.IsDigit(r) || strings.ContainsRune("._:-", r)) {
			return LegacyEnvelope{}, fmt.Errorf("trace-id contains invalid character %q", r)
		}
	}

	return LegacyEnvelope{
		Options: model.RequestOptions{
			Target: model.TargetOptions{
				Host: values["host"], Port: port, User: values["user"],
				Password: values["password"], Database: values["db"],
				SQLMode: values["sql_mode"], Version: values["mysql_version"],
			},
			Check: check, Execute: execute, Backup: backup,
			IgnoreWarnings: ignoreWarnings, SleepMillis: sleepMillis, SleepRows: sleepRows,
			TraceID: traceID,
		},
		SQL: strings.Join(bodyParts, ";\n") + ";",
	}, nil
}

// Windows PowerShell 5.1 commonly writes UTF-16LE to a native process pipe.
// Accept it at the request boundary so the documented Get-Content | executable
// invocation behaves consistently with PowerShell 7 and Unix shells.
func normalizeConsoleEncoding(input string) string {
	b := []byte(input)
	for len(b) >= 3 && b[0] == 0xef && b[1] == 0xbb && b[2] == 0xbf {
		b = b[3:]
	}
	bigEndian := false
	utf16Input := false
	if len(b) >= 2 && b[0] == 0xff && b[1] == 0xfe {
		b = b[2:]
		utf16Input = true
	} else if len(b) >= 2 && b[0] == 0xfe && b[1] == 0xff {
		b = b[2:]
		bigEndian = true
		utf16Input = true
	} else if len(b) >= 4 {
		utf16Input = b[1] == 0 && b[3] == 0
		bigEndian = b[0] == 0 && b[2] == 0
	}
	if !utf16Input && !bigEndian {
		return string(b)
	}
	if len(b)%2 != 0 {
		return input
	}
	units := make([]uint16, len(b)/2)
	for i := range units {
		if bigEndian {
			units[i] = binary.BigEndian.Uint16(b[i*2:])
		} else {
			units[i] = binary.LittleEndian.Uint16(b[i*2:])
		}
	}
	return string(utf16.Decode(units))
}

func splitLeadingDirectives(input string) (string, string, error) {
	trimmed := strings.TrimSpace(input)
	if !strings.HasPrefix(trimmed, "/*") {
		return "", trimmed, nil
	}
	end := strings.Index(trimmed[2:], "*/")
	if end < 0 {
		return "", "", fmt.Errorf("unterminated inception directive comment")
	}
	end += 2
	return trimmed[2:end], strings.TrimSpace(trimmed[end+2:]), nil
}

func parseDirectives(input string) (map[string]string, error) {
	result := make(map[string]string)
	for _, field := range strings.Split(input, ";") {
		field = strings.TrimSpace(strings.TrimLeft(strings.TrimSpace(field), "-"))
		if field == "" {
			continue
		}
		parts := strings.SplitN(field, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("directive %q must use key=value", field)
		}
		key := canonicalDirective(parts[0])
		if _, ok := allowedDirectives[key]; !ok {
			return nil, fmt.Errorf("unknown inception directive %q", key)
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("duplicate inception directive %q", key)
		}
		result[key] = strings.Trim(strings.TrimSpace(parts[1]), "'\"")
	}
	return result, nil
}

func canonicalDirective(key string) string {
	key = strings.ToLower(strings.TrimSpace(key))
	return strings.ReplaceAll(key, "-", "_")
}

func parseNonNegativeInt(value, key string, max int) (int, error) {
	if value == "" {
		return 0, nil
	}
	v, err := strconv.Atoi(value)
	if err != nil || v < 0 || v > max {
		return 0, fmt.Errorf("invalid %s value %q", key, value)
	}
	return v, nil
}

func parsePort(value string) (uint16, error) {
	if value == "" {
		return 3306, nil
	}
	port, err := strconv.ParseUint(value, 10, 16)
	if err != nil || port == 0 {
		return 0, fmt.Errorf("invalid target port %q", value)
	}
	return uint16(port), nil
}

func parseBool(values map[string]string, key string, fallback bool) (bool, error) {
	value, ok := values[key]
	if !ok {
		return fallback, nil
	}
	switch strings.ToLower(value) {
	case "1", "true", "on":
		return true, nil
	case "0", "false", "off":
		return false, nil
	default:
		return false, fmt.Errorf("invalid boolean value %q for %s", value, key)
	}
}

func markerEquals(value, marker string) bool {
	return strings.EqualFold(strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(value), ";")), marker)
}

func splitSQL(input string) ([]string, error) {
	var result []string
	start := 0
	var quote rune
	blockComment := false
	lineComment := false
	escaped := false
	runes := []rune(input)
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		next := rune(0)
		if i+1 < len(runes) {
			next = runes[i+1]
		}
		if lineComment {
			if r == '\n' {
				lineComment = false
			}
			continue
		}
		if blockComment {
			if r == '*' && next == '/' {
				blockComment = false
				i++
			}
			continue
		}
		if quote != 0 {
			if escaped {
				escaped = false
				continue
			}
			if r == '\\' && quote != '\x60' {
				escaped = true
				continue
			}
			if r == quote {
				if next == quote {
					i++
					continue
				}
				quote = 0
			}
			continue
		}
		if r == '/' && next == '*' {
			blockComment = true
			i++
			continue
		}
		if r == '#' || (r == '-' && next == '-' && (i+2 >= len(runes) || unicode.IsSpace(runes[i+2]))) {
			lineComment = true
			if r == '-' {
				i++
			}
			continue
		}
		if r == '\'' || r == '"' || r == '\x60' {
			quote = r
			continue
		}
		if r == ';' {
			part := strings.TrimSpace(string(runes[start:i]))
			if part != "" {
				result = append(result, part)
			}
			start = i + 1
		}
	}
	if quote != 0 || blockComment {
		return nil, fmt.Errorf("unterminated quoted string or comment")
	}
	if part := strings.TrimSpace(string(runes[start:])); part != "" {
		result = append(result, part)
	}
	return result, nil
}
