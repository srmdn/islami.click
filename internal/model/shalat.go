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
	MonthAr string
	Year    string
	Weekday string
}

type ShalatPageData struct {
	City   string
	Cities []string
	Times  PrayerTimes
	Hijri  HijriDate
	Error  string
}
