package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/srmdn/islami.click/internal/hijri"
	"github.com/srmdn/islami.click/internal/model"
	"github.com/srmdn/islami.click/internal/store"
)

var hijriMonthsID = [13]string{
	"", "Muharram", "Safar", "Rabiul Awal", "Rabiul Akhir",
	"Jumadil Awal", "Jumadil Akhir", "Rajab", "Sya'ban",
	"Ramadhan", "Syawal", "Dzulqa'dah", "Dzulhijjah",
}

var masehiMonthsID = [13]string{
	"", "Januari", "Februari", "Maret", "April", "Mei", "Juni",
	"Juli", "Agustus", "September", "Oktober", "November", "Desember",
}

var masehiDaysID = [7]string{"Ahad", "Senin", "Selasa", "Rabu", "Kamis", "Jumat", "Sabtu"}

var indonesianCities = []string{
	"Aceh", "Ambon", "Balikpapan", "Banda Aceh", "Bandar Lampung",
	"Bandung", "Banjarmasin", "Batam", "Bekasi", "Bogor",
	"Denpasar", "Depok", "Jakarta", "Jambi", "Jayapura",
	"Kupang", "Makassar", "Malang", "Manado", "Mataram",
	"Medan", "Padang", "Palembang", "Pekanbaru", "Pontianak",
	"Samarinda", "Semarang", "Surabaya", "Surakarta", "Tangerang",
	"Tasikmalaya", "Yogyakarta",
}

var aladhanClient = &http.Client{Timeout: 10 * time.Second}

type aladhanResponse struct {
	Code int `json:"code"`
	Data struct {
		Timings struct {
			Imsak   string `json:"Imsak"`
			Fajr    string `json:"Fajr"`
			Sunrise string `json:"Sunrise"`
			Dhuhr   string `json:"Dhuhr"`
			Asr     string `json:"Asr"`
			Maghrib string `json:"Maghrib"`
			Isha    string `json:"Isha"`
		} `json:"timings"`
		Date struct {
			Hijri struct {
				Day   string `json:"day"`
				Month struct {
					Number int    `json:"number"`
					En     string `json:"en"`
				} `json:"month"`
				Year    string `json:"year"`
				Weekday struct {
					En string `json:"en"`
				} `json:"weekday"`
			} `json:"hijri"`
		} `json:"date"`
	} `json:"data"`
}

type Handler struct {
	tmpls        map[string]*template.Template
	partialTmpls map[string]*template.Template
	contentStore *store.Store
}

func New(tmpls map[string]*template.Template, partialTmpls map[string]*template.Template, contentStore *store.Store) *Handler {
	return &Handler{tmpls: tmpls, partialTmpls: partialTmpls, contentStore: contentStore}
}

func (h *Handler) renderPartial(w http.ResponseWriter, name string, data any) {
	t, ok := h.partialTmpls[name]
	if !ok {
		http.Error(w, "Partial not found", http.StatusNotFound)
		return
	}
	if err := t.ExecuteTemplate(w, name, data); err != nil {
		log.Printf("renderPartial %s: %v", name, err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
	}
}

func (h *Handler) render(w http.ResponseWriter, page string, data any) {
	t, ok := h.tmpls[page]
	if !ok {
		http.Error(w, "Page not found", http.StatusNotFound)
		return
	}
	if err := t.ExecuteTemplate(w, "base", data); err != nil {
		log.Printf("render %s: %v", page, err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
	}
}

func (h *Handler) Home(w http.ResponseWriter, r *http.Request) {
	h.render(w, "home.html", nil)
}

func (h *Handler) AlMatsurat(w http.ResponseWriter, r *http.Request) {
	h.render(w, "almatsurat.html", nil)
}

func (h *Handler) Kiblat(w http.ResponseWriter, r *http.Request) {
	h.render(w, "kiblat.html", nil)
}

func (h *Handler) Hisab(w http.ResponseWriter, r *http.Request) {
	wib := time.FixedZone("WIB", 7*3600)
	now := time.Now().In(wib)
	hijriDate := hijri.FromGregorian(now)

	months := make([]model.HijriMonthEntry, 12)
	for i := 1; i <= 12; i++ {
		days := 30
		if i%2 != 0 {
			days = 29
		}
		months[i-1] = model.HijriMonthEntry{
			Number: i,
			Name:   hijri.MonthNamesID[i],
			Days:   days,
		}
	}

	data := model.HisabPageData{
		HijriToday:     hijriDate.FormatID(),
		MasehiToday:    hijri.FormatGregorianID(now),
		HijriDay:       hijriDate.Day,
		HijriMonth:     hijriDate.Month,
		HijriMonthName: hijri.MonthNamesID[hijriDate.Month],
		HijriYear:      hijriDate.Year,
		Months:         months,
	}

	h.render(w, "hisab.html", data)
}

func (h *Handler) AsmaulHusna(w http.ResponseWriter, r *http.Request) {
	content, err := h.contentStore.AsmaulHusna(r.Context())
	if err != nil {
		log.Printf("asmaul husna: %v", err)
		http.Error(w, "Failed to load content", http.StatusInternalServerError)
		return
	}

	h.render(w, "asmaul-husna.html", content)
}

func (h *Handler) AlMatsuratSugro(w http.ResponseWriter, r *http.Request) {
	content, err := h.contentStore.AlMatsurat(r.Context(), "almatsurat-sugro")
	if err != nil {
		log.Printf("almatsurat sugro: %v", err)
		http.Error(w, "Failed to load content", http.StatusInternalServerError)
		return
	}

	h.render(w, "almatsurat-sugro.html", content)
}

func (h *Handler) AlMatsuratKubro(w http.ResponseWriter, r *http.Request) {
	content, err := h.contentStore.AlMatsurat(r.Context(), "almatsurat-kubro")
	if err != nil {
		log.Printf("almatsurat kubro: %v", err)
		http.Error(w, "Failed to load content", http.StatusInternalServerError)
		return
	}

	h.render(w, "almatsurat-kubro.html", content)
}

const doaPageSize = 20

func (h *Handler) Doa(w http.ResponseWriter, r *http.Request) {
	page, err := h.contentStore.DoaPage(r.Context(), 1, doaPageSize)
	if err != nil {
		log.Printf("doa page: %v", err)
		http.Error(w, "Failed to load content", http.StatusInternalServerError)
		return
	}

	h.render(w, "doa.html", page)
}

func (h *Handler) DoaMore(w http.ResponseWriter, r *http.Request) {
	pageNum := 2
	if p := r.URL.Query().Get("page"); p != "" {
		if n, err := strconv.Atoi(p); err == nil && n > 1 {
			pageNum = n
		}
	}

	data, err := h.contentStore.DoaPage(r.Context(), pageNum, doaPageSize)
	if err != nil {
		log.Printf("doa more page %d: %v", pageNum, err)
		http.Error(w, "Failed to load content", http.StatusInternalServerError)
		return
	}

	h.renderPartial(w, "doa-more", data)
}

func (h *Handler) Shalat(w http.ResponseWriter, r *http.Request) {
	city := strings.TrimSpace(r.URL.Query().Get("city"))
	if city == "" {
		city = "Jakarta"
	}

	page := model.ShalatPageData{
		City:   city,
		Cities: indonesianCities,
	}

	cacheRow, err := h.fetchShalatCache(r.Context(), city)
	if err != nil {
		log.Printf("shalat cache for %s: %v", city, err)
	}

	if cacheRow != nil {
		page.Times = model.PrayerTimes{
			Imsyak:  stripSeconds(cacheRow.Imsak),
			Subuh:   stripSeconds(cacheRow.Fajr),
			Terbit:  stripSeconds(cacheRow.Sunrise),
			Dhuha:   addMinutes(cacheRow.Sunrise, 16),
			Dzuhur:  stripSeconds(cacheRow.Dhuhr),
			Ashr:    stripSeconds(cacheRow.Asr),
			Maghrib: stripSeconds(cacheRow.Maghrib),
			Isya:    stripSeconds(cacheRow.Isha),
		}

		hijriParts := strings.SplitN(cacheRow.HijriDate, "-", 3)
		hijriDay := ""
		hijriYear := ""
		hijriMonthNum := 0
		if len(hijriParts) == 3 {
			hijriYear = hijriParts[0]
			hijriMonthNum, _ = strconv.Atoi(hijriParts[1])
			hijriDay = hijriParts[2]
		}
		monthID := ""
		if hijriMonthNum >= 1 && hijriMonthNum <= 12 {
			monthID = hijriMonthsID[hijriMonthNum]
		}
		page.Hijri = model.HijriDate{
			Day:   hijriDay,
			Month: monthID,
			Year:  hijriYear,
		}

		wib := time.FixedZone("WIB", 7*3600)
		now := time.Now().In(wib)
		page.MasehiDate = fmt.Sprintf("%s, %d %s %d",
			masehiDaysID[now.Weekday()],
			now.Day(),
			masehiMonthsID[now.Month()],
			now.Year(),
		)

		if hijriMonthNum >= 1 && hijriMonthNum <= 12 {
			page.Hijri.Month = hijriMonthsID[hijriMonthNum]
		}

		h.render(w, "shalat.html", page)
		return
	}

	apiURL := fmt.Sprintf(
		"https://api.aladhan.com/v1/timingsByCity?city=%s&country=Indonesia&method=20",
		url.QueryEscape(city),
	)

	resp, err := aladhanClient.Get(apiURL)
	if err != nil {
		stale, staleErr := h.contentStore.GetShalatCacheStale(r.Context(), city, 20)
		if staleErr != nil {
			log.Printf("stalat stale cache for %s: %v", city, staleErr)
		}
		if stale != nil {
			page.Times = model.PrayerTimes{
				Imsyak:  stripSeconds(stale.Imsak),
				Subuh:   stripSeconds(stale.Fajr),
				Terbit:  stripSeconds(stale.Sunrise),
				Dhuha:   addMinutes(stale.Sunrise, 16),
				Dzuhur:  stripSeconds(stale.Dhuhr),
				Ashr:    stripSeconds(stale.Asr),
				Maghrib: stripSeconds(stale.Maghrib),
				Isya:    stripSeconds(stale.Isha),
			}
			page.MasehiDate = fmt.Sprintf("%s, %d %s %d",
				masehiDaysID[time.Now().Weekday()],
				time.Now().Day(),
				masehiMonthsID[time.Now().Month()],
				time.Now().Year(),
			)
			h.render(w, "shalat.html", page)
			return
		}
		page.Error = "Gagal mengambil data jadwal shalat. Coba lagi."
		h.render(w, "shalat.html", page)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		page.Error = "Gagal membaca data jadwal shalat."
		h.render(w, "shalat.html", page)
		return
	}

	var result aladhanResponse
	if err := json.Unmarshal(body, &result); err != nil || result.Code != 200 {
		page.Error = "Kota tidak ditemukan atau data tidak tersedia."
		h.render(w, "shalat.html", page)
		return
	}

	t := result.Data.Timings
	page.Times = model.PrayerTimes{
		Imsyak:  stripSeconds(t.Imsak),
		Subuh:   stripSeconds(t.Fajr),
		Terbit:  stripSeconds(t.Sunrise),
		Dhuha:   addMinutes(t.Sunrise, 16),
		Dzuhur:  stripSeconds(t.Dhuhr),
		Ashr:    stripSeconds(t.Asr),
		Maghrib: stripSeconds(t.Maghrib),
		Isya:    stripSeconds(t.Isha),
	}

	hijri := result.Data.Date.Hijri
	monthID := hijri.Month.En
	if n := hijri.Month.Number; n >= 1 && n <= 12 {
		monthID = hijriMonthsID[n]
	}
	page.Hijri = model.HijriDate{
		Day:     hijri.Day,
		Month:   monthID,
		Year:    hijri.Year,
		Weekday: hijri.Weekday.En,
	}

	now := time.Now()
	page.MasehiDate = fmt.Sprintf("%s, %d %s %d",
		masehiDaysID[now.Weekday()],
		now.Day(),
		masehiMonthsID[now.Month()],
		now.Year(),
	)

	h.saveShalatToCache(r.Context(), city, &result, body)

	h.render(w, "shalat.html", page)
}

func (h *Handler) ShalatMini(w http.ResponseWriter, r *http.Request) {
	h.renderPartial(w, "shalat-mini", h.fetchShalatMini(r.Context()))
}

func (h *Handler) fetchShalatMini(ctx context.Context) model.ShalatMiniData {
	const city = "Jakarta"

	cacheRow, err := h.fetchShalatCache(ctx, city)
	if err != nil {
		log.Printf("shalat mini cache for %s: %v", city, err)
	}

	if cacheRow != nil {
		return miniDataFromCache(cacheRow)
	}

	resp, err := aladhanClient.Get("https://api.aladhan.com/v1/timingsByCity?city=Jakarta&country=Indonesia&method=20")
	if err != nil {
		stale, _ := h.contentStore.GetShalatCacheStale(ctx, city, 20)
		if stale != nil {
			return miniDataFromCache(stale)
		}
		return model.ShalatMiniData{Error: "Gagal memuat waktu shalat."}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return model.ShalatMiniData{Error: "Gagal memuat waktu shalat."}
	}

	var result aladhanResponse
	if err := json.Unmarshal(body, &result); err != nil || result.Code != 200 {
		return model.ShalatMiniData{Error: "Data tidak tersedia."}
	}

	h.saveShalatToCache(ctx, city, &result, body)

	t := result.Data.Timings
	return miniDataFromTimings(stripSeconds(t.Fajr), stripSeconds(t.Dhuhr), stripSeconds(t.Asr), stripSeconds(t.Maghrib), stripSeconds(t.Isha))
}

func (h *Handler) fetchShalatCache(ctx context.Context, city string) (*model.ShalatCacheRow, error) {
	today := store.TodayDateWIB()
	row, err := h.contentStore.GetShalatCache(ctx, city, today, 20)
	if err != nil {
		return nil, fmt.Errorf("get shalat cache: %w", err)
	}
	if row == nil {
		return nil, nil
	}
	expires, err := time.Parse(time.RFC3339, row.ExpiresAt)
	if err != nil {
		return row, nil
	}
	wib := time.FixedZone("WIB", 7*3600)
	if time.Now().In(wib).After(expires) {
		return nil, nil
	}
	return row, nil
}

func (h *Handler) saveShalatToCache(ctx context.Context, city string, result *aladhanResponse, rawBody []byte) {
	wib := time.FixedZone("WIB", 7*3600)
	now := time.Now().In(wib)
	t := result.Data.Timings
	hijri := result.Data.Date.Hijri
	hijriDate := fmt.Sprintf("%s-%02d-%s", hijri.Year, hijri.Month.Number, hijri.Day)

	row := &model.ShalatCacheRow{
		City:       city,
		PrayerDate: now.Format("2006-01-02"),
		Method:     20,
		Imsak:      stripSeconds(t.Imsak),
		Fajr:       stripSeconds(t.Fajr),
		Sunrise:    stripSeconds(t.Sunrise),
		Dhuhr:      stripSeconds(t.Dhuhr),
		Asr:        stripSeconds(t.Asr),
		Maghrib:    stripSeconds(t.Maghrib),
		Isha:       stripSeconds(t.Isha),
		HijriDate:  hijriDate,
		RawJSON:    string(rawBody),
		FetchedAt:  now.Format(time.RFC3339),
		ExpiresAt:  now.AddDate(0, 0, 1).Truncate(24 * time.Hour).Add(time.Hour).Format(time.RFC3339),
	}
	if err := h.contentStore.SaveShalatCache(ctx, row); err != nil {
		log.Printf("save shalat cache for %s: %v", city, err)
	}
}

func miniDataFromCache(row *model.ShalatCacheRow) model.ShalatMiniData {
	return miniDataFromTimings(row.Fajr, row.Dhuhr, row.Asr, row.Maghrib, row.Isha)
}

func miniDataFromTimings(subuh, dzuhur, ashr, maghrib, isya string) model.ShalatMiniData {
	prayers := []struct{ Name, Time string }{
		{"Subuh", subuh},
		{"Dzuhur", dzuhur},
		{"Ashr", ashr},
		{"Maghrib", maghrib},
		{"Isya", isya},
	}

	wib := time.FixedZone("WIB", 7*3600)
	now := time.Now().In(wib)
	nowMins := now.Hour()*60 + now.Minute()

	parseMins := func(s string) int {
		parts := strings.Split(s, ":")
		if len(parts) < 2 {
			return 0
		}
		hr, _ := strconv.Atoi(parts[0])
		mn, _ := strconv.Atoi(parts[1])
		return hr*60 + mn
	}

	nextIdx := -1
	for i, p := range prayers {
		if parseMins(p.Time) > nowMins {
			nextIdx = i
			break
		}
	}

	rows := make([]model.PrayerMiniRow, len(prayers))
	var nextName, nextTime string
	if nextIdx == -1 {
		for i, p := range prayers {
			rows[i] = model.PrayerMiniRow{Name: p.Name, Time: p.Time, IsNext: i == 0, IsPast: i != 0}
		}
		nextName = prayers[0].Name
		nextTime = prayers[0].Time
	} else {
		for i, p := range prayers {
			rows[i] = model.PrayerMiniRow{Name: p.Name, Time: p.Time, IsNext: i == nextIdx, IsPast: i < nextIdx}
		}
		nextName = prayers[nextIdx].Name
		nextTime = prayers[nextIdx].Time
	}

	parts := strings.Split(nextTime, ":")
	nextHr, _ := strconv.Atoi(parts[0])
	nextMn, _ := strconv.Atoi(parts[1])
	nextT := time.Date(now.Year(), now.Month(), now.Day(), nextHr, nextMn, 0, 0, wib)
	if nextIdx == -1 {
		nextT = nextT.Add(24 * time.Hour)
	}

	return model.ShalatMiniData{City: "Jakarta", Prayers: rows, NextPrayerUnix: nextT.Unix(), NextPrayerName: nextName}
}

func stripSeconds(t string) string {
	// API sometimes returns "HH:MM (timezone)" — take first token
	t = strings.Fields(t)[0]
	parts := strings.Split(t, ":")
	if len(parts) >= 2 {
		return parts[0] + ":" + parts[1]
	}
	return t
}

func addMinutes(t string, mins int) string {
	t = strings.Fields(t)[0]
	parts := strings.Split(t, ":")
	if len(parts) < 2 {
		return t
	}
	h, _ := strconv.Atoi(parts[0])
	m, _ := strconv.Atoi(parts[1])
	total := h*60 + m + mins
	return fmt.Sprintf("%02d:%02d", total/60%24, total%60)
}
