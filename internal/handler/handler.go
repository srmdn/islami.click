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
	tmpls     map[string]*template.Template
	contentFS embed.FS
}

func New(tmpls map[string]*template.Template, contentFS embed.FS) *Handler {
	return &Handler{tmpls: tmpls, contentFS: contentFS}
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
	h.render(w, "doa.html", nil)
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
					Day     string `json:"day"`
					Month   struct {
						En string `json:"en"`
						Ar string `json:"ar"`
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
	page.Hijri = model.HijriDate{
		Day:     hijri.Day,
		Month:   hijri.Month.En,
		MonthAr: hijri.Month.Ar,
		Year:    hijri.Year,
		Weekday: hijri.Weekday.En,
	}

	h.render(w, "shalat.html", page)
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
