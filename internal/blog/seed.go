package blog

import (
	"context"
	"encoding/json"
)

// SeedDemo is explicit opt-in and only operates on an empty instance.
// It exercises the real copy/upload/parse/commit/preview/publish workflow.
func (a *App) SeedDemo(ctx context.Context) error {
	if len(a.Store.Read().Articles) > 0 {
		return nil
	}
	b, err := webFS.ReadFile("web/demo.json")
	if err != nil {
		return err
	}
	var contents []Content
	if err = json.Unmarshal(b, &contents); err != nil {
		return err
	}
	taxIDs := []string{}
	seenTaxonomy := map[string]bool{}
	revisionIDs := []string{}
	articleIDs := []string{}
	for _, content := range contents {
		cp, err := a.WorkingCopy(CopyInput{Kind: content.Kind})
		if err != nil {
			return err
		}
		content.ID = cp.Content.ID
		up, err := a.Upload(UploadInput{CopyID: cp.ID, Content: content, IdempotencyKey: uid()})
		if err != nil {
			return err
		}
		up, err = a.Parse(ctx, up.ID)
		if err != nil {
			return err
		}
		if up.Status != "parsed" {
			return fail("DEMO_PARSE", up.Diagnostic)
		}
		rev, err := a.Commit(CommitInput{UploadID: up.ID, SourceHash: up.Hash, ExpectedDraft: cp.Draft, Epoch: cp.Epoch, IdempotencyKey: uid()})
		if err != nil {
			return err
		}
		revisionIDs = append(revisionIDs, rev.ID)
		for _, id := range append([]string{rev.Content.CategoryID}, rev.Content.TagIDs...) {
			if id != "" && !seenTaxonomy[id] {
				taxIDs = append(taxIDs, id)
				seenTaxonomy[id] = true
			}
		}
		if content.Kind == "article" {
			articleIDs = append(articleIDs, content.ID)
		}
	}
	col, err := a.Collection(CollectionInput{Collection: Collection{Name: "从想法到作品", Slug: "from-thought-to-work", Description: "关于个人写作、简单软件与 AI 协作的几篇思考。", ArticleIDs: articleIDs[:min(3, len(articleIDs))]}})
	if err != nil {
		return err
	}
	ch, err := a.Changes(ChangesInput{Revisions: revisionIDs, Taxonomy: taxIDs, Collections: []string{col.ID}, ExpectedActive: a.Store.Read().Active})
	if err != nil {
		return err
	}
	p, err := a.Preview(ctx, ch.ID)
	if err != nil {
		return err
	}
	p, err = a.Confirm(ConfirmInput{PreviewID: p.ID, ArtifactHash: p.Hash, ChangeSetHash: ch.Hash})
	if err != nil {
		return err
	}
	_, err = a.Publish(PublishInput{PreviewID: p.ID, ConfirmationID: p.Confirmed, ChangeSetHash: ch.Hash, ExpectedActive: ch.Base, IdempotencyKey: "initial-demo"})
	return err
}
