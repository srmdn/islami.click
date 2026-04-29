package model

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

type HijriDate struct {
	Day     string
	Month   string
	Year    string
	Weekday string
}

type ShalatPageData struct {
	Meta       PageMeta
	City       string
	Cities     []string
	Times      PrayerTimes
	Hijri      HijriDate
	MasehiDate string
	Error      string
}

type PrayerMiniRow struct {
	Name   string
	Time   string
	IsNext bool
	IsPast bool
}

type ShalatMiniData struct {
	City           string
	Prayers        []PrayerMiniRow
	NextPrayerUnix int64
	NextPrayerName string
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
