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

// TafsirEditions returns all commentary editions (books) available,
// ordered by id for stable UI listing.
func (s *Store) TafsirEditions(ctx context.Context) ([]model.TafsirEdition, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, slug, title, author, language, source, resource
		FROM tafsir_editions
		ORDER BY id
	`)
	if err != nil {
		return nil, fmt.Errorf("read tafsir editions: %w", err)
	}
	defer rows.Close()

	var editions []model.TafsirEdition
	for rows.Next() {
		var e model.TafsirEdition
		if err := rows.Scan(&e.ID, &e.Slug, &e.Title, &e.Author, &e.Language, &e.Source, &e.Resource); err != nil {
			return nil, fmt.Errorf("scan tafsir edition: %w", err)
		}
		editions = append(editions, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tafsir editions: %w", err)
	}
	return editions, nil
}

// TafsirCoverageByEdition returns, per surah, how many ayahs already carry
// commentary text for one edition.
func (s *Store) TafsirCoverageByEdition(ctx context.Context, editionID string) (map[int]int, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT surah_number, COUNT(*)
		FROM tafsir_texts
		WHERE edition_id = ? AND text <> ''
		GROUP BY surah_number
	`, editionID)
	if err != nil {
		return nil, fmt.Errorf("read tafsir coverage for edition %s: %w", editionID, err)
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

// TafsirAyahsByEditionPage returns verses of a surah on one mushaf page,
// each joined with the commentary of one edition.
func (s *Store) TafsirAyahsByEditionPage(ctx context.Context, editionID string, surahNumber int, pageNumber int) ([]model.TafsirAyah, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT a.ayah_number, a.text_arabic, a.translation, t.text
		FROM quran_ayahs a
		JOIN tafsir_texts t
		  ON t.surah_number = a.surah_number
		 AND t.ayah_number = a.ayah_number
		 AND t.edition_id = ?
		WHERE a.surah_number = ? AND a.page = ?
		ORDER BY a.ayah_number
	`, editionID, surahNumber, pageNumber)
	if err != nil {
		return nil, fmt.Errorf("read %s ayahs for surah %d page %d: %w", editionID, surahNumber, pageNumber, err)
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
	groupIdenticalTafsir(ayahs)
	return ayahs, nil
}

// SearchTafsir returns ayahs whose edition commentary contains the query,
// oldest surah first, capped at limit. Only the tafsir text is matched —
// verse Arabic and translation ride along as display context. Callers build
// display snippets because stored commentary carries HTML.
func (s *Store) SearchTafsir(ctx context.Context, editionID, query string, limit int) ([]model.TafsirSearchResult, error) {
	if limit <= 0 {
		limit = 50
	}
	searchTerm := "%" + query + "%"

	rows, err := s.db.QueryContext(ctx, `
		SELECT s.number, s.name_latin, a.ayah_number, a.page, a.text_arabic, a.translation, t.text
		FROM tafsir_texts t
		JOIN quran_ayahs a
		  ON a.surah_number = t.surah_number
		 AND a.ayah_number = t.ayah_number
		JOIN quran_surahs s ON s.number = t.surah_number
		WHERE t.edition_id = ? AND t.text LIKE ?
		ORDER BY s.number, a.ayah_number
		LIMIT ?
	`, editionID, searchTerm, limit)
	if err != nil {
		return nil, fmt.Errorf("search %s tafsir: %w", editionID, err)
	}
	defer rows.Close()

	var results []model.TafsirSearchResult
	for rows.Next() {
		var r model.TafsirSearchResult
		if err := rows.Scan(&r.SurahNumber, &r.SurahName, &r.AyahNumber, &r.Page, &r.Arabic, &r.Translation, &r.Text); err != nil {
			return nil, fmt.Errorf("scan tafsir search result: %w", err)
		}
		r.EditionID = editionID
		results = append(results, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tafsir search results: %w", err)
	}
	return results, nil
}

// GetTafsirAyah returns one ayah with its edition commentary, or nil when
// the edition has no text for that address.
func (s *Store) GetTafsirAyah(ctx context.Context, editionID string, surahNumber, ayahNumber int) (*model.TafsirSearchResult, error) {
	var r model.TafsirSearchResult
	err := s.db.QueryRowContext(ctx, `
		SELECT s.number, s.name_latin, a.ayah_number, a.page, a.text_arabic, a.translation, t.text
		FROM tafsir_texts t
		JOIN quran_ayahs a
		  ON a.surah_number = t.surah_number
		 AND a.ayah_number = t.ayah_number
		JOIN quran_surahs s ON s.number = t.surah_number
		WHERE t.edition_id = ? AND t.surah_number = ? AND t.ayah_number = ?
	`, editionID, surahNumber, ayahNumber).Scan(&r.SurahNumber, &r.SurahName, &r.AyahNumber, &r.Page, &r.Arabic, &r.Translation, &r.Text)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get %s tafsir %d:%d: %w", editionID, surahNumber, ayahNumber, err)
	}
	r.EditionID = editionID
	return &r, nil
}

// TafsirHeadByEdition returns the first limit ayahs of a surah with one
// edition's commentary, backing surah-name search hits.
func (s *Store) TafsirHeadByEdition(ctx context.Context, editionID string, surahNumber, limit int) ([]model.TafsirSearchResult, error) {
	if limit <= 0 {
		limit = 3
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT s.number, s.name_latin, a.ayah_number, a.page, a.text_arabic, a.translation, t.text
		FROM tafsir_texts t
		JOIN quran_ayahs a
		  ON a.surah_number = t.surah_number
		 AND a.ayah_number = t.ayah_number
		JOIN quran_surahs s ON s.number = t.surah_number
		WHERE t.edition_id = ? AND t.surah_number = ?
		ORDER BY a.ayah_number
		LIMIT ?
	`, editionID, surahNumber, limit)
	if err != nil {
		return nil, fmt.Errorf("read %s tafsir head for surah %d: %w", editionID, surahNumber, err)
	}
	defer rows.Close()

	var results []model.TafsirSearchResult
	for rows.Next() {
		var r model.TafsirSearchResult
		if err := rows.Scan(&r.SurahNumber, &r.SurahName, &r.AyahNumber, &r.Page, &r.Arabic, &r.Translation, &r.Text); err != nil {
			return nil, fmt.Errorf("scan tafsir head: %w", err)
		}
		r.EditionID = editionID
		results = append(results, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tafsir head for surah %d: %w", surahNumber, err)
	}
	return results, nil
}

// groupIdenticalTafsir folds runs of consecutive ayahs carrying byte-identical
// commentary into one card: the first ayah renders it with a "1–6" style
// range label, the rest render verse text only. Sources like the Quran.com
// Ibn Kathir edition attach one group block to every member ayah, so without
// this the same wall of text repeats once per ayah.
func groupIdenticalTafsir(ayahs []model.TafsirAyah) {
	for i := 0; i < len(ayahs); i++ {
		ayahs[i].TafsirFirst = true
		j := i
		for j+1 < len(ayahs) && ayahs[j+1].Tafsir == ayahs[i].Tafsir {
			j++
		}
		if j > i {
			ayahs[i].TafsirRange = fmt.Sprintf("%d–%d", ayahs[i].Number, ayahs[j].Number)
			for k := i + 1; k <= j; k++ {
				ayahs[k].TafsirFirst = false
			}
		}
		i = j
	}
}

// tafsirManifest mirrors a tafsir edition manifest.json.
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
// content/tafsir/NNN.json files (At-Tafsir Al-Muyassar, Arabic) and mirrors
// every row into tafsir_texts as edition "muyassar".
// It must run after seedQuran (higher order) because it UPDATEs ayah rows.
// The manifest checksum also folds in the quran content checksum, so any
// quran reseed (which clears ayah rows) triggers a tafsir reseed as well.
func seedTafsir(ctx context.Context, tx *sql.Tx, contentFS embed.FS, collectionID, path string, order int) error {
	return seedTafsirEdition(ctx, tx, contentFS, collectionID, path,
		"content/tafsir", "muyassar",
		"Tafsir Al-Muyassar", "Tafsir Al-Muyassar (Bahasa Arab) per ayat dari Quran.com API",
		true, order)
}

// seedTafsirIbnKathirEn seeds edition "ibn-kathir-en" (Ibn Kathir abridged,
// English, Quran.com resource 169) from content/tafsir-ibn-kathir-en/.
// The manifest lists only the surahs fetched so far, so a pilot (a few
// short surahs) and the later full 114-surah seed share one code path.
// It never touches the legacy quran_ayahs.tafsir column.
func seedTafsirIbnKathirEn(ctx context.Context, tx *sql.Tx, contentFS embed.FS, collectionID, path string, order int) error {
	return seedTafsirEdition(ctx, tx, contentFS, collectionID, path,
		"content/tafsir-ibn-kathir-en", "ibn-kathir-en",
		"Ibn Kathir (Abridged)", "Tafsir Ibn Kathir ringkas (Bahasa Inggris) per ayat dari Quran.com API",
		false, order)
}

// seedTafsirEdition seeds one commentary edition from dir/NNN.json files
// listed in the manifest at manifestPath. Edition rows come from migration
// 007; the seed only writes texts (INSERT OR REPLACE, so reseed and the
// migration backfill stay idempotent). With updateLegacy it additionally
// keeps the legacy quran_ayahs.tafsir column in sync (Muyassar only,
// while handlers/templates still read it).
func seedTafsirEdition(ctx context.Context, tx *sql.Tx, contentFS embed.FS, collectionID, manifestPath, dir, editionID, title, description string, updateLegacy bool, order int) error {
	data, err := contentFS.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("read %s: %w", manifestPath, err)
	}

	var manifest tafsirManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return fmt.Errorf("parse %s: %w", manifestPath, err)
	}

	if err := insertCollection(ctx, tx, collectionID, "quran", title, description, manifestPath, checksum(data)+quranChecksum(contentFS), order); err != nil {
		return err
	}

	for _, ms := range manifest.Surahs {
		surahPath := fmt.Sprintf("%s/%03d.json", dir, ms.Number)
		surahData, err := contentFS.ReadFile(surahPath)
		if err != nil {
			return fmt.Errorf("read %s tafsir surah %d: %w", editionID, ms.Number, err)
		}

		var surahFile struct {
			Ayahs []struct {
				Number int    `json:"number"`
				Tafsir string `json:"tafsir"`
			} `json:"ayahs"`
		}
		if err := json.Unmarshal(surahData, &surahFile); err != nil {
			return fmt.Errorf("parse %s tafsir surah %d: %w", editionID, ms.Number, err)
		}
		if len(surahFile.Ayahs) != ms.Ayahs {
			return fmt.Errorf("%s tafsir surah %d: manifest lists %d ayahs, file has %d", editionID, ms.Number, ms.Ayahs, len(surahFile.Ayahs))
		}

		for _, ayah := range surahFile.Ayahs {
			if ayah.Number <= 0 || ayah.Tafsir == "" {
				return fmt.Errorf("%s tafsir surah %d: invalid ayah entry #%d", editionID, ms.Number, ayah.Number)
			}
			if _, err := tx.ExecContext(ctx, `
				INSERT OR REPLACE INTO tafsir_texts (edition_id, surah_number, ayah_number, text)
				VALUES (?, ?, ?, ?)
			`, editionID, ms.Number, ayah.Number, ayah.Tafsir); err != nil {
				return fmt.Errorf("seed %s tafsir %d:%d: %w", editionID, ms.Number, ayah.Number, err)
			}
			if !updateLegacy {
				continue
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
