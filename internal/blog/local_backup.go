package blog

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
)

// PrepareBackup revalidates and recompiles offline; the server still performs
// all topology, asset, generation and activation checks on the resulting ZIP.
func (c *Compiler) PrepareBackup(ctx context.Context, data []byte) ([]byte, error) {
	files, manifest, err := readBackupFiles(data, true)
	if err != nil {
		return nil, err
	}
	var state State
	if err := strictJSON(files["state.json"], &state); err != nil {
		return nil, err
	}
	for id, r := range state.Revisions {
		if !hexID(id) || len(id) != 32 || r.ID != id || r.Hash != digest(r.Content) || string(files["content/"+id+".md"]) != r.Content.Markdown {
			return nil, fail("INVALID_BACKUP", "修订与源文不一致")
		}
		compiled, bundle, err := c.CompileBundle(ctx, r.Content)
		if err != nil {
			return nil, err
		}
		r.Compiled = compiled
		r.Bundle = bundle
		r.RendererHash = ""
		state.Revisions[id] = r
	}
	for id, asset := range state.Assets {
		b, ok := files["assets/"+id]
		if !ok || hashBytes(b) != asset.Hash {
			return nil, fail("INVALID_BACKUP", "媒体校验失败")
		}
		if err := c.checkLocalImage(ctx, b); err != nil {
			return nil, err
		}
	}
	files["state.json"], err = json.Marshal(state)
	if err != nil {
		return nil, err
	}
	manifest.Renderer = RendererVersion
	for name, b := range files {
		if name != "manifest.json" {
			manifest.Files[name] = hashBytes(b)
		}
	}
	files["manifest.json"], err = json.Marshal(manifest)
	if err != nil {
		return nil, err
	}
	var out bytes.Buffer
	z := zip.NewWriter(&out)
	for name, b := range files {
		w, err := z.Create(name)
		if err != nil {
			return nil, err
		}
		if _, err = w.Write(b); err != nil {
			return nil, err
		}
	}
	if err := z.Close(); err != nil {
		return nil, err
	}
	if out.Len() > 100<<20 {
		return nil, fail("BACKUP_TOO_LARGE", "编译后的备份超过 100 MiB")
	}
	return out.Bytes(), nil
}
