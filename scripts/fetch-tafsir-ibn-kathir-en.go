//go:build ignore

// Fetch Ibn Kathir (Abridged, English) per ayah from the Quran.com API v4
// (tafsir resource 169, "en-tafisr-ibn-kathir") and vendor it into
// content/tafsir-ibn-kathir-en/NNN.json plus a manifest with per-file hashes.
//
// Usage:
//
//	go run scripts/fetch-tafsir-ibn-kathir-en.go -surahs 112,113,114  # pilot
//	go run scripts/fetch-tafsir-ibn-kathir-en.go                      # full 114
//
// Existing per-surah files with the expected ayah count are skipped,
// so the script is safe to re-run (resume). The manifest lists exactly
// the surahs fetched in that run plus any cached ones, which keeps the
// DB seed (manifest-driven) consistent at every stage.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	resourceID   = 169
	workers      = 6
	maxRetries   = 4
	requestDelay = 200 * time.Millisecond
	outDir       = "content/tafsir-ibn-kathir-en"
)

type SurahMeta struct {
	ID          int `json:"id"`
	TotalVerses int `json:"total_verses"`
}

type TafsirResponse struct {
	Tafsir struct {
		Text string `json:"text"`
	} `json:"tafsir"`
}

type SurahTafsir struct {
	Number int               `json:"number"`
	Source string            `json:"source"`
	Ayahs  []SurahTafsirAyah `json:"ayahs"`
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
	tagRe   = regexp.MustCompile(`<[^>]*>`)
	spaceRe = regexp.MustCompile(`[ \t]+`)
)

// sanitize keeps only the quran.com keyword highlight, normalized to a
// local <mark> element, and strips every other HTML tag verbatim
// (Ibn Kathir English ships h1/h2/p/strong markup; headings collapse
// into plain lines). Placeholders protect our own markup from the stripper.
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

func parseSurahFlag(raw string, metas []SurahMeta) []SurahMeta {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return metas
	}
	want := map[int]bool{}
	for _, part := range strings.Split(raw, ",") {
		n, err := strconv.Atoi(strings.TrimSpace(part))
		if err != nil || n < 1 || n > 114 {
			fmt.Fprintf(os.Stderr, "invalid surah number %q (want 1-114)\n", part)
			os.Exit(1)
		}
		want[n] = true
	}
	var filtered []SurahMeta
	for _, m := range metas {
		if want[m.ID] {
			filtered = append(filtered, m)
		}
	}
	return filtered
}

func main() {
	surahsFlag := flag.String("surahs", "", "comma-separated surah numbers to fetch (default: all 114)")
	flag.Parse()

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
	metas = parseSurahFlag(*surahsFlag, metas)
	if len(metas) == 0 {
		fmt.Fprintln(os.Stderr, "no surahs selected")
		os.Exit(1)
	}

	if err := os.MkdirAll(outDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "mkdir: %v\n", err)
		os.Exit(1)
	}

	client := &http.Client{Timeout: 25 * time.Second}

	type job struct{ surah, ayahs int }
	jobs := make(chan job)
	var wg sync.WaitGroup
	var mu sync.Mutex
	manifest := make([]ManifestSurah, 0, len(metas))
	failed := false

	worker := func() {
		defer wg.Done()
		for j := range jobs {
			filename := filepath.Join(outDir, fmt.Sprintf("%03d.json", j.surah))
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
				Source: "Ibn Kathir (Abridged, English) via Quran.com API v4 (resource en-tafisr-ibn-kathir)",
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
		Source:   "https://api.quran.com/api/v4/tafsirs/169/by_ayah/{surah}:{ayah}",
		Resource: "en-tafisr-ibn-kathir (Ibn Kathir Abridged, English)",
		Language: "english",
		Fetched:  time.Now().UTC().Format(time.RFC3339),
		Surahs:   ordered,
	}
	mData, _ := json.MarshalIndent(m, "", "  ")
	if err := os.WriteFile(filepath.Join(outDir, "manifest.json"), mData, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "write manifest: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("wrote %s/manifest.json — done (%d surahs)\n", outDir, len(ordered))
}
