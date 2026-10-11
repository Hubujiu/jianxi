# 间隙 · 博客视觉原型

WeaveOS 方案统一采用黑白灰：选中背景、按钮、标签、时间轴节点、文章链接、公式容器和 Mermaid 图表均同步使用灰阶；主要按钮使用深灰底与白字。热力图与代码高亮也使用灰阶。

## WeaveOS 风格首页

新增 `weave/index.html`，沿用用户提供的 WeaveOS 视觉：浅灰左侧导航、白色圆角主面板与黑白灰操作元素。首页直接平铺文章，标签筛选与搜索可交互，文章链接复用已有示例阅读页。

预览地址：<http://127.0.0.1:4173/design-prototypes/weave/index.html>。桌面和手机原型图位于 `../output/playwright/jianxi-weave-desktop.png` 与 `../output/playwright/jianxi-weave-mobile.png`。已检查 1440、768、390、320px 布局、分类筛选、搜索、无结果提示和 Esc 关闭。

独立浏览页：`weave/categories.html`（全部分类及分类文章列表）、`weave/tags.html`（全部标签、标签查找及标签文章列表）、`weave/collections.html`（专栏合集与有序文章目录）。首页侧栏及手机导航可进入这些页面。新增专栏支持名称、介绍、至少两篇文章选择和上下调整阅读顺序；保存到当前浏览器 localStorage，刷新保留，未接后端发布。

`weave/catalog.css` 与 `weave/catalog.js` 为浏览页共享样式和交互。更新首页公共外壳后运行 `node design-prototypes/weave/generate-pages.mjs`，同步三个独立页面。浏览页检查脚本为 `../output/playwright/weave-catalog-review.cjs`，已验证目录、标签查找、专栏创建校验、文章排序、刷新保留与手机布局。

侧栏标签以圆角胶囊自动换行，较多时在标签区域滚动。侧栏支持收起为 64px 图标导航；收起后 Logo 悬停或键盘聚焦显示展开图标，点击展开。收起状态保存在当前浏览器，跨原型页面保留；手机继续使用顶部导航。

手机端 Logo 左侧新增菜单按钮，打开侧栏抽屉覆盖正文，配合遮罩及背景滚动锁定。支持关闭按钮、点击遮罩、Esc 关闭，选择导航或筛选后自动收起；键盘焦点留在抽屉内，关闭后回到菜单按钮。桌面侧栏的收起状态独立保留。

文章及排版样板的手机目录改为右侧圆形悬浮按钮，点击展开圆角目录面板，选择章节后自动关闭；支持面板关闭按钮、点击外部或 Esc 关闭。返回顶部仍只在滚动后出现，位于目录按钮下方。桌面继续显示加宽的右侧目录。

顶部为可点击面包屑，文章继承实际进入路径：首页进入显示首页 / 文章名称，分类进入保留全部分类 / 分类名，标签、专栏和时间线同样保留入口。通过会话存储传递入口，再将路径保存到当前浏览器历史记录；刷新与前进后退保留，下一篇沿用阅读入口。直接打开文章默认首页 / 文章名称。手机端独立成行，长标题省略显示。顶部关于图标已移除，关于我通过导航进入。

`weave/about.html` 为独立的关于我页面，包含作者介绍、博客名称由来、写作方向及阅读入口。侧栏、顶部关于入口和手机导航均可进入；作者资料为演示内容。

首页现简化为全宽文章列表，移除介绍横幅、顶部分类卡片及右侧作者、推荐阅读和话题卡片。分类、标签、专栏和关于我通过导航进入独立页面。

`weave/timeline.html` 为独立时间线页面，侧栏与手机导航均有入口。顶部以 365 天绿色热力图展示示例文章的首次发文数量，日期范围为 2025-10-09 至 2026-10-08；悬停显示日期及篇数，点击按日筛选，下方按年、月和发布日期编排文章。提供当天无文章状态与恢复全部记录，手机热力图可横向滚动。截图位于 `../output/playwright/jianxi-weave-timeline.png` 及 `../output/playwright/jianxi-weave-timeline-mobile.png`。

`weave/sample.html` 为文章排版样板，侧栏、首页文章列表标题旁和手机导航均有入口。展示 H1/H2/H3、段落与引用、Go/HTML 代码高亮和复制、行内及块级 LaTeX（含矩阵、多行对齐与积分）、Mermaid 流程图和时序图、列表及表格，附目录与可展开源码。它是样板入口，不计入 5 篇演示文章。

样板正文、KaTeX HTML/MathML、高亮代码与内嵌 Mermaid SVG 均预先生成；读取不依赖公式或图表渲染 JS，也不依赖外部 CDN。公式样式和字体随 `weave/assets/` 提供，`sample.css` 为阅读排版。复制、目录高亮等增强交互需要 JavaScript。构建依赖固定在 `../output/article-renderer/package.json`，运行 `node design-prototypes/weave/build-sample.mjs` 与 `node design-prototypes/weave/generate-pages.mjs` 重建文章；图源码变更后用 `../output/playwright/render-sample-diagrams.cjs` 重新导出 SVG，再重建。此页演示渲染效果，尚未接入 Go/Markdown 正式内容编译链。

五篇示例文章现均生成在 `weave/` 目录，使用与排版样板一致的黑白灰阅读样式、加宽目录、按滚动显示的返回顶部和下一篇导航。首页、搜索、分类、标签、专栏、时间线及关于页的文章入口均链接到这套阅读页。`generate-pages.mjs` 从已有示例数据读取原文，保留段落、章节、引用及阅读笔记；更新公共外壳后运行该脚本同步全部页面。

打开 `index.html` 查看五套方案，或在仓库根目录运行：

```powershell
node design-prototypes/preview-server.mjs
```

浏览器访问 <http://127.0.0.1:4173/design-prototypes/index.html>。服务器仅监听本机。无需安装依赖，也可直接双击 HTML 文件离线查看。

| 方案 | 入口 | 风格 |
| --- | --- | --- |
| A 纸上编辑部 | `editorial/index.html` | 暖纸色、宋体、杂志编排 |
| B 清简索引 | `minimal/index.html` | 冷白、侧栏导航、简洁列表 |
| C 午夜蓝图 | `midnight/index.html` | 深蓝、技术感、网格文章卡片 |
| D 庭院手记 | `garden/index.html` | 浅绿、植物图形、柔和圆角 |
| E 留白之间 | `studio/index.html` | 暖白与墨色、纸页交互、疏朗中文排版 |

每套提供首页及 5 篇完整示例文章，文章间保持相同内容便于比较。页面顶部的 A / B / C / D 可以保留当前文章并切换视觉方案。

交互包括分类与标签筛选、示例全文搜索（快捷键 `/`）、关于我弹窗、文章目录锚点、下一篇文章、返回顶部。弹窗支持 Esc 关闭和键盘操作。手机布局在 767px 及以下切换。

当前只是公开博客的视觉提案。站名、作者、文章和日期均为演示数据，搜索只检索内嵌的 5 篇示例内容；后台、MCP、发布、真正的搜索服务未接入。分类、标签、关于我使用原型弹窗，正式产品应按需求文档落为独立页面。四个风格入口用于评审，不代表增加产品主题系统。

## 文件维护

- `styles.css`：四套视觉样式及响应式规则。
- `interactions.js`：不依赖框架的原型交互。
- `generate.mjs`：示例数据、HTML 模板及总览页；修改后运行 `node design-prototypes/generate.mjs` 重新生成。
- `preview-server.mjs`：只服务原型及预览图片的本机预览服务器。
- `../output/playwright/`：真实浏览器截图与复查脚本。移动整个原型时保留相对目录，总览页需要这里的截图。

无外部字体、图片、CDN 或运行时网络依赖。标题使用本机中文宋体字体回退，正文使用系统中文无衬线字体；不同系统的字体外观可能略有不同。长文和链接为静态 HTML；搜索和筛选依赖 JavaScript。

## 验证

通过 Playwright 检查四套首页与代表文章在 1440、768、390、320px 宽度下的布局；验证分类、标签、正文搜索、无结果状态、键盘关闭、手机搜索、文章目录和方案切换。截图位于 `../output/playwright/`，浏览器检查脚本为 `review.cjs`。

```powershell
npx --yes --package @playwright/cli playwright-cli -s=jianxi-design open http://127.0.0.1:4173/design-prototypes/index.html
npx --yes --package @playwright/cli playwright-cli -s=jianxi-design run-code --filename=output/playwright/review.cjs
```
