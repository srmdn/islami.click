package main

import (
	"context"
	"html/template"
	"log"
	"net/http"
	"os"

	islamiclick "github.com/srmdn/islami.click"
	"github.com/srmdn/islami.click/internal/handler"
	"github.com/srmdn/islami.click/internal/store"
)

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
	}

	pages := []string{
		"home.html",
		"almatsurat.html",
		"almatsurat-sugro.html",
		"almatsurat-kubro.html",
		"doa.html",
		"shalat.html",
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

	h := handler.New(tmpls, partialTmpls, contentStore)

	fs := http.Dir("static")
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(fs)))

	http.HandleFunc("/", h.Home)
	http.HandleFunc("/almatsurat", h.AlMatsurat)
	http.HandleFunc("/almatsurat/sugro", h.AlMatsuratSugro)
	http.HandleFunc("/almatsurat/kubro", h.AlMatsuratKubro)
	http.HandleFunc("/doa", h.Doa)
	http.HandleFunc("/shalat", h.Shalat)
	http.HandleFunc("/shalat/mini", h.ShalatMini)

	log.Printf("islami.click listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
