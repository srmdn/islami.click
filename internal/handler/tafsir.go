package handler

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/srmdn/islami.click/internal/model"
)

// Tafsir renders the index of all 114 surahs with tafsir coverage.
func (h *Handler) Tafsir(w http.ResponseWriter, r *http.Request) {
	surahs, err := h.contentStore.QuranSurahs(r.Context())
	if err != nil {
		log.Printf("tafsir surahs: %v", err)
		http.Error(w, "Failed to load content", http.StatusInternalServerError)
		return
	}

	covered, err := h.contentStore.TafsirCoverage(r.Context())
	if err != nil {
		log.Printf("tafsir coverage: %v", err)
		http.Error(w, "Failed to load content", http.StatusInternalServerError)
		return
	}

	total := 0
	for _, n := range covered {
		total += n
	}

	meta := pageMeta(r, "Tafsir Al-Muyassar", "Baca Tafsir Al-Muyassar per ayat lengkap 114 surah dengan teks Arab, terjemahan Indonesia, dan penjelasan ringkas.")
	meta.JSONLD = breadcrumbJSONLD(homeCrumb(), crumb(2, "Tafsir Al-Muyassar", siteURL+"/tafsir"))
	h.render(w, "tafsir.html", model.TafsirIndexData{
		Meta:        meta,
		Title:       "Tafsir Al-Muyassar",
		Description: "Penjelasan ringkas tiap ayat (Bahasa Arab) beserta teks ayat dan terjemahan Indonesia",
		Surahs:      surahs,
		Covered:     covered,
		TotalAyahs:  total,
	})
}

// TafsirSurah renders one surah with per-ayah Muyassar commentary,
// paginated by mushaf page like the Quran reader.
func (h *Handler) TafsirSurah(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/tafsir/")
	path = strings.TrimSpace(path)
	if path == "" {
		http.Redirect(w, r, "/tafsir", http.StatusSeeOther)
		return
	}

	surahNumber, err := strconv.Atoi(path)
	if err != nil || surahNumber < 1 || surahNumber > 114 {
		http.Error(w, "Surah tidak ditemukan", http.StatusNotFound)
		return
	}

	surah, err := h.contentStore.QuranSurah(r.Context(), surahNumber)
	if err != nil {
		log.Printf("tafsir surah %d: %v", surahNumber, err)
		http.Error(w, "Failed to load content", http.StatusInternalServerError)
		return
	}

	pages, err := h.contentStore.MushafPagesForSurah(r.Context(), surahNumber)
	if err != nil {
		log.Printf("mushaf pages for tafsir surah %d: %v", surahNumber, err)
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

	ayahs, err := h.contentStore.TafsirAyahsByMushafPage(r.Context(), surahNumber, pageNum)
	if err != nil {
		log.Printf("tafsir ayahs %d page %d: %v", surahNumber, pageNum, err)
		http.Error(w, "Failed to load content", http.StatusInternalServerError)
		return
	}

	meta := pageMeta(r, fmt.Sprintf("Tafsir Surah %s", surah.Name), fmt.Sprintf("Tafsir Al-Muyassar Surah %s lengkap per ayat dengan teks Arab dan terjemahan Indonesia.", surah.Name))
	meta.JSONLD = breadcrumbJSONLD(homeCrumb(), crumb(2, "Tafsir Al-Muyassar", siteURL+"/tafsir"), crumb(3, surah.Name, fmt.Sprintf("%s/tafsir/%d", siteURL, surahNumber)))
	data := model.TafsirSurahData{
		Meta:        meta,
		Title:       fmt.Sprintf("Tafsir %s - Al-Muyassar", surah.Name),
		Description: fmt.Sprintf("Surah %s (%s) - %d Ayat", surah.Name, surah.ArabicName, surah.AyahCount),
		Surah:       surah,
		Ayahs:       ayahs,
		Page:        pageNum,
		FirstPage:   firstPage,
		TotalPages:  lastPage,
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

	h.render(w, "tafsir-surah.html", data)
}
