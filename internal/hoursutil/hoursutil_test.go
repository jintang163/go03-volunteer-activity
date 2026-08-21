package hoursutil

import (
	"testing"
	"time"
)

func TestWorkMinutesCapsAndBreak(t *testing.T) {
	in := time.Date(2026, 8, 21, 9, 0, 0, 0, time.UTC)
	out := time.Date(2026, 8, 21, 12, 0, 0, 0, time.UTC)
	if got := RawMinutes(in, out); got != 180 {
		t.Fatalf("raw=%d", got)
	}
	if got := WorkMinutes(in, out, 30, 120); got != 120 {
		t.Fatalf("capped=%d", got)
	}
	if got := WorkMinutes(in, out, 30, 0); got != 150 {
		t.Fatalf("break=%d", got)
	}
	if got := WorkMinutes(out, in, 0, 100); got != 0 {
		t.Fatalf("reversed=%d", got)
	}
}

func TestCheckInWindow(t *testing.T) {
	start := time.Date(2026, 8, 21, 10, 0, 0, 0, time.UTC)
	end := time.Date(2026, 8, 21, 12, 0, 0, 0, time.UTC)
	if InCheckInWindow(start.Add(-40*time.Minute), start, end, 30, 60) {
		t.Fatal("too early")
	}
	if !InCheckInWindow(start.Add(-10*time.Minute), start, end, 30, 60) {
		t.Fatal("should allow")
	}
	if InCheckOutWindow(start.Add(-time.Minute), start, end, 60) {
		t.Fatal("checkout before start")
	}
	if !InCheckOutWindow(end.Add(30*time.Minute), start, end, 60) {
		t.Fatal("within grace")
	}
}
