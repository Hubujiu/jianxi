package blog

import (
	"context"
	"encoding/base64"
	"io"
	"os"
)

type RestoreFileInput struct {
	Path string `json:"path"`
}
type RestoreInput struct {
	ID   string `json:"id"`
	Hash string `json:"artifactHash"`
}

func (b bridge) upload(ctx context.Context, in UploadInput) (any, error) {
	_, bundle, err := b.compiler.CompileBundle(ctx, in.Content)
	if err != nil {
		return nil, err
	}
	var up Upload
	err = b.call(ctx, "upload-compiled", CompiledUploadInput{UploadInput: in, Bundle: bundle}, &up)
	return up, err
}
func (b bridge) parseUpload(ctx context.Context, in IDInput) (any, error) {
	var up Upload
	if err := b.call(ctx, "upload-get", in, &up); err != nil {
		return nil, err
	}
	if up.Status == "parsed" && up.Bundle != nil && up.Bundle.Renderer == RendererVersion {
		err := b.call(ctx, "parse", in, &up)
		return up, err
	}
	_, bundle, err := b.compiler.CompileBundle(ctx, up.Content)
	if err != nil {
		return nil, err
	}
	err = b.call(ctx, "parse-compiled", ParseBundleInput{ID: in.ID, Bundle: bundle}, &up)
	return up, err
}
func (b bridge) asset(ctx context.Context, in AssetInput) (any, error) {
	data, err := base64.StdEncoding.DecodeString(in.Data)
	if err != nil {
		return nil, fail("INVALID_IMAGE", "图片 base64 无效")
	}
	if err := b.compiler.checkLocalImage(ctx, data); err != nil {
		return nil, err
	}
	var out Asset
	err = b.call(ctx, "asset", in, &out)
	return out, err
}
func (b bridge) recompile(ctx context.Context, in GetInput) (any, error) {
	var got struct {
		Revision Revision `json:"revision"`
		Epoch    string   `json:"epoch"`
	}
	if err := b.call(ctx, "get", in, &got); err != nil {
		return nil, err
	}
	_, bundle, err := b.compiler.CompileBundle(ctx, got.Revision.Content)
	if err != nil {
		return nil, err
	}
	var out Revision
	err = b.call(ctx, "recompile", RecompileInput{ContentID: in.ContentID, Revision: got.Revision.ID, SourceHash: got.Revision.Hash, Epoch: got.Epoch, Bundle: bundle}, &out)
	return out, err
}
func (b bridge) rollback(ctx context.Context, in RollbackInput) (any, error) {
	var got struct {
		Revision Revision `json:"revision"`
	}
	if err := b.call(ctx, "get", GetInput{ContentID: in.ContentID, Revision: in.Revision}, &got); err != nil {
		return nil, err
	}
	if got.Revision.Bundle == nil || got.Revision.Bundle.Renderer != RendererVersion {
		if _, err := b.recompile(ctx, GetInput{ContentID: in.ContentID, Revision: in.Revision}); err != nil {
			return nil, err
		}
	}
	var out Revision
	err := b.call(ctx, "rollback", in, &out)
	return out, err
}
func (b bridge) prepareRestore(ctx context.Context, in RestoreFileInput) (any, error) {
	file, err := os.Open(in.Path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, (100<<20)+1))
	if err != nil {
		return nil, err
	}
	prepared, err := b.compiler.PrepareBackup(ctx, data)
	if err != nil {
		return nil, err
	}
	var out any
	err = b.call(ctx, "backup-import", map[string]string{"base64": base64.StdEncoding.EncodeToString(prepared)}, &out)
	return out, err
}
