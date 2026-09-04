package main

import (
	"context"
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	islamiclick "github.com/srmdn/islami.click"
	"github.com/srmdn/islami.click/internal/handler"
	"github.com/srmdn/islami.click/internal/store"
)

var arabicParenRe = regexp.MustCompile(`\(([^()]*)\)`)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	contentStore, err := store.Open(context.Background(), os.Getenv("DB_PATH"), islamiclick.MigrationFS, islamiclick.ContentFS)
	if err != nil {
		log.Fatalf("open content store: %v", err)
	}
	defer contentStore.Close()

	funcMap := template.FuncMap{
		"add": func(a, b int) int { return a + b },
		"hasPrefix": func(s, prefix string) bool { return strings.HasPrefix(s, prefix) },
		"arabicHTML": func(s string) template.HTML {
			escaped := template.HTMLEscapeString(s)
			result := arabicParenRe.ReplaceAllString(escaped, `<span dir="ltr">(<bdi>$1</bdi>)</span>`)
			return template.HTML(result)
		},
		"js": func(s string) template.JS {
			encoded, _ := json.Marshal(s)
			return template.JS(encoded)
		},
		"jsonLD": func(s string) template.JS { return template.JS(s) },
	}

	pages := []string{
		"home.html",
		"almatsurat.html",
		"almatsurat-sugro.html",
		"almatsurat-kubro.html",
		"doa.html",
		"shalat.html",
		"asmaul-husna.html",
		"kiblat.html",
		"hisab.html",
		"quran.html",
		"quran-surah.html",
		"quran-search.html",
		"quiz.html",
		"quiz-category.html",
	}

	tmpls := make(map[string]*template.Template)
	for _, page := range pages {
		tpl := template.New(page).Funcs(funcMap)
		tpl, err := tpl.ParseFS(islamiclick.TemplateFS,
			"templates/layouts/base.html",
			"templates/partials/header.html",
			"templates/partials/footer.html",
			"templates/pages/"+page,
		)
		if err != nil {
			log.Fatalf("parse %s: %v", page, err)
		}
		tmpls[page] = tpl
	}

	partialTmpls := make(map[string]*template.Template)
	{
		tpl := template.New("shalat-mini").Funcs(funcMap)
		tpl, err := tpl.ParseFS(islamiclick.TemplateFS, "templates/partials/shalat-mini.html")
		if err != nil {
			log.Fatalf("parse shalat-mini: %v", err)
		}
		partialTmpls["shalat-mini"] = tpl
	}
	{
		tpl := template.New("doa-more").Funcs(funcMap)
		tpl, err := tpl.ParseFS(islamiclick.TemplateFS, "templates/partials/doa-more.html")
		if err != nil {
			log.Fatalf("parse doa-more: %v", err)
		}
		partialTmpls["doa-more"] = tpl
	}
	{
		tpl := template.New("quran-ayahs").Funcs(funcMap)
		tpl, err := tpl.ParseFS(islamiclick.TemplateFS, "templates/partials/quran-ayahs.html")
		if err != nil {
			log.Fatalf("parse quran-ayahs: %v", err)
		}
		partialTmpls["quran-ayahs"] = tpl
	}

	h := handler.New(tmpls, partialTmpls, contentStore)

	fs := http.Dir("static")
	staticHandler := http.StripPrefix("/static/", http.FileServer(fs))
	http.Handle("/static/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/") {
			http.NotFound(w, r)
			return
		}
		staticHandler.ServeHTTP(w, r)
	}))

	http.HandleFunc("/robots.txt", h.RobotsTxt)
	http.HandleFunc("/sitemap.xml", h.Sitemap)
	http.HandleFunc("/llms.txt", h.LLMsTxt)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		h.Home(w, r)
	})
	http.HandleFunc("/almatsurat", h.AlMatsurat)
	http.HandleFunc("/almatsurat/sugro", h.AlMatsuratSugro)
	http.HandleFunc("/almatsurat/kubro", h.AlMatsuratKubro)
	http.HandleFunc("/doa", h.Doa)
	http.HandleFunc("/doa/more", h.DoaMore)
	http.HandleFunc("/kiblat", h.Kiblat)
	http.HandleFunc("/asmaul-husna", h.AsmaulHusna)
	http.HandleFunc("/hisab", h.Hisab)
	http.HandleFunc("/shalat", h.Shalat)
	http.HandleFunc("/shalat/mini", h.ShalatMini)
	http.HandleFunc("/quran", h.Quran)
	http.HandleFunc("/quran/search", h.QuranSearch)
	http.HandleFunc("/quran/", h.QuranSurah)

	http.HandleFunc("/quiz", h.QuizHome)
	http.HandleFunc("/api/quiz/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if strings.HasSuffix(path, "/start") {
			h.QuizStartAPI(w, r)
		} else if strings.HasSuffix(path, "/answer") {
			h.QuizAnswerAPI(w, r)
		} else if strings.HasSuffix(path, "/leaderboard") {
			h.QuizLeaderboardAPI(w, r)
		} else {
			http.NotFound(w, r)
		}
	})
	http.HandleFunc("/quiz/", h.QuizCategory)

	srv := &http.Server{
		Addr:              "127.0.0.1:" + port,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		Handler:           handler.SecurityHeaders(http.DefaultServeMux),
	}
	log.Printf("islami.click listening on :%s", port)
	log.Fatal(srv.ListenAndServe())
}
