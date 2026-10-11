package blog

import (
	"context"
	"strings"
	"testing"
)

func TestOutlineIncludesNestedHeadings(t *testing.T) {
	a := testApp(t)
	compiled, err := a.Compiler.Compile(context.Background(), Content{Kind: "article", ID: "outline-test", Slug: "outline-test", Title: "目录层级", Markdown: "# 页面标题\n\n## 第一章\n\n### 子节\n\n#### 细节\n\n##### 补充\n\n###### 注记\n\n## 第一章"})
	if err != nil {
		t.Fatal(err)
	}
	if len(compiled.TOC) != 6 {
		t.Fatalf("missing nested headings: %+v", compiled.TOC)
	}
	seen := map[string]bool{}
	for i, heading := range compiled.TOC {
		want := i + 2
		if i == 5 {
			want = 2
		}
		if heading.Level != want || seen[heading.ID] || !strings.Contains(compiled.HTML, `id="`+heading.ID+`"`) {
			t.Fatalf("invalid outline entry: %+v", heading)
		}
		seen[heading.ID] = true
	}
}
