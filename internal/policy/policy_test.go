package policy

import (
	"testing"
	"time"
)

func TestPointsAndCert(t *testing.T) {
	if PointsForMinutes(60) != 10 {
		t.Fatalf("60min -> %d", PointsForMinutes(60))
	}
	if CertificateTier(19*60) != "" {
		t.Fatal("below bronze")
	}
	if CertificateTier(20*60) != "bronze" {
		t.Fatal("bronze")
	}
	if CertificateTier(50*60) != "silver" {
		t.Fatal("silver")
	}
	if CertificateTier(100*60) != "gold" {
		t.Fatal("gold")
	}
	tier, remain := NextCertificateTier(0)
	if tier != "bronze" || remain != BronzeMinutes {
		t.Fatalf("%s %d", tier, remain)
	}
}

func TestOverlap(t *testing.T) {
	a := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	b := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	c := time.Date(2026, 1, 1, 11, 0, 0, 0, time.UTC)
	d := time.Date(2026, 1, 1, 13, 0, 0, 0, time.UTC)
	if !Overlap(a, b, c, d) {
		t.Fatal("should overlap")
	}
	e := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	f := time.Date(2026, 1, 1, 14, 0, 0, 0, time.UTC)
	if Overlap(a, b, e, f) {
		t.Fatal("touching end is not overlap (start.Before end)")
	}
}
