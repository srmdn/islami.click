package model

import "html/template"

// TafsirAyah is one verse with its Arabic text, Indonesian translation
// (both already stored in quran_ayahs) and the At-Tafsir Al-Muyassar
// commentary (Arabic) fetched from the Quran.com API v4.
type TafsirAyah struct {
	Number      int           `json:"number"`
	Arabic      string        `json:"arabic"`
	Translation string        `json:"translation"`
	Tafsir      template.HTML `json:"tafsir"`
	// TafsirFirst marks the ayah that renders the card when consecutive
	// ayahs share an identical group commentary (e.g. Ibn Kathir blocks
	// repeated per ayah by the source API). TafsirRange holds the grouped
	// label ("1–6"), empty when the card covers a single ayah.
	// Render-only hints, never seeded or served as data.
	TafsirFirst bool   `json:"-"`
	TafsirRange string `json:"-"`
}

// TafsirEdition is one commentary book available per ayah
// (e.g. Muyassar in Arabic, Ibn Kathir abridged in English).
type TafsirEdition struct {
	ID       string `json:"id"`
	Slug     string `json:"slug"`
	Title    string `json:"title"`
	Author   string `json:"author"`
	Language string `json:"language"`
	Source   string `json:"source"`
	Resource string `json:"resource"`
}

type TafsirIndexData struct {
	Meta        PageMeta
	Title       string
	Description string
	About       string
	Attribution string
	Surahs      []QuranSurah
	Covered     map[int]int
	TotalAyahs  int
	Editions    []TafsirEdition
	Edition     TafsirEdition
}

type TafsirSurahData struct {
	Meta        PageMeta
	Title       string
	Description string
	Attribution string
	Surah       QuranSurah
	Ayahs       []TafsirAyah
	PrevSurah   *QuranSurah
	NextSurah   *QuranSurah
	Page        int
	FirstPage   int
	TotalPages  int
	Editions    []TafsirEdition
	Edition     TafsirEdition
}

// TafsirSearchResult is one ayah whose edition commentary matched a query,
// with verse context plus the mushaf page and deep-link needed to jump to
// the ayah card on the surah page.
type TafsirSearchResult struct {
	SurahNumber int
	SurahName   string
	AyahNumber  int
	Page        int
	Arabic      string
	Translation string
	// Text is the raw edition commentary (HTML); never rendered directly.
	// Templates use Snippet instead.
	Text      string
	Snippet   template.HTML
	EditionID string
	URL       string
}

type TafsirSearchData struct {
	Meta        PageMeta
	Title       string
	Description string
	Query       string
	Results     []TafsirSearchResult
	ResultCount int
	Editions    []TafsirEdition
	Edition     TafsirEdition
}

// TafsirPeekData is the inline commentary card expanded under one quran
// ayah via htmx. It supports an edition switcher: Tafsir is the active
// edition's text, Editions lists the available tabs, and URL deep-links to
// that edition's page on the full surah reader.
type TafsirPeekData struct {
	SurahNumber int
	AyahNumber  int
	Tafsir      template.HTML
	Range       string
	URL         string
	Editions    []TafsirEdition
	Edition     TafsirEdition
}
