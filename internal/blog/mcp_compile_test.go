package blog

import (
	"context"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMCPCompilesForNodeFreeServer(t *testing.T) {
	ctx := context.Background()
	a, err := NewApp(t.TempDir(), "")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { a.Store.Close() })
	token := strings.Repeat("m", 64)
	if err = a.Auth.write(token); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(a.Handler())
	t.Cleanup(server.Close)
	b := bridge{base: server.URL, token: token, client: server.Client(), compiler: NewCompiler("../../scripts/compile.mjs")}
	copy, err := a.WorkingCopy(CopyInput{})
	if err != nil {
		t.Fatal(err)
	}
	content := copy.Content
	content.Slug = "local-compiler"
	content.Title = "本地编译"
	content.Markdown = "## 中文标题\n\n行内 $E=mc^2$。\n\n$$\n\\frac{1}{2}+\\sqrt{x}\n$$\n\n" + "```go\npackage main\n```\n\n```mermaid\nflowchart LR\n A[本地] --> B[服务器]\n```"
	result, err := b.upload(ctx, UploadInput{CopyID: copy.ID, Content: content, IdempotencyKey: uid()})
	if err != nil {
		t.Fatal(err)
	}
	up := result.(Upload)
	if up.Status != "parsed" || up.Bundle == nil || !up.Compiled.Math || len(up.Compiled.Diagrams) != 1 {
		t.Fatalf("missing local artifacts: %+v", up)
	}
	t.Setenv("PATH", "")
	if _, err = a.Compiler.run(ctx, map[string]any{"kind": "math", "source": "x"}); err == nil {
		t.Fatal("server ran local compiler")
	}
	parsed, err := b.parseUpload(ctx, IDInput{ID: up.ID})
	if err != nil || parsed.(Upload).Hash != up.Hash {
		t.Fatalf("parse replay: %v", err)
	}
	revision, err := a.Commit(CommitInput{UploadID: up.ID, SourceHash: up.Hash, ExpectedDraft: copy.Draft, Epoch: copy.Epoch, IdempotencyKey: uid()})
	if err != nil {
		t.Fatal(err)
	}
	publish(t, a, revision.ID)
	backup, err := a.Export()
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(a.Store.root, "backups", backup))
	if err != nil {
		t.Fatal(err)
	}
	candidate, err := a.InspectBackup(ctx, data)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = a.Restore(candidate.ID, candidate.Hash); err != nil {
		t.Fatal(err)
	}
	active := a.Store.active.Load().Posts[content.ID]
	if active.Compiled.HTML != up.Compiled.HTML || active.Bundle == nil {
		t.Fatal("restore lost compiled content")
	}
}
func TestRenderBundleRejectsTampering(t *testing.T) {
	ctx := context.Background()
	compiler := NewCompiler("../../scripts/compile.mjs")
	content := Content{ID: uid(), Kind: "article", Slug: "bundle", Title: "编译包", Markdown: "```go\npackage main\n```"}
	compiled, bundle, err := compiler.CompileBundle(ctx, content)
	if err != nil {
		t.Fatal(err)
	}
	server := NewCompiler("")
	if got, err := server.VerifyBundle(ctx, content, bundle); err != nil || got.HTML != compiled.HTML {
		t.Fatalf("round trip: %v", err)
	}
	altered := content
	altered.Markdown += "\n修改正文"
	if _, err := server.VerifyBundle(ctx, altered, bundle); err == nil {
		t.Fatal("accepted stale source")
	}
	for _, fragment := range []string{
		"<pre><code>替换源代码</code></pre>",
		"<pre><code><script>alert(1)</script></code></pre>",
		"<pre style=\"position:fixed;background-image:url(https://evil.test)\"><code>package main\n</code></pre>",
		"<pre onclick=\"alert(1)\"><code>package main\n</code></pre>",
	} {
		bad := clone(*bundle)
		bad.Steps[0].Result = map[string]any{"html": []any{fragment}}
		if _, err := server.VerifyBundle(ctx, content, &bad); err == nil {
			t.Fatalf("accepted unsafe fragment: %s", fragment)
		}
	}
	for _, svg := range []string{"<svg xmlns=\"http://www.w3.org/2000/svg\"><script>alert(1)</script></svg>", "<svg xmlns=\"http://www.w3.org/2000/svg\"><use href=\"https://evil.test/a.svg\"/></svg>", "<svg xmlns=\"http://www.w3.org/2000/svg\"><style>@import 'https://evil.test/a.css';</style></svg>", "<!DOCTYPE svg [<!ENTITY x SYSTEM \"file:///etc/passwd\">]><svg>&x;</svg>"} {
		if validateFragment(svg, "svg") == nil {
			t.Fatalf("accepted unsafe SVG: %s", svg)
		}
	}
}

func TestMCPBundleRenderingShowcase(t *testing.T) {
	data, err := os.ReadFile("../../examples/rendering-showcase.md")
	if err != nil {
		t.Fatal(err)
	}
	compiler := NewCompiler("../../scripts/compile.mjs")
	content := Content{ID: uid(), Kind: "article", Slug: "showcase", Title: "渲染展示", Markdown: string(data)}
	compiled, bundle, err := compiler.CompileBundle(context.Background(), content)
	if err != nil {
		t.Fatal(err)
	}
	if !compiled.Math || len(compiled.Diagrams) < 8 || len(bundle.Steps) < 10 {
		t.Fatal("missing rich showcase artifacts")
	}
}

func TestMCPBundleMathCommands(t *testing.T) {
	compiler := NewCompiler("../../scripts/compile.mjs")
	for _, source := range []string{`\phantom{x}+\overbrace{a+b}^{n}`, `\tag{1}x=2`, `\Huge \text{中文}`, `\fcolorbox{blue}{yellow}{x}`, `\frac{a}{b}\quad \rule{1em}{0.5em}`} {
		t.Run(source, func(t *testing.T) {
			content := Content{ID: uid(), Kind: "article", Slug: "math", Title: "公式", Markdown: "```math\n" + source + "\n```"}
			if _, _, err := compiler.CompileBundle(context.Background(), content); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestLegacyReleaseUpgradesThroughMCP(t *testing.T) {
	root := t.TempDir()
	local, err := NewApp(root, "../../scripts/compile.mjs")
	if err != nil {
		t.Fatal(err)
	}
	old := draft(t, local, "", "旧文章", "```go\npackage main\n```")
	publish(t, local, old.ID)
	pending := draft(t, local, old.Content.ID, "待办草稿", "保留此草稿")
	snapshot := clone(*local.Store.active.Load())
	legacy := snapshot.Posts[old.Content.ID]
	legacy.Bundle = nil
	legacy.RendererHash = "legacy-server-compiler"
	snapshot.Posts[old.Content.ID] = legacy
	snapshot.ID = uid()
	hash, err := local.build(snapshot, "releases")
	if err != nil {
		t.Fatal(err)
	}
	state := local.Store.Read()
	state.Revisions[old.ID] = legacy
	state.Active = snapshot.ID
	state.ActiveHash = hash
	if err = local.Store.save(state, &snapshot); err != nil {
		t.Fatal(err)
	}
	if err = local.Store.Close(); err != nil {
		t.Fatal(err)
	}
	server, err := NewApp(root, "")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { server.Store.Close() })
	if server.Store.Read().Active != snapshot.ID {
		t.Fatal("startup replaced legacy release")
	}
	token := strings.Repeat("u", 64)
	if err = server.Auth.write(token); err != nil {
		t.Fatal(err)
	}
	httpServer := httptest.NewServer(server.Handler())
	t.Cleanup(httpServer.Close)
	bridge := bridge{base: httpServer.URL, token: token, client: httpServer.Client(), compiler: NewCompiler("../../scripts/compile.mjs")}
	result, err := bridge.recompile(context.Background(), GetInput{ContentID: old.Content.ID, Revision: old.ID})
	if err != nil {
		t.Fatal(err)
	}
	rebuilt := result.(Revision)
	if rebuilt.ID != old.ID || rebuilt.Hash != legacy.Hash || rebuilt.PublishedAt != legacy.PublishedAt || rebuilt.Bundle == nil {
		t.Fatal("recompile changed source identity or dates")
	}
	if server.Store.Read().Active != snapshot.ID || server.Store.Read().Articles[old.Content.ID].Draft != pending.ID {
		t.Fatal("recompile changed publication or draft")
	}
	t.Setenv("PATH", "")
	publish(t, server, old.ID)
	if server.Store.active.Load().Posts[old.Content.ID].PublishedAt != legacy.PublishedAt {
		t.Fatal("renderer publication changed first publish time")
	}
	if server.Store.Read().Articles[old.Content.ID].Draft != pending.ID {
		t.Fatal("renderer publication replaced unrelated draft")
	}
}
