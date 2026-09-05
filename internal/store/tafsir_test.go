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

// enTafsirTagRe strips the fetch allowlist (block/inline structure tags with
// any attributes) so the remainder must be tag-free.
var enTafsirTagRe = regexp.MustCompile(`</?(?:p|h2|strong|em|br|mark)(?:\s[^>]*)?>`)

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

func TestTafsirEditionsSeeded(t *testing.T) {
	ctx := context.Background()
	contentStore := openTafsirTestStore(t, ctx)

	editions, err := contentStore.TafsirEditions(ctx)
	if err != nil {
		t.Fatalf("tafsir editions: %v", err)
	}
	byID := map[string]string{}
	for _, e := range editions {
		byID[e.ID] = e.Language
		if e.Title == "" || e.Source == "" {
			t.Fatalf("edition %s missing title/source", e.ID)
		}
	}
	if byID["muyassar"] != "arabic" {
		t.Fatalf("muyassar edition language = %q, want arabic", byID["muyassar"])
	}
	if byID["ibn-kathir-en"] != "english" {
		t.Fatalf("ibn-kathir-en edition language = %q, want english", byID["ibn-kathir-en"])
	}
}

func TestTafsirCoverageByEdition(t *testing.T) {
	ctx := context.Background()
	contentStore := openTafsirTestStore(t, ctx)

	muyassar, err := contentStore.TafsirCoverageByEdition(ctx, "muyassar")
	if err != nil {
		t.Fatalf("muyassar coverage: %v", err)
	}
	total := 0
	for _, n := range muyassar {
		total += n
	}
	if total != 6236 {
		t.Fatalf("muyassar total = %d, want 6236", total)
	}

	full, err := contentStore.TafsirCoverageByEdition(ctx, "ibn-kathir-en")
	if err != nil {
		t.Fatalf("ibn-kathir-en coverage: %v", err)
	}
	for surah, want := range map[int]int{1: 7, 2: 286, 112: 4, 113: 5, 114: 6} {
		if full[surah] != want {
			t.Fatalf("ibn-kathir-en surah %d coverage = %d, want %d", surah, full[surah], want)
		}
	}
	totalEn := 0
	for _, n := range full {
		totalEn += n
	}
	if totalEn != 6236 {
		t.Fatalf("ibn-kathir-en total = %d, want 6236", totalEn)
	}
}

func TestTafsirAyahsByEditionPage(t *testing.T) {
	ctx := context.Background()
	contentStore := openTafsirTestStore(t, ctx)

	pages, err := contentStore.MushafPagesForSurah(ctx, 114)
	if err != nil || len(pages) == 0 {
		t.Fatalf("mushaf pages for 114: %v", err)
	}

	muyassar, err := contentStore.TafsirAyahsByEditionPage(ctx, "muyassar", 114, pages[0])
	if err != nil {
		t.Fatalf("muyassar ayahs: %v", err)
	}
	if len(muyassar) != 6 {
		t.Fatalf("muyassar ayahs = %d, want 6", len(muyassar))
	}
	for _, a := range muyassar {
		if !a.TafsirFirst || a.TafsirRange != "" {
			t.Fatalf("muyassar ayah %d should render its own card (distinct per-ayah text)", a.Number)
		}
	}

	en, err := contentStore.TafsirAyahsByEditionPage(ctx, "ibn-kathir-en", 114, pages[0])
	if err != nil {
		t.Fatalf("ibn-kathir-en ayahs: %v", err)
	}
	if len(en) != 6 {
		t.Fatalf("ibn-kathir-en ayahs = %d, want 6", len(en))
	}
	for i, a := range en {
		if string(a.Tafsir) == "" {
			t.Fatalf("ayah %d missing english tafsir", a.Number)
		}
		if string(a.Tafsir) == string(muyassar[i].Tafsir) {
			t.Fatalf("ayah %d english text identical to muyassar (wrong edition?)", a.Number)
		}
		if tafsirTagRe.MatchString(enTafsirTagRe.ReplaceAllString(stripAllowedTafsirMarks(string(a.Tafsir)), "")) {
			t.Fatalf("ayah %d english tafsir has non-allowlist html: %.80s", a.Number, string(a.Tafsir))
		}
	}
	if !strings.Contains(string(en[0].Tafsir), "<h2>Which was revealed in Makkah</h2>") {
		t.Fatal("english tafsir lost its heading structure")
	}
	if !strings.Contains(string(en[0].Tafsir), `<p dir="rtl" lang="ar" class="tafsir-ar">`) {
		t.Fatal("english tafsir lost its RTL quote paragraph")
	}
	if strings.Contains(string(en[0].Tafsir), "Makkahب") {
		t.Fatal("english tafsir glued heading to arabic (structure stripped)")
	}
	// Only allowlist tags may remain.
	clean := enTafsirTagRe.ReplaceAllString(stripAllowedTafsirMarks(string(en[0].Tafsir)), "")
	if tafsirTagRe.MatchString(clean) {
		t.Fatalf("english tafsir has non-allowlist html: %.80s", clean)
	}
	if !en[0].TafsirFirst || en[0].TafsirRange != "1–6" {
		t.Fatalf("ayah 1 should open the 1–6 group card, got first=%v range=%q", en[0].TafsirFirst, en[0].TafsirRange)
	}
	for _, a := range en[1:] {
		if a.TafsirFirst {
			t.Fatalf("ayah %d repeats the group text and must not render a card", a.Number)
		}
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
