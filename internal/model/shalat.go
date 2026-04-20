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
	City        string
	Cities      []string
	Times       PrayerTimes
	Hijri       HijriDate
	MasehiDate  string
	Error       string
}

type PrayerMiniRow struct {
	Name   string
	Time   string
	IsNext bool
	IsPast bool
}

type ShalatMiniData struct {
	City    string
	Prayers []PrayerMiniRow
	Error   string
}
