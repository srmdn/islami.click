package store

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"fmt"

	"github.com/srmdn/islami.click/internal/model"
	"html/template"
)

// TafsirAyahsByMushafPage returns verses of a surah on one mushaf page,
// each joined with its At-Tafsir Al-Muyassar commentary.
func (s *Store) TafsirAyahsByMushafPage(ctx context.Context, surahNumber int, pageNumber int) ([]model.TafsirAyah, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT ayah_number, text_arabic, translation, tafsir
		FROM quran_ayahs
		WHERE surah_number = ? AND page = ?
		ORDER BY ayah_number
	`, surahNumber, pageNumber)
	if err != nil {
		return nil, fmt.Errorf("read tafsir ayahs for surah %d page %d: %w", surahNumber, pageNumber, err)
	}
	defer rows.Close()

	var ayahs []model.TafsirAyah
	for rows.Next() {
		var a model.TafsirAyah
		var tafsir string
		if err := rows.Scan(&a.Number, &a.Arabic, &a.Translation, &tafsir); err != nil {
			return nil, fmt.Errorf("scan tafsir ayah: %w", err)
		}
		a.Tafsir = template.HTML(tafsir)
		ayahs = append(ayahs, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tafsir ayahs for surah %d page %d: %w", surahNumber, pageNumber, err)
	}
	return ayahs, nil
}

// TafsirCoverage returns, per surah, how many ayahs already carry tafsir text.
func (s *Store) TafsirCoverage(ctx context.Context) (map[int]int, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT surah_number, COUNT(*)
		FROM quran_ayahs
		WHERE tafsir <> ''
		GROUP BY surah_number
	`)
	if err != nil {
		return nil, fmt.Errorf("read tafsir coverage: %w", err)
	}
	defer rows.Close()

	covered := make(map[int]int)
	for rows.Next() {
		var surah, count int
		if err := rows.Scan(&surah, &count); err != nil {
			return nil, fmt.Errorf("scan tafsir coverage: %w", err)
		}
		covered[surah] = count
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tafsir coverage: %w", err)
	}
	return covered, nil
}

// tafsirManifest mirrors content/tafsir/manifest.json.
type tafsirManifest struct {
	Source   string `json:"source"`
	Resource string `json:"resource"`
	Language string `json:"language"`
	Surahs   []struct {
		Number int    `json:"number"`
		Ayahs  int    `json:"ayahs"`
		SHA256 string `json:"sha256"`
	} `json:"surahs"`
}

// seedTafsir fills the tafsir column of quran_ayahs from the vendored
// content/tafsir/NNN.json files (At-Tafsir Al-Muyassar, Arabic).
// It must run after seedQuran (higher order) because it UPDATEs ayah rows.
// The manifest checksum also folds in the quran content checksum, so any
// quran reseed (which clears ayah rows) triggers a tafsir reseed as well.
func seedTafsir(ctx context.Context, tx *sql.Tx, contentFS embed.FS, collectionID, path string, order int) error {
	data, err := contentFS.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}

	var manifest tafsirManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}

	if err := insertCollection(ctx, tx, collectionID, "quran", "Tafsir Al-Muyassar", "Tafsir Al-Muyassar (Bahasa Arab) per ayat dari Quran.com API", path, checksum(data)+quranChecksum(contentFS), order); err != nil {
		return err
	}

	for _, ms := range manifest.Surahs {
		surahPath := fmt.Sprintf("content/tafsir/%03d.json", ms.Number)
		surahData, err := contentFS.ReadFile(surahPath)
		if err != nil {
			return fmt.Errorf("read tafsir surah %d: %w", ms.Number, err)
		}

		var surahFile struct {
			Ayahs []struct {
				Number int    `json:"number"`
				Tafsir string `json:"tafsir"`
			} `json:"ayahs"`
		}
		if err := json.Unmarshal(surahData, &surahFile); err != nil {
			return fmt.Errorf("parse tafsir surah %d: %w", ms.Number, err)
		}
		if len(surahFile.Ayahs) != ms.Ayahs {
			return fmt.Errorf("tafsir surah %d: manifest lists %d ayahs, file has %d", ms.Number, ms.Ayahs, len(surahFile.Ayahs))
		}

		for _, ayah := range surahFile.Ayahs {
			if ayah.Number <= 0 || ayah.Tafsir == "" {
				return fmt.Errorf("tafsir surah %d: invalid ayah entry #%d", ms.Number, ayah.Number)
			}
			res, err := tx.ExecContext(ctx, `
				UPDATE quran_ayahs SET tafsir = ? WHERE surah_number = ? AND ayah_number = ?
			`, ayah.Tafsir, ms.Number, ayah.Number)
			if err != nil {
				return fmt.Errorf("seed tafsir %d:%d: %w", ms.Number, ayah.Number, err)
			}
			n, err := res.RowsAffected()
			if err != nil {
				return fmt.Errorf("confirm tafsir %d:%d: %w", ms.Number, ayah.Number, err)
			}
			if n != 1 {
				return fmt.Errorf("tafsir %d:%d: no matching ayah row (seed quran first)", ms.Number, ayah.Number)
			}
		}
	}

	return nil
}
