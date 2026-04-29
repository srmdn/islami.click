package model

type HisabPageData struct {
	Meta           PageMeta
	HijriToday     string
	MasehiToday    string
	HijriDay       int
	HijriMonth     int
	HijriMonthName string
	HijriYear      int
	Months         []HijriMonthEntry
}

type HijriMonthEntry struct {
	Number  int
	Name    string
	Days    int
	IsHaram bool
}