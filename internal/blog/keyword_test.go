package blog

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func keywordContent(markdown string) Content {
	return Content{ID: "keyword-test", Kind: "article", Slug: "keyword-test", Title: "关键词", Markdown: markdown}
}

func TestKeywordCompilationAndBundle(t *testing.T) {
	c := keywordContent("## 理解 [[幂等性|多次执行，结果保持一致。]]\n\n**[[事务|一组操作作为整体执行。]]**，以及 [[幂等性|另一个解释]]。")
	compiler := NewCompiler("")
	compiled, bundle, err := compiler.CompileBundle(context.Background(), c)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(compiled.HTML, `class="keyword-trigger"`) != 3 || !strings.Contains(compiled.HTML, `data-keyword-description="多次执行，结果保持一致。"`) {
		t.Fatalf("missing keyword controls: %s", compiled.HTML)
	}
	if len(compiled.TOC) != 1 || compiled.TOC[0].Text != "理解 幂等性" || !strings.Contains(compiled.Text, "幂等性 多次执行，结果保持一致。") || strings.Contains(compiled.Text, "[[") {
		t.Fatalf("keyword syntax leaked into outline/search: %+v", compiled)
	}
	if len(bundle.Steps) != 0 {
		t.Fatal("keywords should compile entirely in Go")
	}
	encoded, err := json.Marshal(bundle)
	if err != nil {
		t.Fatal(err)
	}
	var received RenderBundle
	if err = json.Unmarshal(encoded, &received); err != nil {
		t.Fatal(err)
	}
	verified, err := compiler.VerifyBundle(context.Background(), c, &received)
	if err != nil || digest(verified) != digest(compiled) {
		t.Fatalf("server verification differs: %v", err)
	}
}

func TestKeywordTextIsEscapedAndAtomic(t *testing.T) {
	compiled, err := NewCompiler("").Compile(context.Background(), keywordContent(`[[<关键字>|<img src=x onerror="alert(1)"> & **文本** $x$ \| \] \\]]`))
	if err != nil {
		t.Fatal(err)
	}
	for _, unsafe := range []string{"<img", "<关键字>", "<strong>", "katex", `description="<`} {
		if strings.Contains(compiled.HTML, unsafe) {
			t.Fatalf("keyword text became markup: %s", compiled.HTML)
		}
	}
	if !strings.Contains(compiled.HTML, "&lt;img") || !strings.Contains(compiled.Text, `**文本** $x$ | ] \`) {
		t.Fatalf("keyword escaping changed content: %+v", compiled)
	}
}

func TestKeywordDiagnostics(t *testing.T) {
	for _, markdown := range []string{
		"[[|解释]]", "[[词| ]]", "[[词|解释", "[[词|跨行\n解释]]",
		"[[" + strings.Repeat("字", 81) + "|解释]]", "[[词|" + strings.Repeat("字", 2001) + "]]",
		"[[[词|解释]]](https://example.com)", "![[[词|解释]]](/media/example.png)",
	} {
		t.Run(markdown[:min(len(markdown), 40)], func(t *testing.T) {
			_, err := NewCompiler("").Compile(context.Background(), keywordContent("第一行\n\n"+markdown))
			if err == nil || !strings.Contains(err.Error(), "第 3 行") {
				t.Fatalf("expected line diagnostic, got %v", err)
			}
		})
	}
}

func TestKeywordDoesNotChangeCodeLinksOrEscapedSyntax(t *testing.T) {
	compiler := NewCompiler("../../scripts/compile.mjs")
	markdown := "`[[词|解释]]`\n\n```text\n[[词|解释]]\n```\n\n[普通链接](https://example.com)\n\n\\[\\[词|解释]]\n\n[[普通文字]]"
	compiled, err := compiler.Compile(context.Background(), keywordContent(markdown))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(compiled.HTML, `class="keyword-trigger"`) || !strings.Contains(compiled.HTML, `<a href="https://example.com">普通链接</a>`) || !strings.Contains(compiled.HTML, "[[普通文字]]") {
		t.Fatalf("keyword parser changed other syntax: %s", compiled.HTML)
	}
}

func TestKeywordInTablesListsAndFootnotes(t *testing.T) {
	markdown := "| 术语 |\n| --- |\n| [[事务\\|全部成功或失败]] |\n\n- [[列表|列表里的说明]]\n\n脚注[^1]\n\n[^1]: [[脚注|脚注里的说明]]"
	compiled, err := NewCompiler("").Compile(context.Background(), keywordContent(markdown))
	if err != nil || strings.Count(compiled.HTML, `class="keyword-trigger"`) != 3 {
		t.Fatalf("keywords missing in table/list/footnote: %v %s", err, compiled.HTML)
	}
}

// Save real compiler output for optional browser QA without publishing data.
func TestKeywordBrowserFixture(t *testing.T) {
	path := os.Getenv("JIANXI_KEYWORD_FIXTURE")
	if path == "" {
		t.Skip("browser fixture output not requested")
	}
	compiled, err := NewCompiler("").Compile(context.Background(), keywordContent("关键词：[[幂等性|无论执行一次还是多次，产生的效果都相同。]]，[[事务|一组操作要么全部成功，要么全部失败。]]。\n\n[[长说明|"+strings.Repeat("这是一段用于验证手机浮窗滚动的说明。", 70)+"]]"))
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(path, []byte(compiled.HTML), 0600); err != nil {
		t.Fatal(err)
	}
}
