package islamiclick_test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	iofs "io/fs"
	"net/url"
	"regexp"
	"strings"
	"testing"

	islamiclick "github.com/srmdn/islami.click"
	"github.com/srmdn/islami.click/internal/model"
)

func sha256Of(t *testing.T, data []byte) string {
	t.Helper()
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func TestDoaContentIsComplete(t *testing.T) {
	data, err := islamiclick.ContentFS.ReadFile("content/doa-harian.json")
	if err != nil {
		t.Fatalf("read doa content: %v", err)
	}

	var page model.DoaPageData
	if err := json.Unmarshal(data, &page); err != nil {
		t.Fatalf("parse doa content: %v", err)
	}

	if strings.TrimSpace(page.Title) == "" {
		t.Fatal("page title is required")
	}
	if len(page.Categories) == 0 {
		t.Fatal("at least one doa category is required")
	}

	for _, category := range page.Categories {
		if strings.TrimSpace(category.ID) == "" {
			t.Fatal("category id is required")
		}
		if strings.TrimSpace(category.Title) == "" {
			t.Fatalf("category %q title is required", category.ID)
		}
		if len(category.Items) == 0 {
			t.Fatalf("category %q must contain items", category.ID)
		}

		for _, item := range category.Items {
			if strings.TrimSpace(item.ID) == "" {
				t.Fatalf("category %q has item with empty id", category.ID)
			}
			checkRequired(t, item.ID, "title", item.Title)
			checkRequired(t, item.ID, "arabic", item.Arabic)
			checkRequired(t, item.ID, "latin", item.Latin)
			checkRequired(t, item.ID, "translation", item.Translation)
			checkRequired(t, item.ID, "source", item.Source)
			checkRequired(t, item.ID, "verification", item.Verification)

			sourceURL := strings.TrimSpace(item.SourceURL)
			if sourceURL == "" {
				t.Fatalf("item %q source_url is required", item.ID)
			}
			parsed, err := url.ParseRequestURI(sourceURL)
			if err != nil {
				t.Fatalf("item %q source_url is invalid: %v", item.ID, err)
			}
			if parsed.Scheme != "https" {
				t.Fatalf("item %q source_url must use https", item.ID)
			}
		}
	}
}

func checkRequired(t *testing.T, itemID, field, value string) {
	t.Helper()
	if strings.TrimSpace(value) == "" {
		t.Fatalf("item %q %s is required", itemID, field)
	}
}

func TestQuranReferencesAreCanonical(t *testing.T) {
	type surah struct {
		Transliteration string `json:"transliteration"`
	}

	surahData, err := islamiclick.ContentFS.ReadFile("content/quran-surahs.json")
	if err != nil {
		t.Fatalf("read quran-surahs.json: %v", err)
	}

	var surahs []surah
	if err := json.Unmarshal(surahData, &surahs); err != nil {
		t.Fatalf("parse quran-surahs.json: %v", err)
	}

	canonical := make(map[string]struct{}, len(surahs))
	for _, s := range surahs {
		canonical[s.Transliteration] = struct{}{}
	}

	re := regexp.MustCompile(`QS\.\s+([A-Za-z'\-]+(?:\s+[A-Za-z'\-]+)*)`)

	files := []string{
		"content/asmaul-husna.json",
		"content/almatsurat-kubro.json",
		"content/almatsurat-sugro.json",
		"content/ayat-doa-ruqyah.json",
		"content/doa-harian.json",
	}

	for _, file := range files {
		data, err := islamiclick.ContentFS.ReadFile(file)
		if err != nil {
			t.Fatalf("read %s: %v", file, err)
		}

		content := string(data)
		matches := re.FindAllStringSubmatchIndex(content, -1)
		for _, m := range matches {
			name := content[m[2]:m[3]]
			if _, ok := canonical[name]; !ok {
				line := 1
				for i := 0; i < m[0]; i++ {
					if content[i] == '\n' {
						line++
					}
				}
				t.Errorf("%s line %d: non-canonical surah name %q", file, line, name)
			}
		}
	}
}

func TestQuranContentAndPageMapAreComplete(t *testing.T) {
	type surahMeta struct {
		ID          int `json:"id"`
		TotalVerses int `json:"total_verses"`
	}
	type quranFile struct {
		Number    int `json:"number"`
		AyahCount int `json:"ayah_count"`
		Ayahs     []struct {
			Number int `json:"number"`
		} `json:"ayahs"`
	}

	surahData, err := islamiclick.ContentFS.ReadFile("content/quran-surahs.json")
	if err != nil {
		t.Fatalf("read quran-surahs.json: %v", err)
	}
	var surahs []surahMeta
	if err := json.Unmarshal(surahData, &surahs); err != nil {
		t.Fatalf("parse quran-surahs.json: %v", err)
	}
	if len(surahs) != 114 {
		t.Fatalf("surah count = %d, want 114", len(surahs))
	}

	totalAyahs := 0
	for _, meta := range surahs {
		path := fmt.Sprintf("content/quran/%03d.json", meta.ID)
		data, err := islamiclick.ContentFS.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		var detail quranFile
		if err := json.Unmarshal(data, &detail); err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		if detail.Number != meta.ID {
			t.Errorf("%s number = %d, want %d", path, detail.Number, meta.ID)
		}
		if detail.AyahCount != meta.TotalVerses || len(detail.Ayahs) != meta.TotalVerses {
			t.Errorf("%s ayah count = %d/%d, metadata = %d", path, detail.AyahCount, len(detail.Ayahs), meta.TotalVerses)
		}
		totalAyahs += len(detail.Ayahs)
	}
	if totalAyahs != 6236 {
		t.Fatalf("total ayah count = %d, want 6236", totalAyahs)
	}

	pageData, err := islamiclick.ContentFS.ReadFile("content/quran-pages.json")
	if err != nil {
		t.Fatalf("read quran-pages.json: %v", err)
	}
	var pages map[string]int
	if err := json.Unmarshal(pageData, &pages); err != nil {
		t.Fatalf("parse quran-pages.json: %v", err)
	}
	if len(pages) != totalAyahs {
		t.Fatalf("page map entries = %d, want %d", len(pages), totalAyahs)
	}
	for key, page := range pages {
		if page < 1 || page > 604 {
			t.Errorf("page map %q = %d, want 1..604", key, page)
		}
	}
	for _, meta := range surahs {
		path := fmt.Sprintf("content/quran/%03d.json", meta.ID)
		data, err := islamiclick.ContentFS.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		var detail quranFile
		if err := json.Unmarshal(data, &detail); err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		for _, ayah := range detail.Ayahs {
			key := fmt.Sprintf("%d:%d", meta.ID, ayah.Number)
			if _, ok := pages[key]; !ok {
				t.Errorf("missing page map entry for %s", key)
			}
		}
	}
}

func TestTafsirMuyassarContentIsComplete(t *testing.T) {
	type manifestSurah struct {
		Number int    `json:"number"`
		Ayahs  int    `json:"ayahs"`
		SHA256 string `json:"sha256"`
	}
	type manifest struct {
		Resource string          `json:"resource"`
		Language string          `json:"language"`
		Surahs   []manifestSurah `json:"surahs"`
	}
	type tafsirFile struct {
		Number int `json:"number"`
		Ayahs  []struct {
			Number int    `json:"number"`
			Tafsir string `json:"tafsir"`
		} `json:"ayahs"`
	}

	manifestData, err := islamiclick.ContentFS.ReadFile("content/tafsir/manifest.json")
	if err != nil {
		t.Fatalf("read tafsir manifest: %v", err)
	}
	var m manifest
	if err := json.Unmarshal(manifestData, &m); err != nil {
		t.Fatalf("parse tafsir manifest: %v", err)
	}
	if m.Resource == "" || m.Language != "arabic" {
		t.Fatalf("tafsir manifest resource/language unexpected: %q/%q", m.Resource, m.Language)
	}
	if len(m.Surahs) != 114 {
		t.Fatalf("tafsir manifest surah count = %d, want 114", len(m.Surahs))
	}

	totalAyahs := 0
	for _, ms := range m.Surahs {
		path := fmt.Sprintf("content/tafsir/%03d.json", ms.Number)
		data, err := islamiclick.ContentFS.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		if got := sha256Of(t, data); got != ms.SHA256 {
			t.Errorf("%s sha256 mismatch: manifest %s, file %s", path, ms.SHA256, got)
		}
		var detail tafsirFile
		if err := json.Unmarshal(data, &detail); err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		if detail.Number != ms.Number {
			t.Errorf("%s number = %d, want %d", path, detail.Number, ms.Number)
		}
		if len(detail.Ayahs) != ms.Ayahs {
			t.Errorf("%s ayah count = %d, manifest = %d", path, len(detail.Ayahs), ms.Ayahs)
		}
		for i, ayah := range detail.Ayahs {
			if ayah.Number != i+1 {
				t.Errorf("%s ayah order broken at index %d (got #%d)", path, i, ayah.Number)
			}
			if strings.TrimSpace(ayah.Tafsir) == "" {
				t.Errorf("%s ayah %d has empty tafsir", path, ayah.Number)
			}
		}
		totalAyahs += len(detail.Ayahs)
	}
	if totalAyahs != 6236 {
		t.Fatalf("total tafsir ayah count = %d, want 6236", totalAyahs)
	}
}

func TestQuizContentMatchesPublishedContract(t *testing.T) {
	type question struct {
		Question    string   `json:"q"`
		Options     []string `json:"options"`
		Answer      int      `json:"answer"`
		Explanation string   `json:"explanation"`
	}
	type category struct {
		Slug      string `json:"slug"`
		Questions struct {
			Basic        []question `json:"basic"`
			Intermediate []question `json:"intermediate"`
			Advanced     []question `json:"advanced"`
		} `json:"questions"`
	}

	entries, err := iofs.ReadDir(islamiclick.ContentFS, "content/quiz")
	if err != nil {
		t.Fatalf("read quiz content directory: %v", err)
	}
	expected := map[string]int{"basic": 15, "intermediate": 15, "advanced": 15}
	categoryCount := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		categoryCount++
		path := "content/quiz/" + entry.Name()
		data, err := islamiclick.ContentFS.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		var cat category
		if err := json.Unmarshal(data, &cat); err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		if strings.TrimSpace(cat.Slug) == "" {
			t.Errorf("%s has empty slug", path)
		}
		difficulties := map[string][]question{
			"basic":        cat.Questions.Basic,
			"intermediate": cat.Questions.Intermediate,
			"advanced":     cat.Questions.Advanced,
		}
		for difficulty, questions := range difficulties {
			if len(questions) != expected[difficulty] {
				t.Errorf("%s %s question count = %d, want %d", path, difficulty, len(questions), expected[difficulty])
			}
			for index, q := range questions {
				if strings.TrimSpace(q.Question) == "" || strings.TrimSpace(q.Explanation) == "" {
					t.Errorf("%s %s question %d has empty text or explanation", path, difficulty, index+1)
				}
				if len(q.Options) != 4 || q.Answer < 0 || q.Answer >= len(q.Options) {
					t.Errorf("%s %s question %d has invalid options/answer", path, difficulty, index+1)
				}
				for optionIndex, option := range q.Options {
					if strings.TrimSpace(option) == "" {
						t.Errorf("%s %s question %d option %d is empty", path, difficulty, index+1, optionIndex+1)
					}
				}
			}
		}
	}
	if categoryCount != 8 {
		t.Fatalf("quiz category count = %d, want 8", categoryCount)
	}
}
