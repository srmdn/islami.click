package model

type AsmaulHusnaEntry struct {
	Number      int    `json:"number"`
	Slug        string `json:"slug"`
	Arabic      string `json:"arabic"`
	Latin       string `json:"latin"`
	Translation string `json:"translation"`
	Source      string `json:"source,omitempty"`
}

type AsmaulHusnaPage struct {
	Title       string             `json:"title"`
	Description string             `json:"description"`
	Names       []AsmaulHusnaEntry `json:"names"`
}
