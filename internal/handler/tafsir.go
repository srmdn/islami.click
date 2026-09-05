package handler

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/srmdn/islami.click/internal/model"
)

// defaultTafsirEdition is the edition rendered when ?edition= is absent
// or unknown. Unknown values fall back silently so old links never break.
const defaultTafsirEdition = "muyassar"

// tafsirBlurb carries the per-edition UI copy: short tagline for the
// index header plus the attribution footnote shown on every tafsir page.
type tafsirBlurb struct {
	tagline     string
	about       string
	attribution string
}

var tafsirBlurbs = map[string]tafsirBlurb{
	"muyassar": {
		tagline: "Penjelasan ringkas tiap ayat (Bahasa Arab) beserta teks ayat dan terjemahan Indonesia",
		about: "At-Tafsir Al-Muyassar adalah penjelasan ringkas makna tiap ayat " +
			"Al-Qur'an dalam Bahasa Arab. Setiap halaman menampilkan teks ayat, " +
			"terjemahan Bahasa Indonesia, dan tafsirnya berdampingan.",
		attribution: "At-Tafsir Al-Muyassar disusun oleh sekelompok ulama di bawah " +
			"bimbingan Syaikh Dr. Shalih Alusy-Syaikh, terbitan Mujamma' Malik Fahd, " +
			"Madinah. Teks Arab via Quran.com API; terjemahan ayat mengikuti teks " +
			"Al-Qur'an di islami.click.",
	},
	"ibn-kathir-en": {
		tagline: "Abridged commentary per verse (English) with Arabic text and Indonesian translation",
		about: "Ibn Kathir (abridged) explains each verse with Qur'an, hadith, and " +
			"reports from the companions — in English. Every page shows the Arabic " +
			"verse, its Indonesian translation, and the commentary side by side. " +
			"Pilot stage: only short surahs are available so far.",
		attribution: "Ibn Kathir (Abridged), by Hafiz Ibn Kathir, English text via " +
			"Quran.com API (resource 169). Verse translations follow the Qur'an " +
			"text on islami.click.",
	},
}

// resolveTafsirEdition loads all editions and picks the one requested via
// ?edition=, falling back to the default when unknown.
func (h *Handler) resolveTafsirEdition(r *http.Request) ([]model.TafsirEdition, model.TafsirEdition) {
	editions, err := h.contentStore.TafsirEditions(r.Context())
	if err != nil {
		log.Printf("tafsir editions: %v", err)
		return nil, model.TafsirEdition{ID: defaultTafsirEdition}
	}
	want := strings.TrimSpace(r.URL.Query().Get("edition"))
	if want == "" {
		want = defaultTafsirEdition
	}
	current := model.TafsirEdition{ID: defaultTafsirEdition}
	for _, e := range editions {
		if e.ID == defaultTafsirEdition && current.Title == "" {
			current = e
		}
		if e.ID == want {
			current = e
		}
	}
	if current.Title == "" && len(editions) > 0 {
		current = editions[0]
	}
	return editions, current
}

func tafsirBlurbFor(editionID string) tafsirBlurb {
	if b, ok := tafsirBlurbs[editionID]; ok {
		return b
	}
	return tafsirBlurb{
		tagline:     "Komentar per ayat beserta teks ayat dan terjemahan Indonesia",
		about:       "Tafsir per ayat beserta teks Arab dan terjemahan Indonesia.",
		attribution: "Teks tafsir via Quran.com API; terjemahan ayat mengikuti teks Al-Qur'an di islami.click.",
	}
}

// Tafsir renders the index of all 114 surahs with per-edition tafsir coverage.
func (h *Handler) Tafsir(w http.ResponseWriter, r *http.Request) {
	surahs, err := h.contentStore.QuranSurahs(r.Context())
	if err != nil {
		log.Printf("tafsir surahs: %v", err)
		http.Error(w, "Failed to load content", http.StatusInternalServerError)
		return
	}

	editions, edition := h.resolveTafsirEdition(r)
	blurb := tafsirBlurbFor(edition.ID)

	covered, err := h.contentStore.TafsirCoverageByEdition(r.Context(), edition.ID)
	if err != nil {
		log.Printf("tafsir coverage: %v", err)
		http.Error(w, "Failed to load content", http.StatusInternalServerError)
		return
	}

	total := 0
	for _, n := range covered {
		total += n
	}

	title := edition.Title
	if title == "" {
		title = "Tafsir"
	}
	meta := pageMeta(r, title, fmt.Sprintf("Baca %s per ayat lengkap dengan teks Arab, terjemahan Indonesia, dan penjelasan. %s", title, blurb.tagline))
	meta.JSONLD = breadcrumbJSONLD(homeCrumb(), crumb(2, title, siteURL+"/tafsir"))
	h.render(w, "tafsir.html", model.TafsirIndexData{
		Meta:        meta,
		Title:       title,
		Description: blurb.tagline,
		About:       blurb.about,
		Attribution: blurb.attribution,
		Surahs:      surahs,
		Covered:     covered,
		TotalAyahs:  total,
		Editions:    editions,
		Edition:     edition,
	})
}

// TafsirSurah renders one surah with per-ayah commentary of one edition,
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

	editions, edition := h.resolveTafsirEdition(r)
	blurb := tafsirBlurbFor(edition.ID)

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

	ayahs, err := h.contentStore.TafsirAyahsByEditionPage(r.Context(), edition.ID, surahNumber, pageNum)
	if err != nil {
		log.Printf("tafsir ayahs %d page %d: %v", surahNumber, pageNum, err)
		http.Error(w, "Failed to load content", http.StatusInternalServerError)
		return
	}

	editionTitle := edition.Title
	if editionTitle == "" {
		editionTitle = "Tafsir"
	}
	meta := pageMeta(r, fmt.Sprintf("%s Surah %s", editionTitle, surah.Name), fmt.Sprintf("%s Surah %s lengkap per ayat dengan teks Arab dan terjemahan Indonesia. %s", editionTitle, surah.Name, blurb.tagline))
	meta.JSONLD = breadcrumbJSONLD(homeCrumb(), crumb(2, editionTitle, siteURL+"/tafsir"), crumb(3, surah.Name, fmt.Sprintf("%s/tafsir/%d", siteURL, surahNumber)))
	data := model.TafsirSurahData{
		Meta:        meta,
		Title:       fmt.Sprintf("Tafsir %s - %s", surah.Name, editionTitle),
		Description: fmt.Sprintf("Surah %s (%s) - %d Ayat", surah.Name, surah.ArabicName, surah.AyahCount),
		Attribution: blurb.attribution,
		Surah:       surah,
		Ayahs:       ayahs,
		Page:        pageNum,
		FirstPage:   firstPage,
		TotalPages:  lastPage,
		Editions:    editions,
		Edition:     edition,
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
