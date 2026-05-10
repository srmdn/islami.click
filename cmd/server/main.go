package main

import (
	"context"
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
		"arabicHTML": func(s string) template.HTML {
			escaped := template.HTMLEscapeString(s)
			result := arabicParenRe.ReplaceAllString(escaped, `<span dir="ltr">(<bdi>$1</bdi>)</span>`)
			return template.HTML(result)
		},
		"js": func(s string) string {
			var result []byte
			for i := 0; i < len(s); i++ {
				c := s[i]
				if c == '\'' || c == '"' || c == '`' || c == '\\' {
					result = append(result, '_')
				} else if c == 0xE2 && i+2 < len(s) && s[i+1] == 0x80 && s[i+2] == 0x99 {
					result = append(result, '_')
					i += 2
				} else if c == 0xE2 && i+2 < len(s) && s[i+1] == 0x80 && s[i+2] == 0x98 {
					result = append(result, '_')
					i += 2
				} else {
					result = append(result, c)
				}
			}
			return string(result)
		},
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
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(fs)))

	http.HandleFunc("/robots.txt", h.RobotsTxt)
	http.HandleFunc("/sitemap.xml", h.Sitemap)
	http.HandleFunc("/llms.txt", h.LLMsTxt)

	http.HandleFunc("/", h.Home)
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
		if strings.HasSuffix(path, "/questions") {
			h.QuizQuestionsAPI(w, r)
		} else if strings.HasSuffix(path, "/leaderboard") {
			h.QuizLeaderboardAPI(w, r)
		} else if strings.HasSuffix(path, "/score") {
			h.QuizScoreAPI(w, r)
		} else {
			http.NotFound(w, r)
		}
	})
	http.HandleFunc("/quiz/", h.QuizCategory)

	srv := &http.Server{
		Addr:              ":" + port,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	log.Printf("islami.click listening on :%s", port)
	log.Fatal(srv.ListenAndServe())
}
