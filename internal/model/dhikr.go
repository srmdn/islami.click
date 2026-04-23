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
	CollectionID string `json:"collection_id,omitempty"`
	Category     string `json:"category,omitempty"`
	SourceType   string `json:"source_type,omitempty"`
	IsRuqyah     bool   `json:"is_ruqyah,omitempty"`
	Title        string `json:"title"`
	Arabic       string `json:"arabic"`
	Latin        string `json:"latin"`
	Translation  string `json:"translation"`
	Source       string `json:"source,omitempty"`
	SourceURL    string `json:"source_url,omitempty"`
	Verification string `json:"verification,omitempty"`
}

// DoaSourceType represents a source type filter tab.
type DoaSourceType struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Count int    `json:"count"`
}

// DoaPageData represents the doa collection page.
type DoaPageData struct {
	Title       string          `json:"title"`
	Description string          `json:"description"`
	Categories  []DoaCategory   `json:"categories"`
	SourceTypes []DoaSourceType `json:"source_types"`
	// Pagination (not in JSON; populated by store)
	Items    []DoaEntry
	HasMore  bool
	NextPage int
}
