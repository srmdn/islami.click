//go:build ignore

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

type QuranComVerse struct {
	VerseNumber int `json:"verse_number"`
	PageNumber  int `json:"page_number"`
}

type QuranComResponse struct {
	Verses []QuranComVerse `json:"verses"`
}

func main() {
	pageMap := make(map[string]int)

	for surah := 1; surah <= 114; surah++ {
		url := fmt.Sprintf("https://api.quran.com/api/v4/verses/by_chapter/%d?language=id&words=false&per_page=300", surah)
		resp, err := http.Get(url)
		if err != nil {
			fmt.Fprintf(os.Stderr, "fetch surah %d: %v\n", surah, err)
			os.Exit(1)
		}

		var result QuranComResponse
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			resp.Body.Close()
			fmt.Fprintf(os.Stderr, "decode surah %d: %v\n", surah, err)
			os.Exit(1)
		}
		resp.Body.Close()

		for _, v := range result.Verses {
			key := fmt.Sprintf("%d:%d", surah, v.VerseNumber)
			pageMap[key] = v.PageNumber
		}

		fmt.Printf("Surah %d: %d ayahs mapped\n", surah, len(result.Verses))
	}

	data, err := json.MarshalIndent(pageMap, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "marshal: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile("content/quran-pages.json", data, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "write file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\nWritten %d mappings to content/quran-pages.json\n", len(pageMap))
}
