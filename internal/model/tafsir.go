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
