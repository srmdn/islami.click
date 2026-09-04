package model

import "html/template"

type HisabPageData struct {
	Meta             PageMeta
	HijriToday       string
	MasehiToday      string
	HijriDay         int
	HijriMonth       int
	HijriMonthName   string
	HijriYear        int
	HijriSourceLabel string
	HijriAnchorsJS   template.JS
	Months           []HijriMonthEntry
}

type HijriMonthEntry struct {
	Number  int
	Name    string
	Days    int
	IsHaram bool
}