package model

import "time"

type PrayerTimes struct {
	Imsyak  string
	Subuh   string
	Terbit  string
	Dhuha   string
	Dzuhur  string
	Ashr    string
	Maghrib string
	Isya    string
}

// PrayerCity is one selectable city with exact coordinates and its
// Indonesian time zone (WIB/WITA/WIT). Coordinates mirror the
// Kemenag-criteria dataset (Subuh 20°, Isya 18°) per city.
type PrayerCity struct {
	Name string  `json:"name"`
	Lat  float64 `json:"lat"`
	Lon  float64 `json:"lon"`
	TZ   string  `json:"tz"`
}

// OffsetSeconds returns the UTC offset for the city's time zone.
func (c PrayerCity) OffsetSeconds() int {
	switch c.TZ {
	case "WITA":
		return 8 * 3600
	case "WIT":
		return 9 * 3600
	default:
		return 7 * 3600
	}
}

// Zone returns the city's fixed time zone.
func (c PrayerCity) Zone() *time.Location {
	return time.FixedZone(c.TZ, c.OffsetSeconds())
}

type HijriDate struct {
	Day     string
	Month   string
	Year    string
	Weekday string
}

type ShalatPageData struct {
	Meta         PageMeta
	City         string
	Cities       []PrayerCity
	TZLabel      string
	TZOffsetMins int
	Lat          float64
	Lon          float64
	Times        PrayerTimes
	Hijri        HijriDate
	MasehiDate   string
	Error        string
}

type PrayerMiniRow struct {
	Name   string
	Time   string
	IsNext bool
	IsPast bool
}

type ShalatMiniData struct {
	City           string
	TZLabel        string
	TZOffsetMins   int
	Prayers        []PrayerMiniRow
	NextPrayerUnix int64
	NextPrayerName string
	NextPrayerTime string
	Error          string
}

type ShalatCacheRow struct {
	City       string
	PrayerDate string
	Method     int
	Imsak      string
	Fajr       string
	Sunrise    string
	Dhuhr      string
	Asr        string
	Maghrib    string
	Isha       string
	HijriDate  string
	RawJSON    string
	FetchedAt  string
	ExpiresAt  string
}
