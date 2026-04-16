package main

import (
	"embed"
	"html/template"
	"log"
	"net/http"
	"os"

	"github.com/srmdn/islami.click/internal/handler"
)

//go:embed templates
var templateFS embed.FS

//go:embed content
var contentFS embed.FS

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	tmpl, err := template.ParseFS(templateFS,
		"templates/layouts/*.html",
		"templates/pages/*.html",
		"templates/partials/*.html",
	)
	if err != nil {
		log.Fatal("failed to parse templates:", err)
	}
	if err != nil {
		log.Fatal("failed to parse templates:", err)
	}

	h := handler.New(tmpl, contentFS)

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
