package blog

import (
	"context"
	"sort"
	"strings"
	"time"
	"unicode/utf8"
)

type SearchResult struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	URL      string `json:"url"`
	Snippet  string `json:"snippet"`
	Date     string `json:"date"`
	score    int
	revision Revision
}

func searchText(r Revision, s Snapshot) string {
	parts := []string{r.Content.Summary, r.Compiled.Text}
	if t, ok := s.Taxonomy[r.Content.CategoryID]; ok {
		parts = append(parts, t.Name)
	}
	for _, id := range r.Content.TagIDs {
		parts = append(parts, s.Taxonomy[id].Name)
	}
	return strings.Join(parts, " ")
}
func (a *App) Search(ctx context.Context, snap Snapshot, q, category, tag string, page int) ([]SearchResult, int, error) {
	q = strings.TrimSpace(q)
	terms := strings.Fields(strings.ToLower(q))
	if utf8.RuneCountInString(q) > 120 || len(terms) > 8 {
		return nil, 0, fail("QUERY_TOO_LONG", "最多 120 字、8 个关键词")
	}
	if len(terms) == 0 {
		return []SearchResult{}, 0, nil
	}
	if page < 1 || page > 1000 {
		return nil, 0, fail("INVALID_PAGE", "页码超出范围")
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	candidates := map[string]bool{}
	fts := []string{}
	for _, term := range terms {
		if utf8.RuneCountInString(term) >= 3 {
			fts = append(fts, `"`+strings.ReplaceAll(term, `"`, `""`)+`"`)
		}
	}
	// Hold the activation lock only while consulting the current FTS table. An
	// already captured older snapshot falls back to its own immutable text.
	useFTS := false
	if len(fts) > 0 {
		a.Store.mu.Lock()
		if a.Store.state.Active == snap.ID {
			rows, err := a.Store.db.QueryContext(ctx, "SELECT id FROM published_search WHERE published_search MATCH ? AND release = ?", strings.Join(fts, " AND "), snap.ID)
			if err == nil {
				useFTS = true
				for rows.Next() {
					var id string
					if rows.Scan(&id) == nil {
						candidates[id] = true
					}
				}
				if rows.Err() != nil {
					useFTS = false
				}
				rows.Close()
			}
		}
		a.Store.mu.Unlock()
	}
	results := []SearchResult{}
	for id, r := range snap.Posts {
		if ctx.Err() != nil {
			return nil, 0, ctx.Err()
		}
		if r.Content.Kind != "article" || (useFTS && !candidates[id]) || (category != "" && r.Content.CategoryID != category) || (tag != "" && !contains(r.Content.TagIDs, tag)) {
			continue
		}
		body := searchText(r, snap)
		all := strings.ToLower(r.Content.Title + " " + body)
		score, matched := 0, true
		for _, term := range terms {
			if !strings.Contains(all, term) {
				matched = false
				break
			}
			if strings.Contains(strings.ToLower(r.Content.Title), term) {
				score += 10
			}
			if strings.Contains(strings.ToLower(r.Content.Summary), term) {
				score += 3
			}
			score++
		}
		if matched {
			snippet := r.Content.Summary
			if snippet == "" {
				snippet = r.Compiled.Text
			}
			for _, term := range terms {
				index := strings.Index(strings.ToLower(r.Compiled.Text), term)
				if index >= 0 {
					before := []rune(r.Compiled.Text[:index])
					after := []rune(r.Compiled.Text[index:])
					snippet = string(before[max(0, len(before)-24):]) + string(after[:min(80, len(after))])
					break
				}
			}
			results = append(results, SearchResult{ID: id, Title: r.Content.Title, URL: contentPath(r.Content), Snippet: snippet, Date: date(r.PublishedAt), score: score, revision: r})
		}
	}
	sort.Slice(results, func(i, j int) bool {
		if results[i].score == results[j].score {
			return results[i].revision.PublishedAt > results[j].revision.PublishedAt
		}
		return results[i].score > results[j].score
	})
	total := len(results)
	return results[min((page-1)*12, total):min(page*12, total)], total, nil
}
