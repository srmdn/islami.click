package store_test

import (
	"context"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	islamiclick "github.com/srmdn/islami.click"
	"github.com/srmdn/islami.click/internal/store"

	_ "modernc.org/sqlite"
)

var tafsirTagRe = regexp.MustCompile(`<[a-zA-Z/][^>]*>`)

func stripAllowedTafsirMarks(s string) string {
	s = strings.ReplaceAll(s, `<mark class="tafsir-key">`, "")
	s = strings.ReplaceAll(s, `</mark>`, "")
	return s
}

func openTafsirTestStore(t *testing.T, ctx context.Context) *store.Store {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "tafsir.db")
	contentStore, err := store.Open(ctx, dbPath, islamiclick.MigrationFS, islamiclick.ContentFS)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { contentStore.Close() })
	return contentStore
}

func TestTafsirSeedCoversAllAyahs(t *testing.T) {
	ctx := context.Background()
	contentStore := openTafsirTestStore(t, ctx)

	covered, err := contentStore.TafsirCoverage(ctx)
	if err != nil {
		t.Fatalf("tafsir coverage: %v", err)
	}

	surahs, err := contentStore.QuranSurahs(ctx)
	if err != nil {
		t.Fatalf("quran surahs: %v", err)
	}
	if len(surahs) != 114 {
		t.Fatalf("surah count: got %d want 114", len(surahs))
	}

	total := 0
	for _, s := range surahs {
		got := covered[s.Number]
		if got != s.AyahCount {
			t.Fatalf("surah %d coverage: got %d want %d", s.Number, got, s.AyahCount)
		}
		total += got
	}
	if total != 6236 {
		t.Fatalf("total tafsir ayahs: got %d want 6236", total)
	}
}

func TestTafsirAyahsByMushafPage(t *testing.T) {
	ctx := context.Background()
	contentStore := openTafsirTestStore(t, ctx)

	ayahs, err := contentStore.TafsirAyahsByMushafPage(ctx, 1, 1)
	if err != nil {
		t.Fatalf("tafsir ayahs: %v", err)
	}
	if len(ayahs) != 7 {
		t.Fatalf("al-fatihah ayahs: got %d want 7", len(ayahs))
	}
	for _, a := range ayahs {
		if a.Arabic == "" || a.Translation == "" {
			t.Fatalf("ayah %d missing arabic/translation", a.Number)
		}
		if string(a.Tafsir) == "" {
			t.Fatalf("ayah %d missing tafsir", a.Number)
		}
		if tafsirTagRe.MatchString(stripAllowedTafsirMarks(string(a.Tafsir))) {
			t.Fatalf("ayah %d tafsir has unexpected html: %.80s", a.Number, string(a.Tafsir))
		}
	}
}
