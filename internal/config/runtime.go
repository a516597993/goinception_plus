package config

import (
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/hanchuanchuan/goinception-plus/internal/audit"
)

type Variable struct{ Name, Value string }
type Level struct {
	Name, Desc string
	Value      int
}

// Runtime is the process-global, non-persistent management configuration.
// Readers always receive value copies, so an audit request keeps one policy
// snapshot even when another connection runs inception set concurrently.
type Runtime struct {
	mu  sync.RWMutex
	cfg Config
}

func NewRuntime(cfg Config) *Runtime { return &Runtime{cfg: cfg} }

func (r *Runtime) Policy() audit.Policy {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.cfg.Policy()
}

func (r *Runtime) Variables() []Variable {
	r.mu.RLock()
	defer r.mu.RUnlock()
	values := legacyVariables(r.cfg.Inc)
	values["max_column_count"] = strconv.Itoa(r.cfg.Inc.MaxColumnCount)
	values["max_update_rows"] = strconv.FormatInt(r.cfg.Inc.MaxUpdateRows, 10)
	values["max_insert_rows"] = strconv.Itoa(r.cfg.Inc.MaxInsertRows)
	values["check_insert_field"] = strconv.FormatBool(r.cfg.Inc.CheckInsertField)
	values["enable_select_star"] = strconv.FormatBool(r.cfg.Inc.EnableSelectStar)
	values["backup_host"], values["backup_port"], values["backup_user"], values["backup_password"] = r.cfg.Backup.Host, strconv.Itoa(r.cfg.Backup.Port), r.cfg.Backup.User, maskSecret(r.cfg.Backup.Password)
	values["max_allowed_packet"] = strconv.Itoa(r.cfg.Server.MaxAllowedPacket)
	items := make([]Variable, 0, len(values))
	for name, value := range values {
		items = append(items, Variable{Name: name, Value: value})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })
	return items
}

func (r *Runtime) Levels() []Level {
	r.mu.RLock()
	defer r.mu.RUnlock()
	bindings := levelBindings(&r.cfg.IncLevel)
	policy := r.cfg.Policy()
	items := make([]Level, 0, len(bindings))
	for name, value := range bindings {
		desc := name
		if definition, ok := audit.RuleByCode(legacyLevelRule[name]); ok {
			desc = definition.Summary
		}
		effective := *value
		if code := legacyLevelRule[name]; code != "" {
			effective = int(policy.Level(code))
		}
		items = append(items, Level{Name: name, Desc: desc, Value: effective})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })
	return items
}

func (r *Runtime) SetVariable(name, value string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	name = strings.ToLower(strings.TrimSpace(name))
	candidate := r.cfg
	switch name {
	case "backup_host":
		candidate.Backup.Host = strings.Trim(value, "'\"")
	case "backup_user":
		candidate.Backup.User = strings.Trim(value, "'\"")
	case "backup_password":
		candidate.Backup.Password, candidate.Backup.PasswordEnv = strings.Trim(value, "'\""), ""
	case "backup_port":
		parsed, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("Incorrect argument type to variable '%s'", name)
		}
		candidate.Backup.Port = parsed
	case "max_allowed_packet":
		parsed, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("Incorrect argument type to variable '%s'", name)
		}
		candidate.Server.MaxAllowedPacket = parsed
	default:
		if err := setIncByTag(&candidate.Inc, name, value); err != nil {
			return err
		}
	}
	if err := validateIncRuntime(candidate.Inc); err != nil {
		return err
	}
	r.cfg = candidate
	return nil
}

func validateIncRuntime(inc Inc) error {
	if inc.ExplainRule != "first" && inc.ExplainRule != "max" {
		return fmt.Errorf("inc.explain_rule must be first or max")
	}
	if inc.SQLSafeUpdates < -1 || inc.SQLSafeUpdates > 1 {
		return fmt.Errorf("inc.sql_safe_updates must be -1, 0, or 1")
	}
	if inc.LockWaitTimeout < -1 {
		return fmt.Errorf("inc.lock_wait_timeout must be -1 or non-negative")
	}
	if inc.MaxColumnCount < 0 || inc.MaxUpdateRows < 0 || inc.MaxInsertRows < 0 || inc.MaxKeyParts < 0 || inc.MaxPrimaryKeyParts < 0 || inc.MaxKeys < 0 {
		return fmt.Errorf("numeric audit limits must be non-negative")
	}
	return nil
}

func (r *Runtime) SetLevel(name string, value int) error {
	if value < 0 || value > 2 {
		return fmt.Errorf("level must be 0, 1, or 2")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if code := strings.ToUpper(strings.TrimSpace(name)); strings.HasPrefix(code, "GIP-") {
		if _, ok := audit.RuleByCode(code); !ok {
			return fmt.Errorf("invalid rule ID %q", name)
		}
		if r.cfg.Rules == nil {
			r.cfg.Rules = map[string]int{}
		}
		r.cfg.Rules[code] = value
		return nil
	}
	name = canonicalLevelAlias(strings.ToLower(strings.TrimSpace(name)))
	if target, ok := levelBindings(&r.cfg.IncLevel)[name]; ok {
		if code := legacyLevelRule[name]; code != "" {
			if _, overridden := r.cfg.Rules[code]; overridden {
				r.cfg.Rules[code] = value
			} else {
				*target = value
			}
		} else {
			*target = value
		}
	} else {
		return fmt.Errorf("invalid level variable %q", name)
	}
	return nil
}

func canonicalLevelAlias(name string) string {
	switch name {
	case "er_column_must_have_comment":
		return "er_column_have_no_comment"
	case "er_insert_field":
		return "er_with_insert_field"
	case "er_sql_no_where":
		return "er_no_where_condition"
	default:
		return name
	}
}

func setIncByTag(inc *Inc, name, text string) error {
	if name == "er_sql_no_where" {
		name = "check_dml_where"
	}
	value := reflect.ValueOf(inc).Elem()
	typeOf := value.Type()
	for i := 0; i < value.NumField(); i++ {
		if typeOf.Field(i).Tag.Get("toml") != name {
			continue
		}
		field := value.Field(i)
		switch field.Kind() {
		case reflect.Bool:
			var parsed bool
			if err := setBool(&parsed, name, text); err != nil {
				return err
			}
			field.SetBool(parsed)
		case reflect.Int, reflect.Int64:
			parsed, err := strconv.ParseInt(text, 10, 64)
			if err != nil {
				return fmt.Errorf("Incorrect argument type to variable '%s'", name)
			}
			field.SetInt(parsed)
		case reflect.String:
			field.SetString(strings.Trim(text, "'\""))
		case reflect.Slice:
			parts := strings.Split(strings.Trim(text, "[]"), ",")
			out := make([]string, 0, len(parts))
			for _, part := range parts {
				if part = strings.Trim(strings.TrimSpace(part), "'\""); part != "" {
					out = append(out, part)
				}
			}
			field.Set(reflect.ValueOf(out))
		default:
			return fmt.Errorf("variable %q is read only", name)
		}
		return nil
	}
	return fmt.Errorf("invalid variable %q", name)
}

func setBool(dst *bool, name, value string) error {
	switch strings.ToLower(value) {
	case "1", "on", "true":
		*dst = true
	case "0", "off", "false":
		*dst = false
	default:
		return fmt.Errorf("Incorrect argument type to variable '%s'", name)
	}
	return nil
}

func maskSecret(value string) string {
	if value == "" {
		return ""
	}
	return "******"
}
