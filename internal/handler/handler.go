package handler

import (
	"embed"
	"encoding/json"
	"html/template"
	"net/http"
)

// Handler holds dependencies for HTTP handlers.
type Handler struct {
	tmpl      *template.Template
	contentFS embed.FS
}

// New creates a new Handler.
func New(tmpl *template.Template, contentFS embed.FS) *Handler {
	return &Handler{tmpl: tmpl, contentFS: contentFS}
}

// Home serves the home page.
func (h *Handler) Home(w http.ResponseWriter, r *http.Request) {
	h.tmpl.ExecuteTemplate(w, "home.html", nil)
}

// AlMatsurat serves the almatsurat picker page.
func (h *Handler) AlMatsurat(w http.ResponseWriter, r *http.Request) {
	h.tmpl.ExecuteTemplate(w, "almatsurat.html", nil)
}

// AlMatsuratSugro serves the Wazifah Sugro page.
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

	h.tmpl.ExecuteTemplate(w, "almatsurat-sugro.html", content)
}

// AlMatsuratKubro serves the Wazifah Kubro page.
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

	h.tmpl.ExecuteTemplate(w, "almatsurat-kubro.html", content)
}

// Doa serves the doa page.
func (h *Handler) Doa(w http.ResponseWriter, r *http.Request) {
	h.tmpl.ExecuteTemplate(w, "doa.html", nil)
}

// Shalat serves the shalat page.
func (h *Handler) Shalat(w http.ResponseWriter, r *http.Request) {
	h.tmpl.ExecuteTemplate(w, "shalat.html", nil)
}
