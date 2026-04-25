package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/srmdn/islami.click/internal/model"
)

func (s *Store) GetShalatCache(ctx context.Context, city, prayerDate string, method int) (*model.ShalatCacheRow, error) {
	var row model.ShalatCacheRow
	err := s.db.QueryRowContext(ctx, `
		SELECT city, prayer_date, method, imsak, fajr, sunrise, dhuhr, asr, maghrib, isha,
		       hijri_date, raw_json, fetched_at, expires_at
		FROM prayer_time_cache
		WHERE city = ? AND prayer_date = ? AND method = ?
	`, city, prayerDate, method).Scan(
		&row.City, &row.PrayerDate, &row.Method,
		&row.Imsak, &row.Fajr, &row.Sunrise, &row.Dhuhr, &row.Asr, &row.Maghrib, &row.Isha,
		&row.HijriDate, &row.RawJSON, &row.FetchedAt, &row.ExpiresAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query shalat cache: %w", err)
	}
	return &row, nil
}

func (s *Store) SaveShalatCache(ctx context.Context, row *model.ShalatCacheRow) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT OR REPLACE INTO prayer_time_cache (
			city, prayer_date, method, imsak, fajr, sunrise, dhuhr, asr, maghrib, isha,
			hijri_date, raw_json, fetched_at, expires_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		row.City, row.PrayerDate, row.Method,
		row.Imsak, row.Fajr, row.Sunrise, row.Dhuhr, row.Asr, row.Maghrib, row.Isha,
		row.HijriDate, row.RawJSON, row.FetchedAt, row.ExpiresAt,
	)
	if err != nil {
		return fmt.Errorf("save shalat cache: %w", err)
	}
	return nil
}

func (s *Store) GetShalatCacheStale(ctx context.Context, city string, method int) (*model.ShalatCacheRow, error) {
	var row model.ShalatCacheRow
	err := s.db.QueryRowContext(ctx, `
		SELECT city, prayer_date, method, imsak, fajr, sunrise, dhuhr, asr, maghrib, isha,
		       hijri_date, raw_json, fetched_at, expires_at
		FROM prayer_time_cache
		WHERE city = ? AND method = ?
		ORDER BY fetched_at DESC
		LIMIT 1
	`, city, method).Scan(
		&row.City, &row.PrayerDate, &row.Method,
		&row.Imsak, &row.Fajr, &row.Sunrise, &row.Dhuhr, &row.Asr, &row.Maghrib, &row.Isha,
		&row.HijriDate, &row.RawJSON, &row.FetchedAt, &row.ExpiresAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query stale shalat cache: %w", err)
	}
	return &row, nil
}

func TodayDateWIB() string {
	wib := time.FixedZone("WIB", 7*3600)
	return time.Now().In(wib).Format("2006-01-02")
}
