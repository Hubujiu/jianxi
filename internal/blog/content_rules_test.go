package blog

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestContentRulesExposeSyntaxThroughAPI(t *testing.T) {
	a := testApp(t)
	r := httptest.NewRequest(http.MethodPost, "/_control/rules", strings.NewReader("{}"))
	result, err := a.dispatch(r, "rules")
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	var rules struct {
		Syntax []contentSyntax `json:"syntax"`
		Limits map[string]int  `json:"limits"`
	}
	if err = json.Unmarshal(data, &rules); err != nil {
		t.Fatal(err)
	}
	names := map[string]bool{}
	for _, syntax := range rules.Syntax {
		if names[syntax.Name] || syntax.Description == "" || syntax.Example == "" {
			t.Fatalf("duplicate or incomplete syntax: %+v", syntax)
		}
		names[syntax.Name] = true
	}
	for _, name := range []string{"headings", "paragraphs", "emphasis", "inline_code", "code_blocks", "lists", "task_lists", "blockquotes", "thematic_breaks", "tables", "links", "images", "footnotes", "keywords", "escaping", "inline_math", "block_math", "math_fences", "mermaid_flowchart", "mermaid_sequence", "mermaid_class", "mermaid_state", "mermaid_er", "mermaid_gantt", "mermaid_pie", "mermaid_timeline", "raw_html"} {
		if !names[name] {
			t.Fatalf("MCP rules missing %s", name)
		}
	}
	if rules.Limits["keywordCharacters"] != 80 || rules.Limits["keywordDescriptionCharacters"] != 2000 {
		t.Fatal("keyword limits missing")
	}
}

func TestContentRulesExamplesCompile(t *testing.T) {
	var markdown strings.Builder
	for _, syntax := range supportedContentSyntax() {
		markdown.WriteString(syntax.Example + "\n\n")
	}
	compiled, _, err := NewCompiler("../../scripts/compile.mjs").CompileBundle(context.Background(), keywordContent(markdown.String()))
	if err != nil {
		t.Fatal(err)
	}
	if !compiled.Math || len(compiled.Diagrams) != 8 || !strings.Contains(compiled.HTML, `class="keyword-trigger"`) {
		t.Fatal("MCP examples lost math, diagrams or keywords")
	}
}
