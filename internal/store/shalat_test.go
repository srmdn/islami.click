package store_test

import (
	"context"
	"path/filepath"
	"testing"

	islamiclick "github.com/srmdn/islami.click"
	"github.com/srmdn/islami.click/internal/model"
	"github.com/srmdn/islami.click/internal/store"
)

func TestShalatCacheRoundTrip(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "content.db")

	s, err := store.Open(ctx, dbPath, islamiclick.MigrationFS, islamiclick.ContentFS)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer s.Close()

	row := &model.ShalatCacheRow{
		City:       "Jakarta",
		PrayerDate: "2025-01-15",
		Method:     20,
		Imsak:      "04:20",
		Fajr:       "04:30",
		Sunrise:    "05:45",
		Dhuhr:      "11:50",
		Asr:        "15:15",
		Maghrib:    "18:05",
		Isha:       "19:15",
		HijriDate:  "1446-07-15",
		RawJSON:    `{"test": true}`,
		FetchedAt:  "2025-01-15T04:20:00+07:00",
		ExpiresAt:  "2025-01-16T01:00:00+07:00",
	}

	if err := s.SaveShalatCache(ctx, row); err != nil {
		t.Fatalf("save shalat cache: %v", err)
	}

	got, err := s.GetShalatCache(ctx, "Jakarta", "2025-01-15", 20)
	if err != nil {
		t.Fatalf("get shalat cache: %v", err)
	}
	if got == nil {
		t.Fatal("expected cache row, got nil")
	}
	if got.City != "Jakarta" {
		t.Fatalf("city mismatch: got %q want %q", got.City, "Jakarta")
	}
	if got.Fajr != "04:30" {
		t.Fatalf("fajr mismatch: got %q want %q", got.Fajr, "04:30")
	}
	if got.Method != 20 {
		t.Fatalf("method mismatch: got %d want %d", got.Method, 20)
	}
}

func TestShalatCacheMiss(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "content.db")

	s, err := store.Open(ctx, dbPath, islamiclick.MigrationFS, islamiclick.ContentFS)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer s.Close()

	got, err := s.GetShalatCache(ctx, "NonexistentCity", "2025-01-15", 20)
	if err != nil {
		t.Fatalf("get shalat cache: %v", err)
	}
	if got != nil {
		t.Fatal("expected nil for cache miss, got row")
	}
}

func TestShalatCacheUpsert(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "content.db")

	s, err := store.Open(ctx, dbPath, islamiclick.MigrationFS, islamiclick.ContentFS)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer s.Close()

	row1 := &model.ShalatCacheRow{
		City:       "Bandung",
		PrayerDate: "2025-01-15",
		Method:     20,
		Imsak:      "04:18",
		Fajr:       "04:28",
		Sunrise:    "05:43",
		Dhuhr:      "11:48",
		Asr:        "15:13",
		Maghrib:    "18:03",
		Isha:       "19:13",
		HijriDate:  "1446-07-15",
		RawJSON:    `{"v": 1}`,
		FetchedAt:  "2025-01-15T04:18:00+07:00",
		ExpiresAt:  "2025-01-16T01:00:00+07:00",
	}
	if err := s.SaveShalatCache(ctx, row1); err != nil {
		t.Fatalf("save row1: %v", err)
	}

	row2 := &model.ShalatCacheRow{
		City:       "Bandung",
		PrayerDate: "2025-01-15",
		Method:     20,
		Imsak:      "04:19",
		Fajr:       "04:29",
		Sunrise:    "05:44",
		Dhuhr:      "11:49",
		Asr:        "15:14",
		Maghrib:    "18:04",
		Isha:       "19:14",
		HijriDate:  "1446-07-15",
		RawJSON:    `{"v": 2}`,
		FetchedAt:  "2025-01-15T06:00:00+07:00",
		ExpiresAt:  "2025-01-16T01:00:00+07:00",
	}
	if err := s.SaveShalatCache(ctx, row2); err != nil {
		t.Fatalf("save row2: %v", err)
	}

	got, err := s.GetShalatCache(ctx, "Bandung", "2025-01-15", 20)
	if err != nil {
		t.Fatalf("get shalat cache: %v", err)
	}
	if got.Fajr != "04:29" {
		t.Fatalf("fajr mismatch after upsert: got %q want %q", got.Fajr, "04:29")
	}
}

func TestShalatStaleCache(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "content.db")

	s, err := store.Open(ctx, dbPath, islamiclick.MigrationFS, islamiclick.ContentFS)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer s.Close()

	row := &model.ShalatCacheRow{
		City:       "Surabaya",
		PrayerDate: "2025-01-14",
		Method:     20,
		Imsak:      "04:10",
		Fajr:       "04:20",
		Sunrise:    "05:35",
		Dhuhr:      "11:40",
		Asr:        "15:05",
		Maghrib:    "17:55",
		Isha:       "19:05",
		HijriDate:  "1446-07-14",
		RawJSON:    `{}`,
		FetchedAt:  "2025-01-14T04:10:00+07:00",
		ExpiresAt:  "2025-01-15T01:00:00+07:00",
	}
	if err := s.SaveShalatCache(ctx, row); err != nil {
		t.Fatalf("save: %v", err)
	}

	stale, err := s.GetShalatCacheStale(ctx, "Surabaya", 20)
	if err != nil {
		t.Fatalf("get stale: %v", err)
	}
	if stale == nil {
		t.Fatal("expected stale row, got nil")
	}
	if stale.City != "Surabaya" {
		t.Fatalf("city mismatch: got %q want %q", stale.City, "Surabaya")
	}
}
