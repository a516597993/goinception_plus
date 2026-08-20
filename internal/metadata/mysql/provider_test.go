package mysql

import "testing"

func TestQuoteIdentifier(t *testing.T) {
	if got, want := QuoteIdentifier("a\x60b"), "\x60a\x60\x60b\x60"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
