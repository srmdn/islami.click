package model

// DhikrEntry represents a single dhikr/doa item.
type DhikrEntry struct {
	ID          string `json:"id"`
	Type        string `json:"type"` // quran, hadith, doa
	Title       string `json:"title"`
	Arabic      string `json:"arabic"`
	Translation string `json:"translation"`
	Repeat      int    `json:"repeat"`
	Source      string `json:"source,omitempty"`
}

// AlMatsurat represents a collection of dhikr (sugro or kubro).
type AlMatsurat struct {
	Title       string       `json:"title"`
	Description string       `json:"description"`
	Sections    []DhikrEntry `json:"sections"`
}

// DoaCategory represents a categorized collection of doa.
type DoaCategory struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Items       []DoaEntry `json:"items"`
}

// DoaEntry represents a single doa item.
type DoaEntry struct {
	ID           string `json:"id"`
	Category     string `json:"category,omitempty"`
	SourceType   string `json:"source_type,omitempty"` // quran or hadith
	Title        string `json:"title"`
	Arabic       string `json:"arabic"`
	Latin        string `json:"latin"`
	Translation  string `json:"translation"`
	Source       string `json:"source,omitempty"`
	SourceURL    string `json:"source_url,omitempty"`
	Verification string `json:"verification,omitempty"`
}

// DoaPageData represents the doa collection page.
type DoaPageData struct {
	Title       string        `json:"title"`
	Description string        `json:"description"`
	Categories  []DoaCategory `json:"categories"`
}
