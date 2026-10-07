# 间隙

间隙（Jianxi）是我们自主搭建的 AI native 中文个人博客系统。

当前目标：面向国内的中文个人博客，采用 Go 后端、静态公开页面和 MCP 内容上传。包含单密钥后台设置、分类标签、目录、搜索、预览回滚、备份导入导出，以及国内 SEO / GEO 推送与第三方分析接入。

调查日期：**2026-10-07**。从 [设计研究导航](docs/ai-native-blog/README.md) 开始阅读。

| 资料 | 内容 |
| --- | --- |
| [产品需求确认稿](docs/ai-native-blog/requirements.md) | 当前已确认范围、页面、后台、历史版本、分析接入与待定项 |
| [技术决策](docs/ai-native-blog/decision-and-requirements.md) | 推荐技术栈、单密钥、静态发布、中文搜索和迁移模型 |
| [内容编译与渲染](docs/ai-native-blog/content-rendering.md) | 受限 Markdown、LaTeX、代码高亮、Mermaid 与静态渲染依赖 |
| [技术栈比较](docs/ai-native-blog/stack-evaluation.md) | Go 模板、Hugo、Astro、templ 与存储方案取舍 |
| [MCP 调查](docs/ai-native-blog/mcp-research.md) | 协议、官方 Go SDK、同一管理 token 和 AI 上传 / 发布契约 |
| [SEO / GEO 调查](docs/ai-native-blog/distribution-research.md) | 国内推送、访问统计、SEO / GEO 服务与报告、反馈边界 |

内容全部由 AI 上传受限 Markdown 和元数据，程序编译 LaTeX 公式、代码块及 Mermaid 并统一生成页面；AI 不能修改 HTML 页面或模板。后台提供配置、预览和回滚；无文章编辑器、主题或插件系统。RSS、Google 与英文 / 多语言页面当前不做。

已确认后端流程为独立副本 → 上传 → 解析 → 新草稿 → 明确变更清单预览 → 确认发布。每篇文章 / 关于我保留当前 + 3 旧版；撤回可回滚或下架，永久删除物理清理；完整备份恢复全部业务数据。支持 AVIF 等图片和外链、适配平台的视频链接嵌入，不上传或托管视频文件。详细边界以产品需求确认稿为准。

目前已完成需求研究与视觉原型，博客系统尚未实现。

已提供五套独立的 [博客视觉 HTML 原型](design-prototypes/index.html)，含首页、示例文章阅读页和可交互的搜索筛选，用于确认视觉方向。最新方案为「留白之间」，打开方式见 [原型说明](design-prototypes/README.md)。
