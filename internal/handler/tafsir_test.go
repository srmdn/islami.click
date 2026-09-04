package handler

import (
	"context"
	"encoding/json"
	"html/template"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

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
		"js": func(s string) template.JS {
			encoded, _ := json.Marshal(s)
			return template.JS(encoded)
		},
		"jsonLD": func(s string) template.JS { return template.JS(s) },
	}

	tmpls := make(map[string]*template.Template)
	for _, page := range []string{"tafsir.html", "tafsir-surah.html"} {
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

	return New(tmpls, nil, contentStore)
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
	for _, want := range []string{"Tafsir Al-Muyassar", "/tafsir/1", "/tafsir/114", "Quran.com"} {
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
	for _, want := range []string{"Tafsir Al-Muyassar", "tafsir-key", "Tafsir Berikutnya", "﴾", "﴿"} {
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
