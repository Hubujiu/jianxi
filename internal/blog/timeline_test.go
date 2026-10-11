package blog

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestTimelineDataGroupsPostsAndAlignsHeatmap(t *testing.T) {
	posts := []Revision{
		{PublishedAt: "2026-01-02T00:00:00Z"},
		{PublishedAt: "2025-12-31T15:00:00Z"},
		{PublishedAt: "2025-12-15T00:00:00Z"},
	}
	data := timelineData(posts, time.Date(2026, 1, 3, 0, 0, 0, 0, time.UTC))
	if data.Count != 3 || data.ActiveDays != 3 {
		t.Fatalf("year stats: %d posts on %d days", data.Count, data.ActiveDays)
	}
	if len(data.Years) != 2 || data.Years[0].Name != "2026" || data.Years[0].Count != 1 || data.Years[1].Name != "2025" || data.Years[1].Count != 2 {
		t.Fatalf("year groups: %+v", data.Years)
	}
	if len(data.Days) != data.Columns*7 || data.Days[0].Date != "2024-12-29" || !data.Days[0].Padding {
		t.Fatalf("heatmap week alignment: first=%+v, cells=%d, columns=%d", data.Days[0], len(data.Days), data.Columns)
	}
	found := false
	for _, day := range data.Days {
		if day.Date == "2026-01-02" {
			found = day.Count == 1 && !day.Padding
		}
	}
	if !found {
		t.Fatal("published day missing from heatmap")
	}
}

func TestSiteTemplateChangeRebuildsPublishedPages(t *testing.T) {
	root := t.TempDir()
	first, err := NewApp(root, "../../scripts/compile.mjs")
	if err != nil {
		t.Fatal(err)
	}
	previous := first.Store.state.Active
	marker := filepath.Join(root, "releases", previous, ".sitehash")
	if err := os.WriteFile(marker, []byte("previous-template"), 0600); err != nil {
		t.Fatal(err)
	}
	hash, err := treeHash(filepath.Dir(marker))
	if err != nil {
		t.Fatal(err)
	}
	state := clone(first.Store.state)
	state.ActiveHash = hash
	if err := first.Store.save(state, nil); err != nil {
		t.Fatal(err)
	}
	if err := first.Store.Close(); err != nil {
		t.Fatal(err)
	}
	second, err := NewApp(root, "../../scripts/compile.mjs")
	if err != nil {
		t.Fatal(err)
	}
	defer second.Store.Close()
	if second.Store.state.Active == previous {
		t.Fatal("site template change left the old release active")
	}
	if second.Store.state.ActiveHash == hash {
		t.Fatal("new release kept the old artifact hash")
	}
}
