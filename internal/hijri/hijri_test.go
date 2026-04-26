package hijri

import (
	"testing"
	"time"
)

func TestFromGregorianKnown(t *testing.T) {
	cases := []struct {
		greg      time.Time
		wantYear  int
		wantMonth int
	}{
		{time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC), 1446, 9},
		{time.Date(2025, 3, 30, 0, 0, 0, 0, time.UTC), 1446, 9},
		{time.Date(2024, 6, 16, 0, 0, 0, 0, time.UTC), 1445, 12},
		{time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), 1446, 7},
	}

	for _, tc := range cases {
		got := FromGregorian(tc.greg)
		if got.Year != tc.wantYear || got.Month != tc.wantMonth {
			t.Errorf("FromGregorian(%v) = %+v, want year/month %d/%d",
				tc.greg.Format("2006-01-02"), got, tc.wantYear, tc.wantMonth)
		}
	}
}

func TestRoundTrip(t *testing.T) {
	orig := Date{Year: 1446, Month: 9, Day: 15}
	g := orig.ToGregorian()
	round := FromGregorian(g)
	if round.Year != orig.Year || round.Month != orig.Month || round.Day != orig.Day {
		t.Errorf("round trip: got %+v, want %+v", round, orig)
	}
}

func TestFormatGregorianID(t *testing.T) {
	got := FormatGregorianID(time.Date(2025, 3, 15, 0, 0, 0, 0, time.UTC))
	want := "Sabtu, 15 Maret 2025"
	if got != want {
		t.Errorf("FormatGregorianID = %q, want %q", got, want)
	}
}

func TestFormatID(t *testing.T) {
	h := Date{Year: 1446, Month: 9, Day: 15}
	got := h.FormatID()
	if got == "" {
		t.Error("FormatID returned empty string")
	}
}