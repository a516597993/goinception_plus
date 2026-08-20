package request

import (
	"encoding/binary"
	"strings"
	"testing"
	"unicode/utf16"
)

func TestParseLegacyEnvelope(t *testing.T) {
	input := "/*--user=test;--password=secret;--host=127.0.0.1;--port=3307;--execute=1;--backup=1;*/\n" +
		"inception_magic_start;\nuse app;\nalter table users add column nickname varchar(32);\ninception_magic_commit;"
	got, err := ParseLegacyEnvelope(input)
	if err != nil {
		t.Fatal(err)
	}
	if got.Options.Target.User != "test" || got.Options.Target.Port != 3307 {
		t.Fatalf("unexpected target: %+v", got.Options.Target)
	}
	if !got.Options.Execute || !got.Options.Backup {
		t.Fatalf("unexpected options: %+v", got.Options)
	}
	if !strings.Contains(got.SQL, "alter table") {
		t.Fatalf("unexpected SQL body: %q", got.SQL)
	}
}

func TestParseArcheryExecutionDirectives(t *testing.T) {
	input := "/*--user=test;--password=secret;--host=127.0.0.1;--port=3306;" +
		"--check=1;--execute=1;--ignore-warnings=1;--backup=0;--sleep=200;--sleep_rows=100;*/" +
		"inception_magic_start;create table t(id bigint primary key);inception_magic_commit;"
	got, err := ParseLegacyEnvelope(input)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Options.IgnoreWarnings || got.Options.SleepMillis != 200 || got.Options.SleepRows != 100 {
		t.Fatalf("unexpected Archery options: %+v", got.Options)
	}
}

func TestParseTraceID(t *testing.T) {
	input := "/*--user=test;--host=127.0.0.1;--trace-id=archery:374;*/" +
		"inception_magic_start;use app;select 1;inception_magic_commit;"
	got, err := ParseLegacyEnvelope(input)
	if err != nil {
		t.Fatal(err)
	}
	if got.Options.TraceID != "archery:374" {
		t.Fatalf("trace id=%q", got.Options.TraceID)
	}
	bad := strings.Replace(input, "archery:374", "archery/374", 1)
	if _, err = ParseLegacyEnvelope(bad); err == nil {
		t.Fatal("invalid trace id accepted")
	}
}

func TestParsePowerShellUTF16Input(t *testing.T) {
	plain := "/*--host=127.0.0.1;--user=test;*/ inception_magic_start;select 1;inception_magic_commit;"
	units := utf16.Encode([]rune(plain))
	bytes := make([]byte, 2+len(units)*2)
	bytes[0], bytes[1] = 0xff, 0xfe
	for i, unit := range units {
		binary.LittleEndian.PutUint16(bytes[2+i*2:], unit)
	}
	if _, err := ParseLegacyEnvelope(string(bytes)); err != nil {
		t.Fatalf("UTF-16 PowerShell input: %v", err)
	}
}

func TestParseRepeatedUTF8BOM(t *testing.T) {
	plain := "/*--host=127.0.0.1;--user=test;*/ inception_magic_start;select 1;inception_magic_commit;"
	if _, err := ParseLegacyEnvelope("\xef\xbb\xbf\xef\xbb\xbf" + plain); err != nil {
		t.Fatalf("repeated UTF-8 BOM: %v", err)
	}
}

func TestParseLegacyEnvelopeRequiresMarkersAndSQL(t *testing.T) {
	for _, input := range []string{
		"select 1",
		"/*--host=127.0.0.1;--user=test;*/ inception_magic_start; inception_magic_commit;",
		"/*--host=127.0.0.1;--user=test;*/ inception_magic_start; select 1;",
	} {
		if _, err := ParseLegacyEnvelope(input); err == nil {
			t.Fatalf("expected error for %q", input)
		}
	}
}

func TestMarkersInsideStringsAndCommentsAreNotControlMarkers(t *testing.T) {
	input := "/*--host=127.0.0.1;--user=test;*/ inception_magic_start;" +
		"select 'inception_magic_commit', 1 /* inception_magic_start */;" +
		"inception_magic_commit;"
	got, err := ParseLegacyEnvelope(input)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got.SQL, "'inception_magic_commit'") {
		t.Fatalf("SQL was truncated: %q", got.SQL)
	}
}

func TestStrictDirectives(t *testing.T) {
	tests := []string{
		"/*--host=127.0.0.1;--user=test;--execute=maybe;*/ inception_magic_start;select 1;inception_magic_commit;",
		"/*--host=127.0.0.1;--host=localhost;--user=test;*/ inception_magic_start;select 1;inception_magic_commit;",
		"/*--host=127.0.0.1;--user=test;--unknown=1;*/ inception_magic_start;select 1;inception_magic_commit;",
		"/*--host=127.0.0.1;--user=test;--backup=1;*/ inception_magic_start;select 1;inception_magic_commit;",
	}
	for _, input := range tests {
		if _, err := ParseLegacyEnvelope(input); err == nil {
			t.Fatalf("expected error for %q", input)
		}
	}
}
