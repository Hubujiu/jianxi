package blog

import (
	"context"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func routeUpload(t *testing.T, a *App) (WorkingCopy, Upload) {
	t.Helper()
	cp, err := a.WorkingCopy(CopyInput{})
	if err != nil {
		t.Fatal(err)
	}
	c := cp.Content
	c.Slug, c.Title, c.Markdown = "custom-article-route", "自定义元数据路由", "## 自定义正文\n\n测试分类与标签。"
	c.Category = &TaxonomyMetadata{Name: "技术", Slug: "my-technology"}
	c.Tags = []TaxonomyMetadata{{Name: "Go", Slug: "my-go"}}
	up, err := a.Upload(UploadInput{CopyID: cp.ID, Content: c, IdempotencyKey: uid()})
	if err != nil {
		t.Fatal(err)
	}
	return cp, up
}

func commitRouteUpload(t *testing.T, a *App, cp WorkingCopy, up Upload) Revision {
	t.Helper()
	up, err := a.Parse(context.Background(), up.ID)
	if err != nil || up.Status != "parsed" {
		t.Fatal("parse", up.Status, err)
	}
	r, err := a.Commit(CommitInput{UploadID: up.ID, SourceHash: up.Hash, ExpectedDraft: cp.Draft, Epoch: cp.Epoch, IdempotencyKey: uid()})
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func publishRouteChanges(t *testing.T, a *App, revisions, taxonomy []string) {
	t.Helper()
	ch, err := a.Changes(ChangesInput{Revisions: revisions, Taxonomy: taxonomy, ExpectedActive: a.Store.Read().Active})
	if err != nil {
		t.Fatal(err)
	}
	p, err := a.Preview(context.Background(), ch.ID)
	if err != nil {
		t.Fatal(err)
	}
	p, err = a.Confirm(ConfirmInput{PreviewID: p.ID, ArtifactHash: p.Hash, ChangeSetHash: ch.Hash})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = a.Publish(PublishInput{PreviewID: p.ID, ConfirmationID: p.Confirmed, ChangeSetHash: ch.Hash, ExpectedActive: ch.Base, IdempotencyKey: uid()}); err != nil {
		t.Fatal(err)
	}
}

func TestExplicitRouteMetadataRequired(t *testing.T) {
	a := testApp(t)
	cp, err := a.WorkingCopy(CopyInput{})
	if err != nil || cp.Content.Slug != "" {
		t.Fatal("new articles must not have a generated route", err)
	}
	for _, tc := range []struct {
		name string
		edit func(*Content)
	}{
		{"missing article slug", func(c *Content) { c.Slug = "" }},
		{"missing category slug", func(c *Content) { c.Category = &TaxonomyMetadata{Name: "技术"} }},
		{"missing tag slug", func(c *Content) { c.Tags = []TaxonomyMetadata{{Name: "Go"}} }},
		{"only category id", func(c *Content) { c.CategoryID = "opaque-id" }},
		{"only tag ids", func(c *Content) { c.TagIDs = []string{"opaque-id"} }},
		{"duplicate tags", func(c *Content) { c.Tags = []TaxonomyMetadata{{Name: "Go", Slug: "go"}, {Name: "Go", Slug: "go"}} }},
		{"path traversal", func(c *Content) { c.Slug = "../custom" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := clone(cp.Content)
			c.Slug = "explicit-route"
			tc.edit(&c)
			if _, err := a.Upload(UploadInput{CopyID: cp.ID, Content: c, IdempotencyKey: uid()}); err == nil {
				t.Fatal("incomplete route metadata accepted")
			}
		})
	}
	for _, kind := range []string{"category", "tag"} {
		for _, slug := range []string{"", strings.Repeat("a", 101), "path/child"} {
			if _, err := a.Taxonomy(TaxonomyInput{Taxon: Taxon{Kind: kind, Name: "测试", Slug: slug}}); err == nil {
				t.Fatal("taxonomy accepted invalid explicit slug")
			}
		}
	}
	if len(a.Store.Read().Taxonomy) != 0 || len(a.Store.Read().Uploads) != 0 {
		t.Fatal("invalid metadata left partial changes")
	}
}

func TestMetadataTaxonomyCommitPublishAndRenames(t *testing.T) {
	a := testApp(t)
	cp, up := routeUpload(t, a)
	if len(a.Store.Read().Taxonomy) != 0 || len(a.Store.active.Load().Taxonomy) != 0 {
		t.Fatal("upload created or published taxonomy")
	}
	r := commitRouteUpload(t, a, cp, up)
	if r.Hash != digest(r.Content) || r.Content.CategoryID != r.Content.Category.ID || r.Content.TagIDs[0] != r.Content.Tags[0].ID {
		t.Fatal("resolved metadata/source hash/associations are inconsistent")
	}
	ch, err := a.Changes(ChangesInput{Revisions: []string{r.ID}, ExpectedActive: a.Store.Read().Active})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = a.Preview(context.Background(), ch.ID); err == nil {
		t.Fatal("new taxonomy was implicitly published without selection")
	}
	publishRouteChanges(t, a, []string{r.ID}, []string{r.Content.CategoryID, r.Content.TagIDs[0]})
	for _, path := range []string{"/posts/custom-article-route/", "/categories/my-technology/", "/tags/my-go/"} {
		w := httptest.NewRecorder()
		a.Handler().ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		if w.Code != 200 {
			t.Fatalf("custom route %s: %d", path, w.Code)
		}
	}
	category := a.Store.Read().Taxonomy[r.Content.CategoryID]
	category.Slug = "technology"
	if _, err = a.Taxonomy(TaxonomyInput{Taxon: category, ExpectedRevision: category.Revision}); err != nil {
		t.Fatal(err)
	}
	publishRouteChanges(t, a, nil, []string{category.ID})
	w := httptest.NewRecorder()
	a.Handler().ServeHTTP(w, httptest.NewRequest("GET", "/categories/my-technology/", nil))
	if w.Code != 301 || w.Header().Get("Location") != "/categories/technology/" {
		t.Fatal("renamed taxonomy did not preserve the old link")
	}
	cp, err = a.WorkingCopy(CopyInput{ContentID: r.Content.ID})
	if err != nil || cp.Content.Category.Slug != "technology" || cp.Content.Category.ID != category.ID {
		t.Fatal("working copy did not hydrate current route metadata", err)
	}
	cp.Content.Slug = "renamed-article"
	up, err = a.Upload(UploadInput{CopyID: cp.ID, Content: cp.Content, IdempotencyKey: uid()})
	if err != nil {
		t.Fatal(err)
	}
	r = commitRouteUpload(t, a, cp, up)
	if len(a.Store.Read().Taxonomy) != 2 {
		t.Fatal("existing taxonomy IDs were not reused")
	}
	publishRouteChanges(t, a, []string{r.ID}, nil)
	w = httptest.NewRecorder()
	a.Handler().ServeHTTP(w, httptest.NewRequest("GET", "/posts/custom-article-route/", nil))
	if w.Code != 301 || w.Header().Get("Location") != "/posts/renamed-article/" {
		t.Fatal("renamed article did not preserve the old link")
	}
	backup, err := a.Export()
	if err != nil {
		t.Fatal(err)
	}
	// Restore validation must accept older revision metadata after taxonomy rename.
	bytes, err := os.ReadFile(filepath.Join(a.Store.root, "backups", backup))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = a.InspectBackup(context.Background(), bytes); err != nil {
		t.Fatal("route metadata broke backup validation", err)
	}
	// Reusing a previous route must restore its page, not create a self-loop.
	category = a.Store.Read().Taxonomy[category.ID]
	category.Slug = "my-technology"
	if _, err = a.Taxonomy(TaxonomyInput{Taxon: category, ExpectedRevision: category.Revision}); err != nil {
		t.Fatal(err)
	}
	publishRouteChanges(t, a, nil, []string{category.ID})
	w = httptest.NewRecorder()
	a.Handler().ServeHTTP(w, httptest.NewRequest("GET", "/categories/my-technology/", nil))
	if w.Code != 200 {
		t.Fatal("reused category route did not restore its page")
	}
	w = httptest.NewRecorder()
	a.Handler().ServeHTTP(w, httptest.NewRequest("GET", "/categories/technology/", nil))
	if w.Code != 301 || w.Header().Get("Location") != "/categories/my-technology/" {
		t.Fatal("reused route created a loop or lost the replacement alias")
	}
}

func TestMetadataTaxonomyConcurrentConflictIsAtomic(t *testing.T) {
	a := testApp(t)
	cp, up := routeUpload(t, a)
	up, err := a.Parse(context.Background(), up.ID)
	if err != nil || up.Status != "parsed" {
		t.Fatal(err)
	}
	if _, err = a.Taxonomy(TaxonomyInput{Taxon: Taxon{Kind: "tag", Name: "Go", Slug: "my-go"}}); err != nil {
		t.Fatal(err)
	}
	_, err = a.Commit(CommitInput{UploadID: up.ID, SourceHash: up.Hash, ExpectedDraft: cp.Draft, Epoch: cp.Epoch, IdempotencyKey: uid()})
	if err == nil || !strings.Contains(err.Error(), "TAXONOMY_CONFLICT") || len(a.Store.Read().Taxonomy) != 1 || len(a.Store.Read().Articles) != 0 {
		t.Fatal("concurrent taxonomy conflict leaked a category or article", err)
	}
}

func TestDemoUsesMetadataTranslationRoutes(t *testing.T) {
	a := testApp(t)
	if err := a.SeedDemo(context.Background()); err != nil {
		t.Fatal(err)
	}
	for _, taxon := range a.Store.Read().Taxonomy {
		if strings.HasPrefix(taxon.Slug, "category-") || strings.HasPrefix(taxon.Slug, "tag-") {
			t.Fatal("demo generated a hash route")
		}
	}
	for _, path := range []string{"/categories/technology/", "/categories/life/", "/categories/reading/", "/categories/thoughts/", "/tags/independent-development/", "/tags/ai/"} {
		w := httptest.NewRecorder()
		a.Handler().ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		if w.Code != 200 {
			t.Fatalf("translated demo route %s: %d", path, w.Code)
		}
	}
}

func TestLegacyRevisionRouteMetadataIsReadCompatible(t *testing.T) {
	a := testApp(t)
	category, err := a.Taxonomy(TaxonomyInput{Taxon: Taxon{Kind: "category", Name: "技术", Slug: "technology"}})
	if err != nil {
		t.Fatal(err)
	}
	r := draft(t, a, "", "旧版文章", "旧版只保存分类 ID。")
	r.Content.CategoryID = category.ID
	r.Hash = digest(r.Content)
	if err = a.Store.Update(func(s *State) error { s.Revisions[r.ID] = r; return nil }); err != nil {
		t.Fatal(err)
	}
	publishRouteChanges(t, a, []string{r.ID}, []string{category.ID})
	cp, err := a.WorkingCopy(CopyInput{ContentID: r.Content.ID})
	if err != nil || cp.Content.Category == nil || cp.Content.Category.Slug != "technology" {
		t.Fatal("legacy working copy did not expose current route metadata", err)
	}
	stored := a.Store.Read().Revisions[r.ID]
	if stored.Content.Category != nil || stored.Hash != r.Hash {
		t.Fatal("hydrating metadata changed an immutable legacy revision")
	}
	name, err := a.Export()
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(a.Store.root, "backups", name))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = a.InspectBackup(context.Background(), data); err != nil {
		t.Fatal("legacy source hashes or backup validation changed", err)
	}
}
