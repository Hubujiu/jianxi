# 间隙 · 博客视觉原型

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
