package blog

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html"
	"net/url"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

type Compiler struct {
	Script string
	gate   chan struct{}
	remote func(context.Context, any) (map[string]any, error)
	record *[]RenderStep
}

func NewCompiler(script string) *Compiler {
	p := ""
	if script != "" {
		p, _ = filepath.Abs(script)
	}
	return &Compiler{Script: p, gate: make(chan struct{}, 1)}
}
func (c *Compiler) run(ctx context.Context, input any) (map[string]any, error) {
	if c.remote != nil {
		return c.remote(ctx, input)
	}
	if c.Script == "" {
		return nil, fail("LOCAL_COMPILATION_REQUIRED", "请通过本地 MCP 编译并上传；服务器不运行 Node 或 Chromium")
	}
	select {
	case c.gate <- struct{}{}:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	defer func() { <-c.gate }()
	ctx, cancel := context.WithTimeout(ctx, 35*time.Second)
	defer cancel()
	b, _ := json.Marshal(input)
	cmd := exec.CommandContext(ctx, "node", "--max-old-space-size=256", c.Script)
	cmd.Stdin = bytes.NewReader(b)
	var stdout, stderr limitedBuffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	var result map[string]any
	if json.Unmarshal(stdout.Bytes(), &result) != nil {
		return nil, fail("BUILD_FAILED", "编译器未返回有效结果；请检查 Node 和 npm ci 安装。"+stderr.String())
	}
	if msg, ok := result["error"].(string); ok {
		return nil, fail("BUILD_FAILED", msg)
	}
	if err != nil {
		return nil, fail("BUILD_FAILED", err.Error())
	}
	if c.record != nil {
		*c.record = append(*c.record, RenderStep{InputHash: digest(input), Result: result})
	}
	return result, nil
}

type limitedBuffer struct{ bytes.Buffer }

func (b *limitedBuffer) Write(p []byte) (int, error) {
	n := len(p)
	if b.Len() < 4<<20 {
		remaining := (4 << 20) - b.Len()
		if len(p) > remaining {
			p = p[:remaining]
		}
		_, _ = b.Buffer.Write(p)
	}
	return n, nil
}

var mathInlineKind = ast.NewNodeKind("JianxiMath")
var mathBlockKind = ast.NewNodeKind("JianxiMathBlock")

type mathInline struct {
	ast.BaseInline
	source string
	offset int
}

func (n *mathInline) Kind() ast.NodeKind   { return mathInlineKind }
func (n *mathInline) Dump(s []byte, l int) { ast.DumpHelper(n, s, l, nil, nil) }

type mathBlock struct {
	ast.BaseBlock
	closed bool
	offset int
}

func (n *mathBlock) Kind() ast.NodeKind   { return mathBlockKind }
func (n *mathBlock) Dump(s []byte, l int) { ast.DumpHelper(n, s, l, nil, nil) }

type inlineMathParser struct{}

func (inlineMathParser) Trigger() []byte { return []byte{'$'} }
func (inlineMathParser) Parse(parent ast.Node, r text.Reader, pc parser.Context) ast.Node {
	line, seg := r.PeekLine()
	if len(line) < 3 || line[1] == '$' || line[1] == ' ' || line[1] == '\n' {
		return nil
	}
	for i := 1; i < len(line); i++ {
		if line[i] == '$' && line[i-1] != '\\' && line[i-1] != ' ' {
			r.Advance(i + 1)
			return &mathInline{source: string(line[1:i]), offset: seg.Start}
		}
	}
	return nil
}

type blockMathParser struct{}

func (blockMathParser) Trigger() []byte { return []byte{'$'} }
func (blockMathParser) Open(parent ast.Node, r text.Reader, pc parser.Context) (ast.Node, parser.State) {
	line, seg := r.PeekLine()
	if strings.TrimSpace(string(line)) != "$$" {
		return nil, parser.NoChildren
	}
	r.Advance(len(line) - 1)
	return &mathBlock{offset: seg.Start}, parser.NoChildren
}
func (blockMathParser) Continue(n ast.Node, r text.Reader, pc parser.Context) parser.State {
	line, seg := r.PeekLine()
	if strings.TrimSpace(string(line)) == "$$" {
		n.(*mathBlock).closed = true
		r.Advance(len(line) - 1)
		return parser.Close
	}
	n.Lines().Append(seg)
	r.Advance(len(line) - 1)
	return parser.Continue | parser.NoChildren
}
func (blockMathParser) Close(ast.Node, text.Reader, parser.Context) {}
func (blockMathParser) CanInterruptParagraph() bool                 { return true }
func (blockMathParser) CanAcceptIndentedLine() bool                 { return false }

type compiledRenderer struct{ nodes map[ast.Node]string }

func (r *compiledRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	for _, kind := range []ast.NodeKind{mathInlineKind, mathBlockKind, keywordKind, ast.KindFencedCodeBlock} {
		reg.Register(kind, r.render)
	}
}
func (r *compiledRenderer) render(w util.BufWriter, source []byte, n ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		_, _ = w.WriteString(r.nodes[n])
	}
	return ast.WalkSkipChildren, nil
}

func validateContent(c Content) error {
	if c.Kind != "article" && c.Kind != "about" {
		return fail("INVALID_CONTENT", "kind 必须为 article 或 about")
	}
	if c.ID == "" || len(c.ID) > 64 {
		return fail("INVALID_CONTENT", "缺少内容 ID")
	}
	if !slugPattern.MatchString(c.Slug) || len(c.Slug) > 100 {
		return fail("INVALID_SLUG", "文章元数据必须明确提供 slug，使用小写字母、数字和连字符，最长 100 字符；不自动生成路由")
	}
	if strings.TrimSpace(c.Title) == "" || utf8.RuneCountInString(c.Title) > 160 {
		return fail("INVALID_TITLE", "标题必填且不超过 160 字")
	}
	if len(c.Markdown) > 256<<10 || !utf8.ValidString(c.Markdown) {
		return fail("CONTENT_TOO_LARGE", "正文必须是有效 UTF-8，且不超过 256 KiB")
	}
	if len(c.TagIDs) > 20 || len(c.Sources) > 30 || len(c.Videos) > 5 || len(c.Summary) > 2000 {
		return fail("INVALID_CONTENT", "元数据超出限制")
	}
	if c.Category != nil {
		if err := validateTaxonomyMetadata(*c.Category); err != nil {
			return err
		}
	}
	if len(c.Tags) > 20 {
		return fail("INVALID_CONTENT", "标签最多 20 个")
	}
	for _, tag := range c.Tags {
		if err := validateTaxonomyMetadata(tag); err != nil {
			return err
		}
	}
	if c.Kind == "about" && c.ID != "about" {
		return fail("INVALID_CONTENT", "关于我使用固定 ID about")
	}
	if c.AvatarURL != "" && !safeURL(c.AvatarURL, true) {
		return fail("INVALID_URL", "头像只允许 HTTPS 或本站 /media/ 资产")
	}
	if len(c.SocialLinks) > 12 {
		return fail("INVALID_CONTENT", "社交链接最多 12 个")
	}
	for _, link := range c.SocialLinks {
		if strings.TrimSpace(link.Name) == "" || utf8.RuneCountInString(link.Name) > 40 || len(link.URL) > 2000 || !safeURL(link.URL, false) {
			return fail("INVALID_CONTENT", "社交链接需填写名称（最多 40 字）与有效 HTTPS 或站内地址")
		}
	}
	for _, link := range append(append([]string{}, c.Sources...), c.Videos...) {
		if !safeURL(link, false) {
			return fail("INVALID_URL", "来源和视频必须使用 HTTPS 地址")
		}
	}
	return nil
}
func safeURL(raw string, image bool) bool {
	if strings.ContainsAny(raw, "\\\r\n\t") {
		return false
	}
	u, err := url.Parse(raw)
	if err != nil || u.User != nil {
		return false
	}
	if u.Scheme == "https" && u.Host != "" {
		return true
	}
	if !image && (strings.HasPrefix(raw, "#") || (strings.HasPrefix(raw, "/") && !strings.HasPrefix(raw, "//"))) {
		return true
	}
	return image && strings.HasPrefix(raw, "/media/") && !strings.Contains(raw, "..")
}
func (c *Compiler) Compile(ctx context.Context, content Content) (Compiled, error) {
	out := Compiled{Diagrams: map[string]string{}}
	if err := validateContent(content); err != nil {
		return out, err
	}
	if strings.HasPrefix(content.AvatarURL, "/media/") {
		out.Assets = append(out.Assets, strings.TrimPrefix(content.AvatarURL, "/media/"))
	}
	source := []byte(content.Markdown)
	special := &compiledRenderer{nodes: map[ast.Node]string{}}
	md := goldmark.New(goldmark.WithExtensions(extension.GFM, extension.Footnote), goldmark.WithParserOptions(parser.WithInlineParsers(util.Prioritized(inlineMathParser{}, 150), util.Prioritized(keywordParser{}, 100)), parser.WithBlockParsers(util.Prioritized(blockMathParser{}, 500))), goldmark.WithRendererOptions(renderer.WithNodeRenderers(util.Prioritized(special, 50))))
	doc := md.Parser().Parse(text.NewReader(source))
	headings := map[string]int{}
	var codeNodes []ast.Node
	var codeBlocks []map[string]string
	nodes, blocks := 0, 0
	var plain strings.Builder
	diagnostic := func(n ast.Node, msg string) error {
		offset := 0
		switch v := n.(type) {
		case *keywordInline:
			offset = v.offset
		case *mathInline:
			offset = v.offset
		case *mathBlock:
			offset = v.offset
		case *ast.RawHTML:
			if v.Segments.Len() > 0 {
				offset = v.Segments.At(0).Start
			}
		default:
			if n.Type() == ast.TypeBlock && n.Lines().Len() > 0 {
				offset = n.Lines().At(0).Start
			}
		}
		return fail("CONTENT_DIAGNOSTIC", fmt.Sprintf("第 %d 行：%s", bytes.Count(source[:min(offset, len(source))], []byte{'\n'})+1, msg))
	}
	err := ast.Walk(doc, func(n ast.Node, enter bool) (ast.WalkStatus, error) {
		if !enter {
			return ast.WalkContinue, nil
		}
		nodes++
		if nodes > 10000 {
			return ast.WalkStop, diagnostic(n, "内容节点超过限制")
		}
		switch v := n.(type) {
		case *keywordInline:
			if v.diagnostic != "" {
				return ast.WalkStop, diagnostic(n, v.diagnostic)
			}
			if v.label == "" || v.description == "" || utf8.RuneCountInString(v.label) > 80 || utf8.RuneCountInString(v.description) > 2000 {
				return ast.WalkStop, diagnostic(n, "关键词和描述不能为空；关键词最多 80 字，描述最多 2000 字")
			}
			for parent := n.Parent(); parent != nil; parent = parent.Parent() {
				if parent.Kind() == ast.KindLink || parent.Kind() == ast.KindImage {
					return ast.WalkStop, diagnostic(n, "关键词描述不能嵌套在链接或图片中")
				}
			}
			label, description := html.EscapeString(v.label), html.EscapeString(v.description)
			special.nodes[n] = `<button type="button" class="keyword-trigger" data-keyword-description="` + description + `" title="` + description + `" aria-haspopup="dialog" aria-expanded="false">` + label + `</button>`
			plain.WriteString(v.label + " " + v.description + " ")
		case *ast.HTMLBlock, *ast.RawHTML:
			return ast.WalkStop, diagnostic(n, "RAW_HTML_NOT_ALLOWED：原始 HTML 请放入 html 代码块")
		case *ast.Link:
			if !safeURL(string(v.Destination), false) {
				return ast.WalkStop, diagnostic(n, "链接只允许 HTTPS、站内地址或锚点")
			}
		case *ast.AutoLink:
			if !safeURL(string(v.URL(source)), false) {
				return ast.WalkStop, diagnostic(n, "链接只允许 HTTPS")
			}
		case *ast.Image:
			dest := string(v.Destination)
			if !safeURL(dest, true) {
				return ast.WalkStop, diagnostic(n, "图片只允许 HTTPS 或本站 /media/ 资产")
			}
			if strings.HasPrefix(dest, "/media/") {
				out.Assets = append(out.Assets, strings.TrimPrefix(dest, "/media/"))
			}
		case *ast.Heading:
			label := string(v.Text(source))
			base := "h-" + hashBytes([]byte(label))[:10]
			headings[base]++
			id := fmt.Sprintf("%s-%d", base, headings[base])
			v.SetAttributeString("id", []byte(id))
			if v.Level >= 2 && v.Level <= 6 {
				out.TOC = append(out.TOC, Heading{ID: id, Text: label, Level: v.Level})
			}
		case *ast.Text:
			plain.Write(v.Segment.Value(source))
			plain.WriteByte(' ')
		}
		var raw, kind string
		display := false
		switch v := n.(type) {
		case *mathInline:
			raw = v.source
			kind = "math"
		case *mathBlock:
			if !v.closed {
				return ast.WalkStop, diagnostic(n, "块公式缺少结束 $$")
			}
			raw = string(v.Lines().Value(source))
			kind = "math"
			display = true
		case *ast.FencedCodeBlock:
			raw = string(v.Lines().Value(source))
			lang := string(v.Language(source))
			kind = lang
			display = true
			if lang != "math" && lang != "mermaid" {
				codeNodes = append(codeNodes, n)
				codeBlocks = append(codeBlocks, map[string]string{"source": raw, "language": lang})
				plain.WriteString(raw)
				return ast.WalkSkipChildren, nil
			}
		}
		if kind == "math" || kind == "mermaid" {
			blocks++
			if blocks > 40 {
				return ast.WalkStop, diagnostic(n, "公式及图表最多 40 个")
			}
			result, err := c.run(ctx, map[string]any{"kind": kind, "source": raw, "display": display})
			if err != nil {
				return ast.WalkStop, diagnostic(n, err.Error())
			}
			if kind == "math" {
				out.Math = true
				special.nodes[n] = result["html"].(string)
			} else {
				svg := result["svg"].(string)
				name := hashBytes([]byte(svg)) + ".svg"
				out.Diagrams[name] = svg
				special.nodes[n] = "<figure class=\"diagram\"><img src=\"/diagrams/" + name + "\" alt=\"文章图表\"><details><summary>查看图表源码</summary><pre><code>" + html.EscapeString(raw) + "</code></pre></details></figure>"
			}
			plain.WriteString(raw)
			return ast.WalkSkipChildren, nil
		}
		return ast.WalkContinue, nil
	})
	if err != nil {
		return out, err
	}
	if len(codeNodes) > 0 {
		// Highlight all fences in one bounded subprocess, sharing a Shiki instance
		// and loading only the grammars needed by this article. Output is static HTML.
		result, err := c.run(ctx, map[string]any{"kind": "code", "blocks": codeBlocks})
		if err != nil {
			return out, diagnostic(codeNodes[0], err.Error())
		}
		fragments, ok := result["html"].([]any)
		if !ok || len(fragments) != len(codeNodes) {
			return out, fail("BUILD_FAILED", "Shiki 返回的代码片段数量不匹配")
		}
		for i, node := range codeNodes {
			fragment, ok := fragments[i].(string)
			if !ok {
				return out, fail("BUILD_FAILED", "Shiki 返回的代码片段无效")
			}
			lang := html.EscapeString(codeBlocks[i]["language"])
			special.nodes[node] = "<div class=\"code-block\"><div class=\"code-label\">" + lang + "<button type=\"button\" data-copy>复制代码</button></div>" + fragment + "</div>"
		}
	}
	var b bytes.Buffer
	if err = md.Renderer().Render(&b, source, doc); err != nil {
		return out, err
	}
	out.HTML = b.String()
	out.Text = plain.String()
	return out, nil
}
