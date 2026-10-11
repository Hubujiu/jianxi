package blog

import "context"

type RecompileInput struct {
	ContentID  string        `json:"contentId"`
	Revision   string        `json:"revision"`
	Epoch      string        `json:"epoch"`
	SourceHash string        `json:"sourceHash"`
	Bundle     *RenderBundle `json:"bundle"`
}

func (a *App) RecompileRevision(ctx context.Context, in RecompileInput) (Revision, error) {
	r, ok := a.Store.Read().Revisions[in.Revision]
	if !ok || r.Content.ID != in.ContentID || r.Hash != in.SourceHash {
		return Revision{}, fail("VERSION_NOT_AVAILABLE", "修订源文不可用")
	}
	compiled, err := a.Compiler.VerifyBundle(ctx, r.Content, in.Bundle)
	if err != nil {
		return Revision{}, err
	}
	err = a.Store.Update(func(s *State) error {
		current, ok := s.Revisions[in.Revision]
		if s.Epoch != in.Epoch || !ok || current.Hash != in.SourceHash {
			return fail("REVISION_CONFLICT", "实例或修订已改变")
		}
		current.Compiled = compiled
		current.Bundle = in.Bundle
		current.RendererHash = a.BuildHash
		s.Revisions[current.ID] = current
		r = current
		return nil
	})
	return r, err
}

type CompiledUploadInput struct {
	UploadInput
	Bundle *RenderBundle `json:"bundle"`
}
type ParseBundleInput struct {
	ID     string        `json:"id"`
	Bundle *RenderBundle `json:"bundle"`
}

func (a *App) UploadCompiled(ctx context.Context, in CompiledUploadInput) (Upload, error) {
	if _, err := a.Compiler.VerifyBundle(ctx, in.Content, in.Bundle); err != nil {
		return Upload{}, err
	}
	up, err := a.Upload(in.UploadInput)
	if err != nil {
		return up, err
	}
	// Retrying an upload never replaces its validated artifact or draft.
	if up.Status == "parsed" {
		return up, nil
	}
	return a.ParseBundle(ctx, up.ID, in.Bundle)
}
func (a *App) ParseBundle(ctx context.Context, id string, bundle *RenderBundle) (Upload, error) {
	up, ok := a.Store.Read().Uploads[id]
	if !ok {
		return Upload{}, fail("UPLOAD_NOT_FOUND", "上传不存在")
	}
	if up.Hash != digest(up.Content) {
		return Upload{}, fail("SOURCE_HASH_MISMATCH", "上传源文 hash 无效")
	}
	compiled, err := a.Compiler.VerifyBundle(ctx, up.Content, bundle)
	if err != nil {
		return up, err
	}
	err = a.Store.Update(func(s *State) error {
		current, ok := s.Uploads[id]
		if !ok {
			return fail("UPLOAD_NOT_FOUND", "上传已失效")
		}
		if current.Hash != up.Hash {
			return fail("SOURCE_HASH_MISMATCH", "上传源文已改变")
		}
		if current.Status == "parsed" && current.RendererHash == a.BuildHash {
			up = current
			return nil
		}
		current.Compiled = compiled
		current.Bundle = bundle
		current.RendererHash = a.BuildHash
		current.Status = "parsed"
		current.Diagnostic = ""
		s.Uploads[id] = current
		up = current
		return nil
	})
	return up, err
}
func (a *App) compileRevision(ctx context.Context, r Revision) (Compiled, *RenderBundle, error) {
	if r.Bundle != nil && r.Bundle.Renderer == RendererVersion {
		compiled, err := a.Compiler.VerifyBundle(ctx, r.Content, r.Bundle)
		return compiled, r.Bundle, err
	}
	return a.Compiler.CompileBundle(ctx, r.Content)
}
