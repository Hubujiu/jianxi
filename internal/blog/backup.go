package blog

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type BackupManifest struct {
	Version   int               `json:"version"`
	Renderer  string            `json:"renderer"`
	CreatedAt string            `json:"createdAt"`
	Sensitive bool              `json:"containsPrivateCredentials"`
	Files     map[string]string `json:"files"`
}
type RestoreCandidate struct {
	ID         string   `json:"id"`
	Generation int64    `json:"generation"`
	Epoch      string   `json:"epoch"`
	State      State    `json:"state"`
	Snapshot   Snapshot `json:"snapshot"`
	Hash       string   `json:"hash"`
	Expires    string   `json:"expires"`
}

func (a *App) Export() (string, error) {
	a.Store.mu.Lock()
	defer a.Store.mu.Unlock()
	return a.exportLocked(a.Store.state, *a.Store.active.Load())
}
func (a *App) exportLocked(s State, snap Snapshot) (string, error) {
	files := map[string][]byte{}
	files["state.json"], _ = json.MarshalIndent(s, "", "  ")
	files["snapshot.json"], _ = json.MarshalIndent(snap, "", "  ")
	for id, r := range s.Revisions {
		files["content/"+id+".md"] = []byte(r.Content.Markdown)
	}
	for id := range s.Assets {
		b, err := os.ReadFile(filepath.Join(a.Store.root, "assets", id))
		if err != nil {
			return "", err
		}
		files["assets/"+id] = b
	}
	manifest := BackupManifest{Version: 1, Renderer: RendererVersion, CreatedAt: now(), Sensitive: true, Files: map[string]string{}}
	for name, b := range files {
		manifest.Files[name] = hashBytes(b)
	}
	mb, _ := json.MarshalIndent(manifest, "", "  ")
	files["manifest.json"] = mb
	name := "jianxi-" + time.Now().UTC().Format("20060102T150405") + "-" + uid()[:8] + ".zip"
	path := filepath.Join(a.Store.root, "backups", name)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return "", err
	}
	z := zip.NewWriter(f)
	for name, b := range files {
		w, err := z.Create(name)
		if err != nil {
			z.Close()
			f.Close()
			return "", err
		}
		if _, err = w.Write(b); err != nil {
			z.Close()
			f.Close()
			return "", err
		}
	}
	if err = z.Close(); err != nil {
		f.Close()
		return "", err
	}
	if err = f.Sync(); err != nil {
		f.Close()
		return "", err
	}
	return name, f.Close()
}
func strictJSON(b []byte, out any) error {
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if err := d.Decode(out); err != nil {
		return err
	}
	if d.Decode(new(any)) != io.EOF {
		return fail("INVALID_BACKUP", "多余 JSON 数据")
	}
	return nil
}
func (a *App) InspectBackup(ctx context.Context, data []byte) (RestoreCandidate, error) {
	var out RestoreCandidate
	files, _, err := readBackupFiles(data, false)
	if err != nil {
		return out, err
	}
	var s State
	var snap Snapshot
	if err = strictJSON(files["state.json"], &s); err != nil {
		return out, err
	}
	if err = strictJSON(files["snapshot.json"], &snap); err != nil {
		return out, err
	}
	if s.Version != 1 || s.Active != snap.ID || s.Articles == nil || s.Revisions == nil || s.Assets == nil || s.Copies == nil || s.Uploads == nil || s.Taxonomy == nil || s.Collections == nil || s.Jobs == nil || s.Reports == nil || s.Keys == nil || s.Previews == nil || s.Changes == nil {
		return out, fail("INVALID_BACKUP", "业务结构或活动版本无效")
	}
	if err = validateSnapshot(snap); err != nil {
		return out, err
	}
	for id, t := range s.Taxonomy {
		if id != t.ID || !slugPattern.MatchString(t.Slug) || (t.Kind != "category" && t.Kind != "tag") {
			return out, fail("INVALID_BACKUP", "私有分类或标签无效")
		}
	}
	for id, c := range s.Collections {
		if id != c.ID || !slugPattern.MatchString(c.Slug) {
			return out, fail("INVALID_BACKUP", "私有专栏无效")
		}
	}
	snap.Assets = map[string]Asset{}
	for id, r := range s.Revisions {
		if !hexID(id) || len(id) != 32 || r.ID != id || r.Hash != digest(r.Content) || string(files["content/"+id+".md"]) != r.Content.Markdown {
			return out, fail("INVALID_BACKUP", "修订与源文不一致")
		}
		compiled, bundle, err := a.compileRevision(ctx, r)
		if err != nil {
			return out, err
		}
		r.Compiled = compiled
		r.Bundle = bundle
		r.RendererHash = a.BuildHash
		s.Revisions[id] = r
	}
	for id, r := range snap.Posts {
		stored, ok := s.Revisions[r.ID]
		if !ok || stored.Content.ID != id || stored.Hash != r.Hash || s.Articles[id].Published != r.ID {
			return out, fail("INVALID_BACKUP", "公开快照与修订不一致")
		}
		if err = validateRefs(stored.Content, snap.Taxonomy, s.Assets); err != nil {
			return out, err
		}
		snap.Posts[id] = stored
		for _, assetID := range stored.Compiled.Assets {
			asset, ok := s.Assets[assetID]
			if !ok {
				return out, fail("ASSET_MISSING", "公开内容引用缺失媒体")
			}
			snap.Assets[assetID] = asset
		}
	}
	for id, article := range s.Articles {
		if id != article.ID || len(article.History) > 4 {
			return out, fail("INVALID_BACKUP", "历史关联无效")
		}
		for _, revID := range append(append([]string{}, article.History...), article.Draft, article.Published) {
			if revID != "" {
				r, ok := s.Revisions[revID]
				if !ok || r.Content.ID != id {
					return out, fail("INVALID_BACKUP", "缺失内容修订")
				}
			}
		}
	}
	out.ID = uid()
	dir := filepath.Join(a.Store.root, "restore", out.ID)
	if err = os.MkdirAll(filepath.Join(dir, "assets"), 0700); err != nil {
		return out, err
	}
	for id, asset := range s.Assets {
		b, ok := files["assets/"+id]
		if !ok || len(id) != 64 || !hexID(id) || asset.ID != id || hashBytes(b) != asset.Hash {
			return out, fail("INVALID_BACKUP", "媒体校验失败")
		}
		metadata, err := inspectImage(b)
		if err != nil {
			return out, err
		}
		if asset.MIME != "image/"+metadata.Format || asset.Width != metadata.Width || asset.Height != metadata.Height {
			return out, fail("INVALID_BACKUP", "媒体真实类型或尺寸不匹配")
		}
		if err = os.WriteFile(filepath.Join(dir, "assets", id), b, 0600); err != nil {
			return out, err
		}
	}
	current := a.Store.Read()
	out.Generation = current.Generation
	out.Epoch = current.Epoch
	snap.ID = out.ID
	out.State = s
	out.Snapshot = snap
	out.Expires = time.Now().Add(time.Hour).UTC().Format(time.RFC3339Nano)
	h, err := a.build(snap, "previews")
	if err != nil {
		return out, err
	}
	out.Hash = h
	if err = writeJSON(filepath.Join(dir, "candidate.json"), out); err != nil {
		return out, err
	}
	return out, nil
}
func (a *App) Restore(id, hash string) (PublishResult, error) {
	st := a.Store
	st.mu.Lock()
	defer st.mu.Unlock()
	var result PublishResult
	if len(id) != 32 || !hexID(id) {
		return result, fail("INVALID_RESTORE", "候选不存在")
	}
	dir := filepath.Join(st.root, "restore", id)
	b, err := os.ReadFile(filepath.Join(dir, "candidate.json"))
	if err != nil {
		return result, err
	}
	var candidate RestoreCandidate
	if err = json.Unmarshal(b, &candidate); err != nil {
		return result, err
	}
	expires, _ := time.Parse(time.RFC3339Nano, candidate.Expires)
	if time.Now().After(expires) || hash != candidate.Hash {
		return result, fail("RESTORE_EXPIRED", "恢复候选已过期或 hash 不匹配")
	}
	if candidate.Generation != st.state.Generation || candidate.Epoch != st.state.Epoch {
		return result, fail("RESTORE_CONFLICT", "预览后业务数据发生改变，请重新导入")
	}
	h, err := treeHash(filepath.Join(st.root, "previews", id))
	if err != nil || h != hash {
		return result, fail("ARTIFACT_CORRUPT", "恢复预览制品校验失败")
	}
	backup, err := a.exportLocked(st.state, *st.active.Load())
	if err != nil {
		return result, err
	}
	s := candidate.State
	s.CleanupPending = false
	s.CleanupBackups = nil
	s.CleanupAssets = nil
	s.Epoch = uid()
	s.Generation = st.state.Generation + 1
	s.Previews = map[string]Preview{}
	s.Changes = map[string]ChangeSet{}
	s.Keys = map[string]Idempotency{}
	for id, j := range s.Jobs {
		if j.Status != "accepted" {
			j.Status = "aborted"
			j.Message = "备份恢复后不自动重放网络请求"
		}
		s.Jobs[id] = j
	}
	for id, up := range s.Uploads {
		up.Status = "uploaded"
		up.Compiled = Compiled{}
		up.Bundle = nil
		up.RendererHash = ""
		up.Revision = ""
		s.Uploads[id] = up
	}
	snap := candidate.Snapshot
	snap.ID = uid()
	for id, asset := range s.Assets {
		data, err := os.ReadFile(filepath.Join(dir, "assets", id))
		if err != nil || hashBytes(data) != asset.Hash {
			return result, fail("ASSET_CORRUPT", "恢复媒体被修改")
		}
		if err = os.WriteFile(filepath.Join(st.root, "assets", id), data, 0600); err != nil {
			return result, err
		}
	}
	finalHash, err := a.build(snap, "releases")
	if err != nil {
		return result, err
	}
	s.Active = snap.ID
	s.ActiveHash = finalHash
	s.event("restored", fmt.Sprintf("整站恢复完成；恢复前备份 %s", backup))
	if err = st.save(s, &snap); err != nil {
		return result, err
	}
	st.state = s
	st.active.Store(&snap)
	a.Auth.Invalidate()
	return PublishResult{Release: snap.ID, ArtifactHash: finalHash, ActivatedAt: now()}, nil
}

func readBackupFiles(data []byte, allowLegacy bool) (map[string][]byte, BackupManifest, error) {
	if len(data) > 100<<20 {
		return nil, BackupManifest{}, fail("BACKUP_TOO_LARGE", "备份压缩文件最多 100 MiB")
	}
	z, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, BackupManifest{}, err
	}
	files := map[string][]byte{}
	var size uint64
	for _, f := range z.File {
		if f.UncompressedSize64 > (256<<20)-size || len(files) > 20000 {
			return nil, BackupManifest{}, fail("BACKUP_TOO_LARGE", "解压后最多 256 MiB / 20000 个文件")
		}
		size += f.UncompressedSize64
		name := f.Name
		if name == "" || strings.Contains(name, "..") || strings.ContainsAny(name, "\\:") || strings.HasPrefix(name, "/") || files[name] != nil {
			return nil, BackupManifest{}, fail("INVALID_BACKUP", "非法或重复的文件路径")
		}
		reader, err := f.Open()
		if err != nil {
			return nil, BackupManifest{}, err
		}
		b, err := io.ReadAll(io.LimitReader(reader, 256<<20))
		reader.Close()
		if err != nil {
			return nil, BackupManifest{}, err
		}
		files[name] = b
	}
	var manifest BackupManifest
	if err = strictJSON(files["manifest.json"], &manifest); err != nil {
		return nil, BackupManifest{}, err
	}
	if manifest.Version != 1 || (manifest.Renderer != RendererVersion && !(allowLegacy && strings.HasPrefix(manifest.Renderer, "jianxi-"))) || !manifest.Sensitive {
		return nil, BackupManifest{}, fail("INCOMPATIBLE_BACKUP", "备份格式或编译器版本不兼容")
	}
	if len(files) != len(manifest.Files)+1 {
		return nil, BackupManifest{}, fail("INVALID_BACKUP", "文件清单不一致")
	}
	for name, hash := range manifest.Files {
		if hashBytes(files[name]) != hash {
			return nil, BackupManifest{}, fail("BACKUP_HASH_MISMATCH", name)
		}
	}
	return files, manifest, nil
}
