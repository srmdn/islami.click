package main

import (
	"html/template"
	"log"
	"net/http"
	"os"

	islamiclick "github.com/srmdn/islami.click"
	"github.com/srmdn/islami.click/internal/handler"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
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
		t, err := template.ParseFS(islamiclick.TemplateFS,
			"templates/layouts/base.html",
			"templates/partials/header.html",
			"templates/partials/footer.html",
			"templates/pages/"+page,
		)
		if err != nil {
			log.Fatalf("parse %s: %v", page, err)
		}
		tmpls[page] = t
	}

	h := handler.New(tmpls, islamiclick.ContentFS)

	fs := http.Dir("static")
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(fs)))

	http.HandleFunc("/", h.Home)
	http.HandleFunc("/almatsurat", h.AlMatsurat)
	http.HandleFunc("/almatsurat/sugro", h.AlMatsuratSugro)
	http.HandleFunc("/almatsurat/kubro", h.AlMatsuratKubro)
	http.HandleFunc("/doa", h.Doa)
	http.HandleFunc("/shalat", h.Shalat)

	log.Printf("islami.click listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
