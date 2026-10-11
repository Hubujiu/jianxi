package blog

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type IDInput struct {
	ID string `json:"id"`
}
type GetInput struct {
	ContentID string `json:"contentId"`
	Revision  string `json:"revision,omitempty"`
	View      string `json:"view,omitempty"`
}
type ListInput struct {
	Query  string `json:"query,omitempty"`
	View   string `json:"view,omitempty"`
	Offset int    `json:"offset,omitempty"`
	Limit  int    `json:"limit,omitempty"`
}

func (a *App) dispatch(r *http.Request, op string) (any, error) {
	switch op {
	case "recompile":
		var in RecompileInput
		if err := decode(r, &in); err != nil {
			return nil, err
		}
		return a.RecompileRevision(r.Context(), in)
	case "backup-import":
		var in struct {
			Data string `json:"base64"`
		}
		if err := decodeLimit(r, &in, 140<<20); err != nil {
			return nil, err
		}
		data, err := base64.StdEncoding.DecodeString(in.Data)
		if err != nil {
			return nil, fail("INVALID_BACKUP", "备份 base64 无效")
		}
		candidate, err := a.InspectBackup(r.Context(), data)
		if err != nil {
			return nil, err
		}
		return map[string]any{"id": candidate.ID, "artifactHash": candidate.Hash, "generation": candidate.Generation, "epoch": candidate.Epoch, "expires": candidate.Expires, "url": a.Auth.adminBase + "previews/" + candidate.ID + "/"}, nil
	case "restore":
		var in RestoreInput
		if err := decode(r, &in); err != nil {
			return nil, err
		}
		return a.Restore(in.ID, in.Hash)
	case "upload-compiled":
		var in CompiledUploadInput
		if err := decode(r, &in); err != nil {
			return nil, err
		}
		return a.UploadCompiled(r.Context(), in)
	case "upload-get":
		var in IDInput
		if err := decode(r, &in); err != nil {
			return nil, err
		}
		up, ok := a.Store.Read().Uploads[in.ID]
		if !ok {
			return nil, fail("UPLOAD_NOT_FOUND", "上传不存在")
		}
		return up, nil
	case "parse-compiled":
		var in ParseBundleInput
		if err := decode(r, &in); err != nil {
			return nil, err
		}
		return a.ParseBundle(r.Context(), in.ID, in.Bundle)
	case "report-csv":
		var in ReportCSVInput
		if err := decode(r, &in); err != nil {
			return nil, err
		}
		return a.ImportCSV(in)
	case "asset-list":
		var in struct{}
		if err := decode(r, &in); err != nil {
			return nil, err
		}
		return a.Store.Read().Assets, nil
	case "delete":
		var in DeleteInput
		if err := decode(r, &in); err != nil {
			return nil, err
		}
		return a.Delete(in)
	case "status":
		var in struct{}
		if err := decode(r, &in); err != nil {
			return nil, err
		}
		s := a.Store.Read()
		legacy := 0
		for _, revision := range s.Revisions {
			if revision.Bundle == nil || revision.Bundle.Renderer != RendererVersion {
				legacy++
			}
		}
		return map[string]any{"activeRelease": s.Active, "epoch": s.Epoch, "generation": s.Generation, "articles": len(s.Articles), "events": s.Events, "jobs": s.Jobs, "renderer": RendererVersion, "compilationLocation": "local-mcp", "recompileNeeded": legacy}, nil
	case "rules":
		var in struct{}
		if err := decode(r, &in); err != nil {
			return nil, err
		}
		return contentRules(), nil
	case "list":
		var in ListInput
		if err := decode(r, &in); err != nil {
			return nil, err
		}
		s := a.Store.Read()
		list := []map[string]any{}
		for _, v := range s.Articles {
			id := v.Draft
			if in.View == "published" || id == "" {
				id = v.Published
			}
			rev, ok := s.Revisions[id]
			if !ok || !strings.Contains(strings.ToLower(rev.Content.Title+rev.Content.Markdown), strings.ToLower(in.Query)) {
				continue
			}
			list = append(list, map[string]any{"article": v, "title": rev.Content.Title, "revision": id, "sourceHash": rev.Hash, "kind": rev.Content.Kind})
		}
		sort.Slice(list, func(i, j int) bool { return list[i]["title"].(string) < list[j]["title"].(string) })
		limit := in.Limit
		if limit <= 0 || limit > 100 {
			limit = 30
		}
		offset := max(0, in.Offset)
		return map[string]any{"items": list[min(offset, len(list)):min(offset+limit, len(list))], "total": len(list), "epoch": s.Epoch}, nil
	case "get":
		var in GetInput
		if err := decode(r, &in); err != nil {
			return nil, err
		}
		s := a.Store.Read()
		v, ok := s.Articles[in.ContentID]
		if !ok {
			return nil, fail("CONTENT_NOT_FOUND", "内容不存在")
		}
		id := in.Revision
		if id == "" {
			id = v.Draft
			if in.View == "published" || id == "" {
				id = v.Published
			}
		}
		rev, ok := s.Revisions[id]
		if !ok || rev.Content.ID != in.ContentID {
			return nil, fail("VERSION_NOT_AVAILABLE", "修订不可用")
		}
		return map[string]any{"article": v, "revision": rev, "epoch": s.Epoch}, nil
	case "working-copy":
		var in CopyInput
		if err := decode(r, &in); err != nil {
			return nil, err
		}
		return a.WorkingCopy(in)
	case "upload":
		var in UploadInput
		if err := decode(r, &in); err != nil {
			return nil, err
		}
		return a.Upload(in)
	case "parse":
		var in IDInput
		if err := decode(r, &in); err != nil {
			return nil, err
		}
		return a.Parse(r.Context(), in.ID)
	case "commit":
		var in CommitInput
		if err := decode(r, &in); err != nil {
			return nil, err
		}
		return a.Commit(in)
	case "taxonomy-list":
		var in struct{}
		if err := decode(r, &in); err != nil {
			return nil, err
		}
		return a.Store.Read().Taxonomy, nil
	case "taxonomy":
		var in TaxonomyInput
		if err := decode(r, &in); err != nil {
			return nil, err
		}
		return a.Taxonomy(in)
	case "collection":
		var in CollectionInput
		if err := decode(r, &in); err != nil {
			return nil, err
		}
		return a.Collection(in)
	case "settings-get":
		var in struct{}
		if err := decode(r, &in); err != nil {
			return nil, err
		}
		s := a.Store.Read()
		return map[string]any{"settings": s.Settings, "integrations": map[string]any{"baiduSite": s.Integrations.BaiduSite, "baiduTokenConfigured": s.Integrations.BaiduToken != "", "enabled": s.Integrations.Enabled, "seoConsole": s.Integrations.SEOConsole, "geoConsole": s.Integrations.GEOConsole}, "collections": s.Collections}, nil
	case "settings":
		var in SettingsInput
		if err := decode(r, &in); err != nil {
			return nil, err
		}
		return a.Settings(in)
	case "changeset":
		var in ChangesInput
		if err := decode(r, &in); err != nil {
			return nil, err
		}
		return a.Changes(in)
	case "preview":
		var in IDInput
		if err := decode(r, &in); err != nil {
			return nil, err
		}
		p, err := a.Preview(r.Context(), in.ID)
		if err != nil {
			return nil, err
		}
		return map[string]any{"previewId": p.ID, "artifactHash": p.Hash, "changeSetHash": a.Store.Read().Changes[p.ChangeSet].Hash, "url": a.Auth.adminBase + "previews/" + p.ID + "/", "expires": p.Expires}, nil
	case "confirm":
		var in ConfirmInput
		if err := decode(r, &in); err != nil {
			return nil, err
		}
		p, err := a.Confirm(in)
		return map[string]string{"confirmationId": p.Confirmed, "previewId": p.ID}, err
	case "publish":
		var in PublishInput
		if err := decode(r, &in); err != nil {
			return nil, err
		}
		return a.Publish(in)
	case "history":
		var in IDInput
		if err := decode(r, &in); err != nil {
			return nil, err
		}
		s := a.Store.Read()
		v, ok := s.Articles[in.ID]
		if !ok {
			return nil, fail("CONTENT_NOT_FOUND", "内容不存在")
		}
		out := []Revision{}
		for _, id := range v.History {
			out = append(out, s.Revisions[id])
		}
		return out, nil
	case "rollback":
		var in RollbackInput
		if err := decode(r, &in); err != nil {
			return nil, err
		}
		return a.Rollback(r.Context(), in)
	case "asset":
		var in AssetInput
		if err := decode(r, &in); err != nil {
			return nil, err
		}
		return a.StoreAsset(r, in)
	case "integrations":
		var in IntegrationInput
		if err := decode(r, &in); err != nil {
			return nil, err
		}
		return map[string]bool{"saved": true}, a.SetIntegrations(in)
	case "report":
		var in Report
		if err := decode(r, &in); err != nil {
			return nil, err
		}
		return a.ImportReport(in)
	case "retry":
		var in IDInput
		if err := decode(r, &in); err != nil {
			return nil, err
		}
		err := a.Store.Update(func(s *State) error {
			j, ok := s.Jobs[in.ID]
			if !ok {
				return fail("JOB_NOT_FOUND", "任务不存在")
			}
			if j.Status == "accepted" {
				return nil
			}
			j.Status = "pending"
			j.NextAt = now()
			s.Jobs[in.ID] = j
			return nil
		})
		return map[string]bool{"queued": err == nil}, err
	case "backup":
		var in struct{}
		if err := decode(r, &in); err != nil {
			return nil, err
		}
		name, err := a.Export()
		return map[string]string{"file": name, "downloadURL": a.Auth.adminBase + "backups/" + name, "sensitivity": "完整备份包含第三方密钥，请妥善保管"}, err
	default:
		return nil, fail("UNKNOWN_OPERATION", op)
	}
}

type AssetInput struct {
	Data string `json:"base64"`
	Alt  string `json:"alt"`
}

func (a *App) StoreAsset(r *http.Request, in AssetInput) (Asset, error) {
	var out Asset
	b, err := base64.StdEncoding.DecodeString(in.Data)
	if err != nil || len(b) > 10<<20 || len(b) < 12 {
		return out, fail("INVALID_IMAGE", "图片不合法或超过 10 MiB")
	}
	metadata, err := inspectImage(b)
	if err != nil {
		return out, err
	}
	format := metadata.Format
	ext := format
	if ext == "jpeg" {
		ext = "jpg"
	}
	id := hashBytes(b)
	out = Asset{ID: id, Hash: id, MIME: "image/" + format, Ext: ext, Width: metadata.Width, Height: metadata.Height, Alt: in.Alt}
	if err = os.WriteFile(filepath.Join(a.Store.root, "assets", id), b, 0600); err != nil {
		return out, err
	}
	err = a.Store.Update(func(s *State) error { s.Assets[id] = out; return nil })
	return out, err
}

type IntegrationInput struct {
	BaiduSite  string `json:"baiduSite"`
	BaiduToken string `json:"baiduToken,omitempty"`
	ClearToken bool   `json:"clearToken"`
	Enabled    bool   `json:"enabled"`
	SEOConsole string `json:"seoConsole"`
	GEOConsole string `json:"geoConsole"`
}

func (a *App) SetIntegrations(in IntegrationInput) error {
	for _, u := range []string{in.BaiduSite, in.SEOConsole, in.GEOConsole} {
		if u != "" && !safeURL(u, false) {
			return fail("INVALID_URL", "服务地址必须使用 HTTPS")
		}
	}
	return a.Store.Update(func(s *State) error {
		s.Integrations.BaiduSite = in.BaiduSite
		s.Integrations.Enabled = in.Enabled
		s.Integrations.SEOConsole = in.SEOConsole
		s.Integrations.GEOConsole = in.GEOConsole
		if in.ClearToken {
			s.Integrations.BaiduToken = ""
		} else if in.BaiduToken != "" {
			s.Integrations.BaiduToken = in.BaiduToken
		}
		return nil
	})
}
func (a *App) ImportReport(in Report) (Report, error) {
	if in.Version != 1 || (in.Kind != "seo" && in.Kind != "geo") || in.Source == "" || in.Definition == "" || len(in.Rows) > 10000 {
		return in, fail("INVALID_REPORT", "需要 version=1、seo/geo 类型、来源与指标定义，最多 10000 行")
	}
	if _, err := time.Parse(time.RFC3339, in.ObservedAt); err != nil {
		return in, fail("INVALID_REPORT", "观察时间必须为 RFC3339")
	}
	for i := range in.Rows {
		r := &in.Rows[i]
		if r.Platform == "" || r.Query == "" || !safeURL(r.URL, false) {
			return in, fail("INVALID_REPORT", "每行需要平台、查询和有效 URL")
		}
		if r.Evidence == "" {
			r.Status = "供应商报告、未核验"
		} else {
			r.Status = "已附证据、待核验"
		}
	}
	in.ID = uid()
	err := a.Store.Update(func(s *State) error { s.Reports[in.ID] = in; return nil })
	return in, err
}
func requestJSON(value any) *strings.Reader {
	b, _ := json.Marshal(value)
	return strings.NewReader(string(b))
}
