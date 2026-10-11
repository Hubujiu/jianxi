package blog

import (
	"archive/zip"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Permanent deletion is two-phase: the caller first previews/confirms withdrawal,
// then explicitly deletes the stable ID. A durable cleanup marker survives a crash.
type DeleteInput struct {
	ContentID          string `json:"contentId"`
	ConfirmContentID   string `json:"confirmContentId"`
	ExpectedGeneration int64  `json:"expectedGeneration"`
}
type DeleteResult struct {
	Status   string `json:"status"`
	Boundary string `json:"boundary"`
}

func (a *App) Delete(in DeleteInput) (DeleteResult, error) {
	result := DeleteResult{Boundary: "已下载到外部的备份、第三方缓存和外链媒体不在本系统删除范围内。"}
	if in.ContentID == "" || in.ConfirmContentID != in.ContentID {
		return result, fail("CONFIRMATION_REQUIRED", "请明确确认要永久删除的内容 ID")
	}
	s := a.Store
	s.mu.Lock()
	defer s.mu.Unlock()
	next := clone(s.state)
	if next.Generation != in.ExpectedGeneration {
		return result, fail("REVISION_CONFLICT", "业务状态已改变，请重新检查删除范围")
	}
	article, ok := next.Articles[in.ContentID]
	if !ok {
		return result, fail("CONTENT_NOT_FOUND", "内容不存在")
	}
	if article.Published != "" {
		return result, fail("WITHDRAW_FIRST", "请先预览并确认下架，确保公开站点不再包含此内容")
	}
	a.files.Lock()
	defer a.files.Unlock()
	for _, col := range next.Collections {
		if !col.Deleted && contains(col.ArticleIDs, in.ContentID) {
			return result, fail("COLLECTION_DEPENDENCY", "请先从专栏中移除该内容并发布变更")
		}
	}
	backups, err := a.affectedBackups(in.ContentID)
	if err != nil {
		return result, err
	}
	next.CleanupBackups = backups
	assetCandidates := map[string]bool{}
	collectAssets := func(content Content) {
		for id := range next.Assets {
			if strings.Contains(content.Markdown, "/media/"+id) {
				assetCandidates[id] = true
			}
		}
	}
	titles := []string{}
	for id, r := range next.Revisions {
		if r.Content.ID == in.ContentID {
			collectAssets(r.Content)
			for _, assetID := range r.Compiled.Assets {
				assetCandidates[assetID] = true
			}
			titles = append(titles, r.Content.Title)
			delete(next.Revisions, id)
		}
	}
	for id, cp := range next.Copies {
		if cp.Content.ID == in.ContentID {
			collectAssets(cp.Content)
			delete(next.Copies, id)
		}
	}
	for id, up := range next.Uploads {
		if up.Content.ID == in.ContentID {
			collectAssets(up.Content)
			delete(next.Uploads, id)
		}
	}
	delete(next.Articles, in.ContentID)
	for id := range assetCandidates {
		next.CleanupAssets = append(next.CleanupAssets, id)
	}
	// Old confirmations and retries must not resurrect deleted payloads.
	next.Previews = map[string]Preview{}
	next.Changes = map[string]ChangeSet{}
	next.Keys = map[string]Idempotency{}
	next.Epoch = uid()
	next.Generation++
	events := []Event{}
	for _, e := range next.Events {
		keep := true
		for _, title := range titles {
			if strings.Contains(e.Message, title) {
				keep = false
			}
		}
		if keep {
			events = append(events, e)
		}
	}
	next.Events = events
	next.event("deletion", "永久删除：正在清理系统内副本")
	next.CleanupPending = true
	if err := s.save(next, nil); err != nil {
		return result, err
	}
	s.state = next
	if err := a.finishDeletionLocked(); err != nil {
		result.Status = "cleanup_pending"
		return result, err
	}
	result.Status = "deleted"
	return result, nil
}
func (a *App) finishDeletionLocked() error {
	s := a.Store
	for _, kind := range []string{"previews", "restore", "releases"} {
		entries, err := os.ReadDir(filepath.Join(s.root, kind))
		if err != nil {
			return err
		}
		for _, entry := range entries {
			if kind == "releases" && entry.Name() == s.state.Active {
				continue
			}
			if err = safeRemove(s.root, filepath.Join(s.root, kind, entry.Name())); err != nil {
				return err
			}
		}
	}
	for _, name := range s.state.CleanupBackups {
		if filepath.Base(name) != name || strings.ContainsAny(name, "/\\") {
			return fail("UNSAFE_PATH", "备份路径不合法")
		}
		if err := os.Remove(filepath.Join(s.root, "backups", name)); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	used := map[string]bool{}
	for _, r := range s.state.Revisions {
		for _, id := range r.Compiled.Assets {
			used[id] = true
		}
	}
	for _, cp := range s.state.Copies {
		for id := range s.state.Assets {
			if strings.Contains(cp.Content.Markdown, "/media/"+id) {
				used[id] = true
			}
		}
	}
	for _, up := range s.state.Uploads {
		for id := range s.state.Assets {
			if strings.Contains(up.Content.Markdown, "/media/"+id) {
				used[id] = true
			}
		}
	}
	next := clone(s.state)
	for id := range next.Assets {
		if !used[id] && contains(next.CleanupAssets, id) {
			if err := os.Remove(filepath.Join(s.root, "assets", id)); err != nil && !os.IsNotExist(err) {
				return err
			}
			delete(next.Assets, id)
		}
	}
	next.event("deleted", "永久删除已完成；系统管理的旧制品与备份已清理")
	if err := s.save(next, nil); err != nil {
		return err
	}
	s.state = next
	// SQLite pages and WAL are system-managed recoverable copies too.
	if _, err := s.db.Exec("PRAGMA wal_checkpoint(TRUNCATE); VACUUM; PRAGMA wal_checkpoint(TRUNCATE);"); err != nil {
		return err
	}
	next.CleanupPending = false
	next.CleanupBackups = nil
	next.CleanupAssets = nil
	if err := s.save(next, nil); err != nil {
		return err
	}
	s.state = next
	return nil
}

func (a *App) affectedBackups(contentID string) ([]string, error) {
	names := []string{}
	entries, err := os.ReadDir(filepath.Join(a.Store.root, "backups"))
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".zip") {
			continue
		}
		z, err := zip.OpenReader(filepath.Join(a.Store.root, "backups", entry.Name()))
		if err != nil {
			return nil, fail("BACKUP_INSPECTION_FAILED", "无法确认旧备份是否包含目标内容")
		}
		var snapshot State
		found := false
		for _, f := range z.File {
			if f.Name != "state.json" {
				continue
			}
			r, err := f.Open()
			if err != nil {
				z.Close()
				return nil, err
			}
			data, err := io.ReadAll(io.LimitReader(r, 256<<20))
			r.Close()
			if err != nil || json.Unmarshal(data, &snapshot) != nil {
				z.Close()
				return nil, fail("BACKUP_INSPECTION_FAILED", "旧备份业务快照无效")
			}
			found = true
			break
		}
		z.Close()
		if !found {
			return nil, fail("BACKUP_INSPECTION_FAILED", "旧备份缺少业务快照")
		}
		affected := false
		if _, ok := snapshot.Articles[contentID]; ok {
			affected = true
		}
		for _, r := range snapshot.Revisions {
			if r.Content.ID == contentID {
				affected = true
			}
		}
		for _, cp := range snapshot.Copies {
			if cp.Content.ID == contentID {
				affected = true
			}
		}
		for _, up := range snapshot.Uploads {
			if up.Content.ID == contentID {
				affected = true
			}
		}
		for _, preview := range snapshot.Previews {
			if _, ok := preview.Snapshot.Posts[contentID]; ok {
				affected = true
			}
		}
		if affected {
			names = append(names, entry.Name())
		}
	}
	return names, nil
}
func safeRemove(root, target string) error {
	root, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	target, err = filepath.Abs(target)
	if err != nil {
		return err
	}
	relative, err := filepath.Rel(root, target)
	if err != nil || relative == "." || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return fail("UNSAFE_PATH", "拒绝清理数据目录之外的路径")
	}
	return os.RemoveAll(target)
}
