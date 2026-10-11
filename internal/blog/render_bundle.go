package blog

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"io"
	"regexp"
	"strings"
)

// Bundles contain only rich fragments from the fixed local compiler. The server
// derives headings, references, text and surrounding HTML from the Markdown.
type RenderStep struct {
	InputHash string         `json:"inputHash"`
	Result    map[string]any `json:"result"`
}
type RenderBundle struct {
	Version    int          `json:"version"`
	Renderer   string       `json:"renderer"`
	SourceHash string       `json:"sourceHash"`
	Steps      []RenderStep `json:"steps"`
}

func renderSourceHash(c Content) string {
	return digest(struct{ Markdown, Avatar string }{c.Markdown, c.AvatarURL})
}
func (c *Compiler) CompileBundle(ctx context.Context, content Content) (Compiled, *RenderBundle, error) {
	bundle := &RenderBundle{Version: 1, Renderer: RendererVersion, SourceHash: renderSourceHash(content), Steps: []RenderStep{}}
	local := &Compiler{Script: c.Script, gate: c.gate, record: &bundle.Steps}
	compiled, err := local.Compile(ctx, content)
	if err != nil {
		return compiled, nil, err
	}
	verified, err := c.VerifyBundle(ctx, content, bundle)
	if err != nil {
		return Compiled{}, nil, err
	}
	if digest(compiled) != digest(verified) {
		return Compiled{}, nil, fail("COMPILED_MISMATCH", "本地与服务端 Markdown 生成结果不一致")
	}
	return compiled, bundle, nil
}
func (c *Compiler) VerifyBundle(ctx context.Context, content Content, bundle *RenderBundle) (Compiled, error) {
	if bundle == nil || bundle.Version != 1 || bundle.Renderer != RendererVersion || bundle.SourceHash != renderSourceHash(content) {
		return Compiled{}, fail("COMPILER_MISMATCH", "编译包版本或源文不匹配，请使用当前本地 MCP 重新编译")
	}
	encoded, _ := json.Marshal(bundle)
	if len(encoded) > 8<<20 || len(bundle.Steps) > 41 {
		return Compiled{}, fail("COMPILED_TOO_LARGE", "编译包最多 8 MiB / 41 次编译任务")
	}
	index := 0
	verifier := &Compiler{remote: func(ctx context.Context, input any) (map[string]any, error) {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if index >= len(bundle.Steps) {
			return nil, fail("COMPILED_MISMATCH", "编译片段缺失")
		}
		step := bundle.Steps[index]
		index++
		if step.InputHash != digest(input) {
			return nil, fail("COMPILED_MISMATCH", "编译片段与源文或任务顺序不一致")
		}
		if err := validateRenderResult(input, step.Result); err != nil {
			return nil, err
		}
		return step.Result, nil
	}}
	compiled, err := verifier.Compile(ctx, content)
	if err == nil && index != len(bundle.Steps) {
		err = fail("COMPILED_MISMATCH", "编译包含额外片段")
	}
	return compiled, err
}

func validateRenderResult(input any, result map[string]any) error {
	request := input.(map[string]any)
	kind, _ := request["kind"].(string)
	if len(result) != 1 {
		return fail("INVALID_COMPILED", "编译结果包含未知字段")
	}
	switch kind {
	case "code":
		blocks := request["blocks"].([]map[string]string)
		fragments, ok := result["html"].([]any)
		if !ok || len(fragments) != len(blocks) {
			return fail("INVALID_COMPILED", "代码片段数量不一致")
		}
		for i, fragment := range fragments {
			s, ok := fragment.(string)
			if !ok {
				return fail("INVALID_COMPILED", "代码片段类型无效")
			}
			if err := validateFragment(s, "code"); err != nil {
				return err
			}
			if fragmentText(s, "code") != xmlText(blocks[i]["source"]) {
				return fail("INVALID_COMPILED", "高亮代码与源代码不一致")
			}
		}
	case "math":
		s, ok := result["html"].(string)
		if !ok {
			return fail("INVALID_COMPILED", "公式片段类型无效")
		}
		if err := validateFragment(s, "math"); err != nil {
			return err
		}
		if fragmentText(s, "annotation") != xmlText(request["source"].(string)) {
			return fail("INVALID_COMPILED", "公式源文注释不一致")
		}
		return validateFragment(s, "math")
	case "mermaid":
		s, ok := result["svg"].(string)
		if !ok {
			return fail("INVALID_COMPILED", "图表片段类型无效")
		}
		return validateFragment(s, "svg")
	default:
		return fail("INVALID_COMPILED", "未知编译任务")
	}
	return nil
}
func xmlText(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(s, "\r\n", "\n"), "\r", "\n")
}
func fragmentText(s, tag string) string {
	d := xml.NewDecoder(strings.NewReader("<root>" + s + "</root>"))
	d.Entity = map[string]string{"nbsp": "\u00a0"}
	depth, inside := 0, 0
	var out strings.Builder
	for {
		token, err := d.Token()
		if err != nil {
			break
		}
		switch t := token.(type) {
		case xml.StartElement:
			depth++
			if t.Name.Local == tag && inside == 0 {
				inside = depth
			}
		case xml.EndElement:
			if depth == inside {
				inside = 0
			}
			depth--
		case xml.CharData:
			if inside > 0 {
				out.Write(t)
			}
		}
	}
	return out.String()
}

var cssURL = regexp.MustCompile(`(?i)url\(\s*['"]?#[a-z0-9_.:-]+['"]?\s*\)`)
var svgKeyframes = regexp.MustCompile(`(?i)@(?:-webkit-)?keyframes\b`)
var unsafeCSS = regexp.MustCompile(`(?i)url\s*\(|@|expression\s*\(|javascript\s*:|data\s*:|https?\s*:|//|\\|behavior\s*:|-moz-binding\s*:`)
var unsafeAttribute = regexp.MustCompile(`(?i)url\s*\(|javascript\s*:|data\s*:|https?\s*:|//|\\`)
var cssValue = regexp.MustCompile(`^[a-zA-Z0-9 .,%()+#\-]*$`)
var fragmentClass = regexp.MustCompile(`^[a-zA-Z0-9 _-]*$`)

func safeFragmentCSS(s string, svg bool) bool {
	clean := cssURL.ReplaceAllString(s, "")
	if svg {
		clean = svgKeyframes.ReplaceAllString(clean, "")
	}
	if unsafeCSS.MatchString(clean) {
		return false
	}
	if svg {
		return true
	}
	allowed := wordSet("color background-color font-size font-style font-weight text-decoration white-space height width min-width max-width top bottom left right margin margin-left margin-right margin-top margin-bottom padding padding-left padding-right padding-top padding-bottom vertical-align border border-color border-top-width border-bottom-width border-right-width border-left-width position display text-align border-width border-style border-right-style text-shadow")
	for _, decl := range strings.Split(s, ";") {
		if strings.TrimSpace(decl) == "" {
			continue
		}
		pair := strings.SplitN(decl, ":", 2)
		if len(pair) != 2 || !allowed[strings.TrimSpace(pair[0])] || !cssValue.MatchString(strings.TrimSpace(pair[1])) {
			return false
		}
		if strings.TrimSpace(pair[0]) == "position" {
			value := strings.TrimSpace(pair[1])
			if value != "relative" && value != "absolute" && value != "static" {
				return false
			}
		}
	}
	return true
}
func wordSet(s string) map[string]bool {
	out := map[string]bool{}
	for _, word := range strings.Fields(s) {
		out[word] = true
	}
	return out
}
func validateFragment(fragment, kind string) error {
	if len(fragment) > 4<<20 {
		return fail("INVALID_COMPILED", "编译片段超过 4 MiB")
	}
	tags := wordSet("span pre code")
	attrs := wordSet("class style tabindex aria-hidden")
	svgTags := wordSet("svg g defs filter feDropShadow marker path rect circle ellipse line polyline polygon text tspan textPath clipPath mask pattern linearGradient radialGradient stop symbol use title desc style")
	svgAttrs := wordSet("id class style xmlns xlink width height viewBox preserveAspectRatio role aria-label aria-roledescription aria-labelledby x y x1 x2 y1 y2 dx dy d points fill fill-opacity fill-rule stroke stroke-width stroke-dasharray stroke-dashoffset stroke-linecap stroke-linejoin stroke-opacity opacity transform text-anchor dominant-baseline alignment-baseline font-family font-size font-weight font-style textLength lengthAdjust marker-start marker-mid marker-end markerUnits markerWidth markerHeight refX refY orient cx cy r rx ry href clip-path clipPathUnits mask maskUnits maskContentUnits patternUnits patternContentUnits patternTransform gradientUnits gradientTransform offset stop-color stop-opacity overflow color version data-id data-et shape-rendering xml space filter flood-color flood-opacity stdDeviation name transform-origin clip-rule")
	if kind == "math" {
		for tag := range wordSet("math semantics annotation mrow mi mo mn mtext mfrac msup msub msubsup munder mover munderover mtable mtr mtd msqrt mroot mspace mstyle mpadded menclose mphantom mlabeledtr") {
			tags[tag] = true
		}
		for attr := range wordSet("xmlns display encoding mathvariant stretchy fence separator form lspace rspace accent accentunder rowalign columnalign columnspacing rowspacing scriptlevel displaystyle width height depth voffset notation columnlines rowlines framespacing frame equalrows equalcolumns minsize maxsize largeop movablelimits symmetric mathcolor mathbackground linebreak linethickness mathsize valign columnspan rowspan") {
			attrs[attr] = true
		}
	}
	if kind != "code" {
		for tag := range svgTags {
			if kind == "math" && tag == "style" {
				continue
			}
			tags[tag] = true
		}
		for attr := range svgAttrs {
			attrs[attr] = true
		}
	}
	decoder := xml.NewDecoder(strings.NewReader("<root>" + fragment + "</root>"))
	decoder.Entity = map[string]string{"nbsp": "\u00a0"}
	stack := []string{}
	nodes := 0
	roots := 0
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fail("INVALID_COMPILED", "编译片段不是有效的受限标记")
		}
		switch t := token.(type) {
		case xml.StartElement:
			name := t.Name.Local
			if len(stack) == 0 && name == "root" {
				stack = append(stack, name)
				continue
			}
			if !tags[name] || (t.Name.Space != "" && t.Name.Space != "http://www.w3.org/2000/svg" && t.Name.Space != "http://www.w3.org/1998/Math/MathML") {
				return fail("INVALID_COMPILED", "编译片段含不允许的元素："+name)
			}
			if len(stack) == 1 {
				roots++
				if (kind == "code" && name != "pre") || (kind == "math" && name != "span") || (kind == "svg" && name != "svg") {
					return fail("INVALID_COMPILED", "编译片段根元素无效")
				}
			}
			nodes++
			if nodes > 100000 || len(stack) > 128 {
				return fail("INVALID_COMPILED", "编译片段复杂度超过限制")
			}
			seen := map[string]bool{}
			for _, attr := range t.Attr {
				key := attr.Name.Local
				value := attr.Value
				if seen[key] || (!attrs[key] && !(kind == "svg" && strings.HasPrefix(key, "data-"))) || strings.HasPrefix(strings.ToLower(key), "on") {
					return fail("INVALID_COMPILED", "编译片段属性无效："+key)
				}
				seen[key] = true
				if key == "xmlns" || attr.Name.Space == "xmlns" {
					if value != "http://www.w3.org/2000/svg" && value != "http://www.w3.org/1998/Math/MathML" && value != "http://www.w3.org/1999/xlink" {
						return fail("INVALID_COMPILED", "编译片段命名空间无效")
					}
					continue
				}
				if key == "href" && (!strings.HasPrefix(value, "#") || len(value) < 2) {
					return fail("INVALID_COMPILED", "图表只允许内部引用")
				}
				if key == "class" && !fragmentClass.MatchString(value) {
					return fail("INVALID_COMPILED", "编译片段 class 无效")
				}
				if key == "style" && !safeFragmentCSS(value, kind == "svg") {
					return fail("INVALID_COMPILED", "编译片段样式无效")
				}
				if key != "style" && unsafeAttribute.MatchString(cssURL.ReplaceAllString(value, "")) {
					return fail("INVALID_COMPILED", "编译片段含外部资源或执行内容")
				}
			}
			stack = append(stack, name)
		case xml.EndElement:
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
		case xml.CharData:
			if len(stack) > 0 && stack[len(stack)-1] == "style" && !safeFragmentCSS(string(t), true) {
				return fail("INVALID_COMPILED", "图表样式含外部资源或执行内容")
			}
			if len(stack) == 1 && strings.TrimSpace(string(t)) != "" {
				return fail("INVALID_COMPILED", "编译片段含额外文本")
			}
		case xml.Directive, xml.ProcInst, xml.Comment:
			return fail("INVALID_COMPILED", "编译片段禁止指令与实体定义")
		}
	}
	if roots != 1 {
		return fail("INVALID_COMPILED", "编译片段必须有单一根元素")
	}
	return nil
}
