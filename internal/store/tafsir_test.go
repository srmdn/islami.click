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

func TestSearchTafsir(t *testing.T) {
	ctx := context.Background()
	contentStore := openTafsirTestStore(t, ctx)

	en, err := contentStore.SearchTafsir(ctx, "ibn-kathir-en", "Which was revealed in Makkah", 50)
	if err != nil {
		t.Fatalf("search en tafsir: %v", err)
	}
	if len(en) == 0 {
		t.Fatal("expected english hits for the surah-114 heading")
	}
	// SQLite LIKE is case-insensitive for ASCII, so compare folded.
	foldedQuery := strings.ToLower("Which was revealed in Makkah")
	for _, r := range en {
		if r.EditionID != "ibn-kathir-en" {
			t.Fatalf("hit edition = %q, want ibn-kathir-en", r.EditionID)
		}
		if r.Page <= 0 || r.SurahName == "" || r.Arabic == "" || r.Translation == "" || r.Text == "" {
			t.Fatalf("hit %d:%d missing context fields", r.SurahNumber, r.AyahNumber)
		}
		if !strings.Contains(strings.ToLower(r.Text), foldedQuery) {
			t.Fatalf("hit %d:%d text lacks query", r.SurahNumber, r.AyahNumber)
		}
	}
	// The heading phrase above repeats on every ayah of every Makki
	// surah (group text is stored per ayah), so limit 50 only exercises
	// the field checks. Precision is checked below with a phrase unique
	// to surah 114.
	distinct, err := contentStore.SearchTafsir(ctx, "ibn-kathir-en", "lordship, sovereignty and divinity", 50)
	if err != nil {
		t.Fatalf("distinct search en tafsir: %v", err)
	}
	if len(distinct) != 6 {
		t.Fatalf("distinct hits = %d, want 6 (all of surah 114)", len(distinct))
	}
	for i, r := range distinct {
		if r.SurahNumber != 114 || r.AyahNumber != i+1 {
			t.Fatalf("distinct hit %d: got %d:%d, want 114:%d", i, r.SurahNumber, r.AyahNumber, i+1)
		}
		if r.Page <= 0 {
			t.Fatalf("distinct hit 114:%d missing page", r.AyahNumber)
		}
	}

	ar, err := contentStore.SearchTafsir(ctx, "muyassar", "الألوهية والعبودية", 50)
	if err != nil {
		t.Fatalf("search ar tafsir: %v", err)
	}
	foundKursi := false
	for _, r := range ar {
		if r.SurahNumber == 2 && r.AyahNumber == 255 {
			foundKursi = true
		}
	}
	if !foundKursi {
		t.Fatal("expected a 2:255 hit for the Muyassar kursi phrase")
	}

	capped, err := contentStore.SearchTafsir(ctx, "muyassar", "الله", 5)
	if err != nil {
		t.Fatalf("capped search: %v", err)
	}
	if len(capped) != 5 {
		t.Fatalf("capped hits = %d, want 5", len(capped))
	}

	none, err := contentStore.SearchTafsir(ctx, "nope", "الله", 10)
	if err != nil {
		t.Fatalf("unknown edition search: %v", err)
	}
	if len(none) != 0 {
		t.Fatalf("unknown edition hits = %d, want 0", len(none))
	}
}

func TestGetTafsirAyah(t *testing.T) {
	ctx := context.Background()
	contentStore := openTafsirTestStore(t, ctx)

	r, err := contentStore.GetTafsirAyah(ctx, "muyassar", 2, 255)
	if err != nil {
		t.Fatalf("get tafsir ayah: %v", err)
	}
	if r == nil {
		t.Fatal("expected 2:255 Muyassar text, got nil")
	}
	if r.SurahNumber != 2 || r.AyahNumber != 255 || r.Page <= 0 {
		t.Fatalf("got %d:%d page %d", r.SurahNumber, r.AyahNumber, r.Page)
	}
	if r.Arabic == "" || r.Translation == "" || r.Text == "" {
		t.Fatal("2:255 missing arabic/translation/text")
	}

	miss, err := contentStore.GetTafsirAyah(ctx, "muyassar", 114, 7)
	if err != nil {
		t.Fatalf("get missing ayah: %v", err)
	}
	if miss != nil {
		t.Fatal("expected nil for 114:7, got a row")
	}

	missEdition, err := contentStore.GetTafsirAyah(ctx, "nope", 1, 1)
	if err != nil {
		t.Fatalf("get unknown edition: %v", err)
	}
	if missEdition != nil {
		t.Fatal("expected nil for unknown edition, got a row")
	}
}

func TestTafsirHeadByEdition(t *testing.T) {
	ctx := context.Background()
	contentStore := openTafsirTestStore(t, ctx)

	head, err := contentStore.TafsirHeadByEdition(ctx, "ibn-kathir-en", 114, 3)
	if err != nil {
		t.Fatalf("tafsir head: %v", err)
	}
	if len(head) != 3 {
		t.Fatalf("head rows = %d, want 3", len(head))
	}
	for i, r := range head {
		if r.AyahNumber != i+1 || r.Text == "" || r.Page <= 0 {
			t.Fatalf("head row %d: ayah=%d page=%d empty=%v", i, r.AyahNumber, r.Page, r.Text == "")
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
