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
	Surahs      []QuranSurah
	Covered     map[int]int
	TotalAyahs  int
}

type TafsirSurahData struct {
	Meta        PageMeta
	Title       string
	Description string
	Surah       QuranSurah
	Ayahs       []TafsirAyah
	PrevSurah   *QuranSurah
	NextSurah   *QuranSurah
	Page        int
	FirstPage   int
	TotalPages  int
}
