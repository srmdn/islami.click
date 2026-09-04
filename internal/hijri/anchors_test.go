package hijri

import (
	"testing"
	"time"
)

func TestFromGregorianUsesKemenagAnchors(t *testing.T) {
	// 1 Rabiul Awal 1448 = Jumat 14 Agustus 2026 (Kemenag + PBNU).
	cases := []struct {
		greg      time.Time
		wantYear  int
		wantMonth int
		wantDay   int
	}{
		{time.Date(2026, 8, 14, 0, 0, 0, 0, time.UTC), 1448, 3, 1},
		{time.Date(2026, 9, 4, 12, 0, 0, 0, time.UTC), 1448, 3, 22},
		{time.Date(2026, 9, 12, 0, 0, 0, 0, time.UTC), 1448, 3, 30},
		{time.Date(2026, 9, 13, 0, 0, 0, 0, time.UTC), 1448, 4, 1},
		{time.Date(2026, 8, 25, 0, 0, 0, 0, time.UTC), 1448, 3, 12}, // Maulid
	}
	for _, tc := range cases {
		got := FromGregorian(tc.greg)
		if got.Year != tc.wantYear || got.Month != tc.wantMonth || got.Day != tc.wantDay {
			t.Errorf("FromGregorian(%s) = %+v, want %d/%d/%d",
				tc.greg.Format("2006-01-02"), got, tc.wantYear, tc.wantMonth, tc.wantDay)
		}
		if Source(tc.greg) != "kemenag" {
			t.Errorf("Source(%s) = %q, want kemenag", tc.greg.Format("2006-01-02"), Source(tc.greg))
		}
	}
}

func TestFromGregorianRespectsLocationDate(t *testing.T) {
	wib := time.FixedZone("WIB", 7*3600)
	// 2026-09-03 18:00 UTC is already Sept 4 in Jakarta.
	got := FromGregorian(time.Date(2026, 9, 3, 18, 0, 0, 0, time.UTC).In(wib))
	if got.Day != 22 || got.Month != 3 {
		t.Fatalf("WIB evening = %+v, want 22/3", got)
	}
}

func TestToGregorianAnchored(t *testing.T) {
	got := Date{Year: 1448, Month: 3, Day: 22}.ToGregorian()
	if got.Format("2006-01-02") != "2026-09-04" {
		t.Fatalf("ToGregorian = %s, want 2026-09-04", got.Format("2006-01-02"))
	}
}

func TestMonthLengthAnchored(t *testing.T) {
	// Rabiul Awal 1448 runs 14 Aug – 12 Sep 2026 = 30 days (Kemenag).
	if got := MonthLength(1448, 3); got != 30 {
		t.Fatalf("MonthLength(1448,3) = %d, want 30", got)
	}
	// Safar 1448: 16 Jul – 13 Aug 2026 = 29 days.
	if got := MonthLength(1448, 2); got != 29 {
		t.Fatalf("MonthLength(1448,2) = %d, want 29", got)
	}
	// Outside coverage: tabular fallback (odd=30, even=29).
	if got := MonthLength(1450, 1); got != 30 {
		t.Fatalf("fallback odd = %d, want 30", got)
	}
	if got := MonthLength(1450, 2); got != 29 {
		t.Fatalf("fallback even = %d, want 29", got)
	}
}

func TestSourceFallsBackOutsideCoverage(t *testing.T) {
	if got := Source(time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)); got != "perkiraan" {
		t.Fatalf("Source(2030) = %q, want perkiraan", got)
	}
	// Arithmetic fallback still resolves a date (old behavior preserved).
	got := FromGregorian(time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC))
	if got.Year != 1451 || got.Month != 8 {
		t.Fatalf("fallback 2030-01-01 = %+v", got)
	}
}
