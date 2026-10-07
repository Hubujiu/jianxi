# MCP 与 Agent 发布接口研究

研究日期：2026-10-07（Asia/Shanghai）。本文针对 Go 后端、公开静态博客、无编辑器/插件/主题系统的方案；协议事实与下文的应用设计建议分开标明。用户最新确认：**仅中国中文站，单主分类、多标签、无层级；单站、单机或单容器；本机 Codex 或 Claude Desktop 经 stdio 接入；文章、About 介绍、配图和分类标签全部由 AI 经 MCP 上传；后台设置站点配置/服务凭据，内容只查看、预览、确认和回滚，不提供编辑；备份导出/导入必需；不提供 RSS/Atom；只有一个 master token，持有它拥有全部控制，没有账号、角色、分权限或 OAuth 系统。**

## 结论与首版取舍

本轮确认更新：每篇文章 / About 当前 + 3 个已发布旧版；AI 只修改独立副本，上传并解析成功后提交新草稿；发布按明确 ChangeSet；撤回分回滚和下架为草稿，永久删除为物理清理；图片含 AVIF / GIF 并支持外链，外部视频 URL 由明确适配器嵌入、不托管视频；整站完整备份恢复全部业务数据及第三方凭据，保留当前管理 token、不复活会话或任务。以下契约按此执行。

MCP 成为所有内容的操作入口：上传/修改文章和 About、管理分类标签、上传配图、搜索、保存草稿、预览、发布、回滚和备份恢复。后台只设置站点配置与服务凭据，并查看/预览/确认/回滚内容。公开站用普通 HTTP 提供完整 HTML 和 sitemap，不生成 RSS/Atom；MCP 不是搜索引擎或生成式搜索自动发现网页的入口，也不保证网站被 AI 回答引用。唯一的 Go 服务承担内容状态、静态文件服务和后台任务；同一个 Go 二进制另提供本地 stdio 桥接子命令，不需要独立部署 MCP 微服务。[MCP 架构](https://modelcontextprotocol.io/specification/2026-07-28/server)、[SDK 服务器入口](https://github.com/modelcontextprotocol/go-sdk/blob/v1.8.0/README.md)

首版建议采用官方 Go SDK，保留协议兼容层，工具围绕设置、分类标签、文章、媒体、搜索及发布流程组织。无需绑定模型供应商、内置聊天框或自行实现 Agent；外部客户端负责模型和写作，Go 服务负责确定性的内容校验与发布。以后需要定时自主研究、检索、写作时，再增加可选的 Go 编排模块。

首版正式选择 **stdio**，客户端为用户本机 Codex 或 Claude Desktop；后台和本地桥接共同使用唯一 master token，不建设 OAuth。之前确认的“AI 写草稿，你确认后发布”保留为工作流程，但同 token 不能在鉴权上区分人和 AI，也不能强制 AI 只有草稿权限。系统能记录针对明确内容版本的确认步骤，不能把该记录描述为第二身份验证。

AI 上传的内容包括 Markdown、LaTeX 公式、HTML/Markdown 等语言的代码示例和受限 Mermaid 图；它们统一经程序编译成静态展示。AI 不直接提交文章 rendered HTML、模板、CSS、JavaScript 或公开文件输出路径。单 master token 赋予所有合法后台操作，内容格式与编译边界是应用输入规则，不是角色或 scope 限权。

本机客户端不等于生产 Go 服务也在本机。stdio 子进程只作为桥接，调用唯一 Go 服务的私有控制面，建议路径 `/_control/`：同机可用 loopback 或 Unix socket；远程部署时可经 SSH tunnel，或受保护 HTTPS 应用 API。这个内部应用 API 校验唯一 master token，不是公开 HTTP MCP `/mcp` endpoint，也不冒充 MCP OAuth。桥接进程不独立维护数据库、发布队列或 active release pointer，所有发布和回滚由唯一服务执行，避免与 daemon 竞争。

## 当前版本与 Go 选型证据

| 项目 | 已核查事实 | 本项目建议 |
| --- | --- | --- |
| 稳定 MCP 规范 | 官网 `specification/latest` 对应 **2026-07-28** | 以该版本设计，兼容目标客户端需要的旧版本 |
| 官方 Go SDK | 官方 `releases/latest` 直接重定向 **v1.8.0**；发布于 2026-09-14 08:05:46 UTC | 固定 `github.com/modelcontextprotocol/go-sdk v1.8.0`，实施时再核验安全修订 |
| SDK 支持协议 | v1.8.0 没有新增协议；最新仍为 2026-07-28，保留旧协议兼容 | 首版使用 stdio；只有以后启用 HTTP MCP 时才涉及 `Stateless: true` |
| SDK 的 Go 最低版本 | v1.8.0 的 `go.mod` 声明 `go 1.25.0` | 最终项目的最低版本由所有依赖共同决定，不能从 SDK 的 1.25 推断整个项目可用 1.25；工具链详见技术栈方案 |
| 许可证 | 新代码 Apache-2.0，尚未完成重许可的既有代码保留 MIT；不是整仓纯 MIT | 发布时保留对应版权与许可文件 |
| cgo | 固定 tag 源码包检查 147 个 `.go` 文件，无 `import "C"`，无 `.c/.h` 文件 | SDK 自身不引入 cgo；整站能否 `CGO_ENABLED=0` 仍须结合 SQLite、图片处理等依赖实际构建验证 |
| 授权能力 | `auth`/`oauthex` 提供 bearer 校验、资源 metadata 与 OAuth 原语 | 本项目仅使用应用的 master token 校验，不引入 OAuth 或 scope 系统 |

来源：[当前规范](https://modelcontextprotocol.io/specification/latest)、[最新发布入口](https://github.com/modelcontextprotocol/go-sdk/releases/latest)、[v1.8.0 发布说明](https://github.com/modelcontextprotocol/go-sdk/releases/tag/v1.8.0)、[固定版本 go.mod](https://github.com/modelcontextprotocol/go-sdk/blob/v1.8.0/go.mod)、[固定版本许可](https://github.com/modelcontextprotocol/go-sdk/blob/v1.8.0/LICENSE)、[固定版本 auth 实现](https://github.com/modelcontextprotocol/go-sdk/blob/v1.8.0/auth/auth.go)。

搜索索引曾返回 v1.7.0 为最新；本次使用官方最新发布重定向、v1.8.0 release 页面及固定 tag 源码交叉确认，避免把搜索摘要当版本依据。没有构建整个博客，也没有对所有传递依赖承诺零 cgo。

## 传输：新旧规范必须区分

| 模式 | 用途与要求 | 首版选择 |
| --- | --- | --- |
| `stdio` | 客户端启动本地子进程，stdin/stdout 传 JSON-RPC；日志写 stderr。HTTP OAuth 规范不用于此模式，凭据通过环境或本地凭据管理传入 | **首版正式选择**；同一 Go 二进制提供桥接子命令，状态与激活由唯一 Go 服务管理 |
| Streamable HTTP，2026-07-28 | 单 MCP endpoint 接收 POST；返回 JSON 或该请求的 SSE 流。每次请求携带协议版本、客户端能力等 `_meta`，HTTP 还携带规定的镜像 headers；正文和 header 不一致须拒绝 | 协议参考，不在当前需求内；本项目的私有 HTTPS 应用 API 不等于该传输 |
| Streamable HTTP，2025-11-25 及更早 | 旧形态含 `initialize`、可选协议 session、独立 GET SSE、DELETE session；不能拿这些规则解释新版本 | SDK兼容实际需要的客户端版本，记录互通结果 |
| 2024-11-05 HTTP+SSE | 旧独立 SSE/POST transport 已弃用 | 首版不新增该传输，确有旧客户端需求再判断 |

弃用旧 HTTP+SSE **不表示取消 SSE 或流式响应**。2026-07-28 仍允许 POST 的响应使用 SSE；删除的是旧独立 GET 流、协议级 session 和流重放等机制。新协议用 `server/discover` 发现服务信息，以逐请求 metadata 取代初始化握手；SDK提供对旧客户端回退的实现，但应用需配置所支持的版本，并用目标客户端实测，不能假定所有应用都支持新协议。业务上跨调用的数据必须用明确的 `postId`、`revision`、`previewId`、`releaseId` 表达，不能寄托在 MCP 会话上。[HTTP 传输规范](https://modelcontextprotocol.io/specification/2026-07-28/basic/transports/streamable-http)、[版本兼容规范](https://modelcontextprotocol.io/specification/2026-07-28/basic/versioning)、[v1.7.0 协议迁移说明](https://github.com/modelcontextprotocol/go-sdk/releases/tag/v1.7.0)

新规范还引入 MRTR 和统一订阅流，roots/sampling/logging 已弃用。博客首版使用普通请求与结果查询即可，不依赖这些高级机制，也不把 MCP sampling 作为内置模型调用的基础。[当前变更记录](https://modelcontextprotocol.io/specification/2026-07-28/changelog)

## 单 master token 的后台和控制面

### 一个凭据、完整权限

服务端只认一个 master token。持有者可操作设置、分类标签、草稿、已发布文章、媒体、构建、发布、回滚、分发和其他后台控制；不建立账号、owner/Agent 两套身份、角色、scope 或每客户端密钥。Codex 与 Claude Desktop 共享同一个 token，日志可以记声明的客户端名，但它不是可信人员身份或权限依据。

公开 HTTP 仅输出已发布内容，私有控制 API、后台数据、预览、MCP resources 和管理工具都检查同一个 token 或由该 token 建立的有效浏览器会话。分类标签、草稿和设置不会因工具名可猜测而公开。内容状态及接口校验是业务规则，不是分权限系统。[工具与资源规范](https://modelcontextprotocol.io/specification/2026-07-28/server/tools)、[资源安全要求](https://modelcontextprotocol.io/specification/2026-07-28/server/resources)

### 后台登录：token 表单换浏览器会话

后台提供一个输入 master token 的登录表单，不设用户名、注册、用户资料或角色管理。POST 登录后用恒定时间方式校验高熵随机 token，创建短期随机 session ID，浏览器只保存 `HttpOnly`、`SameSite=Lax`、`Path=/admin` cookie；正式 HTTPS 部署同时设 `Secure`。浏览器预览及其私有 assets 统一置于 `/admin/previews/...`，使后台 cookie 覆盖所需请求；不把带私有内容的预览发布到公开资源路径。服务端记录会话期限和当前 token generation，退出或轮换 token 时失效。session ID 是服务端自动维护的登录状态，不是要求用户管理的第二把密钥。[浏览器 cookie 语义](https://developer.mozilla.org/en-US/docs/Web/HTTP/Guides/Cookies)、[Go 标准库 Cookie](https://pkg.go.dev/net/http#Cookie)

master token 不放 URL、前端 JavaScript/localStorage、公开 HTML 或日志。后台所有修改使用 POST 等写方法，校验 Origin，并使用会话内 CSRF nonce；这些请求防护不增加账号或第二身份。登录表单本身限制重试，错误响应不回显 token。后台只有设置、内容查看、预览确认、回滚和备份等轻量页面；文章/About/媒体/分类标签的写入入口在 MCP，不提供后台内容编辑表单。

### stdio 桥接与私有应用 API

本地桥接从客户端环境或受保护本地配置读取同一个 master token，每个 `/_control/` 私有应用 API 请求发送 `Authorization: Bearer <master-token>`；该路径不是公网 `/mcp` HTTP endpoint。桥接 stdout 仅输出 MCP JSON-RPC，日志和诊断写 stderr且脱敏；token 不作为普通 tool 参数或 resource 返回。后台表单 cookie 和桥接 header 都兑换同一完整控制权限。客户端的本地 token 副本只用于发送请求，不成为服务端第二个权威来源；后台轮换后需要更新该副本。

服务与客户端同机时使用 loopback/Unix socket；不同机时使用 SSH tunnel 或受保护 HTTPS。私有控制 API 只校验应用 master token，不是远程 HTTP MCP。stdio 规范不要求套用 HTTP OAuth；MCP 也允许双方协商自定义认证，但固定 token 不能宣称实现了标准 OAuth 互通。本项目按已确定的 stdio 客户端验收，不承诺直接接入只支持 OAuth 的托管 MCP 平台。[stdio/传输规范](https://modelcontextprotocol.io/specification/2026-07-28/basic/transports)、[授权边界](https://modelcontextprotocol.io/specification/2026-07-28/basic)

若以后添加真正的 HTTP MCP，必须重新核对传输与目标客户端要求。现有规范要求 Origin 校验；SDK v1.8.0 的 `CrossOriginProtection == nil` 不自动保护。这里只保留协议事实，不将 OAuth 身份体系加回当前产品需求。[HTTP 安全规范](https://modelcontextprotocol.io/specification/2026-07-28/basic/transports/streamable-http)、[固定 tag handler](https://github.com/modelcontextprotocol/go-sdk/blob/v1.8.0/mcp/streamable.go)

### token 轮换、丢失恢复和设置脱敏

master token 采用随机高熵值。**唯一运行权威**是受保护私有 auth 配置文件中的 token verifier/digest 与 generation；该文件独立于内容数据库。部署环境 token 只用于首次初始化该文件，文件存在后重启也不能拿旧环境值覆盖后台已轮换的 token。配置文件须持久化到单机/容器私有 volume，并通过原子更新维护一致性。

后台可轮换 master token，所有持 token 的操作者均拥有这一控制权限。轮换写入同一个 auth 文件的新 digest/generation，服务载入新状态并撤销全部浏览器 sessions；旧 bearer token 随即失效，再更新 Codex/Claude Desktop 桥接的客户端副本。新值只能在生成/设置环节交付，不通过普通设置读取、resource 或日志回显；不保留第二把长期密钥，也不建设双 token 并行轮换。CLI 生成/轮换操作也修改同一个 auth 文件，不要把 token 放普通命令行参数或 shell 历史。

丢失 token 时，通过已有主机/容器管理能力使用 CLI 生成新值，替换同一 auth 文件并 reload/restart、更新客户端；不依赖旧 token，也不生成第二个恢复密钥。没有主机管理访问时，产品不能凭空恢复控制权。auth 文件不随内容数据库导出/导入；恢复内容备份不会把旧 token/generation 带回，主机访问属于部署环境而不是新增博客账号。

设置读取返回普通字段、配置 revision，以及敏感项 `configured=true/false` 或脱敏占位；主 token 永不通过设置页面/API/MCP 读取。国内推送服务等外部 secret 允许持 token 的调用者替换/清空，但读响应不返回原值；省略字段表示保持原值，显式清空单独表达，避免把 `******` 写回真实 secret。secret 脱敏是输出策略，不是给 master token 降权。

同 token 的后台设置也只接受明确 schema：站点配置、服务凭据和程序预设渲染选项。没有任意 `headHTML`、template、CSS、JS、执行插件、服务器文件路径或 shell 字段；修改合法设置不会成为上传可执行页面的旁路。影响公开渲染的设置进入新候选 manifest，经过统一预览发布流程。

## MCP 工具范围

以下名称均为本项目自定义工具，不是 MCP 内置工具。首版没有匿名管理 MCP；公开读取走普通站点与 sitemap，不提供 RSS/Atom。后台设置与资源读取、草稿和发布控制全部共用同一 master token。`readOnlyHint`、`destructiveHint`、`idempotentHint` 等 annotations 只是客户端提示，不产生权限，也不会判断是谁调用。[工具规则](https://modelcontextprotocol.io/specification/2026-07-28/server/tools)

| 自定义工具 | 目的与主要参数 | 结果/边界 |
| --- | --- | --- |
| `blog_settings_get` / `blog_settings_update` | 读取/修改 schema 列出的站点名、域名、SEO、导航、首页、国内分发配置等；修改带 `expectedRevision` | 同 token 全部合法设置权限；敏感值不回显，domain/程序预设渲染配置影响候选manifest；无任意模板/脚本字段 |
| `blog_taxonomy` | `kind=category/tag`，`operation=list/upsert/delete/merge`；ID、中文名称/描述、slug、排序、revision、view | 单主分类与多标签，无 parent/层级；公开改动先存为候选，预览列受影响URL，再确认新release；不静默删文章关系或直接污染线上快照 |
| `blog_search` | 中文 query、分类/标签、`view=draft/published/all`、游标、limit | 查找文章、媒体或内容记录；返回 IDs、标题、摘要、状态、revision，搜索结果不得混入公开 HTML 的草稿 |
| `blog_get` | 获取 `contentId` 的指定 view/revision，支持 article/About | 内容、分类标签、引用、revision、sourceHash；凭 master token可读取仍保留的草稿和发布版本；已裁剪正文返回 VERSION_NOT_AVAILABLE |
| `blog_working_copy` | 从指定内容修订获取 / 修改独立副本；`copyId`、`baseRevision`、副本修订 CAS | 不修改现有草稿或已发布修订；回滚不会覆盖副本 |
| `blog_upload_content` | 上传副本的 strict Markdown 与 typed metadata，带基准修订、幂等键；About 使用固定 key | 返回私有 uploadId / sourceHash，尚未成为新草稿；不接收 renderedHTML/template/CSS/JS/outputPath |
| `blog_parse_upload` | 解析指定 uploadId / sourceHash，返回任务 ID、状态和位置化诊断 | AST / 公式 / 图语法、引用及大小校验；失败保留待修复上传，现有草稿不变 |
| `blog_commit_draft` | 将解析通过的上传提交为新草稿，带 copy/upload hash、基准内容修订、expectedDraftRevision、实例代次与幂等键 | CAS 冲突不覆盖现有稿；返回新 revision/sourceHash；成功提交仍不代表预览渲染已通过 |
| `blog_changeset` | 明确选择内容修订、taxonomy、公开配置与资源变更 | 校验依赖，返回 changeSetId/hash 与受影响 URL；不能自动包含其他未选修改 |
| `blog_preview` | 统一编译指定 source revision/hash 为候选静态站 | 位置化诊断、`previewId`、私有预览、候选release、artifactHash、renderer/配置/字体资源hash，以及受影响URL；失败不激活 |
| `blog_confirm_preview` | 为指定 `previewId` 与 hashes 建立确认记录；client/workflow 来源可声明 | 返回 `confirmationId`；这是确认步骤记录，不能证明调用者是人，仍用同一个 token |
| `blog_publish` | 带 confirmationId、previewId、changeSetHash、相关修订 / source hashes、expectedActiveReleaseId、实例代次与 idempotencyKey | CAS / hash 校验后只执行已绑定的时间定稿规则，返回 release/job ID 和 finalArtifactHash；不重编译已确认内容 |
| `blog_store_asset` | 上传 PNG/JPEG/WebP/AVIF/GIF；媒体类型、大小、内容 hash、文件名/替代文本 | 返回资产 ID，桥接流式上传并验证真实类型和尺寸；不上传视频、HTML、脚本或任意附件；不接受 public 输出路径 |
| `blog_media_reference` | 结构化 HTTPS 外链图片或视频页面 URL、标题 / alt | 图片直连；明确适配的视频平台生成播放器，其他链接普通展示；不接受任意 iframe，不提供服务器抓取代理 |
| `blog_release_status` | 查询 `releaseId`/`jobId` | 构建、验证、激活状态与实际公网 URL；明确失败步骤和可重试性 |
| `blog_rollback` | 选择仍保留的source/metadata版本，带 `expectedActiveReleaseId`、幂等键 | 复制目标形成新候选，统一预览确认后发布；重编译时展示renderer差异，不保证缺失原产物的旧HTML逐字节复现 |
| `blog_history` | 按内容 ID 分页列出当前 / 最后公开修订与保留的 3 旧版 | 每内容独立计数；不将整站 release 当成历史名额 |
| `blog_withdraw` | `mode=rollback/unpublish`；目标历史修订（回滚时必需）、CAS 和幂等键 | 回滚选定内容或下架为草稿；走明确候选 / 确认流程，保留独立副本与现有稿；下架旧址 404 |
| `blog_delete_content` | 内容 ID、预期修订、删除影响清单 / 确认引用及幂等键 | 物理清除内容及系统内可恢复副本，返回清理任务；在全部清理完成前不返回已完成，不创建可恢复该内容的新备份 |
| `blog_delete_asset` | 资产 ID、引用检查、预期版本 | 有其他内容 / 历史引用则返回冲突；独占或无引用资产可物理清理 |
| `blog_import_report` | 标准 SEO / GEO CSV / JSON 上传引用、格式版本、幂等键 | 校验来源、观察时间、指标及证据，保存可完整备份的报告记录 |
| `blog_distribution` | `operation=status/retry`，查询或重试 release/URL 的国内分发任务 | 分渠道 attempted/submitted/accepted/failed，以及已观测索引结果；不把提交成功写成收录成功 |
| `blog_backup` | `operation=export/import/inspect`；完整业务数据及第三方凭据，导入带 manifest hash、全站数据代次与幂等键 | 完整恢复先隔离重建 / 预览再确认覆盖全部业务数据；master 不替换，旧会话 / 任务不复活；内容导出单独标注为非完整备份 |

文章、About 和配图均经 MCP 上传；文章元数据包含单个 `categoryId` 与多个 `tagIds`。媒体上传限定 MIME、大小和内容 hash，资产原件与公开产物纳入版本体系。更新已发布文章、About、分类/标签的公开名称或描述均先形成候选，经预览、确认、发布新的站点 release 后替换公开 artifact；不能因为 taxonomy 修改了数据库就立即改变线上页面。新建尚未使用的 taxonomy 可先私有保存，现有 release 的 taxonomy snapshot 固定。媒体/分类/标签删除和合并必须返回关联影响，不能让 dangling 引用通过校验。这里的校验规则不构成角色权限。首版不提供多语翻译、语言族、hreflang 或切换语言工具。

## 内容和发布契约

### 不需要编辑器，但需要内容结构

外部 Agent 直接提供 Markdown 和结构化 metadata 即可。文章至少包含稳定 ID、slug、title、summary、作者、单个 `categoryId`、多个 `tagIds`、正文、引用/来源、图片及替代文本；分类标签 ID 必须能在同站 taxonomy 中找到且无层级。About 是同一内容模型的固定页面，不强制文章分类。站点固定中文 `zh-CN`。SEO title/description、canonical、公开更新时间、重定向等字段由服务端按明确规则校验或生成。自然语言工具 description 不能替代 JSON Schema 校验。

工具输入采用 JSON Schema 2020-12，明确 required、枚举、长度上限、时间格式和 `additionalProperties: false`；工具输出提供 `outputSchema` 和结构化结果，便于 Agent 读取 revision、状态和冲突。MCP 原生支持 `inputSchema`、`outputSchema`、`structuredContent`。无参数工具也须提供合法对象 schema，不能使用 `null`。[MCP 工具 schema](https://modelcontextprotocol.io/specification/2026-07-28/server/tools#tool)

### Markdown、公式、代码与图的严格输入合同

文章/About API 只接收 UTF-8 `markdown` 字符串与 typed metadata，未知字段拒绝。禁止 `renderedHTML`、`template`、`CSS`、`JS`、`executablePlugin`、`outputPath` 等产物/执行配置字段；媒体上传不接受替换公网站点文件的路径。服务端生成所有 HTML/SVG/样式引用和站内输出路径，客户端不能通过文件名、frontmatter 或备份包改变这些规则。

| 输入结构 | 统一程序行为 | 失败/显示边界 |
| --- | --- | --- |
| 普通 Markdown | 用 Goldmark 解析 AST，校验节点及 typed metadata，再经固定模板编译 | 普通正文的 raw HTML 节点结构性拒绝，报告行列并建议放入 html fence；不能用全正文 regex 把代码示例也误拒 |
| `html` / `markdown` /其他普通代码 fence | 按固定语言别名进行静态转义与高亮，未知语言回退纯文本 | 代码只展示文字；HTML不执行，Markdown不二次编译，JS/shell代码示例也不执行；闭合fence及大小超限给诊断 |
| 特殊 `mermaid` fence | 服务端使用允许图类型、固定配置的 parser/compiler 输出受控静态图 | 禁止用户frontmatter/init覆盖配置、执行脚本选项、click脚本/外部URL和远程资源；拒绝不支持语法，不回退成浏览器动态执行 |
| LaTeX 公式表达式 | `$...$`、`$$...$$` 或特殊 `math` fence 由固定数学渲染器编译 | 只支持公式，不接受任意 `.tex` 文档/工程；拒绝 document环境、write18/shell、文件读写/导入、任意包/外部URL资源；普通 `latex`/`tex`代码fence只展示代码 |

Goldmark AST 是识别正文、raw HTML、fenced code等上下文的起点；Mermaid与公式各自再用语法解析/明确配置做校验，不靠字符串黑名单模拟解析器。比如 HTML教学例子应在 html fence 内合法展示，不能因出现 `<script>`字符串就被当成已执行脚本。图/公式的具体库版本与受支持语法见 [内容编译与渲染](content-rendering.md)，本文件仅定义接口和执行边界。

内容只接受服务端配置的有限语法。图、公式、AST节点数、正文/单块字节数、图节点边数、公式展开/嵌套、编译时间/内存等有明确上限；建议这些上限作为程序配置/read-only content-rules resource 提供，最终数值由渲染器基准确定。诊断结构包含 `code`、`contentId`、源范围（行列/byte offset）、block kind、实际值/上限、修复建议，例如 `RAW_HTML_NOT_ALLOWED`、`UNSUPPORTED_MERMAID_TYPE`、`MERMAID_CONFIG_NOT_ALLOWED`、`MATH_COMMAND_NOT_ALLOWED`、`CONTENT_TOO_LARGE`、`RENDER_LIMIT_EXCEEDED`。摘要错误不得回显 master/服务 secrets。

本站上传图片、公式字体、图形和高亮资源由服务端托管，编译依赖使用固定可信 assets；外链图片由浏览器加载，视频播放器仅由程序的平台适配器生成，AI 不能指定任意 CDN 脚本。发布端只接受校验后的源快照及编译产物，外链固定的是 URL 与配置，不保证远端内容不可变；不提供上传文章 HTML 页面、替换 public 文件或执行任意 TeX/plugin 工具。

MCP 请求结构不合法、未知工具等协议错误走 MCP/JSON-RPC 错误；工具输入校验失败、revision 冲突等可操作失败返回 `isError: true` 和本项目定义的稳定错误码，例如 `REVISION_CONFLICT`、`SOURCE_HASH_MISMATCH`、`PREVIEW_EXPIRED`、`CONFIRMATION_REQUIRED`、`TAXONOMY_IN_USE`、`BUILD_FAILED`。客户端不能见到错误后偷偷降级成无 CAS 的覆盖写入。[工具错误处理](https://modelcontextprotocol.io/specification/2026-07-28/server/tools#error-handling)

### 同 token 下的确认步骤

默认工作流仍是“AI 保存草稿 → 展示预览 → 询问你 → 记录确认 → 发布”。你的确认可在后台预览页、CLI，或回复 Agent 后由 MCP 记录；各入口均使用同一个 master token，不存在 owner 第二凭据。

服务端建立 `confirmationId`，绑定 preview/candidate release、ChangeSet/hash、各选定修订、sourceHash、预览 artifactHash、renderer版本/选项、字体/图形资源、站点配置/资产快照、时间定稿规则及实例代次。发布检查记录有效、目标相符和 CAS 条件；消费记录与幂等发布意图同时落盘，相同幂等键重试返回同一结果。定稿只允许程序填入发布 / 更新时间及直接派生的列表排序、JSON-LD、sitemap、搜索字段，记录 finalArtifactHash，不重新解析正文或编译公式 / 图；不承诺预览和最终文件字节完全相同。确认过期、源或编译环境变化须重新预览。这保障内容一致性，不提供身份隔离。

持 master token 的 AI 同样可以调用确认和发布接口，因此系统**不能从鉴权辨别这次确认来自人还是 AI**。记录的 `clientName`、`confirmedBy` 或自然语言声明只是流程信息，不是人类身份认证。不要增加隐藏的第二密钥、角色或仅 owner 可调用的发布入口；也不要声称系统能禁止拥有 master token 的 Agent 自行发布。要求 Agent 等待你的回复属于可信客户端的操作约定，应写入客户端使用说明和工作流 prompt。

浏览器 session、CSRF nonce、preview ID、confirmation ID 与 revision 都是自动维护的运行状态或业务记录，不改变唯一 master token 的全部控制权。单纯传 `confirmed=true` 不替代明确版本的确认记录，但同 token 调用建立记录是允许的。

### 幂等、并发与预览一致性

1. **业务幂等键独立于 JSON-RPC request ID。** 断流或超时后客户端可能用新的协议 ID 重试；同一个业务幂等键与同一载荷返回已有结果。同键不同载荷拒绝，数据库持久保存去重结果。
2. **每次更新做 revision CAS。** 使用 `expectedRevision` 避免两个 Agent 相互覆盖。冲突返回实际 revision，让客户端重读并明确合并。
3. **预览固定完整构建输入。** 将immutable source hash（正文与 categoryId/tagIds 等元数据）、taxonomy revision/hash、About、站点内容快照、站点配置/canonical base URL、renderer/图与数学编译器版本、固定选项、字体/高亮/媒体assets hashes绑定到candidate manifest。预览列出文章与分类标签页等受影响URL，配置/字体/编译器变更也可能影响全站。只核对Markdown hash不足以保证最终HTML一致。
4. **发布只对已确认候选作时间定稿再激活。** 不在 publish 时重新猜测最新草稿或重编译内容；检查 confirmation 与 ChangeSet/revision/release/实例代次后按固定规则定稿。记录时间字段差异与最终 hash，其余变化拒绝。活动站点被其他发布改变时返回冲突，重新构建须重新确认。检查结果不能证明确认者是人。
5. **激活成功、外部推送成功分开。** 工具超时不代表发布失败；Agent 用幂等键或 job ID 查询实际状态。HTTP 202 或返回 job ID 仅表示工作已接收。

以上是本项目应用契约。MCP 新协议没有替应用提供事务、幂等、审计或静态构建发布语义；其明确鼓励跨调用操作使用显式 handles。[有状态业务工具的处理](https://modelcontextprotocol.io/specification/2026-07-28/server/tools#stateful-tools)、[传输中断规则](https://modelcontextprotocol.io/specification/2026-07-28/basic/transports/streamable-http)

### 回滚、事件与异步分发

在保留窗口内保存 immutable source/typed metadata、renderer版本/选项、字体与媒体引用、compiled artifacts及manifest；事件记录客户端声明、source/artifact hashes、confirmation ID、前后release IDs、时间和结果。同 token 的事件不能当作独立人员身份审计。回滚复制目标源内容形成新候选，经统一程序预览确认后新发布，恢复相应首页/分类标签页、文章/About、sitemap及引用资源；不倒退版本序列或丢后续事件。

若目标仍有原编译产物及完整资源快照，可复用其被验证的内容；若只保留源而旧 renderer/字体/compiled assets 已丢失，必须用当前受控编译器重新预览，展示渲染差异并重新确认，不能承诺与旧 HTML 字节/hash 完全相同。历史保留与备份应尽量记录编译依赖的版本/hash，但不承诺永远保留可执行旧工具链。窗口外被裁剪的源正文不能靠审计记录恢复。

### 三个历史版本与实际裁剪

已确认 **每篇文章 / About 分别保留现行版本 + 3 个已发布旧版**，下架保留最后公开修订及其 3 旧版；草稿不占名额，其他文章发布不挤占。保留策略实际裁剪超额正文快照与产物，而不是只在后台隐藏。整站制品单独清理，不影响每内容历史。普通历史裁剪可留无正文审计 metadata；永久删除则清理全部可恢复内容，详见需求确认稿。`blog_get/rollback` 对不可用版本明确返回错误。

AI 修改工作副本，上传、解析和提交草稿独立记录状态；失败上传不变为新草稿。副本与上传设置期限，不无限保留每次保存正文。构建 / 预览 / 发布及备份任务临时固定所需版本，结束后按窗口裁剪。媒体按副本、上传、草稿、历史及任务引用清理，普通历史裁剪不破坏备份。永久删除优先取消相关任务并清理系统管理的含该内容备份 / 制品；共享资产保留其他引用。外部已下载备份不在控制范围，不能声称已替用户删除。

### 必需的备份导出和导入

完整备份覆盖全部业务数据：文章、About、工作副本 / 上传、草稿、历史、分类标签、所有业务配置和第三方凭据、分析报告 / 证据、媒体、路由及操作 / 推送记录，manifest 标注敏感、schema version 与 hash。不含凭据的便携内容导出不是完整备份。导出采用一致快照，固定所需媒体 / 版本；master token、会话及运行锁排除，恢复保持当前部署 master token。外链媒体只保存引用。

导入先校验 schema / 大小 / hash、路径与引用，再隔离校验并重建公开内容，显示完整覆盖差异及预览；未通过解析的上传作为私有工作数据恢复。确认后检查全站数据代次并创建恢复前完整备份，再覆盖业务数据并激活，不把包中 HTML/SVG/模板/脚本当可信产物。第三方凭据随完整包恢复，master 保持当前；会话失效，历史任务恢复为中止 / 待评估记录，不重放网络操作。新实例代次使旧确认、上传提交句柄与幂等重试无效；恢复事件追加记录。文件鉴权传输，不增加第二个恢复密钥。

数据库提交、产物落盘和外部网络没有共同原子事务：先持久保存已验证最终产物，再事务记录 active release 与 outbox；SQLite 是唯一激活权威，内存快照由它加载，磁盘不另立权威指针。分发 worker 只对已激活、公网可获取且当前仍有效的 URL / 配置 / 实例代次执行；旧任务不能重新提交已下架或删除的内容。恢复对账流程见技术决策。

外部请求属于重试可能重复的交付，使用 URL+release/channel 键去重、限速和退避。状态模型至少分 `queued → attempted → submitted/accepted/failed`；`indexed`、`cited` 只在有外部可验证证据时记录，并保存证据来源/采集时间。搜索引擎或 AI 平台未提供观察接口时，状态应为 `unknown`。回滚也触发当前 URL 的更新事件，已发出的旧提交保留历史，不承诺撤销第三方缓存或引用。

## Resources、Prompts 与模型边界

Resources 适合提供中文写作规范、内容模型、分类标签、内容索引、引用策略等上下文，例如自定义 URI `blog://site/profile`、`blog://site/content-rules`、`blog://site/taxonomy`、`blog://posts/{id}`。全部管理资源读取经过 master token 认证，resource URI 不能用于绕过控制 API；不返回主 token 或第三方 secret，也不让公开静态导出混入草稿或系统路径。缓存按公开 HTML 和私有管理数据隔离。[资源规范](https://modelcontextprotocol.io/specification/2026-07-28/server/resources)

Prompts 可选提供 `draft_article`、`review_sources`、`review_seo` 等中文工作模板，并写明询问你再记录发布确认。它们是客户端工作约定，不是身份验证，也不执行可靠的业务校验。首版有 tools 和少量 resources 就足够；不要用几十个 prompts 模拟插件体系。[Prompts 规范](https://modelcontextprotocol.io/specification/2026-07-28/server/prompts)

服务端不必运行 LLM。事实验证、摘要和引用建议可由客户端模型完成；服务端执行格式、链接、日期、schema、master token、构建和来源完整性校验。以后加入内置模型时，模型生成费用、数据流向和异步状态仍独立于 MCP transport。

公开网页抓取与已连接 MCP 客户端的数据访问属于不同入口。安装 MCP 不会自动把全部博客文章上传到主流 AI 搜索，也不会自动让未配置该服务的模型发现 `resources`。SEO/GEO 的公开 HTML、sitemap、结构化内容与国内平台明确开放的推送接口仍需单独建设，不把 RSS/Atom 加回需求。首版分析以站内统计、SEO/GEO 平台链接及报告导入为主；只有找到明确可用的官方 API 契约后才接自动分析 API，不虚构抓取/引用数据。

## 已确认需求与实现选择

| 需求/选择 | 首版处理 |
| --- | --- |
| 本机 Codex 或 Claude Desktop | 已确定；采用 stdio 桥接，并实测所支持的协议版本与工具交互 |
| 服务部署位置 | 单站单机/容器已确定，但不强求与客户端同主机；经私有控制面调用唯一 Go 服务，不让桥接进程创建第二份发布状态 |
| 唯一 master token | 已确定；后台表单登录、CLI、Codex/Claude Desktop 共用，持有者全部控制；auth 文件中的digest/generation是运行权威，env仅首启初始化；无账号/角色/scope/OAuth |
| AI 草稿、你确认后发布 | 保留为默认步骤；确认记录绑定明确候选，同 token 无法鉴别人和 AI，不采用第二身份/密钥 |
| 仅中国中文与分类标签 | 已确定；固定 zh-CN，单主分类+多标签，无层级，MCP 管理/绑定与公开页面检索/展示，不做多语工具 |
| 文章/About/媒体/分类标签 | 全由 AI 经 MCP 上传；后台只查看、预览、确认、回滚；设置页写站点配置/服务凭据，不提供内容编辑器 |
| 公式、代码示例、Mermaid | 支持受限源语法，统一程序编译静态展示；HTML/Markdown代码fence仅显示，raw HTML正文拒绝；不接受文章HTML/模板/脚本/输出路径 |
| 3 个历史与备份恢复 | 每文章 / About 的 current+3 个发布旧版已确认；完整业务数据及第三方凭据整站恢复，当前 master 保留，不复活旧任务 / 会话 |
| RSS/Atom 与分析 API | 明确不做 feed；分析先统计+平台链接+报告导入，官方 API 契约明确后另接 |
| 远程抓取、定时自主写作、匿名 MCP | 后续扩展，当前不作为必须回答的配置事实或研究阻塞项 |

首版验收应包含：同一 master token 能登录后台并经指定 MCP 客户端上传文章/About/媒体及管理单分类/多标签，中文搜索可定位；后台没有内容编辑入口；无效 token/已失效 session 被拒，轮换后旧 token/旧 session 全部无效；token/设置 secrets 不出现在读响应、公开 HTML 或日志；缺少确认记录/候选 hashes 改变时发布失败，持 token 者可完整确认发布；非法跨源后台写请求被拒；重复发布不产生第二次副作用；并发修改返回冲突；预览与激活 artifact 相符；发布中断可查询恢复；回滚形成新发布版本并恢复分类标签页/sitemap/正文一致；超额旧正文确实裁剪、引用媒体保留、备份可完整导出导入；没有 RSS/Atom 输出；分发状态区分提交、收录和未知。不在这次研究阶段安装依赖或实施这些功能。

输入合同验收另覆盖：html/markdown代码fence中的脚本或嵌套示例仅显示，普通raw HTML准确定位拒绝；受支持公式/图服务端编译且预览资源不依赖AI指定CDN；Mermaid配置覆盖/外部资源、任意TeX工程/文件命令、内容与渲染超限均返回稳定位置化诊断；设置/媒体/备份导入不能注入HTML模板/JS/CSS或public路径；renderer选项/字体资源变化使旧确认失效；旧工具链/产物缺失时回滚重新预览并明确差异。

历史按每内容 current+3 已确认。后台确认与 Agent 对话后经 MCP 确认均使用同一 master token；分类标签删除有引用则先阻止，显式合并 / 迁移后才能完成。完整备份含第三方凭据并标注敏感，恢复覆盖全部业务数据；便携内容导出单独标注。新增副本上传解析、明确变更清单、下架 / 物理删除和媒体适配验收按需求确认稿执行，不重新打开权限模型。

## 研究记录与可信度

共 5 个有效搜索方向：协议版本、Go SDK、HTTP 传输安全、tools/resources schema、Go 授权能力。Exa 查询均 `numResults=5`；有效查询 5 次，另有 1 次因缺失连接器 `objective` 参数失败后重试。按技能口径，`sources_reviewed = 30`（包含失败请求的请求数预算）；实际有效返回 25 项，搜索结果有重复。随后使用 Exa fetch 和官方页面/固定 tag 原始源码核查，未采用第三方教程作为协议或版本结论依据。本轮单 token 调整没有重复版本检索，只补读 MDN cookie 说明与 Go net/http 文档；浏览器会话方案属于本项目建议，不是 MCP 强制要求。

全部主要来源均为 MCP 规范维护方或官方 SDK 项目；规范页属于协议定义的一手来源，固定 tag 源码用于验证实际 SDK 行为。SDK README 仍有旧协议示例，因此涉及 2026-07-28 的握手、会话与 headers 以当前规范和 release/source 为准。浏览器客户端兼容、整个项目构建大小、端到端性能和外部分发效果尚未实测，本文不把这些预测写成事实。
