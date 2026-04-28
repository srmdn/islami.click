//go:build ignore

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
)

type SurahMeta struct {
	ID              int    `json:"id"`
	Name            string `json:"name"`
	Transliteration string `json:"transliteration"`
	Translation     string `json:"translation"`
	Type            string `json:"type"`
	TotalVerses     int    `json:"total_verses"`
}

type Verse struct {
	ID              int    `json:"id"`
	Text            string `json:"text"`
	Translation     string `json:"translation,omitempty"`
	Transliteration string `json:"transliteration,omitempty"`
}

type SurahData struct {
	ID              int     `json:"id"`
	Name            string  `json:"name"`
	Transliteration string  `json:"transliteration"`
	Translation     string  `json:"translation"`
	Type            string  `json:"type"`
	TotalVerses     int     `json:"total_verses"`
	Verses          []Verse `json:"verses"`
}

type OutputSurah struct {
	Number         int          `json:"number"`
	Name           string       `json:"name"`
	ArabicName     string       `json:"arabic_name"`
	Translation    string       `json:"translation"`
	RevelationType string       `json:"revelation_type"`
	AyahCount      int          `json:"ayah_count"`
	Ayahs          []OutputAyah `json:"ayahs"`
}

type OutputAyah struct {
	Number      int    `json:"number"`
	Arabic      string `json:"arabic"`
	Translation string `json:"translation"`
}

func fetchJSON(url string, target any) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return json.NewDecoder(resp.Body).Decode(target)
}

func main() {
	// Fetch Arabic full Quran
	fmt.Println("Fetching Arabic Quran...")
	var arabicData []SurahData
	if err := fetchJSON("https://cdn.jsdelivr.net/npm/quran-json@3.1.2/dist/quran.json", &arabicData); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to fetch Arabic Quran: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Fetched %d surahs (Arabic)\n", len(arabicData))

	// Fetch Indonesian translation
	fmt.Println("Fetching Indonesian translation...")
	var indonesianData []SurahData
	if err := fetchJSON("https://cdn.jsdelivr.net/npm/quran-json@3.1.2/dist/quran_id.json", &indonesianData); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to fetch Indonesian Quran: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Fetched %d surahs (Indonesian)\n", len(indonesianData))

	// Create content directory
	quranDir := "content/quran"
	if err := os.MkdirAll(quranDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create dir: %v\n", err)
		os.Exit(1)
	}

	// Build surah metadata and per-surah files
	surahMetas := make([]SurahMeta, 0, 114)
	for i, arabicSurah := range arabicData {
		indonesianSurah := indonesianData[i]
		if arabicSurah.ID != indonesianSurah.ID {
			fmt.Fprintf(os.Stderr, "Mismatch: arabic %d vs indonesian %d\n", arabicSurah.ID, indonesianSurah.ID)
			os.Exit(1)
		}

		surahMetas = append(surahMetas, SurahMeta{
			ID:              arabicSurah.ID,
			Name:            arabicSurah.Name,
			Transliteration: arabicSurah.Transliteration,
			Translation:     indonesianSurah.Translation,
			Type:            arabicSurah.Type,
			TotalVerses:     arabicSurah.TotalVerses,
		})

		// Build per-surah file with merged Arabic + translation
		output := OutputSurah{
			Number:         arabicSurah.ID,
			Name:           arabicSurah.Transliteration,
			ArabicName:     arabicSurah.Name,
			Translation:    indonesianSurah.Translation,
			RevelationType: arabicSurah.Type,
			AyahCount:      arabicSurah.TotalVerses,
			Ayahs:          make([]OutputAyah, len(arabicSurah.Verses)),
		}

		for j, arabicVerse := range arabicSurah.Verses {
			indonesianVerse := indonesianSurah.Verses[j]
			output.Ayahs[j] = OutputAyah{
				Number:      arabicVerse.ID,
				Arabic:      arabicVerse.Text,
				Translation: indonesianVerse.Translation,
			}
		}

		filename := filepath.Join(quranDir, fmt.Sprintf("%03d.json", arabicSurah.ID))
		data, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to marshal surah %d: %v\n", arabicSurah.ID, err)
			os.Exit(1)
		}
		if err := os.WriteFile(filename, data, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to write %s: %v\n", filename, err)
			os.Exit(1)
		}
		fmt.Printf("Written %s (%d ayahs)\n", filename, len(output.Ayahs))
	}

	// Write surah metadata file
	metaData, err := json.MarshalIndent(surahMetas, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to marshal metadata: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile("content/quran-surahs.json", metaData, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to write metadata: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Written content/quran-surahs.json")
}
