# AI-native 博客：技术栈评估与架构建议

核查日期：2026-10-07（Asia/Shanghai）。本文件是方案研究，记录用户已确认的产品边界及技术建议，未进行系统实现、压测或生产部署。范围为间隙：单站点、以文章阅读为主、通过 Agent/MCP 管理内容的自研博客系统。

## 1. 推荐结论

首版采用 **Go 1.27.1 + 标准库 `net/http` / `html/template` + Goldmark + SQLite + 原生 CSS**。首页、文章、分类页、标签页和 About 在发布时生成完整静态 HTML，读者访问已发布的不可变制品；后台设置、中文搜索、私有管理 API、审核、发布和推送任务由同一个 Go 服务提供。MCP 使用用户本机的 stdio 桥接程序访问私有管理 API，首版不提供公网 HTTP MCP endpoint。内容及任务状态以 SQLite 为权威，Markdown 用于导入、导出和可读备份，磁盘 HTML 是可重新构建的发布制品。

新增已确认的内容能力包括行内/块 LaTeX 公式、HTML/Markdown及常见语言源码高亮、Mermaid声明式图。AI只能改源Markdown/元数据/受控媒体，不能直接写文章HTML页面或模板。建议Chroma在Go编译阶段高亮，KaTeX `renderToString` 与Mermaid CLI在发布阶段输出公式HTML/MathML和SVG；阅读免JS，但**发布环境新增固定Node/Puppeteer/Chromium及字体**，不是纯Go小镜像。详细语法、诊断和安全/资源边界见[内容渲染方案](./content-rendering.md)。

这里的“静态”指内容阅读页：正文、导航、分类、标签、About、元数据与站点地图都预先生成。**公开 `/search` 是明确的动态只读例外**，查询当前发布版的中文全文索引并返回 HTML；后台也使用 Go 模板按请求渲染。两者不改变文章在发布时生成的方式。系统需要 Go 服务承接搜索、后台、MCP 桥接的管理请求和有状态任务；用户本机运行 stdio bridge，不增加服务端 MCP 容器。用户明确不做 RSS/Atom。

前端首版不需要 React、Vue 或 SPA。Go 模板负责 HTML 结构，CSS 负责响应式布局与排版；文章正文、目录锚点跳转、主题色和搜索表单均可在不加载 JS 时使用。后台使用同一 `html/template` 与 CSS，确认提示、状态刷新等确有需要时才加少量 JS。增强交互不能成为正文或 SEO 元数据出现的前提。`html/template` 原生提供上下文相关的转义，标准 HTTP 路由从 Go 1.22 已支持请求方法与通配路径，足够承载本项目首版入口。[Go HTML 模板](https://pkg.go.dev/html/template)、[Go 路由增强](https://go.dev/blog/routing-enhancements)。

用户最新确认范围是 **仅国内中文、个人单站、单机或单容器、低运维**；不做 Google、英文或多语言系统。AI 通过 MCP 上传和修改正文、About、分类、标签和图片，用户确认后发布；必须有后台设置界面，后台与 MCP 使用同一个具有全权限的 master token。首页、单主分类与多标签、分类页、标签页、目录锚点跳转、搜索、发表/更新时间、About、内容预览/回滚和备份导出/导入均为首版必需，分类和标签无层级。后台只编辑站点与第三方服务设置，不做文章富文本编辑器、插件或主题系统。分析首版提供访问统计及 SEO/GEO 外部服务链接/报告导入，明确可用 API 才后接集成。建议具体实现为单台 Linux 主机、单 Go 进程和本地持久化卷，先不引入独立数据库、搜索服务或容器编排平台。

## 2. “轻量”需要分别衡量

| 维度 | 本项目要求 | 不能混为一谈的情况 |
| --- | --- | --- |
| 浏览器负担 | 默认阅读页无可执行 JS；语义化完整 HTML；CSS 小；图片按需要加载 | Astro 可以没有客户端 JS，仍有 Node 构建依赖 |
| 在线运行负担 | 单个 Go 进程；静态阅读请求不查数据库、不调用模型、不渲染 Markdown；搜索和后台明确允许查询数据库 | Go SSR 可做到浏览器 0 JS，但每次访问仍计算页面 |
| 构建与维护负担 | Go控制面 + 固定公式/图构建工具；发布不联网下载新包；依赖/配置/字体有hash | 内容阅读0JS不代表Node/Chrome构建开销不存在；单容器自动发布必须保留辅助工具 |
| 开发体验 | 本地 Windows 可开发、生产 Linux 可验证；依赖少、边界清楚 | 减少运行组件不代表所有生成器功能都值得自己重写 |

以下选型依据是功能、依赖和操作复杂度；没有把网上的性能数字当成本项目实测数据。

## 3. 四条路线比较

| 路线 | 公开页面 | 构建依赖 | 自动发布时的服务依赖 | 主要收益 | 主要代价 | 建议 |
| --- | --- | --- | --- | --- | --- | --- |
| A. Go + `html/template` + Goldmark，发布时 SSG | 内容页完整静态HTML/公式/SVG，默认0JS；后台与搜索按请求渲染 | Go模块 + 固定Node/KaTeX/MermaidCLI/Puppeteer/Chromium | 单Go服务、SQLite、制品；按任务启动辅助渲染进程 | 与MCP、设置、搜索、发布共用模型，不另建前端应用 | 需实现有限生成器功能，额外图构建峰值/镜像要实测 | **首选**：范围明确，但不称纯Go构建 |
| B. Go 控制面 + Hugo 子进程 | 完整静态HTML，是否加载JS取决于布局 | Go、Hugo及同样需要的公式/图渲染工具 | 自动发布节点须运行Hugo/辅助renderer；后台搜索仍为Go | 成熟分类/输出/资源处理 | DB导出适配、两套约定与额外子进程；Hugo本身不消除Math/Mermaid依赖 | 内容生成复杂度明显增长时备选 |
| C. Go 控制面 + Astro SSG | Astro 组件默认输出静态 HTML，默认 0 客户端 JS | Go + Node/npm 或其他 JS 包管理器 | 若在服务端实时构建，Node 也必须在发布节点；若在 CI 构建，在线读服务可不带 Node | 前端组件和 CSS 工具生态好；交互可用 islands | 双语言、双锁文件和构建链；Go 内容数据需要供 Astro 消费 | 有专门前端团队或复杂交互再考虑 |
| D. Go + templ，公开文章在请求时 SSR | 可默认 0 JS，但每次请求生成文章页 | Go + templ 代码生成器 | Go 服务，通常还需要运行时内容查询 | 编译期类型检查、Go 组件复用 | 文章无需每次请求生成；缓存/一致性需另设计 | 首版文章不采用 SSR；后台/搜索用标准模板，templ 静态模式仍可替代 A 的模板层 |

Hugo 官方 CLI 明确用于构建静态站点，并提供输出目录、草稿/未来文章控制等参数；默认 Markdown 渲染器就是 Goldmark，默认关闭 unsafe HTML。采用 Hugo 并不强迫我们提供主题或插件系统，可以只维护一组固定布局。[Hugo CLI](https://gohugo.io/commands/hugo/)、[Hugo Markdown 配置](https://gohugo.io/configuration/markup/)。

Astro 官方组件文档明确默认没有客户端运行时；当前安装文档要求 Node.js **22.12.0 或以上**，因此不能用“零客户端 JS”推导出“没有 Node 构建依赖”。SSG 制品可单独静态托管，只有按需服务器渲染才需要对应在线运行环境。[Astro 组件](https://docs.astro.build/en/basics/astro-components/)、[Astro 安装前提](https://docs.astro.build/en/install-and-setup/)、[Astro 部署](https://docs.astro.build/en/guides/deploy/)。

templ 并非只适用于 SSR。官方静态博客示例直接调用组件 `Render(context, file)` 输出 HTML 文件，再交给静态托管。若后续页面复用与类型安全的收益超过代码生成成本，可将 A 的 `html/template` 替换成 templ，不必改变“发布时静态生成”的架构。[templ 静态博客示例](https://templ.guide/static-rendering/blog-example)。

**为什么不是一开始就用成熟 SSG：** 本项目需要把 MCP、后台设置、内容修订、中文搜索、发布快照与推送队列贯穿起来。A 可以复用同一 Go 领域模型，避免新增“数据库 → 外部构建输入 → 外部生成器 → 回收结果”的适配层。这个优势成立于单站中文博客的有限内容模型；若复杂图片变换或大量自定义输出导致维护成本明显增长，再重新比较 B。

## 4. 候选版本与最小依赖

下表是本次核查时的**起始锁定候选**，不是已经通过漏洞检查、编译和兼容性测试的项目锁文件。实现前需固定版本、校验许可证与传递依赖，并运行 `govulncheck`；不能把任意 `@latest` 留在可重复部署流程中。

| 组件 | 候选版本 / 约束 | 是否首版必需 | 核查依据 |
| --- | --- | --- | --- |
| Go 工具链 | 推荐 1.27.1；兼容最低版本由依赖决定 | 是 | 官方当前列出 1.27.1 和 1.26.8 两条稳定分支；每个大版本仅支持到出现两个更新大版本为止，[发布政策与历史](https://go.dev/doc/devel/release)、[下载版本 JSON](https://go.dev/dl/?mode=json) |
| HTTP、模板、JSON、XML、日志 | Go 标准库；`net/http`、`html/template`、`encoding/json`、`encoding/xml`、`log/slog` | 是 | 不需要再引入 Web 框架、模板框架或日志框架 |
| 后台设置 UI | Go `html/template` + 原生 CSS + 必要的小段 JS | 是 | 与公开站同一部署，不新增前端框架或独立管理服务；后台可动态渲染 |
| MCP | 官方 Go SDK；版本由 MCP 专项核查确定 | 是 | 与其他依赖一起复核最低 Go 版本，不能单独只看 SDK |
| SQLite 访问 | `database/sql` + `modernc.org/sqlite` **v1.60.1**；最低 Go **1.26.0** | 是 | [固定 tag 的 go.mod](https://raw.githubusercontent.com/modernc-org/sqlite/v1.60.1/go.mod)、[驱动版本页](https://pkg.go.dev/modernc.org/sqlite) |
| modernc 间接依赖 | `modernc.org/libc` **v1.77.1**，与上述驱动要求完全一致 | 随驱动 | 官方专门提示 libc 版本脆弱依赖，[驱动说明](https://raw.githubusercontent.com/modernc-org/sqlite/v1.60.1/doc.go) |
| Markdown | `github.com/yuin/goldmark` **v1.8.6**；最低 Go **1.22** | 是 | [固定 tag 的 go.mod](https://raw.githubusercontent.com/yuin/goldmark/v1.8.6/go.mod)、[Goldmark 文档](https://pkg.go.dev/github.com/yuin/goldmark) |
| 正文 HTML 净化 | `github.com/microcosm-cc/bluemonday` **v1.0.27**；最低 Go **1.19** | 建议是 | [固定 tag 的 go.mod](https://raw.githubusercontent.com/microcosm-cc/bluemonday/v1.0.27/go.mod)、[官方 README](https://github.com/microcosm-cc/bluemonday)；传递依赖另做漏洞核查 |
| 代码高亮 | `github.com/alecthomas/chroma/v2` **v2.27.0**；最低Go **1.25** | 是 | HTML/Markdown/latex/tex及常见fence是源码高亮不执行，[固定go.mod](https://raw.githubusercontent.com/alecthomas/chroma/v2.27.0/go.mod)、[官方库](https://github.com/alecthomas/chroma)；不无意升级v3 |
| 公式 | 固定KaTeX构建库；`renderToString` + 对应CSS/fonts | 是 | 发布输出HTML/MathML；不是Goldmark自动支持，[KaTeX API](https://katex.org/docs/api.html) |
| Mermaid图 | 固定MermaidCLI/Puppeteer/Chromium + 本地字体 | 是 | 输出SVG，允许类型全部编译；Node/浏览器开销不能省略，[CLI](https://github.com/mermaid-js/mermaid-cli) |
| CSS与字体 | 原生基础CSS/系统字体 + 独立Chroma/KaTeX资源，渲染用字体固定hash | 是 | 无需Tailwind/前端框架；公式CSS/fonts与图字体不计成整页20KiB，[KaTeX资源](https://katex.org/docs/browser.html) |
| esbuild | 可选固定版本 CLI 或 Go API，用于已有脚本/CSS 需要压缩打包时 | 否 | 首版没有实质 JS 时，引入它没有足够收益，[官方 API](https://esbuild.github.io/api/) |
| 中文全文搜索 | Go + 已有 SQLite；FTS5 trigram + 一二字符字面包含回退 | 是 | 无独立搜索服务、词典或新语言工具链；具体限制与验收见第 9 节，[SQLite FTS5](https://www.sqlite.org/fts5.html#the_trigram_tokenizer) |
| Pagefind | extended 独立二进制与浏览器 JS/WASM | 首版不选 | 作为严格全静态搜索的对照方案；分词不保证任意一二汉字子串召回，[官方中文说明](https://pagefind.app/docs/multilingual/) |

按上述必需依赖组合，最低 Go 至少为 **1.26**，不能因标准库路由在 1.22 就可用而把整体项目写成 Go 1.22。新项目直接使用当前受支持的 Go 1.27 工具链更清楚；若特定部署环境只能用 Go 1.26，则选该分支最新补丁并在实际锁文件下编译验证。

`modernc.org/sqlite` 为 CGo-free，但不是“没有传递依赖”或“比 C 驱动更快”。官方文档承认其转译的 SQLite 在 CPU 密集型查询上存在成本；本项目静态阅读请求不经过数据库，后台写入低频，公开搜索要单独设置并发与查询预算，因此可优先选择无 C 编译器和跨平台构建更简单的路线。其 v1.60.1 平台表包含 Windows amd64/arm64 和 Linux amd64/arm64，所带 SQLite 为 3.53.4。[官方平台和性能说明](https://raw.githubusercontent.com/modernc-org/sqlite/v1.60.1/doc.go)。

替代驱动 `github.com/mattn/go-sqlite3` 当前候选 v1.14.52（[固定 tag 的 module 声明 Go 1.21](https://raw.githubusercontent.com/mattn/go-sqlite3/v1.14.52/go.mod)），需要 `CGO_ENABLED=1` 和 C 编译器，跨编译须额外准备目标平台编译器。其 FTS5 通过 `sqlite_fts5` 构建标签启用。除非实际发现纯 Go 驱动的 CPU 或体积不满足目标，否则首版不承担这套 C 工具链。对任一驱动，FTS5 若进入需求，都应对选定二进制做 `PRAGMA compile_options` 与创建 FTS5 表的能力验证，而不是只由依赖名称推断。[mattn 官方构建说明](https://github.com/mattn/go-sqlite3)。

## 5. 内容权威：SQLite 与文件如何取舍

| 权威来源 | 适合情况 | 优点 | 额外成本 | 本项目判断 |
| --- | --- | --- | --- | --- |
| Markdown 文件 + Git 是内容权威；SQLite 仅存任务/回执 | 单作者希望直接在代码仓库中维护文本，发布走 CI | 文件可直接阅读，修订记录天然可审阅 | MCP 改文件需要 Git 状态协调；内容与任务不同事务；并发改动/回滚需处理冲突 | 可选，但“以 Agent 为管理入口”的首版不默认采用 |
| SQLite 的 Markdown 修订是内容权威；导出 Markdown | MCP 管理草稿、审核、修订和发布任务 | 同一事务记录修订与任务；可做乐观锁；审计、幂等和状态查询明确 | 要做可读导出、迁移和数据库备份 | **推荐** |
| DB 和可编辑 Markdown 文件同时是权威 | 看似兼顾两者 | 无明确收益 | 必须解决同步方向、冲突优先级、丢失修改、回滚与事件循环 | 不采用 |

“无编辑器”不等于“无内容模型”。Agent 仍要创建草稿、读取现有修订、更新摘要和元数据、预览结果并请求发布。以 Markdown 保存正文，让模型与人都能读；不引入富文本块协议、可执行 MDX、任意模板代码或插件配置。

源内容API只接收约束Markdown/元数据/资源：raw HTML按AST拒绝，html/markdown/latex/tex fence按源码显示，math fence及美元分隔符专用公式，mermaid fence专用图编译。AI不能上传页面HTML/SVG附件替换站点DOM；图片上传首版优先PNG/JPEG/WebP。搜索抽取代码文本、图说明与TeX可读源，避免重复收录KaTeX视觉span/隐藏MathML或SVG标签。

建议首版领域表如下；字段为实现建议，最终命名与 MCP 专项设计统一。

| 实体 | 关键字段 | 约束 / 用途 |
| --- | --- | --- |
| `posts` | `id`、稳定 slug、单个主分类 ID、当前草稿 revision、当前发布 revision、`published_at`、公开 `updated_at`、状态 | 稳定 ID 独立于标题；改 slug 创建 redirect；草稿内部更新时间不自动改写公开发表/更新时间 |
| `revisions` | `id`、`post_id`、版本序号、Markdown、title、summary、SEO 字段、引用、author、created_at、content_hash | 修订不可原地覆盖；更新携带 expected revision，防止两个 Agent 覆写彼此内容；发布历史单独计数 |
| `pages` / 单页修订 | About 的稳定 ID、Markdown、元数据、草稿/发布 revision | 内容由 MCP 上传，后台可查看/预览/回滚；About 作为必需单页，不建立任意模板与内容类型平台 |
| `categories`、`tags`、文章标签关联 | 平面分类/标签 ID、name、slug、说明；文章主分类/标签关联 | 单主分类 + 多标签，标签去重，无父级或嵌套分类；公开变更进入待确认 taxonomy 修订与 release 快照 |
| `site_settings` / 修订 | 站点名称、域名、介绍、导航、第三方服务配置引用、版本、修改时间 | 后台表单与 MCP 复用校验/更新逻辑；会影响页面的设置进入 release 构建配置 |
| `search_documents` / FTS 索引 | release ID、content ID、published revision、title、summary、纯文本正文、类别/标签快照、canonical URL | 仅索引被选定发布的内容；搜索固定当前 release，不能混入草稿/未来 release，也不能 join 当前可变 taxonomy |
| `analytics_daily` / 外部报告 | 受控聚合访问指标、日期、来源说明；报告导入记录 | 可先用 SQLite 与后台报表，不承诺第三方无 API 的自动取数；源站计数与 CDN 访问需区分 |
| `assets` | ID、相对存储键、MIME、宽高、hash、alt、归属 | 文件键由服务生成；Markdown 不可任意读本机路径；哈希资产可跨 release 共用 |
| `releases` | release ID、选定revision集、taxonomy/settings snapshot、确认记录、renderer/config/font hash、manifest、状态、时间 | 确认绑定源revision及有效渲染配置；renderer/字体变化需新预览，失败保留旧站；配合历史回滚 |
| `jobs` | type、payload、状态、attempt、next_attempt_at、lease、幂等 key、关联 release | 数据库持久化；进程重启恢复；重试带退避，不靠内存 goroutine 保证可靠性 |
| `delivery_receipts` | channel、URL、动作、提交时间、HTTP 状态、外部回执、尝试次数 | “提交成功”与“已收录/被引用”是不同状态；不从 2xx 推断最终效果 |
| `audit_events` | principal、operation、目标、前后 revision、时间、trace ID | 区分读、改稿、授权发布和外部分发；日志避免存 token 或模型密钥 |
| `redirects` / `tombstones` | 原 URL、目标 / 删除状态、生效 release | 对移动、合并、撤稿和删除给出正确 HTTP 行为与推送动作 |

仅有一个作者时，其介绍与默认社交链接可放站点配置；不为预想的多用户组织设计角色层级、租户或作者商城。

站点固定中文，HTML 声明 `lang="zh-CN"`；单套稳定路径即可，不引入语言前缀、article family、翻译版本或 `hreflang` 输出。文章原始发表时间与已发布实质更新的时间分别保存，重建全站或修改草稿不能把所有文章的公开更新时间刷新为当前时间。

后台历史已确认按 **每篇文章 / About 当前已发布版本 + 3 个已发布旧版** 计算，下架保留最后公开版及其 3 旧版；不把该名额套到审计、回执、任务或备份。回滚只把选定旧修订形成新候选，不覆盖工作副本和当前草稿。AI 修改独立副本，上传、解析通过后 CAS 提交新草稿；副本、上传及整站产物单独清理。永久删除清理全部可恢复内容，详见需求确认稿。

## 6. 数据库与持久化任务

SQLite WAL 允许读写并发，但**同一时刻仍只有一个写者**；WAL 依赖同一主机的共享状态，不适合把数据库放在 NFS 共享盘或多个服务器同时访问。[SQLite WAL](https://sqlite.org/wal.html)。

建议所有写入走受控写事务，持有锁期间只做必要的 SQL，不能把 Markdown 渲染、外网推送或模型请求放进事务。限制 `database/sql` 连接池：管理写入只允许一个受控写连接，公开搜索使用少量只读连接与并发闸门，避免短词扫描占满共享连接池而阻塞发布。访问统计聚合写入，避免为每次静态文件请求制造一笔管理数据库写事务。WAL、foreign keys、busy timeout 等连接相关设置应由 DSN/connection hook 对每个连接生效，不能在池上随手执行一次 PRAGMA 就假设覆盖全部连接。[modernc 连接状态说明](https://raw.githubusercontent.com/modernc-org/sqlite/v1.60.1/doc.go)。

发布任务与待分发事件使用数据库 outbox：写入目标内容修订的事务同时记下构建任务；构建、公开可达检查与 release 激活完成后再生成或启用相应的外部分发任务。公开 URL 未实际可达时不抢先推送。

外网推送采用至少一次执行配合幂等键，记录 attempt 和渠道响应；不能承诺网络外部操作的 exactly-once。内容版本、canonical URL、渠道和动作可组合成幂等键。周期任务也要避免前一轮尚未结束就无限堆积新 goroutine。

完整备份覆盖一致性业务快照、源文、工作副本 / 上传、草稿、历史、媒体、所有业务配置与第三方凭据、分析报告 / 证据、路由与任务 / 回执 / 审计记录，manifest 标注敏感及格式 / 校验信息。**master token 与浏览器会话永不导出**，运行锁和进程状态不恢复；内容导出不等于完整备份。恢复先隔离校验重建并预览，再检查全站写入代次、保存恢复前完整备份后切换全部业务状态。当前 master 保留，旧会话失效，历史任务仅恢复记录、不重放；追加恢复事件并提升实例代次。外链仅保存引用。需在干净实例验证恢复。

备份不能只复制运行中数据库的 `.db` 文件并丢掉 WAL。优先使用 SQLite Online Backup API 形成一致快照；也可在实现中核验驱动支持后采用 `VACUUM INTO`。[SQLite 在线备份](https://sqlite.org/backup.html)。

一致快照只是导出的起点：导出前排除 master/session 与运行锁，不能把运行数据库未经处理原样打包。完整备份包含第三方凭据并明确标注；便携内容导出排除凭据但不提供完整恢复承诺。数据库副本与可读源文件来自同一快照并交叉校验。

## 7. 发布制品、原子切换与部署

建议生产拓扑：一个 Go 容器 + 一个持久化数据卷，HTTPS 由已有反向代理或托管边缘处理。公开页面/搜索无需登录；后台 `/admin/` 与 MCP stdio bridge 请求的私有 control API（建议 `/_control/`）使用同一个 master token 和相同全权限身份，不引入用户表、分级账号、OAuth 服务或额外管理容器。stdio 传输终止于用户本机的 bridge；服务端不开放 `/mcp` 或其他公网 HTTP MCP endpoint。CDN 是可选部署能力，首版实现不依赖特定 CDN API。

浏览器后台推荐通过登录表单验证同一个 master token，换取短时、可失效的 `Secure` / `HttpOnly` / `SameSite` 会话 Cookie；Cookie 写操作与浏览器预览均走 `/admin/` 路径。MCP stdio bridge 到 `/_control/` 的私有管理请求使用 `Authorization: Bearer <master token>`；这不是公网 HTTP MCP 协议，也不是另建一套权限或账号。token 不进入 URL、浏览器 localStorage、站点配置的公开输出或日志；第三方服务凭证在 UI 脱敏。基于 Cookie 的后台写操作需要同源/CSRF 防护，所有变更使用非 GET 请求。内容发布仍须确认具体修订，不能把“拥有全权限 token”与“已确认某篇新草稿”混为一回事。

后台使用 Go 表单编辑站点信息、导航和第三方服务配置；内容列表、预览、历史比较/回滚和发布确认复用 MCP 的领域服务，但不提供正文输入编辑器。About、分类、标签、正文和图片由 MCP 上传与更新。访问统计展示与外部 SEO/GEO 服务链接、报告导入复用本后台；首版不为未知第三方服务设计通用插件或自动取数协议。

发布流程：

1. 先固定 ChangeSet 中明确选择的 revision 集、taxonomy / 设置 / 资源与构建配置，创建候选；AI 副本经上传解析成为新草稿后，相关旧预览失效，不能边读可变数据边构建。
2. 在暂存区以同一AST编译器生成首页、文章、分类/标签页、About、sitemap、目录、代码高亮、公式与图资产，固定taxonomy/settings/renderer/font快照并生成同版搜索索引。图/公式有界任务与hash缓存，任一解析/资源/产物失败阻止ready并保留旧站，后台给源行列诊断。不生成RSS/Atom；公开Markdown仅P1可选。
3. 验证结构、链接、metadata、敏感内容与清单，展示预览再确认；确认绑定内容、预览 hash 与时间定稿规则。发布只填入时间及其直接派生字段 / 列表排序，记录最终 hash，不重编译正文 / 公式 / 图；构建失败保留旧 release。
4. 持久保存最终产物后，在 SQLite 事务中提交 active release、文章时间、搜索版本与 outbox，再更新内存快照。每个请求只读取一次快照，搜索不 join 当前可变 taxonomy；所有公开变更均在已确认 ChangeSet 内。
5. 完成公开可达检查和缓存处理，再启动推送；执行前检查 URL 和配置仍有效。旧制品 / 索引按独立策略清理，每文章 / About 保留当前 + 3 旧版；永久删除取消引用任务并等待相关读取结束再物理清理，未清理完不报告完成。

SQLite 提交、产物落盘与外部推送没有共同事务。SQLite active release 是唯一激活权威，磁盘不维护可独立决定上线版本的指针；先完成产物，再提交数据库，最后刷新内存。启动校验 manifest / hash 并按数据库恢复；未提交候选不能因目录存在就上线，已提交产物丢失时明确报错。具体恢复协议见技术决策，不引入分布式事务系统。

Linux 同一文件系统上的指针文件替换或软链接切换可用于原子部署；Go 的 `os.Rename` 文档明确**非 Unix 平台即使同目录也不保证原子性**。Windows 开发不能拿“先删当前目录再移动新目录”模拟生产保证；可采用不可变目录 + 内存快照切换，崩溃恢复协议另行验证。生产要在 Linux 上验证并发切换与失败回滚。[Go Rename 契约](https://pkg.go.dev/os#Rename)。

`embed.FS` 适合随应用版本固定的模板和 UI 资产；它读取的是编译期文件，不适合把频繁发布的文章全部嵌入二进制，否则每次发布要重新编译部署。公开正文采用外部不可变 release 目录。[Go embed](https://pkg.go.dev/embed)。

公共文件服务只能暴露发布目录，不能直接以项目根目录或数据根目录创建 `http.FileServer`。标准库 `http.Dir` 会跟随符号链接，也可暴露点文件；应以清单或约束 FS 限制可访问文件，禁用目录列表，拒绝工作目录、`.git`、数据库和草稿路径。构建写入及 MCP 资产操作使用相对键和目录约束，例如 `os.Root`；`path` 处理 URL，`filepath` 处理 OS 路径，不能混用。[Go 文件服务行为](https://pkg.go.dev/net/http)、[Go 防目录穿越 API](https://go.dev/blog/osroot)。

哈希 JS/CSS/媒体可用长缓存与 immutable；HTML 与 sitemap 需合适的验证缓存与更新策略，P1 若启用公开 Markdown 表示也采用对应内容更新策略。搜索结果按 query + release ID 隔离，后台、预览和确认页面禁用共享缓存。CDN 缓存失效失败也应成为有回执的任务，不能把“源站已切换”自动当成“所有边缘已更新”。如果业务要求 Go 服务停止时公开阅读页仍在线，可把 release 同步至独立静态托管/CDN；单容器方案本身不提供这种故障隔离，动态搜索和后台也依赖 Go 在线。

301/308 redirect 与有效公开页面由当前 release 路由清单驱动，下架 / 永久删除旧址返回 404，不为删除内容长期保留正文或墓碑。匿名内容资产也只允许当前公开版本引用的资源，旧 release 目录不公开。请求期间无须查 SQLite。若导出到独立静态托管，须另生成并验证对应路由与访问规则。

## 8. Markdown 与 HTML 安全边界

Agent可导入外部文章、引用和生成内容，私有MCP输入仍可能是不可信文本。API不接受文章HTML或可执行Markdown/MDX；统一受约束源内容编译，不允许Agent写模板、脚本或发布目录。公式与Mermaid声明式源码经受控renderer处理，不能扩大为执行任意.tex/JS。

Goldmark默认不渲染raw HTML与危险URL，首版按AST拒绝正文raw HTML，不开 `html.WithUnsafe`。AST分派普通Markdown、代码、公式与图，各用固定结构策略校验/净化后合成模板。普通UGC净化不能删掉MathML/class/style后仍称支持数学，也不能为数学放开任意HTML/SVG。只有程序renderer与对应策略验证的片段进入 `template.HTML`，普通字段继续转义。[Goldmark](https://pkg.go.dev/github.com/yuin/goldmark)、[Go模板](https://pkg.go.dev/html/template)、[KaTeX安全](https://katex.org/docs/security.html)。

bluemonday 官方建议在其他内容处理之后净化，避免后续转换重新引入危险 HTML。策略须允许实际需要的文章元素、经过约束的 heading ID 与高亮 class，使目录锚点在净化后仍存在；同时拒绝 script、事件属性、危险 URI、任意 style/iframe。不能因为净化破坏了样式或目录就跳过它。[bluemonday 官方说明](https://github.com/microcosm-cc/bluemonday)。

代码高亮为首版必需，Chroma发布时输出 `<pre><code>` 与class/CSS，不引入浏览器高亮库；html/markdown/latex/tex fence只展示不执行。图用编译的自托管SVG及img/alt/说明/可展开源码，上传SVG不自动受信；strict不等于进程隔离，净化不能证明SVG直接浏览永无脚本，需资产策略/CSP/浏览器测试。[Chroma](https://github.com/alecthomas/chroma)、[渲染边界](./content-rendering.md)。

外部资源抓取不应成为任意 URL 的服务器代理。MVP 可要求通过受控上传提交媒体，抓取远程资源如需进入范围再单独加入限长、超时、MIME 校验和 SSRF 防护。用户能删除或撤回一篇文章时，旧 release 的可访问性和 CDN 清理也要遵循撤回策略，不能让“保留回滚”导致敏感内容继续公开。

## 9. 必需中文搜索与前台呈现

**推荐首版采用 Go + SQLite 的字面子串搜索：三字符及以上用 FTS5 trigram，一二字符走发布纯文本的字面包含回退。** 这不是词典分词或语义检索；它明确解决个人中文博客中短词、技术名词和跨分词边界的包含检索，并继续使用现有 SQLite，不引入搜索服务器、CGo 中文库或浏览器搜索运行时。

| 方案 | 中文与短查询行为 | 新增依赖 / 运维 | 浏览器与静态边界 | 判断 |
| --- | --- | --- | --- | --- |
| Go + SQLite，trigram + 短词回退 | 无词典的字符子串匹配；三字符以上索引；一二字符仍能包含命中，但需扫描 | 已有 Go/SQLite；增加发布索引、查询预算与一致性验证 | `/search` 动态只读，普通表单和 HTML 结果即可，默认不需要 JS | **首选**：短查询语义明确，个人单机低运维 |
| 仅 SQLite unicode61 | 按连续 Unicode 字母/数字组成 token；无法自动把整段中文切成词 | 无新服务，但要另补中文处理 | 动态只读 | 不作为中文搜索实现 |
| Go + 中文词典分词 + FTS5 | 词语检索与相关性可更丰富；词内/跨词的一二字符包含仍需额外 n-gram/扫描设计 | 分词库、词典、分词配置和更新；gojieba 还需 CGo/C++ | 动态只读 | 首版不为有限检索需求承担这套复杂度 |
| Pagefind extended | 索引与查询均支持中文分词；可匹配已有词，但不保证任意一二字符子词命中 | 固定 extended 构建二进制、额外索引步骤、JS/WASM 和索引分片；独立二进制不强制 Node | 无在线搜索服务，客户端需要 JS/WASM | 严格全静态部署有优势，当前单 Go 服务及短词需求下不选 |

**官方文档确认的边界：** SQLite unicode61 将连续的 token characters 视为同一个 token，因此不能把它当作中文字典分词器。FTS5 trigram 是每三个 Unicode 字符形成一个 token；不足三个字符的 `MATCH` 查询不会命中任何行。`LIKE`/`GLOB` 不具备可利用的连续三字符模式时会回退线性扫描；使用 `LIKE ... ESCAPE` 时也不能靠 trigram 索引优化。因此不能声称“单字 MATCH 正常”或“带通配转义的 LIKE 一律有索引”。[SQLite Unicode61](https://www.sqlite.org/fts5.html#unicode61_tokenizer)、[SQLite Trigram](https://www.sqlite.org/fts5.html#the_trigram_tokenizer)。

Pagefind 1.5.0 已修复索引分词与查询分词不一致的问题，不能继续引用旧版说“只索引分词、查询不分词”。但其维护者明确解释，这项改进不支持任意子词搜索，例如查询“段”不会仅因正文含“一段”就命中；issue #987 关闭代表查询分词功能已发布，不代表变成任意字符包含搜索。该判断来自维护者说明，而非把第三方抱怨当现版本行为。[Pagefind 中文文档](https://pagefind.app/docs/multilingual/)、[官方仓库讨论 #987](https://github.com/Pagefind/pagefind/issues/987)。

Pagefind extended 的中文分词、Rust/WASM 浏览器搜索及静态索引适合完全静态托管；它提供独立预编译二进制，不必强制增加 Node。这里不选择它的原因是既已有运行中的 Go/SQLite 后端，又需要明确保证一二汉字的字面包含，额外浏览器和构建组件未带来足够收益。没有实际构建本站索引，不能宣称它的下载包或峰值内存具体有多小。[Pagefind 安装和运行方式](https://pagefind.app/docs/)。

中文分词库也不是免费的补丁。gojieba 官方说明依赖 CGo 与 C++；gse 是 Go 的 jieba 实现，但仍需加载或嵌入词典。两者可改善词语语义检索，却不能仅靠默认词典承诺所有单字、双字子串召回。本项目暂不加入，若文章量与搜索质量需求升级，再用真实查询样本评估词典和 n-gram 路线。[gojieba](https://github.com/yanyiwu/gojieba)、[gse](https://github.com/go-ego/gse)。

### 9.1 查询与索引约定

1. 发布阶段从已确认的文章正文提取纯文本，并建立对应 release 的搜索文档；标题、摘要、分类、标签与正文参与查询，taxonomy 名称与关联来自该 release 的确认快照。首版公开搜索只返回文章，About 等单页不作为文章结果；草稿、后台配置、凭证和未激活 release 不参与。公开搜索不能 join 当前可变 taxonomy，正文不在请求时重新解析 Markdown。
2. 输入校验 UTF-8，去掉首尾空白并统一基本大小写/空白规则。建议首版按空白分成少量字面检索项并以 AND 组合，不开放任意 FTS 查询语言；每项按 Unicode 字符数分别决定查询路径，不能按 UTF-8 字节数判断汉字长度。
3. 长度至少三字符的检索项作为转义后的 FTS 字面短语，通过参数化 `MATCH` 查询；一二字符项通过参数化 `LIKE` / 字面包含扫描过滤规范化纯文本，转义 `%`、`_` 与转义字符，使用户输入不变成通配规则。这条短词路径明确不宣称使用 trigram 索引。
4. trigram 保留支持长短语查询所需的 `detail=full`。不能为缩小索引改成 `detail=none`/`column` 后仍承诺支持任意长度的 `MATCH`；官方文档列出这些模式对三字符以上 token 的限制。
5. 标题完全匹配、标题包含、分类/标签命中优先于仅正文命中；长查询可辅助使用 FTS 排序，短查询提供稳定的时间/ID 次序。结果至少包含标题、固定文章链接、发表/更新时间和纯文本摘要；摘要经模板转义，不把索引返回片段直接当可信 HTML。
6. 发布时先构建/校验索引，再激活 release。每个搜索请求捕获一次 active release ID 并只查询该版本；回滚时页面和索引同时切换。保留旧版请求结束前不得清理其索引，索引可从内容修订重建，不是第二个内容权威。

### 9.2 短查询的代价与保护

一二汉字查询会扫描当前发布版文档，单字“的”等常见字也可能匹配大部分文章。首版按个人博客规模接受这个代价，并用小型原型验证，不能把“没有独立搜索服务”描述为“所有查询都是常数时间”。若到约 1,000 篇代表性文章时仍不满足预算，先评估发布阶段生成 unigram/bigram 倒排表或专门只读索引，不直接上向量库或集群。

默认使用提交式 HTML 表单，不在用户每输入一个字时向服务器自动发请求。建议技术预算为 1–64 个 Unicode 字符、有限检索项、每页约 20 条、查询超时、IP/连接限频及小量搜索并发；这些值在实现原型中可调。参数化 SQL、字面语法转义和预算缺一不可。查询超时要返回明确失败/重试状态，不能伪装成“没有结果”。搜索结果页设置合适的 `noindex`，避免按关键词生成大量可索引薄页；后台与搜索不能调用模型补答案。

短词保底验收示例：正文“这是一段简单的测试文本”，查询“段”“段简”“简单”“测试文本”均应命中；跨词边界“段简”是字面包含，不依赖词典。另测单字低频词、常见字、多检索项 AND、中文技术词、ASCII 名称、引号、`%`/`_`、空输入和超长输入；明确简繁互转、拼音、纠错与语义同义词不属于首版承诺。

### 9.3 必需页面与样式

首页提供文章列表和分页；每篇文章一个主分类、多个标签，平面分类页/标签页列出各自已发布文章。About 独立页面必须存在。文章显示原始发表时间和已发布更新时间，并由发布阶段生成稳定 heading ID 与目录链接；重复标题要去重 ID，目录使用普通 `#anchor` 链接，不需要 JS 才能跳转。固定页头存在时用 CSS `scroll-margin-top` 防止标题被遮挡。

原生CSS/系统字体负责基础布局；ChromaCSS、KaTeXCSS/fonts与图构建字体另固定hash并单列预算，不称整页只有系统字体/20KiB。CSS-only深浅色跟随系统，持久化切换另加小脚本。复制代码/图源码按钮和图集是P1；图说明及HTML展开源码免JS，后台脚本不加载到普通阅读页。

## 10. 建议验收预算与需用原型验证的事项

以下是**设计预算**，不是已测结果，应在真实内容与目标部署上执行验收。

| 类别 | 建议目标 / 方法 |
| --- | --- |
| 公共页面 | 禁用 JS 后首页、分类/标签页、About、正文、标题、目录锚点、导航、分页、发表/更新时间和文章链接完整可用；HTML 源码包含核心 SEO 数据；不要求 RSS/Atom |
| 浏览器资源 | 阅读可执行JS0KiB；公共基础CSS gzip20KiB预算，ChromaCSS、KaTeXCSS/fonts与SVG另计实际资源；有内容才加载、均自托管，不能称整页20KiB |
| 内容renderer | 公式/所有允许Mermaid类型静态编译，html/markdown/latex/tex源码高亮；code内不解析数学/图；rawHTML拒绝；错误源行列诊断、阻止发布并保留旧站 |
| 内容与发布 | 副本上传解析成功才形成新草稿；失败 / CAS 冲突不覆盖现有稿；发布仅包含 ChangeSet；相同幂等请求不重复生效；定稿仅改变允许的时间派生字段，内容 hash 固定并记录 finalArtifactHash |
| 任务恢复 | 强制终止进程再重启，待处理任务能恢复；外部推送失败可重试且有回执；不会丢失已授权发布或向未激活 URL 推送 |
| 一致性 | 模拟构建、切换、DB 记账三个位置崩溃；启动对账后 active release 与状态一致；并发访问不产生半成品页；内容、taxonomy 快照和搜索索引一同切换 |
| 中文搜索 | HTML 表单在无 JS 时可提交并分页；单字、双字、三字以上和跨词边界包含命中；草稿/未来 release 不出现；测试查询转义、超时和并发，不用空结果掩盖失败 |
| 后台与身份 | 同一个 master token 可登录后台及授权本机 stdio bridge 调用私有 control API；后台设置可保存/预览/确认；目录、分类标签、About、正文无富文本编辑器；浏览器预览只走 admin 会话路径 |
| 历史与回滚 | 每文章 / About 当前 + 3 旧版；回滚只改变目标内容、不覆盖副本 / 草稿；页面与搜索同步；下架为草稿，物理删除实际清理系统内可恢复内容，中断可继续 |
| 安全 | 包含 script、危险 URL、事件属性、目录穿越、超大内容和重复 slug 的样本被拒绝或安全渲染；草稿与密钥不进入公共目录 |
| 部署 | Windows Go开发/Linux生产验证；Go/Node/Puppeteer/Chromium/renderer/font固定；镜像、沙箱与字体兼容要实测，在线发布不能删辅助构建工具 |
| 备份 | 干净实例完整恢复业务数据、报告、记录与第三方凭据；隔离重建预览后切换，写入代次冲突时不覆盖；master 保留、会话失效、旧任务不复活；外链仅恢复引用 |
| 性能 | 约1000篇文章含公式/图/长代码，记录全量/改稿构建、镜像、Go/Chrome峰值与短词查询；无图不启浏览器，hash命中不重复编译；没有RAM/延迟实测承诺 |

下一步原型验证副本上传解析、bridge/API、后台预览确认、AST目录/代码/KaTeX/Mermaid、中文搜索/发布隔离、物理删除、崩溃恢复与完整备份；补测 AVIF/GIF、外链图片与视频平台适配。重点实测 Node/Chrome 峰值、中文字体、SVG/MathML 检查与布局；纯 Go 替代仅研究方向。每内容 current+3 已确认；渲染 / 查询预算属于实现验证，不是另请用户许可的前提。第三方播放器与统计的网络 / JS 依赖单列。

## 11. 研究记录

- 栈评估6次×5=30，中文搜索4次×5=20，本轮renderer4次×5=20，本代理累计 `sources_reviewed=70`。数字是返回槽位含重复/排除，不是独立已验证来源；本轮renderer20只记一次，根文档另统计自身5槽位。
- 新renderer依据KaTeX API/options/security/browser、Mermaid官方CLI/配置/Dockerfile、Chroma源码/Goldmark扩展；Mozilla SVG image-context说明补充浏览器边界，详见[渲染研究](./content-rendering.md)。
- 本轮主要一手来源：SQLite FTS5 的 Unicode61/Trigram 文档；Pagefind 中文/安装文档及维护者在官方 #987 讨论中的现版本说明；gojieba/gse 官方仓库的 CGo/词典要求。已区分 Pagefind 1.5.0 修复的查询分词与仍不支持的任意子词包含。
- 一手资料来自 Go、Hugo、Astro、templ、SQLite、相关 Go 库与 Pagefind 的官方文档/仓库；另外直接核查 Go 下载 JSON 和 Go module proxy 的当前版本元数据，固定 tag 的 `go.mod` 用于确认最低 Go 版本。
- 结论没有采用排行榜、性能营销文或第三方框架比较；这里的架构推荐是根据项目边界作出的判断。没有下载完整上游仓库，也没有安装或实现候选技术栈。
