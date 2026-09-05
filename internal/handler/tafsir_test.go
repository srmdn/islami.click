package handler

import (
	"context"
	"encoding/json"
	"html/template"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"

	islamiclick "github.com/srmdn/islami.click"
	"github.com/srmdn/islami.click/internal/store"
)

func newTafsirTestHandler(t *testing.T) *Handler {
	t.Helper()
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "tafsir-handler.db")
	contentStore, err := store.Open(ctx, dbPath, islamiclick.MigrationFS, islamiclick.ContentFS)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { contentStore.Close() })

	funcMap := template.FuncMap{
		"add": func(a, b int) int { return a + b },
		"hasPrefix": func(s, prefix string) bool {
			return len(s) >= len(prefix) && s[:len(prefix)] == prefix
		},
		"arabicHTML": func(s string) template.HTML {
			return template.HTML(template.HTMLEscapeString(s))
		},
		"tafsirHTML": TafsirHTML,
		"tafsirHTMLFor": TafsirHTMLFor,
		"arabicDigits": ArabicDigits,
		"js": func(s string) template.JS {
			encoded, _ := json.Marshal(s)
			return template.JS(encoded)
		},
		"jsonLD": func(s string) template.JS { return template.JS(s) },
	}

	tmpls := make(map[string]*template.Template)
	for _, page := range []string{"tafsir.html", "tafsir-surah.html", "tafsir-search.html", "quran-surah.html"} {
		tpl := template.New(page).Funcs(funcMap)
		tpl, err := tpl.ParseFS(islamiclick.TemplateFS,
			"templates/layouts/base.html",
			"templates/partials/header.html",
			"templates/partials/footer.html",
			"templates/pages/"+page,
		)
		if err != nil {
			t.Fatalf("parse %s: %v", page, err)
		}
		tmpls[page] = tpl
	}

	partialTmpls := make(map[string]*template.Template)
	for _, partial := range []string{"quran-ayahs", "tafsir-peek"} {
		tpl := template.New(partial).Funcs(funcMap)
		tpl, err := tpl.ParseFS(islamiclick.TemplateFS, "templates/partials/"+partial+".html")
		if err != nil {
			t.Fatalf("parse %s: %v", partial, err)
		}
		partialTmpls[partial] = tpl
	}

	return New(tmpls, partialTmpls, contentStore)
}

func TestTafsirIndexRenders(t *testing.T) {
	h := newTafsirTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/tafsir", nil)
	rec := httptest.NewRecorder()
	h.Tafsir(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{"Tafsir Al-Muyassar", "/tafsir/1", "/tafsir/114", "Quran.com", "Pilih kitab tafsir"} {
		if !strings.Contains(body, want) {
			t.Fatalf("index missing %q", want)
		}
	}
}

func TestTafsirSurahRenders(t *testing.T) {
	h := newTafsirTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/tafsir/1", nil)
	rec := httptest.NewRecorder()
	h.TafsirSurah(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{"Tafsir Al-Muyassar", "tafsir-key", "Tafsir Berikutnya", "﴾", "﴿", "tafsirShare($el)", `data-share-url="/tafsir/1#ayah-1"`, "Gagal menyalin", "x-data", `class="ayah-end"`, ">١<"} {
		if !strings.Contains(body, want) {
			t.Fatalf("surah page missing %q", want)
		}
	}
	for _, bad := range []string{"\u200E", "\u200F", "\u202A", "\u202B", "\u202C", "\u2066", "\u2067", "\u2069"} {
		if strings.Contains(body, bad) {
			t.Fatalf("surah page leaks bidi control U+%04X into copy-pasteable text", []rune(bad)[0])
		}
	}
	if strings.Contains(body, "onerror=") || strings.Contains(body, "onclick=") {
		t.Fatal("surah page contains suspicious inline handlers")
	}
}

func TestTafsirIndexEditionSwitcher(t *testing.T) {
	h := newTafsirTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/tafsir?edition=ibn-kathir-en", nil)
	rec := httptest.NewRecorder()
	h.Tafsir(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{"Ibn Kathir (Abridged)", "Tafsir Al-Muyassar", "?edition=ibn-kathir-en", "4/4 tafsir", "286/286 tafsir"} {
		if !strings.Contains(body, want) {
			t.Fatalf("english index missing %q", want)
		}
	}
}

func TestTafsirIndexUnknownEditionFallsBack(t *testing.T) {
	h := newTafsirTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/tafsir?edition=nope", nil)
	rec := httptest.NewRecorder()
	h.Tafsir(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if body := rec.Body.String(); !strings.Contains(body, "Tafsir Al-Muyassar") {
		t.Fatal("unknown edition should fall back to Muyassar")
	}
}

func TestTafsirSurahEnglishRenders(t *testing.T) {
	h := newTafsirTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/tafsir/114?edition=ibn-kathir-en", nil)
	rec := httptest.NewRecorder()
	h.TafsirSurah(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{"Ibn Kathir (Abridged)", `lang="en"`, "Which was revealed in Makkah", "Quran.com API (resource 169)", "Ayat 1–6", "<h2>Which was revealed in Makkah</h2>", `data-share-url="/tafsir/114?edition=ibn-kathir-en#ayah-1"`, "tafsirShare($el)"} {
		if !strings.Contains(body, want) {
			t.Fatalf("english surah page missing %q", want)
		}
	}
	if got := strings.Count(body, "Which was revealed in Makkah"); got != 1 {
		t.Fatalf("group text rendered %d times, want once", got)
	}
	for _, bad := range []string{"﴾", "﴿", "\u200E", "\u200F", "\u202A", "\u202B", "\u202C", "\u2066", "\u2067", "\u2069"} {
		if strings.Contains(body, bad) {
			t.Fatalf("english surah page leaks %q into latin text", bad)
		}
	}
}

func TestTafsirSurahEnglishFullCoverage(t *testing.T) {
	h := newTafsirTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/tafsir/1?edition=ibn-kathir-en", nil)
	rec := httptest.NewRecorder()
	h.TafsirSurah(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{`lang="en"`, "Ibn Kathir (Abridged)"} {
		if !strings.Contains(body, want) {
			t.Fatalf("english surah page missing %q", want)
		}
	}
	for _, bad := range []string{"belum tersedia", "﴾", "﴿"} {
		if strings.Contains(body, bad) {
			t.Fatalf("english surah page wrongly contains %q", bad)
		}
	}
}

func TestTafsirSurahNotFound(t *testing.T) {
	h := newTafsirTestHandler(t)

	for _, path := range []string{"/tafsir/0", "/tafsir/115", "/tafsir/abc"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		// Route prefix is stripped by the mux in main.go; emulate it here.
		req.URL.Path = strings.TrimPrefix(path, "/tafsir")
		if req.URL.Path == "" {
			req.URL.Path = "/"
		}
		rec := httptest.NewRecorder()
		h.TafsirSurah(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("%s status = %d, want 404", path, rec.Code)
		}
	}
}

func TestTafsirSearchRendersForm(t *testing.T) {
	h := newTafsirTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/tafsir/search", nil)
	rec := httptest.NewRecorder()
	h.TafsirSearch(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{"Pencarian Tafsir", `action="/tafsir/search"`, "Pilih kitab tafsir", "Ibn Kathir (Abridged)", "Cari penjelasan..."} {
		if !strings.Contains(body, want) {
			t.Fatalf("search page missing %q", want)
		}
	}
	if strings.Contains(body, "penjelasan ditemukan") {
		t.Fatal("empty query must not render a result count")
	}
}

func TestTafsirSearchFindsEnglish(t *testing.T) {
	h := newTafsirTestHandler(t)

	// "dangling clot" avoids standalone glue words: the shared reference
	// grammar eats "to" as surah 110 ("Pertolongan"), a pre-existing
	// /quran/search quirk, so multi-word EN queries with "to" resolve oddly.
	req := httptest.NewRequest(http.MethodGet, "/tafsir/search?edition=ibn-kathir-en&q="+url.QueryEscape("dangling clot"), nil)
	rec := httptest.NewRecorder()
	h.TafsirSearch(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{`lang="en"`, `<mark class="tafsir-key">`, `/tafsir/96`, `edition=ibn-kathir-en`, "penjelasan ditemukan"} {
		if !strings.Contains(body, want) {
			t.Fatalf("english search missing %q", want)
		}
	}
	for _, bad := range []string{"<h2>", "﴾", "﴿"} {
		if strings.Contains(body, bad) {
			t.Fatalf("english snippet leaks %q", bad)
		}
	}
}

func TestTafsirSearchDirectReference(t *testing.T) {
	h := newTafsirTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/tafsir/search?edition=ibn-kathir-en&q="+url.QueryEscape("114:1"), nil)
	rec := httptest.NewRecorder()
	h.TafsirSearch(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if body := rec.Body.String(); !strings.Contains(body, `/tafsir/114?edition=ibn-kathir-en#ayah-1`) {
		t.Fatal("direct reference 114:1 should deep-link without a page param on the single-page surah")
	}
}

func TestTafsirSearchArabicFindsKursi(t *testing.T) {
	h := newTafsirTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/tafsir/search?q="+url.QueryEscape("الألوهية"), nil)
	rec := httptest.NewRecorder()
	h.TafsirSearch(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{"/tafsir/2?page=", "#ayah-255", `<mark class="tafsir-key">`, `dir="rtl"`} {
		if !strings.Contains(body, want) {
			t.Fatalf("arabic search missing %q", want)
		}
	}
	if !utf8.ValidString(body) {
		t.Fatal("arabic search page is not valid UTF-8")
	}
}

func TestTafsirSearchUnknownEditionFallsBack(t *testing.T) {
	h := newTafsirTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/tafsir/search?edition=nope&q="+url.QueryEscape("الألوهية"), nil)
	rec := httptest.NewRecorder()
	h.TafsirSearch(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if body := rec.Body.String(); !strings.Contains(body, "Tafsir Al-Muyassar") {
		t.Fatal("unknown edition should fall back to Muyassar")
	}
}

func TestTafsirSearchIndonesianViaTranslation(t *testing.T) {
	h := newTafsirTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/tafsir/search?q="+url.QueryEscape("Kursi"), nil)
	rec := httptest.NewRecorder()
	h.TafsirSearch(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{"/tafsir/2?page=", "#ayah-255", "penjelasan ditemukan"} {
		if !strings.Contains(body, want) {
			t.Fatalf("indonesian search missing %q", want)
		}
	}
}

func TestTafsirSnippetStripsTagsAndMarks(t *testing.T) {
	raw := `<h2>Which was revealed in Makkah</h2><p>In the Name of Allah.</p>`
	out := string(tafsirSnippet(raw, "makkah"))
	if !strings.Contains(out, `<mark class="tafsir-key">Makkah</mark>`) {
		t.Fatalf("snippet missing highlight: %q", out)
	}
	for _, bad := range []string{"<h2>", "</h2>", "<p>", "</p>"} {
		if strings.Contains(out, bad) {
			t.Fatalf("snippet leaks tag %q: %q", bad, out)
		}
	}
}

func TestQuranTafsirPeekRenders(t *testing.T) {
	h := newTafsirTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/quran/1/1/tafsir", nil)
	rec := httptest.NewRecorder()
	h.QuranSurah(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{"Tafsir Al-Muyassar", "tafsir-body", "Buka tafsir lengkap", `/tafsir/1?page=`, "#ayah-1"} {
		if !strings.Contains(body, want) {
			t.Fatalf("peek fragment missing %q", want)
		}
	}
	// Fragment only: none of the full-page chrome may leak in.
	for _, bad := range []string{"Murottal", "Beranda", "Surah Berikutnya"} {
		if strings.Contains(body, bad) {
			t.Fatalf("peek fragment leaks page chrome %q", bad)
		}
	}
	for _, bad := range []string{"\u200E", "\u200F", "\u202A", "\u202B", "\u202C", "\u2066", "\u2067", "\u2069"} {
		if strings.Contains(body, bad) {
			t.Fatalf("peek leaks bidi control U+%04X", []rune(bad)[0])
		}
	}
	if strings.Contains(body, "onerror=") || strings.Contains(body, "onclick=") {
		t.Fatal("peek fragment contains suspicious inline handlers")
	}
}

func TestQuranTafsirPeekNotFound(t *testing.T) {
	h := newTafsirTestHandler(t)

	for _, path := range []string{"/quran/1/99/tafsir", "/quran/115/1/tafsir"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		h.QuranSurah(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("%s status = %d, want 404", path, rec.Code)
		}
	}

	// Existing surah route still intact beside the peek shape.
	req := httptest.NewRequest(http.MethodGet, "/quran/2", nil)
	rec := httptest.NewRecorder()
	h.QuranSurah(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("/quran/2 status = %d, want 200", rec.Code)
	}
}

func TestQuranSurahPeekEnhancement(t *testing.T) {
	h := newTafsirTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/quran/114", nil)
	rec := httptest.NewRecorder()
	h.QuranSurah(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{
		`x-data="{open:false}"`,
		`hx-get="/quran/114/1/tafsir"`,
		`hx-trigger="click once"`,
		`hx-target="#tafsir-peek-1"`,
		`aria-controls="tafsir-peek-1"`,
		`:aria-expanded=`,
		`x-show="open"`,
		`Memuat tafsir…`,
		// No-JS fallback stays a plain deep-link.
		`href="/tafsir/114?page=`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("surah page missing peek wiring %q", want)
		}
	}
}

func TestQuranAyahsPartialPeekEnhancement(t *testing.T) {
	h := newTafsirTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/quran/114", nil)
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()
	h.QuranSurah(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if body := rec.Body.String(); !strings.Contains(body, `hx-get="/quran/114/1/tafsir"`) {
		t.Fatal("htmx ayah partial missing peek wiring")
	}
}

func TestTafsirSnippetCentersAndTruncates(t *testing.T) {
	raw := strings.Repeat("kata ", 200) + "TARGET" + strings.Repeat(" kata", 200)
	out := string(tafsirSnippet(raw, "target"))
	if !strings.HasPrefix(out, "… ") || !strings.HasSuffix(out, " …") {
		t.Fatalf("long snippet must truncate both ends: %.60q…", out)
	}
	if !strings.Contains(out, `<mark class="tafsir-key">TARGET</mark>`) {
		t.Fatalf("snippet must center on the match: %.120q", out)
	}
	if n := len([]rune(out)); n > tafsirSnippetWidth+64 {
		t.Fatalf("snippet too long: %d runes", n)
	}
}

func TestTafsirSnippetEscapesStrayMarkup(t *testing.T) {
	// Tags are stripped before windowing, so a bare "<" that forms no
	// tag must survive as escaped text instead.
	out := string(tafsirSnippet("iman 5 < 7 kuat", "kuat"))
	if strings.Contains(out, "< 7") {
		t.Fatalf("snippet leaks raw markup: %q", out)
	}
	if !strings.Contains(out, "&lt;") {
		t.Fatalf("snippet must escape stray markup: %q", out)
	}
	if !strings.Contains(out, `<mark class="tafsir-key">kuat</mark>`) {
		t.Fatalf("snippet missing highlight: %q", out)
	}
}
