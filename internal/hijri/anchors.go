package hijri

import (
	"encoding/json"
	"html/template"
	"log"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	islamiclick "github.com/srmdn/islami.click"
)

// hijriAnchor is one official month start from Kalender Hijriah Indonesia
// (Ditjen Bimas Islam, Kemenag RI): the Gregorian date of Hijri day 1.
// Anchors are facts scraped once from the public SIHAT calendar bundle —
// refresh yearly. Dates outside coverage fall back to arithmetic
// (tabular) calculation and must be labeled "perkiraan".
type hijriAnchor struct {
	Year  int
	Month int
	Start time.Time // UTC midnight
}

var (
	anchorOnce sync.Once
	anchors    []hijriAnchor
)

func loadAnchors() []hijriAnchor {
	anchorOnce.Do(func() {
		data, err := islamiclick.ContentFS.ReadFile("content/hijri-anchors.json")
		if err != nil {
			log.Printf("hijri anchors: %v", err)
			return
		}
		var raw map[string]string
		if err := json.Unmarshal(data, &raw); err != nil {
			log.Printf("hijri anchors parse: %v", err)
			return
		}
		for k, v := range raw {
			ym := strings.SplitN(k, "-", 2)
			if len(ym) != 2 {
				continue
			}
			y, err1 := strconv.Atoi(ym[0])
			m, err2 := strconv.Atoi(ym[1])
			start, err3 := time.Parse("2006-01-02", v)
			if err1 != nil || err2 != nil || err3 != nil || m < 1 || m > 12 {
				continue
			}
			anchors = append(anchors, hijriAnchor{Year: y, Month: m, Start: start})
		}
		sort.Slice(anchors, func(i, j int) bool { return anchors[i].Start.Before(anchors[j].Start) })
	})
	return anchors
}

// fromAnchors resolves a Gregorian date via official month starts.
// The last anchored month has no known end, so dates at/after the
// following unknown boundary are left to the arithmetic fallback.
func fromAnchors(t time.Time) (Date, bool) {
	a := loadAnchors()
	if len(a) < 2 {
		return Date{}, false
	}
	g := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
	for i := len(a) - 1; i >= 0; i-- {
		if g.Before(a[i].Start) {
			continue
		}
		if i+1 >= len(a) {
			return Date{}, false
		}
		day := int(g.Sub(a[i].Start).Hours()/24) + 1
		return Date{Year: a[i].Year, Month: a[i].Month, Day: day}, true
	}
	return Date{}, false
}

func anchorStart(year, month int) (time.Time, bool) {
	for _, a := range loadAnchors() {
		if a.Year == year && a.Month == month {
			return a.Start, true
		}
	}
	return time.Time{}, false
}

// Source reports whether a Gregorian date falls inside official
// Kemenag coverage ("kemenag") or uses the arithmetic estimate
// ("perkiraan").
func Source(t time.Time) string {
	if _, ok := fromAnchors(t); ok {
		return "kemenag"
	}
	return "perkiraan"
}

// isLeapTabular reports 30-year-cycle leap years for the arithmetic fallback.
func isLeapTabular(y int) bool { return (11*y+14)%30 < 11 }

// AnchorsJS exposes the anchor table for client-side converters
// (e.g. the /hisab Alpine widget) so server and browser agree.
func AnchorsJS() template.JS {
	type jsAnchor struct {
		Y     int    `json:"y"`
		M     int    `json:"m"`
		Start string `json:"start"`
	}
	a := loadAnchors()
	out := make([]jsAnchor, 0, len(a))
	for _, an := range a {
		out = append(out, jsAnchor{Y: an.Year, M: an.Month, Start: an.Start.Format("2006-01-02")})
	}
	b, err := json.Marshal(out)
	if err != nil {
		return template.JS("[]")
	}
	return template.JS(b)
}

// MonthLength returns the real month length when covered by official
// anchors, else the tabular value (odd months 30 days, even 29,
// Dzulhijjah 30 in leap years).
func MonthLength(year, month int) int {
	nm, ny := month+1, year
	if nm > 12 {
		nm, ny = 1, year+1
	}
	if s, ok := anchorStart(year, month); ok {
		if n, ok := anchorStart(ny, nm); ok {
			return int(n.Sub(s).Hours() / 24)
		}
	}
	if month == 12 {
		if isLeapTabular(year) {
			return 30
		}
		return 29
	}
	if month%2 == 1 {
		return 30
	}
	return 29
}
