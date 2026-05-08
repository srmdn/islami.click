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
	"sync"
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

const siteURL = "https://islami.click"
const defaultOGImage = "https://islami.click/static/images/og-image.png"

func pageMeta(r *http.Request, title, description string) model.PageMeta {
	return model.PageMeta{
		CanonicalURL:  siteURL + r.URL.Path,
		OGTitle:       title + " | islami.click",
		OGDescription: description,
		OGImage:       defaultOGImage,
	}
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
	h.render(w, "home.html", struct{ Meta model.PageMeta }{
		Meta: pageMeta(r, "Beranda", "Portal islami lengkap: Al-Matsurat, jadwal shalat, Al-Quran, doa harian, Asmaul Husna, dan kiblat untuk Muslim Indonesia."),
	})
}

func (h *Handler) AlMatsurat(w http.ResponseWriter, r *http.Request) {
	h.render(w, "almatsurat.html", struct{ Meta model.PageMeta }{
		Meta: pageMeta(r, "Al-Matsurat", "Kumpulan dzikir pagi dan petang dari Al-Matsurat Sugro dan Kubro, panduan wirid harian Muslim."),
	})
}

func (h *Handler) Kiblat(w http.ResponseWriter, r *http.Request) {
	h.render(w, "kiblat.html", struct{ Meta model.PageMeta }{
		Meta: pageMeta(r, "Arah Kiblat", "Cek arah kiblat dari lokasi Anda secara akurat menggunakan kompas berbasis GPS."),
	})
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
		isHaram := i == 1 || i == 7 || i == 11 || i == 12
		months[i-1] = model.HijriMonthEntry{
			Number:  i,
			Name:    hijri.MonthNamesID[i],
			Days:    days,
			IsHaram: isHaram,
		}
	}

	data := model.HisabPageData{
		Meta:           pageMeta(r, "Hisab Kalender Hijriyah", "Konversi tanggal Hijriyah dan Masehi, kalender bulan Hijriyah lengkap."),
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
	content.Meta = pageMeta(r, "Asmaul Husna", "99 Asmaul Husna lengkap dengan teks Arab, transliterasi Latin, dan terjemahan makna dalam bahasa Indonesia.")
	h.render(w, "asmaul-husna.html", content)
}

func (h *Handler) Quran(w http.ResponseWriter, r *http.Request) {
	surahs, err := h.contentStore.QuranSurahs(r.Context())
	if err != nil {
		log.Printf("quran surahs: %v", err)
		http.Error(w, "Failed to load content", http.StatusInternalServerError)
		return
	}

	data := model.QuranPageData{
		Meta:        pageMeta(r, "Al-Quran", "Baca Al-Quran 30 juz lengkap dengan teks Arab, terjemahan Indonesia, dan audio murottal per surah."),
		Title:       "Al-Qur'an",
		Description: "Baca Al-Qur'an lengkap dengan terjemahan Bahasa Indonesia",
		Surahs:      surahs,
	}

	h.render(w, "quran.html", data)
}

func audioURLForSurah(surahNumber int) string {
	return fmt.Sprintf("https://download.quranicaudio.com/quran/mishaari_raashid_al_3afaasee/%03d.mp3", surahNumber)
}

func (h *Handler) QuranSurah(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/quran/")
	path = strings.TrimSpace(path)
	if path == "" {
		http.Redirect(w, r, "/quran", http.StatusSeeOther)
		return
	}

	surahNumber, err := strconv.Atoi(path)
	if err != nil || surahNumber < 1 || surahNumber > 114 {
		http.Error(w, "Surah tidak ditemukan", http.StatusNotFound)
		return
	}

	surah, err := h.contentStore.QuranSurah(r.Context(), surahNumber)
	if err != nil {
		log.Printf("quran surah %d: %v", surahNumber, err)
		http.Error(w, "Failed to load content", http.StatusInternalServerError)
		return
	}

	pages, err := h.contentStore.MushafPagesForSurah(r.Context(), surahNumber)
	if err != nil {
		log.Printf("mushaf pages for surah %d: %v", surahNumber, err)
		http.Error(w, "Failed to load content", http.StatusInternalServerError)
		return
	}
	if len(pages) == 0 {
		http.Error(w, "Surah tidak ditemukan", http.StatusNotFound)
		return
	}

	firstPage := pages[0]
	lastPage := pages[len(pages)-1]

	pageNum := firstPage
	if p := r.URL.Query().Get("page"); p != "" {
		if n, err := strconv.Atoi(p); err == nil {
			if n >= firstPage && n <= lastPage {
				pageNum = n
			}
		}
	}

	ayahs, err := h.contentStore.QuranAyahsByMushafPage(r.Context(), surahNumber, pageNum)
	if err != nil {
		log.Printf("quran ayahs %d page %d: %v", surahNumber, pageNum, err)
		http.Error(w, "Failed to load content", http.StatusInternalServerError)
		return
	}

	data := model.SurahReaderData{
		Meta:        pageMeta(r, fmt.Sprintf("Surah %s", surah.Name), fmt.Sprintf("Baca Surah %s lengkap dengan teks Arab, terjemahan Indonesia, dan audio murottal.", surah.Name)),
		Title:       fmt.Sprintf("%s - Al-Qur'an", surah.Name),
		Description: fmt.Sprintf("Surah %s (%s) - %d Ayat", surah.Name, surah.ArabicName, surah.AyahCount),
		Surah:       surah,
		Ayahs:       ayahs,
		Page:        pageNum,
		PageSize:    0,
		TotalPages:  lastPage,
		AudioURL:    audioURLForSurah(surahNumber),
	}

	if surahNumber > 1 {
		prev, err := h.contentStore.GetQuranSurahByNumber(r.Context(), surahNumber-1)
		if err == nil && prev != nil {
			data.PrevSurah = prev
		}
	}
	if surahNumber < 114 {
		next, err := h.contentStore.GetQuranSurahByNumber(r.Context(), surahNumber+1)
		if err == nil && next != nil {
			data.NextSurah = next
		}
	}

	if r.Header.Get("HX-Request") == "true" {
		h.renderPartial(w, "quran-ayahs", data)
		return
	}

	h.render(w, "quran-surah.html", data)
}

func (h *Handler) QuranSearch(w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(r.URL.Query().Get("q"))

	data := model.QuranSearchData{
		Meta:        pageMeta(r, "Pencarian Al-Qur'an", "Cari ayat dalam Al-Qur'an"),
		Title:       "Pencarian Al-Qur'an",
		Description: "Cari ayat dalam Al-Qur'an",
		Query:       query,
	}

	if query != "" {
		results, err := h.smartQuranSearch(r.Context(), query)
		if err != nil {
			log.Printf("quran search %q: %v", query, err)
			http.Error(w, "Gagal mencari", http.StatusInternalServerError)
			return
		}
		data.Results = results
		data.ResultCount = len(results)
	}

	h.render(w, "quran-search.html", data)
}

func (h *Handler) smartQuranSearch(ctx context.Context, query string) ([]model.QuranSearchResult, error) {
	seen := make(map[string]bool)
	var results []model.QuranSearchResult

	add := func(r model.QuranSearchResult) {
		key := fmt.Sprintf("%d:%d", r.SurahNumber, r.AyahNumber)
		if !seen[key] {
			seen[key] = true
			results = append(results, r)
		}
	}

	surahNum, ayahNum, cleanedQuery, err := h.extractQuranReference(ctx, query)
	if err != nil {
		return nil, err
	}

	if surahNum > 0 && ayahNum > 0 {
		r, err := h.contentStore.GetQuranAyah(ctx, surahNum, ayahNum)
		if err != nil {
			return nil, err
		}
		if r != nil {
			add(*r)
		}
	}

	if surahNum > 0 && ayahNum == 0 {
		ayahs, err := h.contentStore.QuranAyahs(ctx, surahNum)
		if err != nil {
			return nil, err
		}
		for _, a := range ayahs {
			if a.Number > 3 {
				break
			}
			add(model.QuranSearchResult{
				SurahNumber: surahNum,
				AyahNumber:  a.Number,
				Arabic:      a.Arabic,
				Translation: a.Translation,
			})
		}
	}

	if strings.TrimSpace(cleanedQuery) != "" {
		contentResults, err := h.contentStore.SearchQuran(ctx, cleanedQuery, 50)
		if err != nil {
			return nil, err
		}
		for _, cr := range contentResults {
			add(cr)
		}
	}

	surahResults, err := h.contentStore.SearchQuranSurahs(ctx, query)
	if err != nil {
		return nil, err
	}
	for _, sr := range surahResults {
		ayahs, err := h.contentStore.QuranAyahs(ctx, sr.SurahNumber)
		if err != nil {
			return nil, err
		}
		if len(ayahs) > 0 {
			add(model.QuranSearchResult{
				SurahNumber: sr.SurahNumber,
				SurahName:   sr.SurahName,
				AyahNumber:  ayahs[0].Number,
				Arabic:      ayahs[0].Arabic,
				Translation: ayahs[0].Translation,
			})
		}
	}

	return results, nil
}

func stripNoiseWords(s string) string {
	noise := []string{"qs.", "qs", "surah", "surat", "ayat", "ayah"}
	words := strings.Fields(s)
	result := make([]string, 0, len(words))
	for _, w := range words {
		isNoise := false
		for _, n := range noise {
			if w == n {
				isNoise = true
				break
			}
		}
		if !isNoise {
			result = append(result, w)
		}
	}
	return strings.Join(result, " ")
}

func (h *Handler) extractQuranReference(ctx context.Context, query string) (surah int, ayah int, cleaned string, err error) {
	normalized := strings.ToLower(strings.TrimSpace(query))

	words := strings.Fields(normalized)
	for i, w := range words {
		for _, sep := range []string{":", "/"} {
			parts := strings.Split(w, sep)
			if len(parts) == 2 {
				if s, err1 := strconv.Atoi(parts[0]); err1 == nil && s >= 1 && s <= 114 {
					if a, err2 := strconv.Atoi(parts[1]); err2 == nil && a >= 1 {
						cleanedWords := make([]string, 0, len(words)-1)
						cleanedWords = append(cleanedWords, words[:i]...)
						cleanedWords = append(cleanedWords, words[i+1:]...)
						cleaned = stripNoiseWords(strings.Join(cleanedWords, " "))
						return s, a, cleaned, nil
					}
				}
			}
		}
	}

	punctNormalized := normalized
	for _, p := range []string{":", "/", ".", ",", ";", "!", "?", "'", "\"", "(", ")", "[", "]"} {
		punctNormalized = strings.ReplaceAll(punctNormalized, p, " ")
	}

	stripped := stripNoiseWords(punctNormalized)
	strippedWords := strings.Fields(stripped)

	surahNum := 0
	surahStart, surahEnd := -1, -1

	for i := 0; i < len(strippedWords)-1; i++ {
		candidate := strippedWords[i] + " " + strippedWords[i+1]
		num, err := h.findSurahNumber(ctx, candidate)
		if err != nil {
			return 0, 0, "", err
		}
		if num > 0 {
			surahNum = num
			surahStart, surahEnd = i, i+2
			break
		}
	}

	if surahNum == 0 {
		for i, w := range strippedWords {
			if _, err := strconv.Atoi(w); err == nil {
				continue
			}
			num, err := h.findSurahNumber(ctx, w)
			if err != nil {
				return 0, 0, "", err
			}
			if num > 0 {
				surahNum = num
				surahStart, surahEnd = i, i+1
				break
			}
		}
	}

	if surahNum == 0 {
		return 0, 0, stripped, nil
	}

	remaining := make([]string, 0, len(strippedWords))
	for i := 0; i < len(strippedWords); i++ {
		if i < surahStart || i >= surahEnd {
			remaining = append(remaining, strippedWords[i])
		}
	}

	for i := 0; i < len(remaining); i++ {
		if a, err := strconv.Atoi(remaining[i]); err == nil && a >= 1 {
			final := make([]string, 0, len(remaining)-1)
			final = append(final, remaining[:i]...)
			final = append(final, remaining[i+1:]...)
			cleaned = strings.Join(final, " ")
			return surahNum, a, cleaned, nil
		}
	}

	cleaned = strings.Join(remaining, " ")
	return surahNum, 0, cleaned, nil
}

var (
	surahNameCache     map[string]int
	surahNameCacheOnce sync.Once
	surahNameCacheErr  error
)

func (h *Handler) loadSurahNameCache(ctx context.Context) (map[string]int, error) {
	surahNameCacheOnce.Do(func() {
		surahs, err := h.contentStore.QuranSurahs(ctx)
		if err != nil {
			surahNameCacheErr = err
			return
		}
		surahNameCache = make(map[string]int)
		for _, s := range surahs {
			base := strings.ToLower(s.Name)
			variations := []string{
				base,
				strings.ReplaceAll(base, "-", " "),
				strings.ReplaceAll(base, "-", ""),
				strings.ReplaceAll(base, " ", "-"),
				strings.ReplaceAll(base, " ", ""),
				strings.ReplaceAll(base, "'", ""),
				strings.ReplaceAll(strings.ReplaceAll(base, "-", ""), "'", ""),
			}
			for _, v := range variations {
				surahNameCache[v] = s.Number
			}
		}
	})
	if surahNameCacheErr != nil {
		return nil, surahNameCacheErr
	}
	return surahNameCache, nil
}

func (h *Handler) findSurahNumber(ctx context.Context, name string) (int, error) {
	results, err := h.contentStore.SearchQuranSurahs(ctx, name)
	if err != nil {
		return 0, err
	}
	if len(results) > 0 {
		return results[0].SurahNumber, nil
	}

	normalizedName := strings.ReplaceAll(name, "-", " ")
	normalizedName = strings.ReplaceAll(normalizedName, "'", "")
	normalizedName = strings.ReplaceAll(normalizedName, "`", "")
	results, err = h.contentStore.SearchQuranSurahs(ctx, normalizedName)
	if err != nil {
		return 0, err
	}
	if len(results) > 0 {
		return results[0].SurahNumber, nil
	}

	cache, err := h.loadSurahNameCache(ctx)
	if err != nil {
		return 0, err
	}

	keys := []string{
		strings.ToLower(name),
		strings.ToLower(strings.ReplaceAll(name, "-", "")),
		strings.ToLower(strings.ReplaceAll(name, " ", "")),
		strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(name, "-", ""), " ", "")),
	}
	for _, k := range keys {
		if n, ok := cache[k]; ok {
			return n, nil
		}
	}
	return 0, nil
}

func (h *Handler) AlMatsuratSugro(w http.ResponseWriter, r *http.Request) {
	content, err := h.contentStore.AlMatsurat(r.Context(), "almatsurat-sugro")
	if err != nil {
		log.Printf("almatsurat sugro: %v", err)
		http.Error(w, "Failed to load content", http.StatusInternalServerError)
		return
	}
	content.Meta = pageMeta(r, "Al-Matsurat Sugro", "Dzikir pagi dan petang Al-Matsurat Sugro lengkap dengan teks Arab, transliterasi, dan terjemahan.")
	h.render(w, "almatsurat-sugro.html", content)
}

func (h *Handler) AlMatsuratKubro(w http.ResponseWriter, r *http.Request) {
	content, err := h.contentStore.AlMatsurat(r.Context(), "almatsurat-kubro")
	if err != nil {
		log.Printf("almatsurat kubro: %v", err)
		http.Error(w, "Failed to load content", http.StatusInternalServerError)
		return
	}
	content.Meta = pageMeta(r, "Al-Matsurat Kubro", "Dzikir pagi dan petang Al-Matsurat Kubro lengkap dengan teks Arab, transliterasi, dan terjemahan.")
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
	page.Meta = pageMeta(r, "Doa Harian", "Kumpulan doa harian Islam lengkap: doa makan, tidur, bepergian, masuk rumah, dan ratusan doa lainnya.")
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
		Meta:   pageMeta(r, "Jadwal Shalat", "Jadwal shalat harian akurat berdasarkan lokasi kota di Indonesia, dilengkapi waktu imsak dan terbit."),
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

func (h *Handler) RobotsTxt(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	fmt.Fprintf(w, "User-agent: *\nAllow: /\n\nSitemap: https://islami.click/sitemap.xml\n")
}

func (h *Handler) Sitemap(w http.ResponseWriter, r *http.Request) {
	staticURLs := []string{
		"/",
		"/almatsurat",
		"/almatsurat/sugro",
		"/almatsurat/kubro",
		"/doa",
		"/shalat",
		"/asmaul-husna",
		"/kiblat",
		"/hisab",
		"/quran",
		"/quiz",
	}

	surahs, err := h.contentStore.QuranSurahs(r.Context())
	if err != nil {
		log.Printf("sitemap: quran surahs: %v", err)
		http.Error(w, "failed to generate sitemap", http.StatusInternalServerError)
		return
	}

	quizCats, err := h.contentStore.QuizCategories(r.Context())
	if err != nil {
		log.Printf("sitemap: quiz categories: %v", err)
		http.Error(w, "failed to generate sitemap", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	fmt.Fprint(w, `<?xml version="1.0" encoding="UTF-8"?>`)
	fmt.Fprint(w, "\n<urlset xmlns=\"http://www.sitemaps.org/schemas/sitemap/0.9\">\n")

	for _, path := range staticURLs {
		fmt.Fprintf(w, "  <url><loc>%s%s</loc><changefreq>weekly</changefreq><priority>0.8</priority></url>\n",
			siteURL, path)
	}

	for _, s := range surahs {
		fmt.Fprintf(w, "  <url><loc>%s/quran/%d</loc><changefreq>monthly</changefreq><priority>0.6</priority></url>\n",
			siteURL, s.Number)
	}

	for _, c := range quizCats {
		fmt.Fprintf(w, "  <url><loc>%s/quiz/%s</loc><changefreq>monthly</changefreq><priority>0.6</priority></url>\n",
			siteURL, c.Slug)
	}

	fmt.Fprint(w, "</urlset>\n")
}

func (h *Handler) LLMsTxt(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	content := `# islami.click
> Portal islami lengkap untuk Muslim Indonesia.

Author: srmdn
Language: id
Topics: Al-Quran, dzikir, doa Islam, jadwal shalat, Asmaul Husna, kiblat, kalender Hijriyah, quiz Islam
Content-Type: Islamic reference tools and content
Update-Cadence: irregular

## Key Pages
- /: Beranda - portal islami utama
- /quran: Al-Quran 30 juz dengan terjemahan Indonesia
- /almatsurat: Al-Matsurat Sugro dan Kubro (dzikir pagi-petang)
- /doa: Kumpulan doa harian Islam
- /shalat: Jadwal shalat harian per kota Indonesia
- /asmaul-husna: 99 Asmaul Husna dengan makna
- /kiblat: Penunjuk arah kiblat berbasis GPS
- /hisab: Kalender dan konversi Hijriyah-Masehi
- /quiz: Quiz Islami interaktif dengan leaderboard bersama

## Quiz Categories
- /quiz/aqidah: Aqidah dan tauhid
- /quiz/rukun-islam: Rukun Islam dan ibadah wajib
- /quiz/alquran: Al-Quran dan ilmu Al-Quran
- /quiz/hadits: Hadits dan ilmu hadits
- /quiz/sirah: Sirah Nabawiyah
- /quiz/fiqh: Fiqh dan hukum Islam
- /quiz/sejarah-islam: Sejarah Islam
- /quiz/akhlak: Akhlak dan adab Islami

## About
islami.click adalah referensi ibadah harian untuk Muslim Indonesia.
Menyediakan bacaan Al-Quran, dzikir, doa, jadwal shalat, dan alat bantu ibadah lainnya.
`
	fmt.Fprint(w, content)
}
