package xraypoller

import (
	"testing"
	"time"
)

func TestParseCursor(t *testing.T) {
	if _, ok, err := parseCursor(""); err != nil || ok {
		t.Fatalf("empty: ok %v err %v", ok, err)
	}
	if _, ok, err := parseCursor("0"); err != nil || ok {
		t.Fatalf("zero: ok %v err %v", ok, err)
	}
	got, ok, err := parseCursor("2026-10-26T12:00:00Z")
	if err != nil || !ok {
		t.Fatal(err)
	}
	if !got.Equal(time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC)) {
		t.Fatalf("got %s", got)
	}
}
