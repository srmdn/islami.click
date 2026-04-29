package model

type QuranSurah struct {
	Number         int    `json:"number"`
	Name           string `json:"name"`
	ArabicName     string `json:"arabic_name"`
	Translation    string `json:"translation"`
	RevelationType string `json:"revelation_type"`
	AyahCount      int    `json:"ayah_count"`
}

type QuranAyah struct {
	Number      int    `json:"number"`
	Arabic      string `json:"arabic"`
	Translation string `json:"translation"`
}

type QuranPageData struct {
	Meta        PageMeta
	Title       string
	Description string
	Surahs      []QuranSurah
}

type SurahReaderData struct {
	Meta        PageMeta
	Title       string
	Description string
	Surah       QuranSurah
	Ayahs       []QuranAyah
	PrevSurah   *QuranSurah
	NextSurah   *QuranSurah
	Page        int
	PageSize    int
	TotalPages  int
	AudioURL    string
}

type QuranSearchResult struct {
	SurahNumber  int
	SurahName    string
	AyahNumber   int
	Arabic       string
	Translation  string
}

type QuranSearchData struct {
	Meta        PageMeta
	Title       string
	Description string
	Query       string
	Results     []QuranSearchResult
	ResultCount int
}
