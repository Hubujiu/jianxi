package blog

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"slices"
	"time"
)

func (a *App) Preview(ctx context.Context, id string) (Preview, error) {
	s := a.Store.Read()
	ch, ok := s.Changes[id]
	if !ok {
		return Preview{}, fail("CHANGESET_NOT_FOUND", "变更清单不存在")
	}
	if err := checkChanges(s, ch); err != nil {
		return Preview{}, err
	}
	snap := clone(*a.Store.active.Load())
	snap.ID = uid()
	if ch.IncludeSettings {
		snap.Settings = s.Settings
	}
	for _, id := range ch.Taxonomy {
		v := s.Taxonomy[id]
		if old, ok := snap.Taxonomy[id]; ok && old.Slug != v.Slug {
			for from, to := range snap.Redirects {
				if to == taxonPath(old) {
					snap.Redirects[from] = taxonPath(v)
				}
			}
			snap.Redirects[taxonPath(old)] = taxonPath(v)
		}
		if v.Deleted {
			delete(snap.Taxonomy, id)
		} else {
			snap.Taxonomy[id] = v
		}
	}
	for _, id := range ch.Collections {
		v := s.Collections[id]
		if v.Deleted {
			delete(snap.Collections, id)
		} else {
			snap.Collections[id] = v
		}
	}
	for _, id := range ch.Revisions {
		r := s.Revisions[id]
		if old, ok := snap.Posts[r.Content.ID]; ok && old.Content.Slug != r.Content.Slug {
			for from, to := range snap.Redirects {
				if to == contentPath(old.Content) {
					snap.Redirects[from] = contentPath(r.Content)
				}
			}
			snap.Redirects[contentPath(old.Content)] = contentPath(r.Content)
		}
		if r.PublishedAt == "" {
			r.PublishedAt = s.Articles[r.Content.ID].FirstPublished
		}
		snap.Posts[r.Content.ID] = r
	}
	for _, id := range ch.Withdraw {
		delete(snap.Posts, id)
	}
	slugs := map[string]bool{}
	snap.Assets = map[string]Asset{}
	for id, r := range snap.Posts {
		if r.RendererHash != a.BuildHash {
			if a.Compiler.Script == "" && (r.Bundle == nil || r.Bundle.Renderer != RendererVersion) {
				current, ok := a.Store.active.Load().Posts[id]
				if !ok || current.ID != r.ID {
					return Preview{}, fail("LOCAL_COMPILATION_REQUIRED", "旧修订请通过本地 MCP 重新编译后预览")
				}
			} else {
				compiled, bundle, err := a.compileRevision(ctx, r)
				if err != nil {
					return Preview{}, err
				}
				r.Compiled = compiled
				r.Bundle = bundle
				r.RendererHash = a.BuildHash
				snap.Posts[id] = r
			}
		}
		if err := validateRefs(r.Content, snap.Taxonomy, s.Assets); err != nil {
			return Preview{}, err
		}
		p := contentPath(r.Content)
		if slugs[p] {
			return Preview{}, fail("SLUG_CONFLICT", "公开地址重复")
		}
		slugs[p] = true
		for _, assetID := range r.Compiled.Assets {
			v, ok := s.Assets[assetID]
			if !ok {
				return Preview{}, fail("ASSET_MISSING", assetID)
			}
			snap.Assets[assetID] = v
		}
	}
	for _, t := range snap.Taxonomy {
		slugs[taxonPath(t)] = true
	}
	for _, col := range snap.Collections {
		slugs["/collections/"+col.Slug+"/"] = true
	}
	for old, dest := range snap.Redirects {
		// A custom route may deliberately reuse an earlier URL. Live pages win
		// over old aliases; removed destinations must not leave broken redirects.
		if slugs[old] || !slugs[dest] {
			delete(snap.Redirects, old)
		}
	}
	for _, col := range snap.Collections {
		for _, id := range col.ArticleIDs {
			if _, ok := snap.Posts[id]; !ok {
				return Preview{}, fail("COLLECTION_DEPENDENCY", "专栏包含未发布文章；请在清单中包含相应修订或调整专栏")
			}
		}
	}
	p := Preview{ID: snap.ID, ChangeSet: id, BuildHash: a.BuildHash, Snapshot: snap, Expires: time.Now().Add(2 * time.Hour).UTC().Format(time.RFC3339Nano)}
	h, err := a.build(snap, "previews")
	if err != nil {
		return Preview{}, err
	}
	p.Hash = h
	err = a.Store.Update(func(st *State) error {
		if err := checkChanges(*st, ch); err != nil {
			return err
		}
		st.Previews[p.ID] = p
		return nil
	})
	return p, err
}

type ConfirmInput struct {
	PreviewID     string `json:"previewId"`
	ArtifactHash  string `json:"artifactHash"`
	ChangeSetHash string `json:"changeSetHash"`
}

func (a *App) Confirm(in ConfirmInput) (Preview, error) {
	var out Preview
	err := a.Store.Update(func(s *State) error {
		p, ok := s.Previews[in.PreviewID]
		if !ok {
			return fail("PREVIEW_NOT_FOUND", "预览不存在")
		}
		ch := s.Changes[p.ChangeSet]
		if p.Hash != in.ArtifactHash || ch.Hash != in.ChangeSetHash {
			return fail("HASH_MISMATCH", "确认内容与预览不一致")
		}
		if err := a.checkPreview(*s, p); err != nil {
			return err
		}
		p.Confirmed = uid()
		s.Previews[p.ID] = p
		out = p
		return nil
	})
	return out, err
}
func (a *App) checkPreview(s State, p Preview) error {
	expiry, _ := time.Parse(time.RFC3339Nano, p.Expires)
	if time.Now().After(expiry) {
		return fail("PREVIEW_EXPIRED", "预览已过期")
	}
	if p.BuildHash != buildFingerprint(a.Compiler.Script) {
		return fail("RENDERER_CHANGED", "模板或编译环境已改变，请重新启动服务并预览")
	}
	ch, ok := s.Changes[p.ChangeSet]
	if !ok {
		return fail("CHANGESET_NOT_FOUND", "清单已失效")
	}
	if err := checkChanges(s, ch); err != nil {
		return err
	}
	h, err := treeHash(filepath.Join(a.Store.root, "previews", p.ID))
	if err != nil || h != p.Hash {
		return fail("ARTIFACT_CORRUPT", "预览制品已改变")
	}
	return nil
}

type PublishInput struct {
	PreviewID      string `json:"previewId"`
	ConfirmationID string `json:"confirmationId"`
	ChangeSetHash  string `json:"changeSetHash"`
	ExpectedActive string `json:"expectedActiveRelease"`
	IdempotencyKey string `json:"idempotencyKey"`
}
type PublishResult struct {
	Release      string `json:"release"`
	ArtifactHash string `json:"finalArtifactHash"`
	ActivatedAt  string `json:"activatedAt"`
}

func (a *App) Publish(in PublishInput) (PublishResult, error) {
	st := a.Store
	st.mu.Lock()
	defer st.mu.Unlock()
	s := clone(st.state)
	var result PublishResult
	if reused, err := replay(&s, "publish:"+in.IdempotencyKey, in, &result); reused || err != nil {
		return result, err
	}
	p, ok := s.Previews[in.PreviewID]
	if !ok {
		return result, fail("PREVIEW_NOT_FOUND", "预览不存在")
	}
	if p.Confirmed == "" || p.Confirmed != in.ConfirmationID {
		return result, fail("CONFIRMATION_REQUIRED", "请先确认预览")
	}
	ch := s.Changes[p.ChangeSet]
	if ch.Hash != in.ChangeSetHash || s.Active != in.ExpectedActive {
		return result, fail("RELEASE_CONFLICT", "清单或公开版本不匹配")
	}
	if err := a.checkPreview(s, p); err != nil {
		return result, err
	}
	snap := clone(p.Snapshot)
	snap.ID = uid()
	stamp := now()
	// The compiled body is frozen in the preview. Only timestamps and their derived
	// template fields are finalized here; no Markdown/KaTeX/Mermaid call is made.
	for _, id := range ch.Revisions {
		r := s.Revisions[id]
		r.Compiled = p.Snapshot.Posts[r.Content.ID].Compiled
		r.Bundle = p.Snapshot.Posts[r.Content.ID].Bundle
		r.RendererHash = p.Snapshot.Posts[r.Content.ID].RendererHash
		article := s.Articles[r.Content.ID]
		if article.FirstPublished == "" {
			article.FirstPublished = stamp
		}
		r.PublishedAt = article.FirstPublished
		r.UpdatedAt = stamp
		if old, ok := st.active.Load().Posts[r.Content.ID]; ok && old.Hash == r.Hash {
			r.UpdatedAt = old.UpdatedAt
		}
		article.Published = r.ID
		article.History = append([]string{r.ID}, removeString(article.History, r.ID)...)
		if len(article.History) > 4 {
			article.History = article.History[:4]
		}
		if article.Draft == r.ID {
			article.Draft = ""
		}
		s.Articles[article.ID] = article
		s.Revisions[r.ID] = r
		snap.Posts[article.ID] = r
	}
	for _, id := range ch.Withdraw {
		v := s.Articles[id]
		if v.Draft == "" {
			v.Draft = v.Published
		}
		v.Published = ""
		s.Articles[id] = v
	}
	hash, err := a.build(snap, "releases")
	if err != nil {
		return result, err
	}
	s.Active = snap.ID
	for _, r := range snap.Posts {
		if stored, ok := s.Revisions[r.ID]; ok {
			stored.Compiled = r.Compiled
			stored.Bundle = r.Bundle
			stored.RendererHash = r.RendererHash
			s.Revisions[r.ID] = stored
		}
	}
	s.ActiveHash = hash
	s.Generation++
	result = PublishResult{Release: snap.ID, ArtifactHash: hash, ActivatedAt: now()}
	s.event("published", "静态版本已生效")
	urls := []string{}
	for _, id := range ch.Revisions {
		r := s.Revisions[id]
		if r.Content.Kind == "article" {
			urls = append(urls, snap.Settings.Domain+contentPath(r.Content))
		}
	}
	if len(urls) > 0 {
		job := DiscoveryJob{ID: uid(), Release: snap.ID, URLs: urls, Status: "not_configured", NextAt: now()}
		if s.Integrations.Enabled && s.Integrations.BaiduToken != "" {
			job.Status = "pending"
		}
		s.Jobs[job.ID] = job
	}
	remember(&s, "publish:"+in.IdempotencyKey, in, result)
	// Every previous confirmation is stale after activation. Consumed working
	// copies are retired, while unsubmitted copies and the current draft survive.
	s.Previews = map[string]Preview{}
	s.Changes = map[string]ChangeSet{}
	for id, up := range s.Uploads {
		if slices.Contains(ch.Revisions, up.Revision) {
			delete(s.Copies, up.CopyID)
			delete(s.Uploads, id)
		}
	}
	pruneRevisions(&s)
	a.files.Lock()
	defer a.files.Unlock()
	if err = st.save(s, &snap); err != nil {
		return PublishResult{}, err
	}
	st.state = s
	st.active.Store(&snap)
	if err := a.cleanArtifactsLocked(); err != nil {
		log.Printf("旧制品清理待重试：%v", err)
	}
	return result, nil
}
func removeString(list []string, value string) []string {
	result := []string{}
	for _, v := range list {
		if v != value {
			result = append(result, v)
		}
	}
	return result
}
func pruneRevisions(s *State) {
	keep := map[string]bool{}
	for _, a := range s.Articles {
		keep[a.Draft] = true
		keep[a.Published] = true
		for _, id := range a.History {
			keep[id] = true
		}
	}
	for id, cp := range s.Copies {
		created, err := time.Parse(time.RFC3339Nano, cp.CreatedAt)
		if err == nil && time.Since(created) > 30*24*time.Hour {
			delete(s.Copies, id)
			for upID, up := range s.Uploads {
				if up.CopyID == id {
					delete(s.Uploads, upID)
				}
			}
		}
	}
	for _, p := range s.Previews {
		expiry, _ := time.Parse(time.RFC3339Nano, p.Expires)
		if time.Now().Before(expiry) {
			for _, r := range p.Snapshot.Posts {
				keep[r.ID] = true
			}
		}
	}
	for id := range s.Revisions {
		if !keep[id] {
			delete(s.Revisions, id)
		}
	}
}

func (a *App) cleanArtifactsLocked() error {
	for _, kind := range []string{"releases", "previews"} {
		entries, err := os.ReadDir(filepath.Join(a.Store.root, kind))
		if err != nil {
			return err
		}
		for _, entry := range entries {
			if kind == "releases" && entry.Name() == a.Store.state.Active {
				continue
			}
			if kind == "previews" {
				if _, ok := a.Store.state.Previews[entry.Name()]; ok {
					continue
				}
				if _, err := os.Stat(filepath.Join(a.Store.root, "restore", entry.Name(), "candidate.json")); err == nil {
					continue
				}
			}
			if err = safeRemove(a.Store.root, filepath.Join(a.Store.root, kind, entry.Name())); err != nil {
				return err
			}
		}
	}
	return nil
}

type RollbackInput struct {
	ContentID string `json:"contentId"`
	Revision  string `json:"revision"`
}

func (a *App) Rollback(ctx context.Context, in RollbackInput) (Revision, error) {
	s := a.Store.Read()
	article := s.Articles[in.ContentID]
	if !slices.Contains(article.History, in.Revision) {
		return Revision{}, fail("VERSION_NOT_AVAILABLE", "该历史已不在保留范围")
	}
	r := s.Revisions[in.Revision]
	if current, ok := a.Store.active.Load().Posts[in.ContentID]; ok {
		r.Content.Slug = current.Content.Slug
	} else if len(article.History) > 0 {
		r.Content.Slug = s.Revisions[article.History[0]].Content.Slug
	}
	compiled, bundle, err := a.compileRevision(ctx, r)
	if err != nil {
		return Revision{}, err
	}
	r.ID = uid()
	r.Hash = digest(r.Content)
	r.Compiled = compiled
	r.Bundle = bundle
	r.RendererHash = a.BuildHash
	r.CreatedAt = now()
	r.UpdatedAt = ""
	err = a.Store.Update(func(s *State) error { s.Revisions[r.ID] = r; return nil })
	return r, err
}
func treeHash(dir string) (string, error) {
	files := map[string]string{}
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(dir, path)
		files[filepath.ToSlash(rel)] = hashBytes(b)
		return nil
	})
	if err != nil {
		return "", err
	}
	return digest(files), nil
}
func writeJSON(path string, value any) error {
	b, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0600)
}
