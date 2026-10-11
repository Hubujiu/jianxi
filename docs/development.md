# 间隙：开发与运行

## GHCR 自动构建与更新

`.github/workflows/check.yml` 在推送 `main`、推送 `vX.Y.Z` 标签和手动运行时，执行资源构建、Go 测试与静态检查，构建 Linux/amd64 服务端和 MCP 镜像，并通过真实 MCP 上传、编译、发布及备份恢复的容器验收。全部通过后才使用仓库的 `GITHUB_TOKEN` 发布镜像，无需另配发布密钥。Pull Request 执行验收，不发布镜像。

- 服务端：`ghcr.io/hubujiu/jianxi:latest`。
- MCP：`ghcr.io/hubujiu/jianxi-mcp:latest`。
- 每次发布同时保留 `sha-完整提交哈希`；稳定版本标签额外发布去掉 `v` 的版本号，例如 `1.0.0`。推送 main 或发布稳定版本后更新 latest。

已有版本可从 Actions 页面手动运行，ref 填写 `v1.0.0`；也可以执行 `gh workflow run check.yml --repo Hubujiu/jianxi --ref main -f ref=v1.0.0`。这会使用当前工作流验证并发布已有标签对应的源码，不移动 Git 标签。

首次发布后，将两个 GHCR package 的可见性设为 Public，使部署电脑和服务器无需保存拉取凭据；镜像不包含站点数据、管理密钥或浏览器会话。Chromium 的无沙箱配置仅用于临时 CI runner 和 MCP 容器，公开服务器不运行 Chromium。

1Panel 编排使用 [deploy/1panel.compose.yaml](../deploy/1panel.compose.yaml)。它保持原 `jianxi-data` 数据卷、容器名、loopback 端口、512 MiB 内存和 2 CPU，镜像改为 GHCR，并设置 `pull_policy: always`。以后在 Actions 成功后，在 1Panel 更新该编排并勾选强制拉取镜像；服务器将拉取并启动新版。镜像发布本身不会自动重启服务器。更新前使用面板备份，回退时将 image 改为已验证的版本号或提交标签，继续使用同一数据卷。

Docker MCP 使用 [examples/mcp-docker-config.json](../examples/mcp-docker-config.json)，每次启动通过 `--pull always` 获取镜像；更新后重新连接 MCP。原生本地 MCP 仍需更新可执行程序，该工作流仅发布容器镜像。

更新：2026-10-11。需求基线仍为 `ai-native-blog/requirements.md`；本文件记录实际实现，不把本地验证等同于生产验收。

当前依赖清单及统一动画方案见 [当前依赖与动效决策](technical-decisions.md)。公开页面和后台共用 `web/static/motion.js`：GSAP 是唯一动画引擎，统一弹簧曲线、时长表和减少动效设置；CSS 保留布局及即时状态反馈。

## 已实现的代码

- `cmd/jianxi`：单服务、带本地编译器的 stdio MCP、初始化信息读取、主机密钥恢复和 Go 原生健康检查。
- `internal/blog/model.go`、`store.go`：SQLite 业务快照、FTS5 trigram、事务激活、进程级独占协调器锁。一个数据目录只允许一个服务进程。
- `workflow.go`、`publish.go`：副本、上传、解析、草稿 CAS、明确清单、预览 hash、确认、发布幂等、下架、历史回滚。首次发布时间由成功发布定稿；确认后不再编译正文、公式或图表。
- `compile.go`、`render_bundle.go`、`scripts/compile.mjs`：本地 MCP 生成 Shiki / KaTeX / Mermaid 片段；服务器用 Goldmark 校验源文、重建正文结构，并验证片段元素、属性、样式与内部资源引用。`image_metadata.go` 在 Go 中检查图片容器、类型与像素尺寸；Sharp 解码校验在本地 MCP 中执行。
- `http.go`、`render.go`、`web/`：静态博客、动态只读搜索、单密钥后台。所有私有预览和下载都在 `/admin/{随机安全码}/`；桥接 API 在 `/_control/`。
- `backup.go`、`delete.go`：完整敏感备份、隔离重建、数据代次冲突、确认恢复、删除重启清理。完整备份包含第三方凭据，不包含管理 token。
- `discovery.go`、`report.go`：百度持久推送队列、有限重试、标准 JSON/CSV SEO/GEO 报告。状态区分发布成功、平台接收和收录未知。

公开页面包括首页及分页、文章、分类和标签总览/列表、专栏及有序目录、时间线、关于我、搜索和真实 404。未配置内容时展示空状态，不自动生成生产文章。

正文中未包裹链接的图片（包括静态图表）支持点击或 Enter / 空格打开放大预览，背景模糊，图片下方显示名称：优先 Markdown 图片标题，其次图片描述，最后 URL 文件名。原生 dialog 管理焦点与背景隔离，GSAP 使用共享动效策略；点击图片、遮罩、关闭按钮或 Esc 退出，关闭后恢复焦点和滚动。支持手机、减少动效和站内页面切换清理，无新增第三方依赖。`output/playwright/jianxi-image-viewer-review.cjs` 在真实前台注入同源图片 fixture 验证以上行为，不发布或修改文章内容。

### 关键词描述语法

文章和关于页的 Markdown 支持 `[[关键词|描述]]`，例如：

```markdown
接口需要保证 [[幂等性|无论执行一次还是多次，产生的效果都相同。]]。
```

关键词显示虚线下划线，点击或 Enter / 空格打开附近的说明浮窗；支持关闭按钮、Esc、点击外部关闭，键盘关闭后焦点回到关键词。浮窗不锁定正文滚动，随关键词位置调整，在手机上限制宽高并允许长说明滚动；站内导航时清理。无 JavaScript 时通过原生 title 查看描述。

关键词与描述必须写在同一行且非空，关键词最多 80 字、描述最多 2000 字。内容按纯文本转义，不解析内部 Markdown、HTML 或公式；使用 `\|`、`\]`、`\\` 表示分隔符、右方括号和反斜杠。支持段落、标题、强调、列表、表格和脚注；不能嵌套在链接或图片内。代码、公式和转义后的标记保留原义。标题目录只显示关键词，搜索索引包含关键词及描述。编译与服务端校验共用 Go AST 扩展，不增加本地渲染任务；错误返回源文行号。渲染器版本为 `jianxi-5`，旧编译包需要本地 MCP 重新编译后预览发布。

表格遵循 GFM 的竖线转义规则，单元格中写成 `[[关键词\|描述]]`；描述里的其他竖线同样使用 `\|`。

验证：`keyword_test.go` 覆盖编译包一致性、文本转义、目录与索引、错误行号和与既有语法的兼容。`go test ./... -count=1`、`go vet ./...` 已通过；Playwright 使用真实编译结果在隔离本地服务验证桌面、390/320px 手机、横屏、触摸、键盘、关闭、长说明滚动、减少动效和实际站内导航，不修改现有文章或发布数据。浏览器检查脚本与截图位于 `output/playwright/keyword-*`。

`examples/rendering-showcase.md` 包含六处关键词描述（含强调和表格），`examples/rich-content.md` 包含四处；首次 `--demo` 导入的“做小一点的软件”文章包含检索、复杂度、导出三个术语。已有本地演示站可运行 `node scripts/update-demo-keywords.mjs` 查看更新范围，再用 `--apply` 通过真实 MCP 编译、提交、预览并激活这两篇示例。脚本只接受 loopback 地址，保留其他文章和不一致的未发布草稿，重复运行无变更。

### MCP 正文语法契约

写作前调用 `blog_content_rules`，其 `syntax` 数组返回 27 项语法的名称、中文说明、可复制示例和限制：标题与段落、行内样式、代码、列表、引用、分隔线、表格、链接、图片、脚注、关键词、转义、三种公式写法、八类 Mermaid 图和原始 HTML 的处理。`media` 补充图片资产、视频和来源元数据约定，`limits` 包含关键词长度限制；上传工具描述也明确引导先读取规则。`content_rules_test.go` 将全部示例合并后实际编译并校验编译包，`node scripts/check-mcp.mjs` 通过 stdio MCP 检查工具发现与规则返回，不只检查源码中的描述。

## 安装与启动

```powershell
npm ci
npm run assets
go run ./cmd/jianxi serve --data data/dev --demo --addr 127.0.0.1:8080
```

`--demo` 只用于空的开发站点，导入已确认原型的五篇演示文章和演示作者介绍，走实际上传至发布链路。演示首次发布时间是导入成功时间，不伪造原型日期。正式站点使用独立数据目录，不带 `--demo`。

首次运行在私有数据目录生成 `auth.json`（校验值）与 `initial-token.txt`（一次初始化交付文件）。保存密钥后自行移除 `initial-token.txt`。也可首次启动前在环境中提供不少于 32 字符的 `JIANXI_TOKEN`；已有 `auth.json` 后，旧环境值不会覆盖轮换结果。Windows 部署还应给数据目录设置仅服务账户可读的 ACL；Unix 创建文件采用 0600/0700。

后台地址 `/admin/{随机安全码}/`，完整路径保存在私有数据目录的 `admin-route.txt`，重启与管理密钥轮换后保持不变。安全码由密码学随机数生成；旧 `/admin/` 和错误安全码返回 404，正确路径仍须管理密钥验证。管理页面使用 same-origin 来源策略，同站表单保留来源校验，跨站请求不发送 Referer，公开页面不展示管理入口，robots.txt 仅封禁通用 `/admin/` 前缀，不包含安全码。发布或恢复不会暴露管理 token。轮换后须更新 MCP 客户端；丢失密钥时，在服务主机运行：

```powershell
bin/jianxi.exe token-reset --data data/dev
```

该命令有意向当前终端显示新生成的密钥。服务每次鉴权读取同一权威文件，旧 bearer 与会话立即失效。

## MCP

在发文电脑安装 Node 22 并执行 `npm ci`、`npm run assets`、构建 Go 程序后，使用 `examples/mcp-config.json`，替换实际可执行文件、`--renderer` 的编译脚本绝对路径、服务器 HTTPS 地址和私有 token。MCP 客户端不再只是 HTTP 转发器：公式、代码高亮、图表和Sharp 图片元数据检查都在此电脑执行。服务器不需要 Node、npm 或 Chromium。该文件只有占位符，不应将实际 token 提交到 Git。`JIANXI_URL` 只允许 loopback HTTP 或 HTTPS，不跟随可能泄漏凭据的重定向。

桥接使用[官方 Go MCP SDK](https://github.com/modelcontextprotocol/go-sdk)，本地验证协商协议 `2025-06-18`，完成 `initialize`、`tools/list` 和真实 `blog_status` 调用。MCP stdout 仅包含 JSON-RPC；本地 Node 编译进程的输出由 Go 捕获，不污染协议。也可构建 `docker build --target mcp -t jianxi-mcp:latest .`，在发文电脑通过 `examples/mcp-docker-config.json` 运行；该编译镜像包含 Node / Chromium，不部署在博客服务器。Docker MCP 读取本地备份时需额外挂载备份目录，并在工具参数中使用容器内路径。

推荐发文顺序：

1. `blog_content_rules`、`blog_status`、`blog_taxonomy_list` 读取约定、实例代次与当前公开版本。
2. `blog_working_copy` 获取副本；新建不传 `contentId`，关于我传 `kind=about`。
3. 修改返回的 `content`，必须明确填写文章 `slug`；关联分类填写 `category: {name, slug}`，标签填写 `tags: [{name, slug}]`。已有分类标签保留副本返回的稳定 ID 和路由元数据；新建时不填 ID。调用 `blog_upload_content`，指定 `copyId` 和唯一幂等键。本地固定编译器先完成全部编译，然后上传源文和内部编译包；服务端校验通过即返回 `status=parsed`。编译失败不会提交上传或覆盖旧草稿。完整示例见 [文章元数据](../examples/article-metadata.json)。
4. `blog_parse_upload` 传上传 `id`；对当前编译包执行校验，不启动浏览器。旧的源文上传由 MCP 在本地编译后补交片段；必须得到 `status=parsed`。
5. `blog_commit_draft` 携带上传返回的 `sourceHash`、`uploadId`、副本的 `expectedDraftRevision`、`epoch` 与幂等键。提交时从文章元数据创建新分类标签候选，与文章草稿原子保存；上传或解析失败不会创建分类标签。冲突必须重读合并。
6. `blog_changeset` 明确列出修订 ID、依赖的分类/标签 ID、专栏 ID，以及是否包含站点设置，绑定当前公开版本。
7. `blog_preview` 传清单 ID，展示私有预览地址、清单与 hash，等待用户批准。
8. `blog_confirm_preview` 绑定预览与清单 hash；`blog_publish` 携带确认 ID、清单 hash、预期公开版本与幂等键。

同一个 token 拥有全部管理能力；确认流程不能区分本人和 AI 身份。后台同样可查看并确认 MCP 草稿，没有文章编辑器。

`blog_collection` 管理有序专栏，至少两篇文章。分类/标签删除前必须通过内容修订迁移或解除当前关联。历史版本仍保留原关联；回滚引用已删除分类时，预览会报告依赖冲突。

文章、分类和标签的路由末段均由作者元数据的 `slug` 指定，仅允许小写字母、数字与连字符，最长 100 字符，分别生成 `/posts/{slug}/`、`/categories/{slug}/`、`/tags/{slug}/`。新文章副本没有默认 slug，缺失或非法时拒绝上传；不能只提交分类/标签 ID 而省略其路由元数据。新分类标签须在发布清单中明确选择，不随文章自动上线。已有分类标签改名或改路由仍用 `blog_taxonomy` 的 `expectedRevision`；普通文章上传不会隐式修改其他文章共用的 taxonomy。发布后旧地址维护 301。

演示分类使用 `technology`、`life`、`reading`、`thoughts`，标签使用 `ai`、`go`、`personal-blog` 等英文译名；路由直接写在 `web/demo.json` 的文章元数据中，导入器不再生成名称哈希。已有开发演示站可显式执行 `node scripts/migrate-demo-routes.mjs --data data/dev` 查看迁移清单，再加 `--apply` 通过正常发布流程迁移原始哈希分类/标签并保留 301；仅处理匹配原始演示名称和哈希的路由，保留自定义 slug、稳定 ID、文章及历史数据。默认读取初始化密钥文件；若已轮换，使用 `--token-file` 指向当前密钥文件。该脚本仅连接本机开发服务，不在生产启动时自动迁移。

完整备份可从后台导出，或使用 `blog_backup` 获取受保护下载地址。含当前编译包的新备份支持后台导入：服务器不信任缓存 HTML，而是从源文与验证后的片段重建。旧备份、缺少编译包的备份或升级编译器后的备份，使用 `blog_prepare_restore {path}` 在本地重编译并上传恢复候选；展示候选预览后，明确批准才能调用 `blog_restore_backup {id,artifactHash}`。报告支持 `blog_import_report` 和 `blog_import_report_csv`，模板在 `examples/`。

## 内容和媒体

`kind=about` 使用独立个人介绍页，居中显示头像、作者名称（`author`）和简介（`summary`），下方显示个人链接及可选的 Markdown 介绍，不显示文章分类、发布日期和目录。未发布介绍时仍使用站点作者与简介展示个人页。

关于我内容可配置 `avatarUrl`（HTTPS 或已上传的 `/media/{assetId}`）与 `socialLinks`（最多 12 个 `{ "name": "平台名称", "url": "https://…" }`）。未配置头像时显示名字头像，未配置社交链接时不显示社交账号。站内链接和本地头像适配私有预览前缀，头像资产随发布和备份保留。

`examples/rich-content.md` 提供公式、代码、图表、列表与表格样例，`examples/rendering-showcase.md` 提供完整排版展示。原始 HTML 按 AST 拒绝，放在 `html` 围栏内的示例则转义展示。行内代码和代码围栏不递归解析公式。代码围栏由服务端 Shiki 4.5.0（`github-light`）批量生成静态 HTML；浏览器不加载 Shiki/WASM，禁用 JS 仍显示高亮，未知语言回退纯文本。正文 H2 至 H6 生成确定、去重的中文目录锚点。

约束：正文 256 KiB；每篇最多 40 个公式/图表；单公式 8 KiB，展开上限 100；Mermaid 16 KiB、最多 150 边。编译子进程并发 1，Node 堆上限 256 MiB，35 秒超时；Mermaid CLI 内部 25 秒超时。Chrome 进程的整体内存由部署容器限制，不把 Node 堆限制当作完整浏览器内存限制。

Mermaid 使用固定配置和纯 SVG 文本，拒绝文章配置覆盖、回调、远程资源、HTML、自定义样式；不开放任意 TeX 文档或插件。编译错误保留诊断，不改变既有草稿或公开站点。

`blog_store_asset` 接收最多 10 MiB 的 PNG/JPEG/WebP/AVIF/GIF，检查实际格式与尺寸；最多 4000 万像素。`/media/{id}` 只有当前公开版本引用的资产可匿名读取，历史资产由鉴权路径读取。HTTPS 外链图片浏览器直连，不由后端抓取。

视频支持[哔哩哔哩官方外链播放器](https://player.bilibili.com/)的标准 BV 视频地址，默认不自动播放；其他平台保留普通链接。私有预览一律只显示链接，不加载第三方播放器。不托管视频文件。

## 动效

- 公开站点的普通站内点击保留侧栏、顶栏和搜索表面，立即请求下一份完整 HTML 并更新正文、面包屑及页面元数据。正文始终可见，只给新标题 6px 的轻微位移；不再重新绘制侧栏或执行透明度/裁剪入场。侧栏已移除重复的“全部文章”入口。
- 侧栏使用同一个高亮块在主导航和具体分类之间移动，约 480ms，采用与侧栏展开、搜索面板一致的阻尼弹簧曲线，带轻微越过终点后的回弹；连续点击从当前渲染位置接续，同一目标的页面到达不会重启动画。取消旧请求，只有最新目标可以替换正文。首屏绘制前读取已保存的侧栏宽度和收起状态。
- 桌面预留固定滚动条空间，长短页面切换不再横向移动正文和顶栏。
- 所有滚动区域统一使用透明轨道、灰色细圆角滑块，不显示原生箭头；横向和纵向均为 14px 轨道、4px 透明内边框。主内容、侧栏、文章目录、代码块、表格、公式、时间线、搜索面板和后台共用 `site.css` 规则及 `scrollbars.js` 的捕获阶段滚动监听，动态加载的正文无需重复绑定。各区域独立在滚动时显示、停止 1.2 秒后隐藏；无 JS 时保留可见滑块，Firefox 使用标准 scrollbar 属性。侧栏和目录仍预留滚动条空间。`output/playwright/unified-scrollbars-review.cjs` 验证统一外观、两种方向、独立显隐、稳定宽度、手机页面 / 目录和无 JS 回退。
- 每次替换正文清理旧目录观察器和动画，重新绑定目录、复制按钮及时间线筛选。前进后退恢复阅读位置和入口面包屑；跨页锚点及公式样式随页面加载。历史返回、正文锚点和减少动效模式直接显示内容，键盘聚焦会立即完成标题动画。
- 禁用 JavaScript、修改键点击、新窗口、后台/私有路径保持原生导航；站内请求失败回退到原生导航。`output/playwright/jianxi-navigation-review.cjs` 验证侧栏/高亮/搜索 DOM 持续存在、中间帧、正文不透明、连续请求、历史位置、搜索表单、手机导航/目录、减少动效及无 JS 阅读。公式样式加载用真实文章响应添加样式链接的浏览器 fixture 验证，未发布测试内容。

仅实际界面交互，没有循环演示、模拟游标或自动镜头运动。

- 搜索入口与展开面板使用同一持续存在的 DOM 表面；尺寸、圆角和颜色连续变化，内容短暂模糊淡换，输入不会因开关而重建。支持 `Ctrl/⌘ K`、`/`、Escape、焦点约束和真实搜索。
- 手机文章目录从同一个圆形控件展开为面板，关闭回到原控件；导航抽屉支持背景遮罩、键盘与滚动锁。
- 桌面侧栏支持真实指针拖曳调宽、方向键调宽、收起和跨页面记忆。
- 发布、恢复和密钥操作从原按钮所在表面展开确认面板；取消后回归原位。
- 使用近临界弹簧曲线，微小超调；不使用 bounce、粒子、光晕或渐层边框。运行中切换减少动效设置会直接落到业务终态。
- 时间线热力图的全部日期格子（含 0 篇）共享 GSAP 悬停 / 键盘聚焦反馈，轻微放大至 1.08 倍、移开恢复。滚动容器四周保留 8px，保证边缘格子的缩放、边框和焦点轮廓不裁切。减少动效时关闭缩放，触屏点击不保留悬停缩放；页面切换清理事件与动画。`output/playwright/jianxi-heat-hover-review.cjs` 验证桌面 / 手机、滚动边缘、空日期筛选、键盘、减少动效、触屏与导航后重新绑定。
- 前台与后台统一使用自托管霞鹜文楷，WOFF2 字形分片通过 `unicode-range` 按需并行加载，只有一个常用分片预加载；内容哈希 URL 享有一年不可变缓存，下载期间显示系统字体。代码 / 公式保留专用字体。完整策略见 [技术决策](technical-decisions.md#霞鹜文楷加载策略)。黑白组件，浅暖灰画布，无运行时 CDN 依赖。

## 保留、删除与恢复

每份内容保留当前及 3 个历史已发布修订。已消费的上传与副本在其修订发布后回收；未消费副本保留最多 30 天，在后续发布清理。当前草稿不被回滚覆盖。过期正文不通过幂等缓存继续保留；已经回收的上传句柄重试会返回过期诊断。旧确认在公开版本变化时失效，旧静态制品会清理。

永久删除必须先预览确认下架，再明确确认稳定内容 ID；专栏依赖需先解除。删除清理目标全部修订、副本、上传和独占资产，同时清理系统管理的旧预览、旧 release 与包含该内容的备份（保留无关备份），防止从旧整站副本恢复目标。共享资产按其他内容引用保留。SQLite 启用 secure_delete，并执行 WAL 截断与 VACUUM；不是取证级介质擦除。清理中的持久标记使重启继续任务。外部已下载备份、第三方缓存与外链媒体不受本系统控制。

完整 ZIP 含清单和 hash、规范源文、业务数据、资产与第三方凭据。导入最多压缩 100 MiB、展开 256 MiB、20000 文件，拒绝路径穿越、哈希不匹配和版本不兼容；不信任备份里的缓存 HTML，所有有效修订从源文与验证后的编译片段重新装配；旧版本备份先经本地 MCP 重编译。导入快照、修订、路由与资源关系校验后才产生私有候选。

确认恢复检查全站数据代次，期间任何业务写入会使候选冲突。先保存恢复前完整备份，再激活新业务数据与索引。管理 token 不变，旧会话和确认失效；历史任务作为记录恢复，不自动重放网络推送。

## 验证与当前限制

已执行 Go 单元/集成测试和 `go vet`，涵盖发布隔离、CAS、预览失效、幂等、短中文/标点搜索、历史裁剪、下架、回滚保留草稿、备份恢复与并发冲突、认证与 CSRF、恶意 Markdown、静态公式/图表、图片五种格式、协调器独占、永久删除清理和制品篡改检测。

实施时重新核查并升级了渲染依赖：Mermaid CLI 12.0.0 / Mermaid 12.1.0、Puppeteer 25.12.0、KaTeX 0.19.0、Sharp 0.35.5；具体树锁定在 `package-lock.json`。全树 KaTeX 使用同一固定版本，避免旧嵌套依赖。`npm audit --omit=dev --registry=https://registry.npmjs.org` 当前报告 0 项漏洞。修复依据包括 [Sharp 上游安全公告](https://github.com/advisories/GHSA-f88m-g3jw-g9cj)；更新后重新执行富内容与五种图片格式回归。

基础 CSS gzip 约 7.3 KiB，公开界面增强 JS（含导航、统一动效、页面加载序列、搜索筛选和首屏侧栏偏好）gzip 合计约 14.8 KiB，GSAP gzip 约 27.6 KiB；KaTeX 样式/字体另计。上述为本地 gzip 测量，实际压缩传输需反向代理开启。

通过 Playwright 在 1440、768、390、320px 验证实际点击、拖曳、连续反向开关、节点/输入保留、导航与目录、减少动效、无 JavaScript 阅读/搜索；后台所有页面与确认取消也已检查。脚本位于 `output/playwright/jianxi-app-review.cjs` 和 `jianxi-admin-review.cjs`，截图使用 `jianxi-app-` 前缀。没有采样完整帧时，不宣称恒定 60fps。

早期 `output/playwright/jianxi-publish-review.cjs` 还通过实际浏览器表单验证了配置草稿 → 明确清单 → 私有 iframe 预览 → 同一按钮表面变形确认 → 静态版本激活的完整流程；保留原有五篇演示文章，没有夹带正文修改。

仍须部署环境验证：

- 2026-10-09 已完成编译器迁移：本地 MCP 处理 Shiki / KaTeX / Mermaid / Sharp，服务器镜像基于 scratch，仅有静态 Go 程序、CA 和私有数据/临时目录。已验证真实 stdio MCP 初始化、工具 schema、本地编译上传、草稿、预览确认发布、富内容静态页面、全部图表类型、五种图片格式、修订缓存重编译、离线备份重建、无 Node 服务端恢复、重启持久化、初始化密钥清理和 Go 健康检查。`scripts/docker-smoke.mjs` 只使用自己创建的容器和临时目录，结束时清理这些测试数据。
- 真正的域名、HTTPS、百度站点验证和 token 尚未提供；未向真实搜索平台提交 URL，也未验证访问统计账号。推送前检查主域名是否返回当前 release 标识，私网请求被拒绝。
- 100/1000/5000 篇负载、构建峰值、长时间运行与故障注入尚未完成。当前业务状态是 SQLite 中的事务 JSON 快照；大站性能需要实测，不能等同于已经通过生产规模验收。
- 第三方分析查询 API、分类批量合并和媒体管理的完整后台界面尚未提供。迁移支持后台当前格式备份导入和 MCP 离线重编译恢复，分类迁移经明确内容修订，分析用标准报告导入。

## 容器部署

2026-10-11 已使用 computer use（鼠标、键盘、截图）在 1Panel 更新 `blog.hubujiu.com`：镜像为 `jianxi:20261011-keywords`，编排与容器仍为 `jianxi`，数据卷仍为 `jianxi-data`，保留原有 loopback 8080 端口、512 MiB 内存、2 CPU 和 HTTPS 代理。更新前通过 1Panel 创建包含挂载数据的编排备份，并启用停机备份以保证一致性；日志确认备份和恢复启动成功。原 `jianxi:latest` 镜像保留用于回退。新版容器 healthy，公网首页可访问，`/healthz` 返回 `ok`，公网 `static/site.js` 已包含关键词浮窗实现。远程站点原有 0 篇文章，本次只更新服务程序。

本次更新包是 `output/docker/jianxi-keywords-20261011-linux-amd64.tar.gz`（SHA-256：`9a1954f1b5a879f11d3d004443bdc422db0a87d0184f7b28a3392f259bee8c99`），对应编排位于 `output/deployment-1panel/compose-keywords.yaml`。本机 Docker Desktop 启动报错，因此使用 Go 交叉编译静态 Linux/amd64 程序，按 Docker save 格式替换原 scratch 镜像的程序层，保留已验证的证书层、目录层和运行配置；打包脚本校验 ELF 架构、全部层摘要和程序内容，1Panel 导入及运行验证成功。旧的 `jianxi-linux-amd64.tar.gz` 和 Docker MCP 包仍是 2026-10-09 版本，不能作为本轮更新包使用。原镜像回退需在 1Panel 将该编排的 image 改回 `jianxi:latest`，保留同一数据卷和升级前备份。

```sh
docker compose up --build -d
docker compose ps
```

默认端口仅映射到本机，供 HTTPS 反向代理转发。数据卷包含数据库、认证配置、媒体和备份。服务端容器采用 UID/GID 10001:10001，设置 1 GiB 内存和 2 CPU 上限；无需 Chromium 或专用共享内存设置。MCP 编译镜像在发文电脑运行，保留独立的 Chromium 内存和 256 MiB `/dev/shm` 预算。

### 将已构建镜像搬到服务器

已导出的部署包位于 `output/docker/jianxi-linux-amd64.tar.gz`，同目录提供 `compose.yaml` 与 SHA256 校验文件。适用于 x86_64 Linux 服务器；ARM64 需要另外构建对应架构。镜像只包含程序和依赖，不包含当前开发站的文章、数据或密钥。迁移后 Docker 显示服务端镜像约 47 MB（此前同机编译精简版约 1.69 GB）。服务端压缩包约 17 MiB（原同机编译包约 419 MiB）；另有 `output/docker/jianxi-mcp-linux-amd64.tar.gz`，约 419 MiB，仅供发文电脑运行 Docker MCP。服务器另需预留文章、图片和备份空间。

在本地 Docker Desktop Linux 引擎中，以 2 CPU、256 MiB 内存、禁用 swap 的服务端容器验证五篇演示内容、完整渲染展示文章、五种图片、MCP 离线恢复和重启：空闲与重启后约 13–14 MiB，未出现服务进程 OOM。此值是对应时点的 Docker stats 内存，并非峰值。发文电脑的编译器仍需单独的 Node / Chromium 资源。宿主机本身不是 2 GiB 服务器；尚未测大站、大图、批量恢复与高并发，默认继续保留 1 GiB 服务端上限。
将镜像包和 `compose.yaml` 放到服务器的同一目录，在该目录执行：

```sh
docker load -i jianxi-linux-amd64.tar.gz
docker compose up -d --no-build --pull never
docker compose ps
curl -f http://127.0.0.1:8080/healthz
```

首次启动创建空站；通过管理后台或 MCP 发布内容。需要迁移旧开发站内容时，先导出完整备份，再经本地 MCP 的 `blog_prepare_restore` 重编译，预览确认后恢复。旧站升级不会在服务器启动时编译旧正文；原发布页继续可读。`blog_status.recompileNeeded` 指出需要升级编译包的修订数量，`blog_recompile_revision` 可更新指定修订缓存，随后通过正常清单、预览和发布上线，保留原修订 ID、源文与发布时间。

在服务器私有终端读取后台路径和初始化密钥：

```sh
docker compose exec -T jianxi /jianxi init-info --data /data
```

妥善保存密钥后移除初始化交付文件：`docker compose exec -T jianxi /jianxi init-info --data /data --clear-initial-token`。`docker compose down` 保留数据卷；`docker compose down -v` 会删除站点数据。一个数据卷只运行一个服务实例。

公开访问前，将服务器上的 HTTPS 反向代理转发至 `127.0.0.1:8080`，并在后台设置真实站点地址。默认 Compose 的本机端口绑定不能从其他机器直接访问；尚未验证真实域名、搜索平台及生产负载。

重新构建和容器回归：

```sh
docker build -t jianxi:latest .
go build -o bin/jianxi ./cmd/jianxi
node scripts/docker-smoke.mjs
```

可通过 `JIANXI_TEST_IMAGE` 选择测试镜像、`JIANXI_TEST_MEMORY` 设置测试内存；脚本将 swap 上限设置为同一值以禁止 swap。PowerShell 中验证低内存预算：

```powershell
$env:JIANXI_TEST_MEMORY = '256m'
node scripts/docker-smoke.mjs
```

还可设置 `JIANXI_TEST_MCP_IMAGE=jianxi-mcp:latest`，让验收使用发文电脑上的独立 Docker MCP 进程，而不是本机 `bin/jianxi`。该模式为测试临时共享容器网络，实际远程发文使用 HTTPS。服务端镜像没有 shell、cat 或 rm；运维使用 `/jianxi init-info`、`/jianxi token-reset`，或 `docker compose cp` 复制私有文件。
