//go:build ignore

// One-off: fetch city list + coordinates + GMT offset from
// jadwalsholat.org monthly pages (facts only: name, lat/long, timezone).
// Prayer times themselves are computed at runtime via Aladhan lat/long API.
// Run: go run scripts/fetch-prayer-cities.go
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

type PrayerCity struct {
	Name string  `json:"name"`
	Lat  float64 `json:"lat"`
	Lon  float64 `json:"lon"`
	TZ   string  `json:"tz"` // WIB (UTC+7) | WITA (UTC+8) | WIT (UTC+9)
}

var (
	optRe   = regexp.MustCompile(`<option value="(\d+)">([^<]+)</option>`)
	h1Re    = regexp.MustCompile(`Jadwal Sholat untuk (.+?), GMT \+(\d)`)
	coordRe = regexp.MustCompile(`Untuk Kota <b>.+?</b> (\d+)&deg;(\d+)' (LS|LU) (\d+)&deg;(\d+)' BT`)
	client  = &http.Client{Timeout: 20 * time.Second}
)

func get(url string) (string, error) {
	var last error
	for i := 0; i < 3; i++ {
		req, _ := http.NewRequest("GET", url, nil)
		req.Header.Set("User-Agent", "Mozilla/5.0 (islami.click one-off dataset fetch)")
		resp, err := client.Do(req)
		if err != nil {
			last = err
			time.Sleep(time.Second)
			continue
		}
		body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
		resp.Body.Close()
		if err != nil {
			last = err
			time.Sleep(time.Second)
			continue
		}
		if resp.StatusCode != 200 {
			last = fmt.Errorf("status %d", resp.StatusCode)
			time.Sleep(time.Second)
			continue
		}
		return string(body), nil
	}
	return "", last
}

func parseDMS(deg, min string, hemi string, isLat bool) (float64, error) {
	d, err := strconv.ParseFloat(deg, 64)
	if err != nil {
		return 0, err
	}
	m, err := strconv.ParseFloat(min, 64)
	if err != nil {
		return 0, err
	}
	v := d + m/60
	if isLat && hemi == "LS" {
		v = -v
	}
	return v, nil
}

func main() {
	base := "https://jadwalsholat.org/jadwal-sholat/monthly.php"
	html, err := get(base)
	if err != nil {
		fmt.Fprintln(os.Stderr, "fetch city list:", err)
		os.Exit(1)
	}
	opts := optRe.FindAllStringSubmatch(html, -1)
	fmt.Printf("options found: %d\n", len(opts))

	seen := map[string]bool{}
	var cities []PrayerCity
	var skipped []string
	for i, o := range opts {
		id, name := o[1], strings.TrimSpace(o[2])
		key := strings.ToLower(name)
		if seen[key] {
			skipped = append(skipped, name)
			continue
		}
		page, err := get(base + "?id=" + id)
		if err != nil {
			fmt.Fprintf(os.Stderr, "WARN %s (id=%s): %v\n", name, id, err)
			continue
		}
		h1 := h1Re.FindStringSubmatch(page)
		cm := coordRe.FindStringSubmatch(page)
		if h1 == nil || cm == nil {
			fmt.Fprintf(os.Stderr, "WARN %s (id=%s): parse failed (h1=%v coord=%v)\n", name, id, h1 != nil, cm != nil)
			continue
		}
		lat, err1 := parseDMS(cm[1], cm[2], cm[3], true)
		lon, err2 := parseDMS(cm[4], cm[5], "", false)
		if err1 != nil || err2 != nil {
			fmt.Fprintf(os.Stderr, "WARN %s (id=%s): bad coords\n", name, id)
			continue
		}
		var tz string
		switch h1[2] {
		case "7":
			tz = "WIB"
		case "8":
			tz = "WITA"
		case "9":
			tz = "WIT"
		default:
			fmt.Fprintf(os.Stderr, "WARN %s (id=%s): unknown GMT +%s\n", name, id, h1[2])
			continue
		}
		seen[key] = true
		cities = append(cities, PrayerCity{Name: name, Lat: lat, Lon: lon, TZ: tz})
		if (i+1)%50 == 0 {
			fmt.Printf("... %d/%d\n", i+1, len(opts))
		}
		time.Sleep(250 * time.Millisecond)
	}

	sort.Slice(cities, func(a, b int) bool { return cities[a].Name < cities[b].Name })
	out, _ := json.MarshalIndent(cities, "", "  ")
	path := filepath.Join("content", "prayer-cities.json")
	if err := os.WriteFile(path, append(out, '\n'), 0644); err != nil {
		fmt.Fprintln(os.Stderr, "write:", err)
		os.Exit(1)
	}
	fmt.Printf("wrote %s: %d cities, %d dupes skipped\n", path, len(cities), len(skipped))
	for _, s := range skipped {
		fmt.Println("  dupe:", s)
	}
}
