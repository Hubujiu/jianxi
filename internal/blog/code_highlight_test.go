package blog

import (
	"context"
	"html"
	"strings"
	"testing"
)

func TestCodeHighlightDoesNotEmitDocumentStyles(t *testing.T) {
	compiler := NewCompiler("../../scripts/compile.mjs")
	for _, language := range []string{"go", "html", "", "unknown-language"} {
		t.Run(language, func(t *testing.T) {
			compiled, err := compiler.Compile(context.Background(), Content{
				ID: "code-fragment", Kind: "article", Slug: "code-fragment", Title: "代码片段",
				Markdown: "```" + language + "\n<button onclick=\"alert('example')\">示例</button>\n```\n",
			})
			if err != nil {
				t.Fatal(err)
			}
			lower := strings.ToLower(compiled.HTML)
			for _, forbidden := range []string{"<!doctype", "<html", "<head", "<body", "<style", "<script", "<button onclick="} {
				if strings.Contains(lower, forbidden) {
					t.Fatalf("code leaked document markup or active HTML %q: %s", forbidden, compiled.HTML)
				}
			}
			if !strings.Contains(compiled.HTML, "data-copy") || (!strings.Contains(compiled.HTML, "&lt;") && !strings.Contains(compiled.HTML, "&#x3C;")) || !strings.Contains(compiled.HTML, "<pre") {
				t.Fatalf("code fragment lost copying, escaping or preformatted text: %s", compiled.HTML)
			}
			if !strings.Contains(compiled.HTML, `class="shiki github-light"`) {
				t.Fatal("code was not rendered by Shiki")
			}
		})
	}
}

func TestShikiRendersBatchWithAliasesAndPlainTextFallback(t *testing.T) {
	compiler := NewCompiler("../../scripts/compile.mjs")
	compiled, err := compiler.Compile(context.Background(), Content{
		ID: "shiki-batch", Kind: "article", Slug: "shiki-batch", Title: "Shiki",
		Markdown: "```js\nconst greeting = \"中文\";\n```\n\n```python\ndef hello():\n    return 42\n```\n\n```unknown-language\n<script>alert(1)</script>\n$not_math$\n```\n\n```\n纯文本\n```\n",
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(compiled.HTML, `class="shiki github-light"`) != 4 || !strings.Contains(compiled.HTML, `<span style="color:`) {
		t.Fatal("missing statically highlighted tokens or batch code blocks")
	}
	if compiled.Math || strings.Contains(compiled.HTML, "<script") || !strings.Contains(html.UnescapeString(compiled.HTML), "<script>alert(1)</script>") || !strings.Contains(compiled.HTML, "$not_math$") {
		t.Fatal("unknown-language code was executed or recursively parsed")
	}
	if !strings.Contains(compiled.Text, `const greeting = "中文";`) || !strings.Contains(compiled.Text, "return 42") {
		t.Fatal("highlighting lost searchable source text")
	}
}

func TestStartupRecompilesPublishedCodeAfterRendererUpgrade(t *testing.T) {
	root := t.TempDir()
	app, err := NewApp(root, "../../scripts/compile.mjs")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { app.Store.Close() })
	revision := draft(t, app, "", "代码文章", "```go\npackage main\n```\n")
	publish(t, app, revision.ID)
	unpublished := draft(t, app, revision.Content.ID, "未发布修改", "保持这份草稿")
	snapshot := clone(*app.Store.active.Load())
	legacy := snapshot.Posts[revision.Content.ID]
	legacy.RendererHash = "legacy-standalone-renderer"
	legacy.Compiled.HTML = "<style>body { background: white; }</style><pre>package main</pre>"
	snapshot.Posts[revision.Content.ID] = legacy
	snapshot.ID = uid()
	hash, err := app.build(snapshot, "releases")
	if err != nil {
		t.Fatal(err)
	}
	state := app.Store.Read()
	state.Active, state.ActiveHash = snapshot.ID, hash
	state.Revisions[revision.ID] = legacy
	if err := app.Store.save(state, &snapshot); err != nil {
		t.Fatal(err)
	}
	if err := app.Store.Close(); err != nil {
		t.Fatal(err)
	}
	app, err = NewApp(root, "../../scripts/compile.mjs")
	if err != nil {
		t.Fatal(err)
	}
	current := app.Store.active.Load().Posts[revision.Content.ID]
	if strings.Contains(current.Compiled.HTML, "<style") || current.RendererHash != app.BuildHash {
		t.Fatal("startup retained legacy code styles")
	}
	if current.ID != revision.ID || current.Hash != revision.Hash || current.PublishedAt != legacy.PublishedAt {
		t.Fatal("renderer upgrade changed article identity, source or publication time")
	}
	state = app.Store.Read()
	if state.Articles[revision.Content.ID].Draft != unpublished.ID || state.Revisions[revision.ID].Compiled.HTML != current.Compiled.HTML {
		t.Fatal("renderer upgrade changed the draft or failed to update the published cache")
	}
}
