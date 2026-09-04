//go:build ignore

// Fetch At-Tafsir Al-Muyassar (Arabic) per ayah from the Quran.com API v4
// (tafsir resource 16, "ar-tafsir-muyassar") and vendor it into
// content/tafsir/NNN.json plus a manifest with per-file hashes.
//
// Usage: go run scripts/fetch-tafsir-muyassar.go
// Existing per-surah files with the expected ayah count are skipped,
// so the script is safe to re-run (resume).
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

const (
	resourceID  = 16
	workers      = 10
	maxRetries   = 4
	requestDelay = 100 * time.Millisecond
)

type SurahMeta struct {
	ID          int    `json:"id"`
	TotalVerses int    `json:"total_verses"`
}

type TafsirResponse struct {
	Tafsir struct {
		Text string `json:"text"`
	} `json:"tafsir"`
}

type SurahTafsir struct {
	Number int                  `json:"number"`
	Source string               `json:"source"`
	Ayahs  []SurahTafsirAyah    `json:"ayahs"`
}

type SurahTafsirAyah struct {
	Number int    `json:"number"`
	Tafsir string `json:"tafsir"`
}

type ManifestSurah struct {
	Number int    `json:"number"`
	Ayahs  int    `json:"ayahs"`
	SHA256 string `json:"sha256"`
}

type Manifest struct {
	Source   string          `json:"source"`
	Resource string          `json:"resource"`
	Language string          `json:"language"`
	Fetched  string          `json:"fetched_at"`
	Surahs   []ManifestSurah `json:"surahs"`
}

var (
	tagRe  = regexp.MustCompile(`<[^>]*>`)
	spaceRe = regexp.MustCompile(`[ \t]+`)
)

// sanitize keeps only the quran.com keyword highlight, normalized to a
// local <mark> element, and strips every other HTML tag verbatim.
// Placeholders protect our own markup from the tag stripper.
func sanitize(s string) string {
	s = strings.ReplaceAll(s, `<span class="green">`, "")
	s = strings.ReplaceAll(s, "</span>", "")
	s = tagRe.ReplaceAllString(s, "")
	s = strings.ReplaceAll(s, "", "<mark class=\"tafsir-key\">")
	s = strings.ReplaceAll(s, "", "</mark>")
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		lines[i] = strings.TrimSpace(spaceRe.ReplaceAllString(l, " "))
	}
	out := strings.Join(lines, "\n")
	for strings.Contains(out, "\n\n\n") {
		out = strings.ReplaceAll(out, "\n\n\n", "\n\n")
	}
	return strings.TrimSpace(out)
}

func fetchTafsir(client *http.Client, surah, ayah int) (string, error) {
	url := fmt.Sprintf("https://api.quran.com/api/v4/tafsirs/%d/by_ayah/%d:%d", resourceID, surah, ayah)
	var lastErr error
	for attempt := 1; attempt <= maxRetries; attempt++ {
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return "", err
		}
		req.Header.Set("User-Agent", "islami.click-fetch/1.0 (contact: srmdn)")
		req.Header.Set("Accept", "application/json")
		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			time.Sleep(time.Duration(attempt) * time.Second)
			continue
		}
		var tr TafsirResponse
		err = json.NewDecoder(resp.Body).Decode(&tr)
		resp.Body.Close()
		if err != nil {
			lastErr = err
			time.Sleep(time.Duration(attempt) * time.Second)
			continue
		}
		if strings.TrimSpace(tr.Tafsir.Text) == "" {
			lastErr = fmt.Errorf("empty tafsir text for %d:%d", surah, ayah)
			time.Sleep(time.Duration(attempt) * time.Second)
			continue
		}
		return sanitize(tr.Tafsir.Text), nil
	}
	return "", fmt.Errorf("fetch %d:%d: %w", surah, ayah, lastErr)
}

func fileSHA256(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func main() {
	metaData, err := os.ReadFile("content/quran-surahs.json")
	if err != nil {
		fmt.Fprintf(os.Stderr, "read quran-surahs.json: %v\n", err)
		os.Exit(1)
	}
	var metas []SurahMeta
	if err := json.Unmarshal(metaData, &metas); err != nil {
		fmt.Fprintf(os.Stderr, "parse quran-surahs.json: %v\n", err)
		os.Exit(1)
	}

	dir := "content/tafsir"
	if err := os.MkdirAll(dir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "mkdir: %v\n", err)
		os.Exit(1)
	}

	client := &http.Client{Timeout: 25 * time.Second}

	type job struct{ surah, ayahs int }
	jobs := make(chan job)
	var wg sync.WaitGroup
	var mu sync.Mutex
	manifest := make([]ManifestSurah, 0, 114)
	failed := false

	worker := func() {
		defer wg.Done()
		for j := range jobs {
			filename := filepath.Join(dir, fmt.Sprintf("%03d.json", j.surah))
			if existing, err := os.ReadFile(filename); err == nil {
				var st SurahTafsir
				if json.Unmarshal(existing, &st) == nil && len(st.Ayahs) == j.ayahs {
					fmt.Printf("skip %s (%d ayahs, cached)\n", filename, j.ayahs)
					mu.Lock()
					manifest = append(manifest, ManifestSurah{Number: j.surah, Ayahs: j.ayahs, SHA256: fileSHA256(filename)})
					mu.Unlock()
					continue
				}
			}

			time.Sleep(requestDelay)
			st := SurahTafsir{
				Number: j.surah,
				Source: "At-Tafsir Al-Muyassar via Quran.com API v4 (resource ar-tafsir-muyassar)",
				Ayahs:  make([]SurahTafsirAyah, 0, j.ayahs),
			}
			ok := true
			for a := 1; a <= j.ayahs; a++ {
				text, err := fetchTafsir(client, j.surah, a)
				if err != nil {
					fmt.Fprintf(os.Stderr, "FAILED %d:%d: %v\n", j.surah, a, err)
					ok = false
					break
				}
				st.Ayahs = append(st.Ayahs, SurahTafsirAyah{Number: a, Tafsir: text})
				time.Sleep(requestDelay)
			}
			if !ok {
				mu.Lock()
				failed = true
				mu.Unlock()
				continue
			}
			data, err := json.MarshalIndent(st, "", "  ")
			if err != nil {
				fmt.Fprintf(os.Stderr, "marshal surah %d: %v\n", j.surah, err)
				mu.Lock()
				failed = true
				mu.Unlock()
				continue
			}
			if err := os.WriteFile(filename, data, 0644); err != nil {
				fmt.Fprintf(os.Stderr, "write %s: %v\n", filename, err)
				mu.Lock()
				failed = true
				mu.Unlock()
				continue
			}
			sum := sha256.Sum256(data)
			fmt.Printf("wrote %s (%d ayahs)\n", filename, j.ayahs)
			mu.Lock()
			manifest = append(manifest, ManifestSurah{Number: j.surah, Ayahs: j.ayahs, SHA256: hex.EncodeToString(sum[:])})
			mu.Unlock()
		}
	}

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go worker()
	}
	for _, m := range metas {
		jobs <- job{surah: m.ID, ayahs: m.TotalVerses}
	}
	close(jobs)
	wg.Wait()

	if failed {
		fmt.Fprintln(os.Stderr, "some surahs failed; re-run to resume")
		os.Exit(1)
	}

	ordered := make([]ManifestSurah, 0, len(manifest))
	byNumber := map[int]ManifestSurah{}
	for _, ms := range manifest {
		byNumber[ms.Number] = ms
	}
	for _, m := range metas {
		ms, ok := byNumber[m.ID]
		if !ok {
			fmt.Fprintf(os.Stderr, "missing manifest entry for surah %d\n", m.ID)
			os.Exit(1)
		}
		ordered = append(ordered, ms)
	}
	m := Manifest{
		Source:   "https://api.quran.com/api/v4/tafsirs/16/by_ayah/{surah}:{ayah}",
		Resource: "ar-tafsir-muyassar (Tafsir Muyassar, Arabic)",
		Language: "arabic",
		Fetched:  time.Now().UTC().Format(time.RFC3339),
		Surahs:   ordered,
	}
	mData, _ := json.MarshalIndent(m, "", "  ")
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), mData, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "write manifest: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("wrote content/tafsir/manifest.json — done")
}
