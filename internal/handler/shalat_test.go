package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/srmdn/islami.click/internal/model"
)

func TestPrayerCitiesDatasetLoads(t *testing.T) {
	cities, byName := getPrayerCities()
	if len(cities) < 500 {
		t.Fatalf("prayer cities = %d, want >= 500", len(cities))
	}
	for _, c := range cities {
		if c.Name == "" {
			t.Fatal("empty city name")
		}
		if c.TZ != "WIB" && c.TZ != "WITA" && c.TZ != "WIT" {
			t.Fatalf("%s: invalid tz %q", c.Name, c.TZ)
		}
		if c.Lat < -11 || c.Lat > 6 || c.Lon < 95 || c.Lon > 141 {
			t.Fatalf("%s: coords out of Indonesia range (%f, %f)", c.Name, c.Lat, c.Lon)
		}
	}
	jkt, ok := byName["Jakarta"]
	if !ok {
		t.Fatal("Jakarta missing from dataset")
	}
	if jkt.TZ != "WIB" || jkt.OffsetSeconds() != 7*3600 {
		t.Fatalf("Jakarta zone = %s/%d", jkt.TZ, jkt.OffsetSeconds())
	}
}

func TestCityFromRequest(t *testing.T) {
	_, byName := getPrayerCities()

	// Explicit valid query wins over cookie.
	r := httptest.NewRequest(http.MethodGet, "/shalat?city=Jayapura", nil)
	r.AddCookie(&http.Cookie{Name: prayerCityCookie, Value: "Depok"})
	city, explicit := cityFromRequest(r, byName)
	if city.Name != "Jayapura" || !explicit {
		t.Fatalf("query = %+v explicit=%v", city, explicit)
	}

	// No query: valid cookie is remembered.
	r = httptest.NewRequest(http.MethodGet, "/shalat", nil)
	r.AddCookie(&http.Cookie{Name: prayerCityCookie, Value: "Depok"})
	city, explicit = cityFromRequest(r, byName)
	if city.Name != "Depok" || explicit {
		t.Fatalf("cookie = %+v explicit=%v", city, explicit)
	}

	// Escaped cookie values (spaces, commas) decode.
	r = httptest.NewRequest(http.MethodGet, "/shalat", nil)
	r.AddCookie(&http.Cookie{Name: prayerCityCookie, Value: "Sambas%2C+Kalbar"})
	city, _ = cityFromRequest(r, byName)
	if city.Name != "Sambas, Kalbar" {
		t.Fatalf("escaped cookie = %q", city.Name)
	}

	// Invalid query and unknown cookie fall back to Jakarta.
	r = httptest.NewRequest(http.MethodGet, "/shalat?city=Kota+Tidak+Ada", nil)
	city, explicit = cityFromRequest(r, byName)
	if city.Name != "Jakarta" || explicit {
		t.Fatalf("bad query = %+v explicit=%v", city, explicit)
	}
	r = httptest.NewRequest(http.MethodGet, "/shalat", nil)
	r.AddCookie(&http.Cookie{Name: prayerCityCookie, Value: "Atlantis"})
	city, _ = cityFromRequest(r, byName)
	if city.Name != "Jakarta" {
		t.Fatalf("bad cookie = %q", city.Name)
	}

	// Nothing at all: Jakarta.
	r = httptest.NewRequest(http.MethodGet, "/shalat", nil)
	city, _ = cityFromRequest(r, byName)
	if city.Name != "Jakarta" {
		t.Fatalf("default = %q", city.Name)
	}
}

func TestPrayerTimesFromRawAppliesIhtiyati(t *testing.T) {
	// Raw Aladhan values for Jakarta 2026-09-04 (method=20).
	got := prayerTimesFromRaw("04:24", "04:34", "05:52", "11:52", "15:08", "17:52", "19:02")
	want := model.PrayerTimes{
		Imsyak:  "04:26", // Subuh+2-10
		Subuh:   "04:36", // +2
		Terbit:  "05:50", // -2
		Dhuha:   "06:14", // Terbit+24
		Dzuhur:  "11:54", // +2
		Ashr:    "15:10", // +2
		Maghrib: "17:54", // +2
		Isya:    "19:04", // +2
	}
	if got != want {
		t.Fatalf("ihtiyati mismatch:\n got %+v\nwant %+v", got, want)
	}
}

func TestPrayerTimesFromRowMatchesRaw(t *testing.T) {
	row := &model.ShalatCacheRow{
		Imsak: "04:24", Fajr: "04:34", Sunrise: "05:52", Dhuhr: "11:52",
		Asr: "15:08", Maghrib: "17:52", Isha: "19:02",
	}
	got := prayerTimesFromRow(row)
	if got.Subuh != "04:36" || got.Terbit != "05:50" || got.Dhuha != "06:14" {
		t.Fatalf("row transform = %+v", got)
	}
}

func TestHijriFromStored(t *testing.T) {
	got := hijriFromStored("1448-03-22")
	if got.Year != "1448" || got.Day != "22" || got.Month != "Rabiul Awal" {
		t.Fatalf("hijri = %+v", got)
	}
	if bad := hijriFromStored("bogus"); bad != (model.HijriDate{}) {
		t.Fatalf("bogus hijri = %+v", bad)
	}
}

func TestAddMinutesWrapsMidnight(t *testing.T) {
	if got := addMinutes("23:59", 2); got != "00:01" {
		t.Fatalf("wrap = %s", got)
	}
	if got := addMinutes("00:05", -10); got != "23:55" {
		t.Fatalf("negative = %s", got)
	}
}
