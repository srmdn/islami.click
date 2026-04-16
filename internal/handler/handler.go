package handler

import (
	"embed"
	"encoding/json"
	"html/template"
	"log"
	"net/http"
)

type Handler struct {
	tmpls     map[string]*template.Template
	contentFS embed.FS
}

func New(tmpls map[string]*template.Template, contentFS embed.FS) *Handler {
	return &Handler{tmpls: tmpls, contentFS: contentFS}
}

func (h *Handler) render(w http.ResponseWriter, page string, data any) {
	t, ok := h.tmpls[page]
	if !ok {
		http.Error(w, "Page not found", http.StatusNotFound)
		return
	}
	if err := t.ExecuteTemplate(w, "base", data); err != nil {
		log.Printf("render %s: %v", page, err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
	}
}

func (h *Handler) Home(w http.ResponseWriter, r *http.Request) {
	h.render(w, "home.html", nil)
}

func (h *Handler) AlMatsurat(w http.ResponseWriter, r *http.Request) {
	h.render(w, "almatsurat.html", nil)
}

func (h *Handler) AlMatsuratSugro(w http.ResponseWriter, r *http.Request) {
	data, err := h.contentFS.ReadFile("content/almatsurat-sugro.json")
	if err != nil {
		http.Error(w, "Failed to load content", http.StatusInternalServerError)
		return
	}

	var content struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		Sections    []struct {
			ID          string `json:"id"`
			Type        string `json:"type"`
			Title       string `json:"title"`
			Arabic      string `json:"arabic"`
			Translation string `json:"translation"`
			Repeat      int    `json:"repeat"`
			Source      string `json:"source"`
		} `json:"sections"`
	}

	if err := json.Unmarshal(data, &content); err != nil {
		http.Error(w, "Failed to parse content", http.StatusInternalServerError)
		return
	}

	h.render(w, "almatsurat-sugro.html", content)
}

func (h *Handler) AlMatsuratKubro(w http.ResponseWriter, r *http.Request) {
	data, err := h.contentFS.ReadFile("content/almatsurat-kubro.json")
	if err != nil {
		http.Error(w, "Failed to load content", http.StatusInternalServerError)
		return
	}

	var content struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		Sections    []struct {
			ID          string `json:"id"`
			Type        string `json:"type"`
			Title       string `json:"title"`
			Arabic      string `json:"arabic"`
			Translation string `json:"translation"`
			Repeat      int    `json:"repeat"`
			Source      string `json:"source"`
		} `json:"sections"`
	}

	if err := json.Unmarshal(data, &content); err != nil {
		http.Error(w, "Failed to parse content", http.StatusInternalServerError)
		return
	}

	h.render(w, "almatsurat-kubro.html", content)
}

func (h *Handler) Doa(w http.ResponseWriter, r *http.Request) {
	h.render(w, "doa.html", nil)
}

func (h *Handler) Shalat(w http.ResponseWriter, r *http.Request) {
	h.render(w, "shalat.html", nil)
}
