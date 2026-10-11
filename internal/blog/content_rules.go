package blog

// The MCP writing contract lives beside the compiler, and is returned by the
// authenticated rules endpoint so clients can discover examples before writing.
type contentSyntax struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Example     string `json:"example"`
	Notes       string `json:"notes,omitempty"`
}

func contentRules() map[string]any {
	return map[string]any{
		"workflow":    "副本 → 本地 MCP 编译并上传 → 校验 → 提交草稿 → 明确清单 → 预览 → 展示给用户 → 确认 → 发布",
		"routes":      "文章 slug 必须由元数据明确提供；关联分类 category:{name,slug} 和 tags:[{name,slug}] 也必须提供 slug。categoryId/tagIds 只用于稳定关联，不能代替路由元数据。新分类标签在提交草稿时创建，须在发布清单显式包含其 ID。路由为 /posts/{slug}/、/categories/{slug}/、/tags/{slug}/；slug 最长 100 字符，仅小写字母、数字、连字符。不生成默认值或哈希。已有分类标签改路由用 blog_taxonomy 的 CAS；发布后旧地址 301。",
		"markdown":    "受限 GFM、脚注、[[关键词|纯文本描述]]、$行内公式$、独占行 $$ 块公式、math 围栏、mermaid 围栏；完整写法、示例和限制见 syntax。禁止原始 HTML；代码中的 HTML 仅展示。",
		"syntax":      supportedContentSyntax(),
		"limits":      map[string]int{"markdownBytes": 256 << 10, "mathAndDiagrams": 40, "imageBytes": 10 << 20, "keywordCharacters": 80, "keywordDescriptionCharacters": 2000},
		"renderer":    RendererVersion,
		"permissions": "同一管理 token 拥有全部权限；确认记录不是身份验证。",
		"media":       "/media/{assetId} 或 HTTPS 图片。先用 blog_store_asset 上传 PNG/JPEG/WebP/AVIF/GIF 获得资产 ID，再写入 Markdown；图片点击可放大。视频写在 content.videos 的 HTTPS URL 数组中，不接收视频文件：公开页适配哔哩哔哩 BV 地址，其余平台和私有预览显示链接。来源 URL 写在 content.sources 中。",
	}
}

func supportedContentSyntax() []contentSyntax {
	return []contentSyntax{
		{"headings", "一至六级标题；二至六级进入目录，锚点由程序生成。", "# 一级标题\n\n## 二级标题\n\n### 三级标题\n\n#### 四级标题\n\n##### 五级标题\n\n###### 六级标题", "文章标题填写 content.title；正文标题无需手写 ID。"},
		{"paragraphs", "空行分隔段落；两个行尾空格形成段内换行。", "第一段。\n\n第二段第一行。  \n第二段第二行。", ""},
		{"emphasis", "加粗、强调、组合强调和删除线。", "**加粗**、*强调*、***加粗并强调***、~~删除线~~", ""},
		{"inline_code", "反引号包裹行内代码，内部标记保留原文。", "命令是 `go test ./...`，标记是 `[[词|描述]]`。", "包含反引号时可用更长的反引号包裹。"},
		{"code_blocks", "围栏代码块可标注语言，使用 Shiki 静态高亮并提供复制按钮。缩进四空格的代码块作为普通代码显示。", "```go\npackage main\n\nfunc main() {}\n```", "无语言或未知语言按纯文本处理。HTML、Markdown、公式和关键词标记在普通代码块中仅展示，不执行。math 和 mermaid 围栏有专门含义。"},
		{"lists", "无序、有序和嵌套列表。", "- 第一项\n  - 子项\n- 第二项\n\n1. 第一步\n2. 第二步", "嵌套列表按父项的正文起始位置缩进。"},
		{"task_lists", "任务列表展示已完成和未完成状态。", "- [x] 已完成\n- [ ] 待完成", "阅读页的复选框用于展示，不保存读者修改。"},
		{"blockquotes", "引用、嵌套引用，可包含正文行内样式。", "> 一段引用。\n>\n> > 嵌套引用。", ""},
		{"thematic_breaks", "分隔线。", "上一段。\n\n---\n\n下一段。", "分隔线前后留空行，以免被识别为标题。"},
		{"tables", "GFM 表格，支持列对齐；单元格中的竖线必须转义。", "| 术语 | 说明 |\n| :--- | ---: |\n| [[事务\\|一组操作全部成功或全部失败]] | 点击左侧术语 |", "表格关键词的分隔符写成 \\|；描述中的其他竖线也写成 \\|。"},
		{"links", "显式链接和自动 HTTPS 链接。", "[关于我](/about/)、[外部资料](https://example.com)、https://example.com", "只允许 HTTPS、以单个 / 开头的站内路径和 #锚点；图片另受图片地址规则约束。"},
		{"images", "Markdown 图片，支持替代文本及可选标题，正文图片可点击放大。", "![图示说明](https://example.com/diagram.png \"图片名称\")", "本站图片写成 ![说明](/media/{assetId})；外链只能使用 HTTPS。名称优先标题，其次替代文本。包裹在链接中的图片按链接交互。"},
		{"footnotes", "脚注引用和文末定义。", "正文补充说明[^note]。\n\n[^note]: 这是脚注内容。", "同一标记可多次引用；定义应与引用对应。"},
		{"keywords", "点击关键词，弹出纯文本描述；支持正文、标题、强调、列表、表格和脚注。", "接口应保证 [[幂等性|无论执行一次还是多次，产生的效果都相同。]]。", "必须在同一行，关键词与描述非空，分别最多 80/2000 字；不解析内部 Markdown、HTML 或公式，不能嵌套在链接或图片中。使用 \\|、\\]、\\\\ 表示字面竖线、右方括号和反斜杠。表格使用 [[关键词\\|描述]]。目录只显示关键词，搜索包含描述。"},
		{"escaping", "使用反斜杠显示 Markdown 标点及字面美元符号。", "\\*星号\\*、\\[方括号\\]、\\$20、\\[\\[词|描述]]", "需要展示整段语法时优先使用行内代码或普通代码围栏。"},
		{"inline_math", "美元符号包裹 LaTeX 行内公式，由 KaTeX 在发布前编译。", "能量公式：$E=mc^2$。", "$ 内侧不能以空白开始或结束；字面美元符号写成 \\$。禁止受信任命令、外部资源和不支持的语法。"},
		{"block_math", "独占一行的 $$ 开始和结束块级公式。", "$$\n\\int_0^1 x^2\\,dx = \\frac{1}{3}\n$$", "不要写成同一行 $$公式$$。每个公式源最多 8 KiB；公式和图表合计最多 40 个。"},
		{"math_fences", "math 围栏也可书写块级 LaTeX 公式。", "```math\n\\frac{a}{b}+\\sqrt{x}\n```", "与块公式使用相同的编译和限制。"},
		{"mermaid_flowchart", "Mermaid 流程图，支持 flowchart 或 graph。", "```mermaid\nflowchart LR\n A[开始] --> B[结束]\n```", mermaidSyntaxNotes},
		{"mermaid_sequence", "Mermaid 时序图。", "```mermaid\nsequenceDiagram\n participant A as 客户端\n participant B as 服务端\n A->>B: 请求\n B->>A: 响应\n```", mermaidSyntaxNotes},
		{"mermaid_class", "Mermaid 类图。", "```mermaid\nclassDiagram\n class Article {\n  +String title\n }\n```", mermaidSyntaxNotes},
		{"mermaid_state", "Mermaid 状态图，支持 stateDiagram 和 stateDiagram-v2。", "```mermaid\nstateDiagram-v2\n [*] --> Draft\n Draft --> Published\n Published --> [*]\n```", mermaidSyntaxNotes},
		{"mermaid_er", "Mermaid 实体关系图。", "```mermaid\nerDiagram\n AUTHOR ||--o{ ARTICLE : writes\n```", mermaidSyntaxNotes},
		{"mermaid_gantt", "Mermaid 甘特图。", "```mermaid\ngantt\n title 写作计划\n dateFormat YYYY-MM-DD\n section 写作\n 初稿 :a, 2026-10-11, 2d\n```", mermaidSyntaxNotes},
		{"mermaid_pie", "Mermaid 饼图。", "```mermaid\npie\n title 内容比例\n \"技术\" : 60\n \"随想\" : 40\n```", mermaidSyntaxNotes},
		{"mermaid_timeline", "Mermaid 时间线。", "```mermaid\ntimeline\n title 项目历程\n 2026 : 开始写作\n```", mermaidSyntaxNotes},
		{"raw_html", "正文禁止原始 HTML，包括 script、iframe、自定义标签和样式；HTML 教学内容必须放在代码块中。", "```html\n<button onclick=\"alert('示例')\">仅展示源码</button>\n```", "不要用 HTML 实现关键词浮窗、图片或视频；使用对应 Markdown 语法或元数据。"},
	}
}

const mermaidSyntaxNotes = "mermaid 围栏在发布前编译成静态 SVG。仅支持规则中列出的八类图；单图源最多 16 KiB、最多 150 条边，公式和图表合计最多 40 个。禁止配置指令、frontmatter、HTML、click 回调、外部资源和自定义样式。"
