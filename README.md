# 间隙

间隙（Jianxi）是我们自主搭建的 AI native 中文个人博客系统。

当前版本：**1.0.0**（Git 标签 `v1.0.0`）。

当前目标：面向国内的中文个人博客，采用 Go 后端、静态公开页面和 MCP 内容上传。包含单密钥后台设置、分类标签、目录、搜索、预览回滚、备份导入导出，以及国内 SEO / GEO 推送与第三方分析接入。

调查日期：**2026-10-07**。从 [设计研究导航](docs/ai-native-blog/README.md) 开始阅读。

| 资料 | 内容 |
| --- | --- |
| [产品需求确认稿](docs/ai-native-blog/requirements.md) | 当前已确认范围、页面、后台、历史版本、分析接入与待定项 |
| [技术决策](docs/ai-native-blog/decision-and-requirements.md) | 推荐技术栈、单密钥、静态发布、中文搜索和迁移模型 |
| [当前依赖与动效决策](docs/technical-decisions.md) | 已实施的依赖清单、GSAP 统一动效入口与维护边界 |
| [内容编译与渲染](docs/ai-native-blog/content-rendering.md) | 受限 Markdown、LaTeX、代码高亮、Mermaid 与静态渲染依赖 |
| [技术栈比较](docs/ai-native-blog/stack-evaluation.md) | Go 模板、Hugo、Astro、templ 与存储方案取舍 |
| [MCP 调查](docs/ai-native-blog/mcp-research.md) | 协议、官方 Go SDK、同一管理 token 和 AI 上传 / 发布契约 |
| [SEO / GEO 调查](docs/ai-native-blog/distribution-research.md) | 国内推送、访问统计、SEO / GEO 服务与报告、反馈边界 |

内容全部由 AI 上传受限 Markdown 和元数据，程序编译 LaTeX 公式、代码块及 Mermaid 并统一生成页面；AI 不能修改 HTML 页面或模板。后台提供配置、预览和回滚；无文章编辑器、主题或插件系统。RSS、Google 与英文 / 多语言页面当前不做。

已确认后端流程为独立副本 → 上传 → 解析 → 新草稿 → 明确变更清单预览 → 确认发布。每篇文章 / 关于我保留当前 + 3 旧版；撤回可回滚或下架，永久删除物理清理；完整备份恢复全部业务数据。支持 AVIF 等图片和外链、适配平台的视频链接嵌入，不上传或托管视频文件。详细边界以产品需求确认稿为准。

项目已进入开发，正式代码采用 **Go + SQLite + 静态发布 + 官方 Go SDK 的本地 MCP 编译器**，界面延续最新 Weave 黑白灰原型，使用自托管霞鹜文楷（按需并行加载 WOFF2 分片）和 GSAP 连续变形交互。只实现实际界面，不包含自动循环演示。

## 本地运行

需要 Go 1.27.1、Node 22。首次安装会下载本地 MCP 编译所需浏览器；博客服务器只运行 Go。

```powershell
npm ci
npm run assets
go run ./cmd/jianxi serve --demo --data data/dev
```

打开 [博客](http://127.0.0.1:8080/)。管理入口为 `/admin/{随机安全码}/`，完整路径保存在私有目录 `data/dev/admin-route.txt`；博客不展示管理链接，旧 `/admin/` 和错误安全码返回 404。首次生成的管理密钥位于私有目录 `data/dev/initial-token.txt`，后台输入该密钥解锁。`--demo` 仅在空站导入原型中的演示内容；正式部署去掉此参数。所有数据、密钥、会话与备份均不进入版本控制。

```powershell
go test ./... -count=1
go vet ./...
go build -o bin/jianxi.exe ./cmd/jianxi
```

[开发与部署说明](docs/development.md) 包含 MCP 配置、真实发布流程、动效说明、备份与删除规则、验证结果和当前限制。纯 Go 服务端镜像 `jianxi:latest` 已完成 Linux/amd64 构建与容器验证；富内容编译、图表浏览器和 Sharp 图片检查在发文电脑的 MCP 中运行，可使用 Docker Compose 部署；镜像迁移、初始密钥和数据卷说明见该文档的容器部署章节。

[视觉原型](design-prototypes/README.md) 保留用于对照；最新方案为 `design-prototypes/weave/`。
