package handler

import (
	"context"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"regexp"
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

// TafsirSearch renders the per-edition commentary search form and matches,
// mirroring QuranSearch: the same reference grammar (5:7, "ayat 7 al
// maidah", surah names), with content search over one edition's tafsir text
// plus verse Arabic and Indonesian translation so Indonesian keywords work
// before any Indonesian commentary edition exists.
func (h *Handler) TafsirSearch(w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if len(query) > 200 {
		query = query[:200]
	}

	editions, edition := h.resolveTafsirEdition(r)

	data := model.TafsirSearchData{
		Meta:        pageMeta(r, "Pencarian Tafsir", "Cari penjelasan ayat dalam kitab tafsir"),
		Title:       "Pencarian Tafsir",
		Description: "Cari penjelasan ayat dalam kitab tafsir",
		Query:       query,
		Editions:    editions,
		Edition:     edition,
	}

	if query != "" {
		results, err := h.smartTafsirSearch(r.Context(), edition.ID, query)
		if err != nil {
			log.Printf("tafsir search %q: %v", query, err)
			http.Error(w, "Gagal mencari", http.StatusInternalServerError)
			return
		}
		data.Results = results
		data.ResultCount = len(results)
	}

	h.render(w, "tafsir-search.html", data)
}

func (h *Handler) smartTafsirSearch(ctx context.Context, editionID, query string) ([]model.TafsirSearchResult, error) {
	seen := make(map[string]bool)
	var results []model.TafsirSearchResult
	firstPages := make(map[int]int)

	// snippetQuery centers excerpts on the content words; a bare reference
	// like "2:255" leaves no content words, so excerpts fall back to the
	// raw query (no match → head of the commentary).
	snippetQuery := query

	add := func(r model.TafsirSearchResult) {
		key := fmt.Sprintf("%d:%d", r.SurahNumber, r.AyahNumber)
		if seen[key] {
			return
		}
		seen[key] = true
		r.EditionID = editionID
		r.Snippet = tafsirSnippet(r.Text, snippetQuery)
		r.URL = tafsirSearchURL(ctx, h, editionID, r, firstPages)
		results = append(results, r)
	}

	surahNum, ayahNum, cleanedQuery, err := h.extractQuranReference(ctx, query)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(cleanedQuery) != "" {
		snippetQuery = cleanedQuery
	}

	if surahNum > 0 && ayahNum > 0 {
		r, err := h.contentStore.GetTafsirAyah(ctx, editionID, surahNum, ayahNum)
		if err != nil {
			return nil, err
		}
		if r != nil {
			add(*r)
		}
	}

	if surahNum > 0 && ayahNum == 0 {
		head, err := h.contentStore.TafsirHeadByEdition(ctx, editionID, surahNum, 3)
		if err != nil {
			return nil, err
		}
		for _, a := range head {
			add(a)
		}
	}

	if strings.TrimSpace(cleanedQuery) != "" {
		hits, err := h.contentStore.SearchTafsir(ctx, editionID, cleanedQuery, 50)
		if err != nil {
			return nil, err
		}
		for _, cr := range hits {
			add(cr)
		}
	}

	surahResults, err := h.contentStore.SearchQuranSurahs(ctx, query)
	if err != nil {
		return nil, err
	}
	for _, sr := range surahResults {
		head, err := h.contentStore.TafsirHeadByEdition(ctx, editionID, sr.SurahNumber, 1)
		if err != nil {
			return nil, err
		}
		for _, a := range head {
			add(a)
		}
	}

	return results, nil
}

// tafsirSearchURL deep-links a search hit to its ayah card, dropping the
// edition param for Muyassar and the page param on the surah's first
// mushaf page so URLs stay clean like the share buttons. firstPages caches
// the per-surah lookup across hits.
func tafsirSearchURL(ctx context.Context, h *Handler, editionID string, r model.TafsirSearchResult, firstPages map[int]int) string {
	url := fmt.Sprintf("/tafsir/%d", r.SurahNumber)
	var qs []string
	if editionID != defaultTafsirEdition {
		qs = append(qs, "edition="+editionID)
	}
	fp, ok := firstPages[r.SurahNumber]
	if !ok {
		if pages, err := h.contentStore.MushafPagesForSurah(ctx, r.SurahNumber); err == nil && len(pages) > 0 {
			fp = pages[0]
		}
		firstPages[r.SurahNumber] = fp
	}
	if fp != 0 && r.Page != fp {
		qs = append(qs, "page="+strconv.Itoa(r.Page))
	}
	if len(qs) > 0 {
		url += "?" + strings.Join(qs, "&")
	}
	return fmt.Sprintf("%s#ayah-%d", url, r.AyahNumber)
}

// tafsirSnippetWidth is the excerpt length (runes) shown per search hit.
const tafsirSnippetWidth = 220

var tafsirTagStripper = regexp.MustCompile(`(?s)<[^>]*>`)

// tafsirSnippet centers a plain-text window on the first case-insensitive
// match of query. Stored commentary carries HTML, so tags are stripped
// before windowing (never cut mid-tag); output is escaped with the match
// marked, safe to render directly.
func tafsirSnippet(raw, query string) template.HTML {
	plain := tafsirTagStripper.ReplaceAllString(raw, " ")
	plain = strings.Join(strings.Fields(plain), " ")
	if plain == "" {
		return template.HTML("")
	}
	runes := []rune(plain)
	start := 0
	if q := strings.TrimSpace(query); q != "" {
		if i := indexFoldRunes(plain, q); i >= 0 {
			start = i - (tafsirSnippetWidth-len([]rune(q)))/2
		}
	}
	if start < 0 {
		start = 0
	}
	end := start + tafsirSnippetWidth
	if end > len(runes) {
		end = len(runes)
		start = end - tafsirSnippetWidth
		if start < 0 {
			start = 0
		}
	}
	var b strings.Builder
	if start > 0 {
		b.WriteString("… ")
	}
	b.WriteString(markFold(string(runes[start:end]), query))
	if end < len(runes) {
		b.WriteString(" …")
	}
	return template.HTML(b.String())
}

// markFold escapes s and wraps every case-insensitive occurrence of q in
// <mark class="tafsir-key"> (styled in input.css). Rune-based so multibyte
// Arabic slices never split.
func markFold(s, q string) string {
	q = strings.TrimSpace(q)
	if q == "" {
		return template.HTMLEscapeString(s)
	}
	qlen := len([]rune(q))
	var b strings.Builder
	for rest := s; ; {
		i := indexFoldRunes(rest, q)
		if i < 0 {
			b.WriteString(template.HTMLEscapeString(rest))
			break
		}
		r := []rune(rest)
		b.WriteString(template.HTMLEscapeString(string(r[:i])))
		b.WriteString(`<mark class="tafsir-key">`)
		b.WriteString(template.HTMLEscapeString(string(r[i : i+qlen])))
		b.WriteString(`</mark>`)
		rest = string(r[i+qlen:])
	}
	return b.String()
}

// indexFoldRunes reports the rune index of the first case-insensitive
// occurrence of q in s, or -1.
func indexFoldRunes(s, q string) int {
	lq := []rune(strings.ToLower(q))
	if len(lq) == 0 {
		return 0
	}
	ls := []rune(strings.ToLower(s))
outer:
	for i := 0; i+len(lq) <= len(ls); i++ {
		for j := range lq {
			if ls[i+j] != lq[j] {
				continue outer
			}
		}
		return i
	}
	return -1
}
