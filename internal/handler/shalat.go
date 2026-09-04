package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	islamiclick "github.com/srmdn/islami.click"
	"github.com/srmdn/islami.click/internal/model"
)

// aladhanMethodKemenag is the Aladhan calculation method matching the
// Kemenag Indonesia criteria (Subuh 20°, Isya 18°, Ashar Syafi'i).
const aladhanMethodKemenag = 20

var (
	prayerCitiesOnce   sync.Once
	prayerCities       []model.PrayerCity
	prayerCityByName   map[string]model.PrayerCity
	prayerCitiesLoaded bool
)

// getPrayerCities loads the embedded city dataset (name + exact
// coordinates + WIB/WITA/WIT zone) once. Coordinates mirror the
// Kemenag-criteria per-city dataset; times are computed at runtime.
func getPrayerCities() ([]model.PrayerCity, map[string]model.PrayerCity) {
	prayerCitiesOnce.Do(func() {
		data, err := islamiclick.ContentFS.ReadFile("content/prayer-cities.json")
		if err != nil {
			log.Printf("prayer cities: %v", err)
			return
		}
		var cities []model.PrayerCity
		if err := json.Unmarshal(data, &cities); err != nil {
			log.Printf("prayer cities parse: %v", err)
			return
		}
		prayerCities = cities
		prayerCityByName = make(map[string]model.PrayerCity, len(cities))
		for _, c := range cities {
			if c.Name == "" || c.TZ == "" {
				continue
			}
			prayerCityByName[c.Name] = c
		}
		if _, ok := prayerCityByName["Jakarta"]; ok {
			prayerCitiesLoaded = true
		}
	})
	if !prayerCitiesLoaded {
		fallback := model.PrayerCity{Name: "Jakarta", Lat: -6.1667, Lon: 106.8167, TZ: "WIB"}
		return []model.PrayerCity{fallback}, map[string]model.PrayerCity{"Jakarta": fallback}
	}
	return prayerCities, prayerCityByName
}

// lookupPrayerCity resolves a city name, defaulting to Jakarta.
func lookupPrayerCity(byName map[string]model.PrayerCity, name string) model.PrayerCity {
	if c, ok := byName[name]; ok {
		return c
	}
	return byName["Jakarta"]
}

// prayerTimesFromRaw applies the Kemenag-style safety margin on top of
// raw Aladhan values (all "HH:MM", seconds already truncated upstream):
//
//   - +2 min (ihtiyati) on start times: Subuh, Dzuhur, Ashar, Maghrib, Isya
//   - -2 min on Terbit (end of the Subuh window)
//   - Imsak locked to Subuh-10
//   - Dhuha = Terbit+24 (≈ sun altitude 3,5°)
func prayerTimesFromRaw(imsak, fajr, sunrise, dhuhr, asr, maghrib, isha string) model.PrayerTimes {
	_ = imsak // derived from Subuh to keep the pair consistent
	subuh := addMinutes(stripSeconds(fajr), 2)
	terbit := addMinutes(stripSeconds(sunrise), -2)
	return model.PrayerTimes{
		Imsyak:  addMinutes(subuh, -10),
		Subuh:   subuh,
		Terbit:  terbit,
		Dhuha:   addMinutes(terbit, 24),
		Dzuhur:  addMinutes(stripSeconds(dhuhr), 2),
		Ashr:    addMinutes(stripSeconds(asr), 2),
		Maghrib: addMinutes(stripSeconds(maghrib), 2),
		Isya:    addMinutes(stripSeconds(isha), 2),
	}
}

// prayerTimesFromRow applies the safety margin to a stored cache row.
func prayerTimesFromRow(row *model.ShalatCacheRow) model.PrayerTimes {
	return prayerTimesFromRaw(row.Imsak, row.Fajr, row.Sunrise, row.Dhuhr, row.Asr, row.Maghrib, row.Isha)
}

// hijriFromStored parses the stored "YYYY-MM-NAME-or-DD" hijri date.
func hijriFromStored(stored string) model.HijriDate {
	var out model.HijriDate
	parts := splitHijri(stored)
	if len(parts) != 3 {
		return out
	}
	out.Year = parts[0]
	if n := atoiHijri(parts[1]); n >= 1 && n <= 12 {
		out.Month = hijriMonthsID[n]
	}
	out.Day = parts[2]
	return out
}

func splitHijri(s string) []string { return strings.SplitN(s, "-", 3) }

func atoiHijri(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}

func fetchTimingsByCoords(ctx context.Context, city model.PrayerCity, date time.Time) (*aladhanResponse, []byte, error) {
	apiURL := fmt.Sprintf(
		"https://api.aladhan.com/v1/timings/%s?latitude=%f&longitude=%f&method=%d",
		date.Format("02-01-2006"), city.Lat, city.Lon, aladhanMethodKemenag,
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, nil, err
	}
	resp, err := aladhanClient.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if err != nil {
		return nil, nil, err
	}
	var result aladhanResponse
	if err := json.Unmarshal(body, &result); err != nil || result.Code != 200 {
		return nil, nil, fmt.Errorf("aladhan: code=%d", result.Code)
	}
	return &result, body, nil
}
