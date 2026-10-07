# 中文个人博客：国内 SEO / GEO 分发与分析研究

检索日期：2026-10-07（Asia/Shanghai）。本轮使用 Exa 进行 5 个角度检索及一次百度统计授权补检，`numResults` 合计 30；搜索请求失败 0 次。页面 fetch 有 1 次失败（百度统计旧章节根地址），已通过官方手册入口和具体章节补齐。前一轮搜索预算为 35，累计 65；本文的活跃方案以最新用户范围为准。来源仅采用平台官方文档、规范原文和服务商官网；服务商自述只证明其宣传或公开接口说明，不证明监测完整性、效果或实际可用性。

## 1. 已确认的范围与实际目标

本项目服务国内中文读者，是个人单站，部署在一台机器或一个容器里。Go 后端提供内容、发布、后台设置、MCP 和分发任务；访客端以静态 HTML 展示。AI 先生成草稿，用户确认后发布；不做编辑器、插件或主题系统。

首版必须有文章、单主分类、多标签、文章目录、站内搜索、发布时间、更新时间和 About。全部站点内容由 AI 上传，后台用于预览、保留 3 个历史版本与回滚，并提供备份导入导出。增加轻量后台设置，用于站点信息、百度推送 token、统计服务和第三方分析入口。后台与 MCP 使用同一个管理 token，拥有同样的全部管理权限，不引入 owner / editor 角色体系。

分析首版范围已经确认：先接访问统计；SEO / GEO 先提供服务链接和报告导入，有明确 API 且用户决定启用后再扩展，不要求首版对所有供应商实现自动 dashboard API。具体统计服务尚未选定；百度统计可作为国内场景的候选。RSS / Atom 当前不做。

活跃方案只规划中文国内分发；不规划 Google 提交、英文路径、翻译工作流、hreflang 或以国际 AI 爬虫为中心的配置。国内必应 / IndexNow 是否启用列为可选待确认项，默认不启用，不作为核心流程依赖。

“SEO / GEO 推送系统”应拆成四条独立链路：

| 链路 | 本系统可以提供的结果 | 不应混淆的结果 |
| --- | --- | --- |
| SEO 技术产物 | 可抓取 HTML、metadata、canonical、sitemap、robots、真实状态码 | 技术校验通过不等于搜索排名 |
| 百度变化通知 | 按已验证站点和额度提交新 URL，记录平台接收反馈 | 接收成功不等于已收录，更不等于进入百度 AI 答案 |
| 第三方 SEO 分析 | 按指定词、地区、设备和时间查看排名快照或导入报告 | 排名快照不等于实际访客数或官方全量曝光 |
| GEO 监测 | 对一组明确问题采样，记录品牌提及、回答中的 URL 引用和证据 | 采样答案不等于全平台实际曝光；不是通用 GEO 提交 API |

MCP 是已连接 AI 客户端的内容与管理接口；不把 MCP 接入成功当作搜索收录。百度普通收录公开入口明确表示主动提交可加快发现，但无法保证收录。[百度普通收录](https://ziyuan.baidu.com/linksubmit/index)

## 2. 国内平台提交矩阵

“未知”表示本次未找到或未完成核验的公开规范，不证明平台绝不存在私有合作或其他入口。

| 平台 / 场景 | 首版位置 | 可用通道 | 证据与边界 |
| --- | --- | --- | --- |
| 百度普通搜索 | P0 | 普通收录 API 推送、sitemap、手动提交 | 当前公开入口可查；历史官方手册有 API 调用和响应说明。实际站点权限、token、TLS 端点及额度须后台核验；不保证收录 |
| 百度 AI 搜索 / 文心相关 AI 答案 | 观察对象 | 复用公开内容与百度搜索发现；AI 答案抽样或第三方报告 | 普通收录成功不能推导 AI 引用。本次未确认普通博客另有可保证进入 AI 答案的公开网站推送规范 |
| 豆包 | 观察对象 | 可选第三方采样报告 / 经公开接口确认后的监测 | 网站收录 push、专用 bot 管理规范和官方站点曝光反馈本次未确认 |
| 腾讯元宝 | 观察对象 | 同上 | 不把腾讯生态内容可被检索的现象推导为本站有官方直推 API |
| Kimi | 观察对象 | 同上 | 模型或联网问答 API 与网站索引提交是不同能力；本站收录 push 本次未确认 |
| DeepSeek | 观察对象 | 同上 | 模型调用、RAG 注入和模型回答认识站名不证明网址已索引；本站收录 push 本次未确认 |
| 国内必应 / IndexNow | 可选待确认 | IndexNow 变化通知及 sitemap | 协议有公开文档，但最新需求未将此通道设为必需。启用后仍区分 accepted 与 indexed |

百度依据：[当前普通收录入口](https://ziyuan.baidu.com/linksubmit/index)、[官方普通收录手册](https://ziyuan.baidu.com/college/courseinfo?id=267&page=2)。可选 IndexNow 依据：[协议](https://www.indexnow.org/documentation.html)、[FAQ](https://www.indexnow.org/faq)。

本次发现的百度千帆 [AI 搜索 API 文档](https://ai.baidu.com/ai-doc/AppBuilder/amaxd2det)说明的是调用者提交问题、获得生成回答及参考来源的能力。此类 query / chat 接口可以用于受控抽样，但不是“把博客 URL 提交到消费端 AI 搜索索引”的接口。付费调用也不自动等于掌握消费端全量曝光数据。

### 百度普通收录的实现边界

当前入口仍说明可主动提交，并且不保证收录；可读取的官方学院手册标注最新更新章节为 2020-08-14，列出 API、sitemap 和手动提交方式。手册介绍站点验证、token、按行提交 URL、`success` / `remain` / 不合法 URL 等反馈，提醒重复提交会浪费额度。[当前入口](https://ziyuan.baidu.com/linksubmit/index)、[历史手册](https://ziyuan.baidu.com/college/courseinfo?id=267&page=2)

首版后台提供站点地址、推送 token、控制台实际给出的 API endpoint、quota 状态、最近提交时间和错误信息。不要把历史 HTTP 示例直接当作当前可安全使用的端点，也不要把示例批次数字、响应额度写成所有站点固定权益。上线前用真实账号确认站点已验证、接口可用、凭据有效、额度和批次限制；本文没有登录账号、试投 URL 或购买服务。

如果没有推送权限，状态显示 `configuration_required` 或 `provider_unavailable`，继续提供 sitemap、待提交 URL 导出和官方控制台链接。删除 / 改 URL 的死链与改版能力同样按实际账号接口核验，不能将新 URL 推送接口假当通用删除接口。

## 3. 百度统计、SEO 排名与 GEO 监测不是同一服务

### 3.1 百度统计：浏览器访问分析

百度统计官方说明：在网站页面加入 JavaScript 跟踪代码，浏览器执行代码后把访问参数送到百度统计；创建站点时得到对应的代码跟踪 ID。官方推荐异步代码，示例脚本来自 `hm.baidu.com/hm.js`。[百度统计代码介绍](https://tongji.baidu.com/web/help/article?id=174&type=0)

因此可以在所有**公开内容页面**注入统计代码，记录 PV、访客、来源、入口页等可用报告。不在后台、MCP、鉴权预览中加入访问统计脚本。脚本未执行、被拦截、访客未点击或机器人不执行 JS 的情况不能从此统计到；它不提供关键词排名，也不能证明某次 AI 回答曾引用文章。

有两个不同字段：

- `trackingId`：公开统计脚本的站点签名，用于页面采集；官方 JS API 称其为 `site_signature`。
- `apiSiteId`：后台报告查询使用的 `site_id`；不要假设它等于脚本签名。

字段依据：[百度统计 JS API](https://tongji.baidu.com/holmes/Analytics/%E6%8A%80%E6%9C%AF%E6%8E%A5%E5%85%A5%E6%8C%87%E5%8D%97/JS%20API/JS%20API%20%E4%BD%BF%E7%94%A8%E6%89%8B%E5%86%8C)、[报告 API](https://tongji.baidu.com/api/manual/Chapter1/getData.html)。

### 3.2 百度统计：有公开报告 API，但账号需要真实授权

官方 Tongji API 有按 `site_id`、报告 method、日期和 metrics 获取数据的说明，并区分百度账号与百度商业账号。普通百度账号文档列出 access token 方式，商业账号示例使用对应账号凭据。[getData 文档](https://tongji.baidu.com/api/manual/Chapter1/getData.html)

官方说明 API 数据导出服务在“管理”中开通，需要站点主账号和有效站点权限；这是百度统计外部服务的开通要求，不意味着本博客后台需要另设 owner 角色。授权文档介绍 API Key、Secret Key、授权码、access token 和 refresh token。[数据 API 说明](https://tongji.baidu.com/holmes/Analytics/%E6%93%8D%E4%BD%9C%E6%89%8B%E5%86%8C/%E6%95%B0%E6%8D%AE%E7%AE%A1%E7%90%86/%E6%95%B0%E6%8D%AEapi)、[百度账号授权](https://tongji.baidu.com/api/manual/Chapter2/openapi.html)

产品分两级接入：

1. 填统计跟踪 ID、开启异步脚本、提供官方 dashboard 链接。
2. 用户确有报告 API 权限时配置服务端凭据，用 Go 拉取所需报告；没有权限时保留 dashboard 链接或报告导入，不虚构“自动同步成功”。

部分官方授权示例较旧并含 HTTP URL；配置真实授权时应使用当前受支持的 HTTPS 流程，不能将长效凭据明文发送或记录到日志。具体账号限额、授权寿命和可用报告以配置时的平台返回与控制台为准，本轮未调用认证接口。

### 3.3 SEO 排名监测：存在第三方公开 API 候选

[5118 官方 PC 排名查询 API](https://www.5118.com/apistore/detail/0d5b519e-d2a2-e711-b5b0-d4ae52d0f72c)公开了指定网址和关键词提交任务、返回 task ID、随后获取关键词排名结果的流程；说明文档提供 Authorization 凭据示例、错误码和额度 / 频率限制。其排名监控服务页面也介绍定制监控与公开 API 方式。[5118 排名监控方案](https://www.5118.com/service/keyword_rank_monitor)

这是后续可选自动 adapter 的依据，不是已选服务或首版实现承诺。首版使用服务链接和报告导入。官网页面同时存在旧 HTTP 描述和 HTTPS 示例，后续接入前需要确认当前 TLS 端点、账号套餐、PC / 移动覆盖、地区、刷新频率和结果授权。

排名记录至少带：服务商、搜索引擎、设备、地区、关键词、目标站点 / URL、排名、观察时间、task ID 和结果原文。第三方“权重”是服务商指标，不标为百度官方权重；不能把一次快照或 `site:` 估计数量当成官方全量索引证明。

### 3.4 GEO 监测：公开 API 与报告服务分开对待

以下仅用于说明可接入形态，具体商用服务待用户选择；本轮未注册、付费、调用 API 或验证任何供应商的准确率与国内部署可达性。

| 候选 / 形态 | 官方页面能确立什么 | 本系统接入方式 | 仍需验证 |
| --- | --- | --- | --- |
| GEOBase / AIBase | 官网宣传对豆包、DeepSeek、元宝等进行品牌曝光和引用来源分析 | 先保存 dashboard URL，用户提供报告时导入；本轮未确认公开 API | 采样方法、可导出格式、URL 引用明细、套餐、历史数据和 API 权益 |
| AIPOGEO 诊断报告 | 官网介绍按问题模拟测试、报告查看与导出 | 可选报告链接 / 文件导入；本轮未确认公开 API | 供应商部分“引用率”实际按品牌提及占比定义，需字段映射核验 |
| GEO Insights / GEO Tool Hub | 服务商首页公开异步监测 API 示例：提交问题、查询任务、取得回答与参考 URL；列出国内 AI 平台选项 | 首版链接 / 报告；后续可选 Go API adapter，用户选择且验证账号后启用 | 服务商身份和所在地、国内可达、数据合规使用条件、是否真实消费端采样、限额、样本证据、错误处理 |
| 其他 SEO / GEO 服务 | 以用户选定服务的第一方合同 / API 文档为准 | 有公开 API 再联通，否则 dashboard 链接和报告导入 | 不因营销页面写“支持多模型”而自动认定具有开放 API |

来源：[GEOBase 官网](https://geo.aibase.com/)、[AIPOGEO 诊断说明](https://www.aipogeo.cn/AI%E8%83%BD%E8%A7%81%E5%BA%A6%E6%A3%80%E6%B5%8B.html)、[GEO Insights 公开接口说明](https://geo.geotoolhub.com/)。

GEO Insights 首页公开的是**监测问题提交接口**：返回任务、采样回答、referenceList / citationList。它不是向百度、豆包、元宝、Kimi 或 DeepSeek 推送博客收录的通道。服务商宣称的覆盖、可用性或“与网页一致”仍需抽样验证，不能照搬为本项目 SLA。

GEO 数据应至少拆成：

| 字段 / 指标 | 定义 |
| --- | --- |
| 品牌提及 `mentioned` | 回答正文出现站点名、作者名或已配置别名 |
| 本站 URL 引用 `cited` | 回答附可核验的本站链接，按 canonical / 合法重定向归并 |
| 第三方来源引用 | 回答引用别的媒体或平台提到本站，单独记录，不算本站文章链接 |
| 采样提及率 / 引用率 | 明确问题集、平台、时间、有效回答数与相应分母；失败请求不能默认为未引用 |
| 实际曝光 / 点击 | 只有服务实际提供对应观测数据时展示；不能由采样命中率乘以平台用户数推算 |

例如某服务用“100 个问题中 12 个答案提到品牌”计算“引用率”，在本系统中应映射成**采样提及率**；只有返回明确 URL 引用证据时才计入本站引用率。用户应能查看问题、平台 / 模式、采样时间、原始回答、链接或供应商证据截图。付费联网问答 API 的输出也不当然等于同名消费端产品的真实用户回答。

## 4. 后台结构化配置与秘密管理

后台只做必要管理与设置，不成为编辑器。所有后台 API、设置修改、任务重试、内容发布和 MCP 管理能力统一使用一个 token，不再增加角色、权限等级或第二个“owner token”。第三方 API 凭据是调用外部服务的配置，与本系统管理 token 分开存储。

| 配置对象 | 公开 / 非秘密字段 | 服务端私密字段 | 状态与操作 |
| --- | --- | --- | --- |
| 站点 | domain、名称、描述、作者、默认语言 `zh-CN` | 无 | 校验规范域名和站点产物；About 由 AI 上传，后台预览 |
| 百度推送 | enabled、verifiedSite、endpoint、dashboardURL、batchLimit / quota 状态 | pushToken | 保存、替换 / 清空 token；检查配置；查看结果；按真实权益启用 |
| 访问统计 | provider、serviceID / trackingId、scriptURL、dashboardURL、enabled | 可选报告 apiKey / secretKey、access / refresh token | 公开采集与私密报告同步分别开启 |
| SEO 分析 | provider、serviceID、dashboardURL、keywords、devices、region；pollInterval 后续预留 | 后续启用 API 时的 apiKey / token | 首版链接 / 报告导入；以后启用已确认 API 才显示拉取状态 |
| GEO 分析 | provider、serviceID、dashboardURL、prompts、平台列表；采样周期后续预留 | 后续启用 API 时的 apiKey / token | 首版报告导入 / 链接；字段映射和证据可查询 |

设计规则：

- 常规设置读取只返回私密字段的“已配置”状态和 mask；保存 mask 不应覆盖原值。替换、清空和轮换是明确操作。
- 私密值只持久化在后端的非公开配置 / 数据库中；可使用服务端主密钥加密，主密钥来自环境或非公开持久配置。公开 HTML、sitemap、搜索索引、JSON-LD、llms / Markdown 输出和前端 JavaScript 中都不包含管理 token、推送 token 或私密 API key。
- trackingId 这类公开采集标识可以进入页面；报告 `apiSiteId` 等业务标识也不应被误当作 API 认证凭据。公开可用字段不允许夹带真实 secret。
- `scriptURL` 使用验证后的 HTTPS URL和按 provider 约束的域名；保存 dashboard URL 不代表将任意网站 HTML 嵌入管理页。首版不用任意脚本 / HTML 粘贴框，避免配置混入凭据或破坏页面。
- 接入方式结构化为 `script`、`api`、`report_import`、`dashboard_link`，每项显示 `configured`、`connected`、`awaiting_authorization`、`quota_exhausted`、`last_error` 等真实状态。
- 日志过滤 Authorization、URL query 中的 token、secret 和原始认证响应；导入的报告和外部文本不执行脚本，不把它们当作系统指令。
- 配置变更影响公开统计代码时需要重新构建产物；私密 API key 轮换只影响服务端调用。任何 provider 的 API 查询、页面访问或报告导入不得自动发布文章。

分析报告导入可由后台选择文件，也可通过 MCP 提交结构化报告 / 文件；它属于私密分析数据，不属于“AI 上传的站点正文”，不触发发布、不进入公共内容和搜索索引。报告导入与整站备份恢复使用分开的类型、格式校验和操作入口，不能把供应商 CSV 当作备份包覆盖站点。

整站完整备份与恢复覆盖内容、副本 / 上传、草稿、每内容保留历史、分类标签、本站媒体、全部业务配置及第三方凭据、分析报告 / 证据和操作 / 推送记录，manifest 标注敏感并通过鉴权接口传输。内容导出不能冒充完整备份。认证配置不迁移，当前 token 保持，旧会话失效；任务历史恢复为记录，不自动重放。恢复先隔离校验重建并预览，再检查全站写入代次、创建恢复前完整备份并确认切换，域名 / 渠道核对后才允许推送。每文章 / About 当前 + 3 旧版已确认，回滚更新同一 URL 的页面与搜索；下架及永久删除返回 404。外链媒体只备份 URL。

## 5. 中文静态站的 SEO 与信息结构

以下属于本项目技术基线，不声称某个标签一定提高百度排名或 AI 引用率。

新增富内容要求：AI 仅提交受限 Markdown 与结构化元数据，HTML / Markdown 代码块显示转义源码，LaTeX 和 Mermaid 在发布前编译。公开阅读不依赖公式 / 图表运行脚本；公式保留 HTML + MathML，图表以自托管 SVG 图片配图名 / 说明，搜索纯文本从校验过的源内容提取，不直接索引渲染 HTML、SVG 样式或重复公式节点。代码示例保留可读源码，分类与正文仍同属一个发布快照。[内容编译与渲染](content-rendering.md)

外部 AI / 搜索平台对公式或图表的理解按实际观察记录，不因生成 MathML / SVG 就声称引用效果改善。站内搜索可匹配代码示例以及公式 / 图表的文字说明，不增加 OCR、公式相似度或视觉搜索能力。源语法错误或渲染失败阻止发布，图表资源与 CSS / 字体沿用版本化静态资产规则。

| 需求 | 首版行为 | 验收点 |
| --- | --- | --- |
| 文章正文 | 完整静态 HTML，标题、摘要、正文、作者和时间可直接阅读 | 禁用 JS 仍能读正文；公开页面真实 200 |
| 分类 / 标签 | 每篇文章一个主分类、多个标签；各自稳定 URL、中文名称和可跟随文章链接；正文显示分类标签 | 不存在分类 / 标签是真实 404；空集合不伪造可索引内容 |
| 文章目录 | 从真实标题生成稳定锚点；目录能跳转正文 | 标题重名处理，不把页内锚点当独立 sitemap 页面 |
| 站内搜索 | 只索引已发布文章；搜索结果显示标题、摘要、分类 / 标签和原文 URL | 草稿、后台、私密配置和分析报告不进入公开搜索索引 |
| 发布时间 / 更新时间 | `published_at` 保留首次发布时间；正文实质变更才更新 `modified_at` | 页面、JSON-LD、sitemap 一致；日期含明确时区 |
| About | 固定公开页，说明作者、站点主题与可核验背景；参与导航和公开索引 | 真实来源与作者说明，避免虚构资历 |
| metadata | 独立 title、description；可选 OG 作为分享字段 | 摘要对应正文，不以堆砌关键词代替说明 |
| canonical | 自指规范 URL；内部链接、sitemap、分享地址一致；slug 改动保留真实重定向 | 规范 URL 表示偏好，不当作平台必须索引的指令 |
| JSON-LD | BlogPosting 与实际作者、标题、语言、分类、发布时间和重要更新时间一致 | 可解析，字段与可见正文一致；不宣称百度专用 GEO schema |
| sitemap / robots | sitemap 列已发布规范 URL与真实 lastmod；robots 给 sitemap 入口 | 百度实际 sitemap 限额和索引型文件支持按其后台核验 |
| RSS / Atom | 当前不做 | 不作为首版产物、配置或验收条件 |
| 状态码 / 删除 | 下架 / 永久删除旧址真实 404；永久迁移用 301 / 308 | 不把不存在路径返回首页 200，不为物理删除内容保留正文墓碑 |
| 性能 | 样式和字体克制、图片有尺寸、交互 JS 按需；第三方统计异步且可关闭 | 统计或监测服务故障不阻断正文加载 |

通用规范依据：[canonical link RFC 6596](https://www.rfc-editor.org/rfc/rfc6596)、[robots RFC 9309](https://www.rfc-editor.org/rfc/rfc9309)、[BlogPosting](https://schema.org/BlogPosting)。百度 sitemap 入口、配额与格式描述以 [官方普通收录手册](https://ziyuan.baidu.com/college/courseinfo?id=267&page=2)及实际后台为准。

robots 是爬虫访问约定，不是身份验证。草稿、后台和 MCP 必须鉴权隔离；预览可额外 noindex，但不依赖 noindex 保密。不用同一个 robots 禁止抓取后，再承诺机器人能读取页面上的移除指令。[RFC 9309](https://www.rfc-editor.org/rfc/rfc9309)

可选 Markdown / llms.txt 只用于已连接工具或主动使用它们的 agent，列为后续实验，不把它当国内 AI 平台官方收录规范。中文站没有英文页面、多语言索引或 hreflang 生成任务。

## 6. Go 发布与分发流程

以下是单站实现建议：一个 Go 进程、数据库持久任务和静态文件即可；不要求独立队列、额外大模型常驻服务或插件宿主。

1. **副本上传与草稿。** MCP 修改独立副本，上传、解析成功后 CAS 提交新草稿；建立明确 ChangeSet，后台不编辑正文。
2. **预览构建与确认。** 按清单构建文章、列表、分类 / 标签、About、目录、搜索、sitemap 与路由，验证后展示预览；确认绑定修订 / 内容 hash、清单与时间定稿规则。其他草稿不公开，RSS / Atom 不生成。
3. **定稿与持久发布意图。** 发布只填入服务端时间及其直接派生字段，保存最终产物 hash 与 URL 差异，不重编译正文；SQLite 提交 active release 与 outbox 为唯一激活权威。待分发任务暂不派发，失败保留旧公开版本。
4. **切换与外网验证。** 原子切换公开 release，验证文章、canonical、sitemap 和相关状态码；没有公网可访问域名时分发任务保持待配置。外网检查不宣称验证所有 CDN 节点。
5. **激活 outbox。** 在事务中标记 `public_verified` 并激活百度等已启用 adapter 的任务。重启后根据发布意图和公开版本补齐，避免部署成功而通知丢失。
6. **平台通知。** 首版百度按真实权益提交新增有价值 URL；更新、删除和迁移按该平台明确支持的通道处理。IndexNow 只有用户启用后才发送差异 URL。幂等键包含站点、provider、URL、published revision 和操作，不因刷新后台重复消耗配额。
7. **退避与配额。** 429、暂时网络错误和适用 5xx 进入退避加抖动；遵守可用 Retry-After；凭据 / 验证 / 参数错误停止盲目重试并显示原因。远端接收不确定的 timeout 不能直接写“提交失败”或“已索引”。
8. **独立分析任务。** 首版加载已配置访问统计、提供 SEO / GEO 服务链接和报告导入。以后启用已确认 API 时再按服务限额低频拉取，记录时间、provider、task ID 与证据。未选 provider 就展示待选择，不自动购买或注册。
9. **后台和 MCP 报告。** 分别显示静态发布、通知接收、平台收录反馈、访问统计、关键词排名、采样提及与 URL 引用；一种数据缺失不改写成另一种数据为零。

建议状态：`queued → public_verified → sending → accepted / retry_wait / failed / configuration_required`。索引、访问、排名和 AI 引用是独立数据，不从 `accepted` 自动跳到 `indexed` 或 `cited`。

下架和永久删除文章应移出公开 HTML / Markdown、分类标签列表、搜索索引和 sitemap，旧址返回 404；回滚保持 URL 并发布历史内容的新修订。永久删除同时清理系统内可恢复副本与独占媒体，外部缓存不在本系统控制范围。百度删除 / 死链反馈需使用实际支持的通道；后台展示证据，不能承诺即时从搜索或 AI 结果消失。

## 7. 观测数据和可验收要求

### 必须带来源的证据

| 证据层 | 能证明什么 | 不能推导什么 |
| --- | --- | --- |
| 外网 HTTP 检查 | 本站当前版本可访问，路由与产物正确 | 搜索已抓取 / 已索引 |
| 百度推送反馈 | 某次批次或 URL 已被平台接收，或有明确失败 | 排名、用户曝光、AI 引用 |
| 百度统计 | 脚本采集到的访问和可用来源报告 | 全部访客、全部机器人、未点击 AI 引用 |
| SEO vendor 结果 | 某时某设备地区的指定词排名样本 | 百度官方全量流量、站点绝对权重 |
| GEO vendor 样本 / 导入报告 | 指定问题集中的提及和有证据的引用 | 全平台每次真实用户问答或真实曝光 |
| 实际访问日志 | 某 UA / IP 曾请求某路径、收到某状态 | 单凭 UA 就确定 bot 身份，或确定其 AI 答案引用 |

单机直接提供页面时可以记录源站访问；以后接 CDN 时，缓存命中可能不会到 Go origin，需要宿主可用的边缘日志或接受采样 / 源站观测范围。仅浏览器 JS 统计不能覆盖不执行 JS 的机器人。模拟 bot UA 的 curl 只验证配置，不计真实抓取。

### 首版可以自动或人工验收的行为

1. 文章、分类、标签、About、目录和搜索满足正文直接可读；日期与公开索引一致；缺失路径真实 404。
2. 草稿、管理 token、推送 token 和私密 API key 不进入公共 HTML、静态目录、搜索索引或 sitemap。
3. 全后台与 MCP 使用同一管理 token；常规设置读取返回 mask / configured 状态，mask 保存不覆盖已有 secret。
4. 百度无凭据 / 权限时明确待配置；真实站点接入时验证 ownership、token、endpoint 和 quota，未验证不展示推送已成功。
5. 统计脚本仅在启用且配置正确的公开页面出现；脚本失败不会阻断正文；API 报告授权与公开采集单独展示。
6. 首版 SEO / GEO 服务链接和报告导入可用，报告保留来源和时间；没有选择 API 接入时不显示自动同步状态。报告导入与备份恢复的入口和校验分开。
7. GEO 报告区分品牌提及与本站链接引用，带有效样本分母和证据；服务端错误 / 超时不当作未提及。
8. 构建失败、公开验证失败、429、无效 token、超时、部分失败和重启恢复的状态有明确行为；重复无变化发布不增加通知。
9. 删除内容同步更新所有公开派生产物；平台移除反馈展示已知状态和时间。
10. 报告含 provider、采集 / 导入日期和范围；未提供的官方曝光、索引或引用数据显示未知，不编造数值。
11. 全部正文由 AI 上传，后台可预览 3 个历史版本和回滚；备份可导出、校验并导入恢复，报告文件不能冒充备份覆盖站点。RSS / Atom 不作为首版检查。

**不可承诺的结果：**固定分钟内百度收录、关键词首页、一定被豆包 / 元宝 / Kimi / DeepSeek 引用、完整获得每个平台曝光、付费工具对全站所有主题都准确、固定时间内删除搜索结果。它们可以成为运营观察目标，不能作为系统功能通过标准。

## 8. 首版优先级和待选服务

| 优先级 | 范围 | 状态 |
| --- | --- | --- |
| P0 | 中文静态文章、单主分类多标签 / 目录 / 搜索 / 时间 / About；SEO 技术基线；AI 上传与确认后发布；3 个历史版本与回滚；备份导入导出；持久任务 | 已确认 |
| P0 | 百度普通收录 adapter、sitemap、后台 token 设置、真实额度 / 错误反馈、手动提交退路 | 已规划；上线启用依站点账号能力 |
| P0 | 一个 token 管理后台与 MCP；结构化第三方设置、secret mask、dashboard 链接和报告导入能力 | 已确认 |
| P0 | 可配置异步访问统计；SEO / GEO 服务链接、报告导入与证据模型 | 已确认该接入范围，具体统计 / 商用服务待用户选择 |
| P1 | 用户选择后的百度统计报告 API、SEO 排名 API、GEO 监测 API adapter | 依服务选择、公开文档、凭据与实测权限 |
| 可选 | 国内必应 / IndexNow；Markdown / llms.txt agent 入口 | 待确认，不作为默认首版依赖 |
| 当前不做 | RSS / Atom；Google、英文与多语言 | 按用户最新范围排除 |

待落实：域名与部署环境；百度已验证站点账号及当前推送权限；采用哪一个访问统计、SEO 排名和 GEO 监测服务；是否需要付费 API 自动同步，还是先使用报告导入 / dashboard 链接。用户尚未选服务时只完成可配置接入结构，不绑定某个商业供应商。

本轮已核验的关键第一方页面：

- 百度收录：[普通收录入口](https://ziyuan.baidu.com/linksubmit/index)、[历史官方手册](https://ziyuan.baidu.com/college/courseinfo?id=267&page=2)。
- 百度统计：[统计代码](https://tongji.baidu.com/web/help/article?id=174&type=0)、[API 手册](https://tongji.baidu.com/api/manual/)、[getData](https://tongji.baidu.com/api/manual/Chapter1/getData.html)、[账号授权](https://tongji.baidu.com/api/manual/Chapter2/openapi.html)、[数据 API 与权限](https://tongji.baidu.com/holmes/Analytics/%E6%93%8D%E4%BD%9C%E6%89%8B%E5%86%8C/%E6%95%B0%E6%8D%AE%E7%AE%A1%E7%90%86/%E6%95%B0%E6%8D%AEapi)。
- SEO API 候选：[5118 PC 排名查询 API](https://www.5118.com/apistore/detail/0d5b519e-d2a2-e711-b5b0-d4ae52d0f72c)。
- GEO 接入形态：[GEOBase](https://geo.aibase.com/)、[AIPOGEO 报告说明](https://www.aipogeo.cn/AI%E8%83%BD%E8%A7%81%E5%BA%A6%E6%A3%80%E6%B5%8B.html)、[GEO Insights API 示例](https://geo.geotoolhub.com/)。
- 国内 AI API 与网站提交的区分：[百度千帆 AI 搜索](https://ai.baidu.com/ai-doc/AppBuilder/amaxd2det)。
