package main

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type Digest struct {
	Date     string        `json:"date"`
	Intro    string        `json:"intro"`
	Articles []DigestEntry `json:"articles"`
}

type DigestEntry struct {
	Title   string `json:"title"`
	Link    string `json:"link"`
	Source  string `json:"source"`
	Summary string `json:"summary"`
}

var tmpl = template.Must(template.ParseGlob("templates/*.html"))

func handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	sources, err := loadFeedSources("feeds.json")
	if err != nil {
		http.Error(w, "failed to load feeds: "+err.Error(), 500)
		return
	}

	articles := fetchAllArticles(sources, 7*24*time.Hour)

	seen := map[string]bool{}
	var categories []string
	for _, a := range articles {
		if a.Category != "" && !seen[a.Category] {
			seen[a.Category] = true
			categories = append(categories, a.Category)
		}
	}
	sort.Strings(categories)

	tmpl.ExecuteTemplate(w, "layout.html", map[string]any{
		"Page":       "index",
		"Articles":   articles,
		"Categories": categories,
	})
}

func handleCreateDigest(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	selected := r.Form["selected"]
	if len(selected) == 0 {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	var entries []DigestEntry
	for _, val := range selected {
		parts := splitSelected(val)
		if len(parts) == 3 {
			entries = append(entries, DigestEntry{
				Title:  parts[0],
				Link:   parts[1],
				Source: parts[2],
			})
		}
	}

	today := time.Now().Format("2006-01-02")

	tmpl.ExecuteTemplate(w, "layout.html", map[string]any{
		"Page":     "digest",
		"Date":     today,
		"Articles": entries,
	})
}

func handleEditDigest(w http.ResponseWriter, r *http.Request) {
	date := r.URL.Query().Get("date")
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}

	digest, err := loadDigest(date)
	if err != nil {
		http.Error(w, "digest not found", 404)
		return
	}

	tmpl.ExecuteTemplate(w, "layout.html", map[string]any{
		"Page":     "digest",
		"Date":     digest.Date,
		"Intro":    digest.Intro,
		"Articles": digest.Articles,
	})
}

func handleSaveDigest(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	date := r.FormValue("date")
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}

	intro := r.FormValue("intro")
	titles := r.Form["title"]
	links := r.Form["link"]
	sources := r.Form["source"]
	summaries := r.Form["summary"]

	var entries []DigestEntry
	for i := range titles {
		entries = append(entries, DigestEntry{
			Title:   titles[i],
			Link:    links[i],
			Source:  getIndex(sources, i),
			Summary: getIndex(summaries, i),
		})
	}

	digest := Digest{
		Date:     date,
		Intro:    intro,
		Articles: entries,
	}

	if err := saveDigest(digest); err != nil {
		http.Error(w, "failed to save: "+err.Error(), 500)
		return
	}

	http.Redirect(w, r, "/digests/"+date, http.StatusSeeOther)
}

func handleListDigests(w http.ResponseWriter, r *http.Request) {
	entries, _ := os.ReadDir("digests")

	var dates []string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".json") {
			dates = append(dates, strings.TrimSuffix(e.Name(), ".json"))
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(dates)))

	tmpl.ExecuteTemplate(w, "layout.html", map[string]any{
		"Page":  "digests",
		"Dates": dates,
	})
}

func handleViewDigest(w http.ResponseWriter, r *http.Request) {
	date := r.PathValue("date")

	digest, err := loadDigest(date)
	if err != nil {
		http.Error(w, "digest not found", 404)
		return
	}

	tmpl.ExecuteTemplate(w, "layout.html", map[string]any{
		"Page":     "digest",
		"Date":     digest.Date,
		"Intro":    digest.Intro,
		"Articles": digest.Articles,
	})
}

func loadDigest(date string) (Digest, error) {
	data, err := os.ReadFile(filepath.Join("digests", date+".json"))
	if err != nil {
		return Digest{}, err
	}
	var d Digest
	if err := json.Unmarshal(data, &d); err != nil {
		return Digest{}, err
	}
	return d, nil
}

func saveDigest(d Digest) error {
	os.MkdirAll("digests", 0755)
	data, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join("digests", d.Date+".json"), data, 0644)
}

func splitSelected(val string) []string {
	// Format: "title\x1flink\x1fsource"
	return strings.Split(val, "\x1f")
}

func getIndex(s []string, i int) string {
	if i < len(s) {
		return s[i]
	}
	return ""
}

func init() {
	if err := os.MkdirAll("digests", 0755); err != nil {
		log.Fatal(err)
	}
}
