package handler

import (
	"embed"
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

	"github.com/srmdn/islami.click/internal/model"
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

type Handler struct {
	tmpls        map[string]*template.Template
	partialTmpls map[string]*template.Template
	contentFS    embed.FS
}

func New(tmpls map[string]*template.Template, partialTmpls map[string]*template.Template, contentFS embed.FS) *Handler {
	return &Handler{tmpls: tmpls, partialTmpls: partialTmpls, contentFS: contentFS}
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

func (h *Handler) AlMatsuratSugro(w http.ResponseWriter, r *http.Request) {
	data, err := h.contentFS.ReadFile("content/almatsurat-sugro.json")
	if err != nil {
		http.Error(w, "Failed to load content", http.StatusInternalServerError)
		return
	}

	var content struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		Sections    []struct {
			ID          string `json:"id"`
			Type        string `json:"type"`
			Title       string `json:"title"`
			Arabic      string `json:"arabic"`
			Translation string `json:"translation"`
			Repeat      int    `json:"repeat"`
			Source      string `json:"source"`
		} `json:"sections"`
	}

	if err := json.Unmarshal(data, &content); err != nil {
		http.Error(w, "Failed to parse content", http.StatusInternalServerError)
		return
	}

	h.render(w, "almatsurat-sugro.html", content)
}

func (h *Handler) AlMatsuratKubro(w http.ResponseWriter, r *http.Request) {
	data, err := h.contentFS.ReadFile("content/almatsurat-kubro.json")
	if err != nil {
		http.Error(w, "Failed to load content", http.StatusInternalServerError)
		return
	}

	var content struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		Sections    []struct {
			ID          string `json:"id"`
			Type        string `json:"type"`
			Title       string `json:"title"`
			Arabic      string `json:"arabic"`
			Translation string `json:"translation"`
			Repeat      int    `json:"repeat"`
			Source      string `json:"source"`
		} `json:"sections"`
	}

	if err := json.Unmarshal(data, &content); err != nil {
		http.Error(w, "Failed to parse content", http.StatusInternalServerError)
		return
	}

	h.render(w, "almatsurat-kubro.html", content)
}

func (h *Handler) Doa(w http.ResponseWriter, r *http.Request) {
	data, err := h.contentFS.ReadFile("content/doa-harian.json")
	if err != nil {
		http.Error(w, "Failed to load content", http.StatusInternalServerError)
		return
	}

	var page model.DoaPageData
	if err := json.Unmarshal(data, &page); err != nil {
		http.Error(w, "Failed to parse content", http.StatusInternalServerError)
		return
	}

	h.render(w, "doa.html", page)
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

	apiURL := fmt.Sprintf(
		"https://api.aladhan.com/v1/timingsByCity?city=%s&country=Indonesia&method=20",
		url.QueryEscape(city),
	)

	resp, err := aladhanClient.Get(apiURL)
	if err != nil {
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

	var result struct {
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

	h.render(w, "shalat.html", page)
}

func (h *Handler) ShalatMini(w http.ResponseWriter, r *http.Request) {
	h.renderPartial(w, "shalat-mini", h.fetchShalatMini())
}

func (h *Handler) fetchShalatMini() model.ShalatMiniData {
	resp, err := aladhanClient.Get("https://api.aladhan.com/v1/timingsByCity?city=Jakarta&country=Indonesia&method=20")
	if err != nil {
		return model.ShalatMiniData{Error: "Gagal memuat waktu shalat."}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return model.ShalatMiniData{Error: "Gagal memuat waktu shalat."}
	}

	var result struct {
		Code int `json:"code"`
		Data struct {
			Timings struct {
				Fajr    string `json:"Fajr"`
				Dhuhr   string `json:"Dhuhr"`
				Asr     string `json:"Asr"`
				Maghrib string `json:"Maghrib"`
				Isha    string `json:"Isha"`
			} `json:"timings"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &result); err != nil || result.Code != 200 {
		return model.ShalatMiniData{Error: "Data tidak tersedia."}
	}

	t := result.Data.Timings
	prayers := []struct{ Name, Time string }{
		{"Subuh", stripSeconds(t.Fajr)},
		{"Dzuhur", stripSeconds(t.Dhuhr)},
		{"Ashr", stripSeconds(t.Asr)},
		{"Maghrib", stripSeconds(t.Maghrib)},
		{"Isya", stripSeconds(t.Isha)},
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
		// All prayers passed — Subuh is next (tomorrow)
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

	// Compute Unix timestamp for next prayer
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
