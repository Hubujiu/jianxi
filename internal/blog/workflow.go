package blog

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
)

type App struct {
	files     sync.RWMutex
	Store     *Store
	Compiler  *Compiler
	Auth      *Auth
	BuildHash string
	SiteHash  string
}

func NewApp(root, script string) (*App, error) {
	s, err := OpenStore(root)
	if err != nil {
		return nil, err
	}
	a, err := OpenAuth(root)
	if err != nil {
		s.Close()
		return nil, err
	}
	app := &App{Store: s, Compiler: NewCompiler(script), Auth: a, SiteHash: siteFingerprint()}
	app.BuildHash = buildFingerprint(script)
	if s.state.CleanupPending {
		if err = app.finishDeletionLocked(); err != nil {
			s.Close()
			return nil, err
		}
	}
	if s.active.Load() == nil {
		snap := Snapshot{ID: uid(), Settings: s.state.Settings, Posts: map[string]Revision{}, Taxonomy: map[string]Taxon{}, Collections: map[string]Collection{}, Redirects: map[string]string{}, Assets: map[string]Asset{}}
		hash, err := app.build(snap, "releases")
		if err != nil {
			s.Close()
			return nil, err
		}
		s.state.Active = snap.ID
		s.state.ActiveHash = hash
		if err = s.save(s.state, &snap); err != nil {
			s.Close()
			return nil, err
		}
		s.active.Store(&snap)
	} else {
		marker, err := os.ReadFile(filepath.Join(s.root, "releases", s.state.Active, ".sitehash"))
		needsBuild := err != nil || string(marker) != app.SiteHash
		snap := clone(*s.active.Load())
		// Renderer upgrades must also replace cached published fragments. Merely
		// rebuilding templates would preserve the old code block's global CSS.
		for id, revision := range snap.Posts {
			if revision.RendererHash == app.BuildHash {
				continue
			}
			// Existing published releases remain readable after a compiler upgrade.
			// A Node-free server never recompiles legacy rich content on startup.
			if script == "" && (revision.Bundle == nil || revision.Bundle.Renderer != RendererVersion) {
				continue
			}
			compiled, bundle, compileErr := app.compileRevision(context.Background(), revision)
			if compileErr != nil {
				s.Close()
				return nil, compileErr
			}
			revision.Compiled = compiled
			revision.Bundle = bundle
			revision.RendererHash = app.BuildHash
			snap.Posts[id] = revision
			needsBuild = true
		}
		if needsBuild {
			snap.ID = uid()
			hash, buildErr := app.build(snap, "releases")
			if buildErr != nil {
				s.Close()
				return nil, buildErr
			}
			state := clone(s.state)
			for _, revision := range snap.Posts {
				state.Revisions[revision.ID] = revision
			}
			state.Active = snap.ID
			state.ActiveHash = hash
			state.Generation++
			if err := s.save(state, &snap); err != nil {
				s.Close()
				return nil, err
			}
			s.state = state
			s.active.Store(&snap)
		}
	}
	return app, nil
}

type CopyInput struct {
	ContentID string `json:"contentId,omitempty"`
	Revision  string `json:"revision,omitempty"`
	Kind      string `json:"kind,omitempty"`
}

func (a *App) WorkingCopy(in CopyInput) (WorkingCopy, error) {
	var copy WorkingCopy
	err := a.Store.Update(func(s *State) error {
		copy = WorkingCopy{ID: uid(), Epoch: s.Epoch, CreatedAt: now()}
		if in.ContentID == "" {
			in.ContentID = uid()
		}
		if in.Kind == "about" {
			in.ContentID = "about"
		}
		copy.Content = Content{ID: in.ContentID, Kind: "article", Title: "新文章", TagIDs: []string{}}
		if in.Kind == "about" {
			copy.Content.Kind = "about"
			copy.Content.Slug = "about"
			copy.Content.Title = "关于我"
		}
		if article, ok := s.Articles[in.ContentID]; ok {
			r := in.Revision
			if r == "" {
				r = article.Draft
				if r == "" {
					r = article.Published
				}
				if r == "" && len(article.History) > 0 {
					r = article.History[0]
				}
			}
			rev, ok := s.Revisions[r]
			if !ok || rev.Content.ID != in.ContentID {
				return fail("VERSION_NOT_AVAILABLE", "修订不存在")
			}
			copy.Content = rev.Content
			copy.Base = r
			copy.Draft = article.Draft
			copy.PublishedBase = article.Published
		}
		copy.Content = contentWithTaxonomyMetadata(copy.Content, s.Taxonomy)
		s.Copies[copy.ID] = copy
		return nil
	})
	return copy, err
}

type UploadInput struct {
	CopyID         string  `json:"copyId"`
	Content        Content `json:"content"`
	IdempotencyKey string  `json:"idempotencyKey"`
}

func (a *App) Upload(in UploadInput) (Upload, error) {
	var out Upload
	err := a.Store.Update(func(s *State) error {
		if reused, err := replay(s, "upload:"+in.IdempotencyKey, in, &out); reused || err != nil {
			return err
		}
		cp, ok := s.Copies[in.CopyID]
		if !ok || cp.Epoch != s.Epoch {
			return fail("COPY_NOT_FOUND", "请重新获取独立副本")
		}
		if cp.Content.ID != in.Content.ID || cp.Content.Kind != in.Content.Kind {
			return fail("INVALID_CONTENT", "不能修改副本的稳定 ID 或种类")
		}
		if err := validateContent(in.Content); err != nil {
			return err
		}
		content, err := normalizeUploadTaxonomy(in.Content, s.Taxonomy)
		if err != nil {
			return err
		}
		out = Upload{ID: uid(), CopyID: in.CopyID, Content: content, Hash: digest(content), Status: "uploaded"}
		s.Uploads[out.ID] = out
		remember(s, "upload:"+in.IdempotencyKey, in, out)
		return nil
	})
	return out, err
}
func (a *App) Parse(ctx context.Context, id string) (Upload, error) {
	s := a.Store.Read()
	up, ok := s.Uploads[id]
	if !ok {
		return up, fail("UPLOAD_NOT_FOUND", "上传不存在")
	}
	if up.Bundle != nil {
		return a.ParseBundle(ctx, id, up.Bundle)
	}
	compiled, bundle, compileErr := a.Compiler.CompileBundle(ctx, up.Content)
	err := a.Store.Update(func(s *State) error {
		v, ok := s.Uploads[id]
		if !ok {
			return fail("UPLOAD_NOT_FOUND", "上传已失效")
		}
		up = v
		up.Compiled = compiled
		up.Bundle = bundle
		up.RendererHash = a.BuildHash
		if compileErr != nil {
			up.Status = "invalid"
			up.Diagnostic = compileErr.Error()
		} else {
			up.Status = "parsed"
			up.Diagnostic = ""
		}
		s.Uploads[id] = up
		return nil
	})
	return up, err
}

type CommitInput struct {
	UploadID       string `json:"uploadId"`
	SourceHash     string `json:"sourceHash"`
	ExpectedDraft  string `json:"expectedDraftRevision"`
	Epoch          string `json:"epoch"`
	IdempotencyKey string `json:"idempotencyKey"`
}

func (a *App) Commit(in CommitInput) (Revision, error) {
	var out Revision
	err := a.Store.Update(func(s *State) error {
		if reused, err := replay(s, "commit:"+in.IdempotencyKey, in, &out); reused || err != nil {
			return err
		}
		if in.Epoch != s.Epoch {
			return fail("INSTANCE_CONFLICT", "实例已恢复，请重新获取副本")
		}
		up, ok := s.Uploads[in.UploadID]
		if !ok || up.Status != "parsed" {
			return fail("PARSE_REQUIRED", "请先完成上传解析")
		}
		if up.Hash != in.SourceHash {
			return fail("SOURCE_HASH_MISMATCH", "上传内容不匹配")
		}
		cp, ok := s.Copies[up.CopyID]
		if !ok || cp.Epoch != s.Epoch {
			return fail("COPY_NOT_FOUND", "副本已失效")
		}
		article := s.Articles[up.Content.ID]
		if article.Published != cp.PublishedBase {
			return fail("REVISION_CONFLICT", "副本基准公开版本已改变，请重新获取并合并")
		}
		if article.Draft != in.ExpectedDraft || cp.Draft != in.ExpectedDraft {
			return fail("REVISION_CONFLICT", "当前草稿已改变，请获取新副本并合并")
		}
		if up.Revision != "" {
			out = s.Revisions[up.Revision]
			return nil
		}
		if err := commitUploadTaxonomy(up.Content, s.Taxonomy); err != nil {
			return err
		}
		if err := validateRefs(up.Content, s.Taxonomy, s.Assets); err != nil {
			return err
		}
		out = Revision{ID: uid(), Content: up.Content, Hash: up.Hash, Compiled: up.Compiled, Bundle: up.Bundle, CreatedAt: now(), RendererHash: up.RendererHash}
		article.ID = up.Content.ID
		article.Draft = out.ID
		s.Articles[article.ID] = article
		s.Revisions[out.ID] = out
		up.Revision = out.ID
		cp.CommittedRevision = out.ID
		s.Copies[cp.ID] = cp
		s.Uploads[up.ID] = up
		s.event("draft", "已创建新草稿："+out.Content.Title)
		remember(s, "commit:"+in.IdempotencyKey, in, out)
		return nil
	})
	return out, err
}
func replay(s *State, key string, input, out any) (bool, error) {
	if strings.HasSuffix(key, ":") || len(key) > 200 {
		return false, fail("IDEMPOTENCY_KEY_REQUIRED", "请提供长度 1–160 的幂等键")
	}
	if v, ok := s.Keys[key]; ok {
		if v.Hash != digest(input) {
			return false, fail("IDEMPOTENCY_CONFLICT", "同一幂等键不能用于不同请求")
		}
		if err := json.Unmarshal(v.Result, out); err != nil {
			return true, err
		}
		switch value := out.(type) {
		case *Revision:
			current, ok := s.Revisions[value.ID]
			if !ok {
				return true, fail("VERSION_NOT_AVAILABLE", "该修订已按保留规则清理")
			}
			*value = current
		case *Upload:
			current, ok := s.Uploads[value.ID]
			if !ok {
				return true, fail("UPLOAD_EXPIRED", "已发布上传句柄已回收，请获取新副本")
			}
			*value = current
		}
		return true, nil
	}
	return false, nil
}
func remember(s *State, key string, input, out any) {
	switch value := out.(type) {
	case Revision:
		out = map[string]string{"revision": value.ID}
	case Upload:
		out = map[string]string{"id": value.ID}
	}
	b, _ := json.Marshal(out)
	s.Keys[key] = Idempotency{Hash: digest(input), Result: b}
}
func validateRefs(c Content, tax map[string]Taxon, assets map[string]Asset) error {
	if c.CategoryID != "" {
		t, ok := tax[c.CategoryID]
		if !ok || t.Kind != "category" || t.Deleted {
			return fail("TAXONOMY_MISSING", "分类未包含在候选中")
		}
	}
	seen := map[string]bool{}
	for _, id := range c.TagIDs {
		t, ok := tax[id]
		if !ok || t.Kind != "tag" || t.Deleted || seen[id] {
			return fail("TAXONOMY_MISSING", "标签不存在或重复")
		}
		seen[id] = true
	}
	return nil
}

type TaxonomyInput struct {
	Taxon            Taxon `json:"taxon"`
	ExpectedRevision int   `json:"expectedRevision"`
}

func (a *App) Taxonomy(in TaxonomyInput) (Taxon, error) {
	t := in.Taxon
	err := a.Store.Update(func(s *State) error {
		if t.Kind != "category" && t.Kind != "tag" {
			return fail("INVALID_TAXONOMY", "类型必须为 category 或 tag")
		}
		if err := validateTaxonomyMetadata(TaxonomyMetadata{Name: t.Name, Slug: t.Slug, Description: t.Description}); err != nil {
			return err
		}
		if t.ID == "" {
			t.ID = uid()
		}
		old := s.Taxonomy[t.ID]
		if old.Revision != in.ExpectedRevision {
			return fail("REVISION_CONFLICT", "分类或标签已改变")
		}
		if old.ID != "" && old.Kind != t.Kind {
			return fail("INVALID_TAXONOMY", "不能改变分类类型")
		}
		for id, other := range s.Taxonomy {
			if id != t.ID && other.Kind == t.Kind && !other.Deleted && (other.Slug == t.Slug || other.Name == t.Name) {
				return fail("TAXONOMY_CONFLICT", "名称或 slug 重复")
			}
		}
		if t.Deleted {
			for _, article := range s.Articles {
				for _, revID := range []string{article.Draft, article.Published} {
					r, ok := s.Revisions[revID]
					if ok && (r.Content.CategoryID == t.ID || slices.Contains(r.Content.TagIDs, t.ID)) {
						return fail("TAXONOMY_IN_USE", "请先通过内容修订明确迁移或解除当前文章关联，再发布变更")
					}
				}
			}
		}
		t.Revision = old.Revision + 1
		s.Taxonomy[t.ID] = t
		return nil
	})
	return t, err
}

type CollectionInput struct {
	Collection       Collection `json:"collection"`
	ExpectedRevision int        `json:"expectedRevision"`
}

func (a *App) Collection(in CollectionInput) (Collection, error) {
	v := in.Collection
	err := a.Store.Update(func(s *State) error {
		if v.ID == "" {
			v.ID = uid()
		}
		old := s.Collections[v.ID]
		if old.Revision != in.ExpectedRevision {
			return fail("REVISION_CONFLICT", "专栏已改变")
		}
		if v.Name == "" || !slugPattern.MatchString(v.Slug) || (!v.Deleted && len(v.ArticleIDs) < 2) {
			return fail("INVALID_COLLECTION", "专栏需要名称、slug 和至少两篇文章")
		}
		seen := map[string]bool{}
		for _, id := range v.ArticleIDs {
			if _, ok := s.Articles[id]; !ok || id == "about" || seen[id] {
				return fail("INVALID_COLLECTION", "文章不存在或重复")
			}
			seen[id] = true
		}
		for id, other := range s.Collections {
			if id != v.ID && !other.Deleted && other.Slug == v.Slug {
				return fail("SLUG_CONFLICT", "专栏地址重复")
			}
		}
		v.Revision = old.Revision + 1
		s.Collections[v.ID] = v
		return nil
	})
	return v, err
}

type SettingsInput struct {
	Settings         Settings `json:"settings"`
	ExpectedRevision int      `json:"expectedRevision"`
}

func (a *App) Settings(in SettingsInput) (Settings, error) {
	v := in.Settings
	err := a.Store.Update(func(s *State) error {
		if s.Settings.Revision != in.ExpectedRevision {
			return fail("REVISION_CONFLICT", "设置已改变")
		}
		u, err := url.Parse(v.Domain)
		if err != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") || (u.Scheme != "https" && !(u.Scheme == "http" && (u.Hostname() == "localhost" || u.Hostname() == "127.0.0.1"))) {
			return fail("INVALID_DOMAIN", "主域名必须为 HTTPS 域名，或本机 HTTP 地址")
		}
		if strings.TrimSpace(v.Name) == "" || len(v.Name) > 120 || len(v.Description) > 1000 {
			return fail("INVALID_SETTINGS", "站名或说明不合法")
		}
		if v.AnalyticsID != "" && (len(v.AnalyticsID) != 32 || !hexID(v.AnalyticsID)) {
			return fail("INVALID_ANALYTICS", "百度统计 ID 必须是 32 位十六进制标识")
		}
		v.Domain = strings.TrimRight(v.Domain, "/")
		v.Revision = s.Settings.Revision + 1
		s.Settings = v
		return nil
	})
	return v, err
}
func hexID(s string) bool {
	for _, r := range s {
		if !(r >= '0' && r <= '9' || r >= 'a' && r <= 'f') {
			return false
		}
	}
	return true
}

type ChangesInput struct {
	Revisions       []string `json:"revisions,omitempty"`
	Taxonomy        []string `json:"taxonomy,omitempty"`
	Collections     []string `json:"collections,omitempty"`
	IncludeSettings bool     `json:"includeSettings,omitempty"`
	Withdraw        []string `json:"withdraw,omitempty"`
	ExpectedActive  string   `json:"expectedActiveRelease"`
}

func (a *App) Changes(in ChangesInput) (ChangeSet, error) {
	var out ChangeSet
	err := a.Store.Update(func(s *State) error {
		if s.Active != in.ExpectedActive {
			return fail("RELEASE_CONFLICT", "公开版本已改变")
		}
		out = ChangeSet{ID: uid(), Base: s.Active, Epoch: s.Epoch, Revisions: in.Revisions, Taxonomy: in.Taxonomy, Collections: in.Collections, IncludeSettings: in.IncludeSettings, Withdraw: in.Withdraw, Bindings: map[string]string{}}
		if len(in.Revisions)+len(in.Taxonomy)+len(in.Collections)+len(in.Withdraw) == 0 && !in.IncludeSettings {
			return fail("EMPTY_CHANGESET", "请选择需要发布的修改")
		}
		seen := map[string]bool{}
		for _, id := range out.Revisions {
			r, ok := s.Revisions[id]
			if !ok || seen[r.Content.ID] {
				return fail("REVISION_NOT_FOUND", "修订不存在或同一内容重复")
			}
			seen[r.Content.ID] = true
			out.Bindings["revision:"+id] = r.Hash
			out.Bindings["compiled:"+id] = digest(r.Compiled)
			out.Bindings["draft:"+r.Content.ID] = s.Articles[r.Content.ID].Draft
			out.URLs = append(out.URLs, contentPath(r.Content))
		}
		for _, id := range out.Withdraw {
			v, ok := s.Articles[id]
			if !ok || v.Published == "" || seen[id] {
				return fail("INVALID_WITHDRAW", "内容未发布或在清单重复")
			}
			out.Bindings["withdraw:"+id] = v.Published
			out.URLs = append(out.URLs, contentPath(s.Revisions[v.Published].Content))
		}
		for _, id := range out.Taxonomy {
			t, ok := s.Taxonomy[id]
			if !ok {
				return fail("TAXONOMY_MISSING", id)
			}
			out.Bindings["taxonomy:"+id] = digest(t)
		}
		for _, id := range out.Collections {
			v, ok := s.Collections[id]
			if !ok {
				return fail("COLLECTION_MISSING", id)
			}
			out.Bindings["collection:"+id] = digest(v)
		}
		if out.IncludeSettings {
			out.Bindings["settings"] = digest(s.Settings)
		}
		out.Hash = digest(out)
		s.Changes[out.ID] = out
		return nil
	})
	return out, err
}
func checkChanges(s State, ch ChangeSet) error {
	if ch.Epoch != s.Epoch || ch.Base != s.Active {
		return fail("RELEASE_CONFLICT", "实例或公开版本已改变，请重新预览")
	}
	for k, want := range ch.Bindings {
		parts := strings.SplitN(k, ":", 2)
		actual := ""
		switch parts[0] {
		case "settings":
			actual = digest(s.Settings)
		case "revision":
			actual = s.Revisions[parts[1]].Hash
		case "compiled":
			actual = digest(s.Revisions[parts[1]].Compiled)
		case "draft":
			actual = s.Articles[parts[1]].Draft
		case "withdraw":
			actual = s.Articles[parts[1]].Published
		case "taxonomy":
			actual = digest(s.Taxonomy[parts[1]])
		case "collection":
			actual = digest(s.Collections[parts[1]])
		}
		if want != actual {
			return fail("PREVIEW_STALE", fmt.Sprintf("%s 已改变，请重新预览", k))
		}
	}
	return nil
}
