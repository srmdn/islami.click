package store

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"path"
	"strings"

	"github.com/srmdn/islami.click/internal/model"
)

func (s *Store) QuranSurahs(ctx context.Context) ([]model.QuranSurah, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT number, name_arabic, name_latin, name_translation, revelation_place, ayah_count
		FROM quran_surahs
		ORDER BY number
	`)
	if err != nil {
		return nil, fmt.Errorf("read quran surahs: %w", err)
	}
	defer rows.Close()

	var surahs []model.QuranSurah
	for rows.Next() {
		var s model.QuranSurah
		if err := rows.Scan(&s.Number, &s.ArabicName, &s.Name, &s.Translation, &s.RevelationType, &s.AyahCount); err != nil {
			return nil, fmt.Errorf("scan quran surah: %w", err)
		}
		surahs = append(surahs, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate quran surahs: %w", err)
	}
	return surahs, nil
}

func (s *Store) QuranSurah(ctx context.Context, number int) (model.QuranSurah, error) {
	var surah model.QuranSurah
	err := s.db.QueryRowContext(ctx, `
		SELECT number, name_arabic, name_latin, name_translation, revelation_place, ayah_count
		FROM quran_surahs
		WHERE number = ?
	`, number).Scan(&surah.Number, &surah.ArabicName, &surah.Name, &surah.Translation, &surah.RevelationType, &surah.AyahCount)
	if err != nil {
		if err == sql.ErrNoRows {
			return surah, fmt.Errorf("quran surah %d not found", number)
		}
		return surah, fmt.Errorf("read quran surah %d: %w", number, err)
	}
	return surah, nil
}

func (s *Store) QuranAyahs(ctx context.Context, surahNumber int) ([]model.QuranAyah, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT ayah_number, text_arabic, translation
		FROM quran_ayahs
		WHERE surah_number = ?
		ORDER BY ayah_number
	`, surahNumber)
	if err != nil {
		return nil, fmt.Errorf("read quran ayahs for surah %d: %w", surahNumber, err)
	}
	defer rows.Close()

	var ayahs []model.QuranAyah
	for rows.Next() {
		var a model.QuranAyah
		if err := rows.Scan(&a.Number, &a.Arabic, &a.Translation); err != nil {
			return nil, fmt.Errorf("scan quran ayah: %w", err)
		}
		ayahs = append(ayahs, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate quran ayahs for surah %d: %w", surahNumber, err)
	}
	return ayahs, nil
}

func (s *Store) QuranAyahsByMushafPage(ctx context.Context, surahNumber int, pageNumber int) ([]model.QuranAyah, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT ayah_number, text_arabic, translation
		FROM quran_ayahs
		WHERE surah_number = ? AND page = ?
		ORDER BY ayah_number
	`, surahNumber, pageNumber)
	if err != nil {
		return nil, fmt.Errorf("read quran ayahs for surah %d page %d: %w", surahNumber, pageNumber, err)
	}
	defer rows.Close()

	var ayahs []model.QuranAyah
	for rows.Next() {
		var a model.QuranAyah
		if err := rows.Scan(&a.Number, &a.Arabic, &a.Translation); err != nil {
			return nil, fmt.Errorf("scan quran ayah: %w", err)
		}
		ayahs = append(ayahs, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate quran ayahs for surah %d page %d: %w", surahNumber, pageNumber, err)
	}
	return ayahs, nil
}

func (s *Store) MushafPagesForSurah(ctx context.Context, surahNumber int) ([]int, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT DISTINCT page FROM quran_ayahs
		WHERE surah_number = ?
		ORDER BY page
	`, surahNumber)
	if err != nil {
		return nil, fmt.Errorf("read mushaf pages for surah %d: %w", surahNumber, err)
	}
	defer rows.Close()

	var pages []int
	for rows.Next() {
		var p int
		if err := rows.Scan(&p); err != nil {
			return nil, fmt.Errorf("scan page: %w", err)
		}
		pages = append(pages, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate mushaf pages for surah %d: %w", surahNumber, err)
	}
	return pages, nil
}

func (s *Store) SearchQuran(ctx context.Context, query string, limit int) ([]model.QuranSearchResult, error) {
	if limit <= 0 {
		limit = 20
	}
	searchTerm := "%" + query + "%"

	rows, err := s.db.QueryContext(ctx, `
		SELECT s.number, s.name_latin, a.ayah_number, a.text_arabic, a.translation
		FROM quran_ayahs a
		JOIN quran_surahs s ON s.number = a.surah_number
		WHERE a.text_arabic LIKE ? OR a.translation LIKE ?
		ORDER BY s.number, a.ayah_number
		LIMIT ?
	`, searchTerm, searchTerm, limit)
	if err != nil {
		return nil, fmt.Errorf("search quran: %w", err)
	}
	defer rows.Close()

	var results []model.QuranSearchResult
	for rows.Next() {
		var r model.QuranSearchResult
		if err := rows.Scan(&r.SurahNumber, &r.SurahName, &r.AyahNumber, &r.Arabic, &r.Translation); err != nil {
			return nil, fmt.Errorf("scan quran search result: %w", err)
		}
		results = append(results, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate quran search results: %w", err)
	}
	return results, nil
}

func seedQuran(ctx context.Context, tx *sql.Tx, contentFS embed.FS, collectionID, path string, order int) error {
	metaData, err := contentFS.ReadFile("content/quran-surahs.json")
	if err != nil {
		return fmt.Errorf("read quran metadata: %w", err)
	}

	var metaList []struct {
		ID              int    `json:"id"`
		Name            string `json:"name"`
		Transliteration string `json:"transliteration"`
		Translation     string `json:"translation"`
		Type            string `json:"type"`
		TotalVerses     int    `json:"total_verses"`
	}
	if err := json.Unmarshal(metaData, &metaList); err != nil {
		return fmt.Errorf("parse quran metadata: %w", err)
	}

	pageMapData, err := contentFS.ReadFile("content/quran-pages.json")
	if err != nil {
		return fmt.Errorf("read quran page map: %w", err)
	}
	var pageMap map[string]int
	if err := json.Unmarshal(pageMapData, &pageMap); err != nil {
		return fmt.Errorf("parse quran page map: %w", err)
	}

	if err := insertCollection(ctx, tx, collectionID, "quran", "Al-Qur'an", "Al-Qur'an dengan terjemahan Bahasa Indonesia", path, quranChecksum(contentFS), order); err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, "DELETE FROM quran_surahs"); err != nil {
		return fmt.Errorf("clear quran surahs: %w", err)
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM quran_ayahs"); err != nil {
		return fmt.Errorf("clear quran ayahs: %w", err)
	}

	for _, meta := range metaList {
		_, err := tx.ExecContext(ctx, `
			INSERT INTO quran_surahs (number, slug, name_arabic, name_latin, name_translation, revelation_place, ayah_count)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`, meta.ID, slugify(meta.Transliteration), meta.Name, meta.Transliteration, meta.Translation, meta.Type, meta.TotalVerses)
		if err != nil {
			return fmt.Errorf("seed quran surah %d: %w", meta.ID, err)
		}

		surahPath := fmt.Sprintf("content/quran/%03d.json", meta.ID)
		surahData, err := contentFS.ReadFile(surahPath)
		if err != nil {
			return fmt.Errorf("read quran surah %d: %w", meta.ID, err)
		}

		var surahFile struct {
			Ayahs []model.QuranAyah `json:"ayahs"`
		}
		if err := json.Unmarshal(surahData, &surahFile); err != nil {
			return fmt.Errorf("parse quran surah %d: %w", meta.ID, err)
		}

		for _, ayah := range surahFile.Ayahs {
			key := fmt.Sprintf("%d:%d", meta.ID, ayah.Number)
			pageNum := pageMap[key]
			_, err := tx.ExecContext(ctx, `
				INSERT INTO quran_ayahs (surah_number, ayah_number, text_arabic, translation, page)
				VALUES (?, ?, ?, ?, ?)
			`, meta.ID, ayah.Number, ayah.Arabic, ayah.Translation, pageNum)
			if err != nil {
				return fmt.Errorf("seed quran ayah %d:%d: %w", meta.ID, ayah.Number, err)
			}
		}
	}

	return nil
}

func quranChecksum(contentFS embed.FS) string {
	h := sha256.New()

	metaData, err := contentFS.ReadFile("content/quran-surahs.json")
	if err == nil {
		h.Write(metaData)
	}

	pageMapData, err := contentFS.ReadFile("content/quran-pages.json")
	if err == nil {
		h.Write(pageMapData)
	}

	entries, err := fs.ReadDir(contentFS, "content/quran")
	if err == nil {
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
				continue
			}
			data, err := contentFS.ReadFile(path.Join("content/quran", entry.Name()))
			if err == nil {
				h.Write(data)
			}
		}
	}

	return hex.EncodeToString(h.Sum(nil))
}

func slugify(name string) string {
	return strings.ToLower(strings.ReplaceAll(name, " ", "-"))
}

func (s *Store) GetQuranSurahByNumber(ctx context.Context, number int) (*model.QuranSurah, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT number, name_arabic, name_latin, name_translation, revelation_place, ayah_count
		FROM quran_surahs
		WHERE number = ?
	`, number)
	var surah model.QuranSurah
	if err := row.Scan(&surah.Number, &surah.ArabicName, &surah.Name, &surah.Translation, &surah.RevelationType, &surah.AyahCount); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get quran surah %d: %w", number, err)
	}
	return &surah, nil
}

func (s *Store) GetQuranAyah(ctx context.Context, surahNumber, ayahNumber int) (*model.QuranSearchResult, error) {
	var r model.QuranSearchResult
	err := s.db.QueryRowContext(ctx, `
		SELECT s.number, s.name_latin, a.ayah_number, a.text_arabic, a.translation
		FROM quran_ayahs a
		JOIN quran_surahs s ON s.number = a.surah_number
		WHERE a.surah_number = ? AND a.ayah_number = ?
	`, surahNumber, ayahNumber).Scan(&r.SurahNumber, &r.SurahName, &r.AyahNumber, &r.Arabic, &r.Translation)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get quran ayah %d:%d: %w", surahNumber, ayahNumber, err)
	}
	return &r, nil
}

func (s *Store) SearchQuranSurahs(ctx context.Context, query string) ([]model.QuranSearchResult, error) {
	q := "%" + query + "%"
	rows, err := s.db.QueryContext(ctx, `
		SELECT number, name_latin, 0, '', ''
		FROM quran_surahs
		WHERE LOWER(name_latin) LIKE LOWER(?) OR LOWER(name_translation) LIKE LOWER(?)
		ORDER BY number
		LIMIT 10
	`, q, q)
	if err != nil {
		return nil, fmt.Errorf("search quran surahs: %w", err)
	}
	defer rows.Close()

	var results []model.QuranSearchResult
	for rows.Next() {
		var r model.QuranSearchResult
		if err := rows.Scan(&r.SurahNumber, &r.SurahName, &r.AyahNumber, &r.Arabic, &r.Translation); err != nil {
			return nil, fmt.Errorf("scan quran surah search: %w", err)
		}
		results = append(results, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate quran surah search: %w", err)
	}
	return results, nil
}


