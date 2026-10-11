import { createRequire } from 'node:module';
import { readFile, writeFile, mkdir, cp } from 'node:fs/promises';
const require=createRequire(new URL('../../output/article-renderer/package.json',import.meta.url));
const katex=require('katex'),hljs=require('highlight.js');
const asset=new URL('assets/',import.meta.url);await mkdir(asset,{recursive:true});
await cp(new URL('../../output/article-renderer/node_modules/katex/dist/katex.min.css',import.meta.url),new URL('katex.min.css',asset));
await cp(new URL('../../output/article-renderer/node_modules/katex/dist/fonts/',import.meta.url),new URL('fonts/',asset),{recursive:true});
await cp(new URL('../../output/article-renderer/node_modules/katex/LICENSE',import.meta.url),new URL('KATEX-LICENSE.txt',asset));
const escape=text=>text.replaceAll('&','&amp;').replaceAll('<','&lt;').replaceAll('>','&gt;');
const code=(source,language)=>`<div class="code-example"><div class="code-heading"><span>${language}</span><button type="button" class="copy-code">复制代码</button></div><pre><code class="hljs language-${language}">${hljs.highlight(source,{language}).value}</code></pre></div>`;
const math=(source,displayMode=false)=>katex.renderToString(source,{displayMode,output:'htmlAndMathml',trust:false,throwOnError:true,maxExpand:1000,maxSize:10});
const mathSource=String.raw`\begin{aligned}
\mathcal{L}(\theta) &= \frac{1}{n}\sum_{i=1}^{n}\bigl(y_i-f_{\theta}(x_i)\bigr)^2 \\
\theta_{t+1} &= \theta_t - \eta\nabla_{\theta}\mathcal{L}(\theta_t)
\end{aligned}`;
const flow=String.raw`flowchart LR
    A[Markdown 源内容] --> B[解析与校验]
    B --> C[代码高亮]
    B --> D[LaTeX 公式]
    B --> E[Mermaid 图]
    C --> F[静态文章页]
    D --> F
    E --> F`;
const sequence=String.raw`sequenceDiagram
    participant A as 作者
    participant B as 内容编译器
    participant C as 读者
    A->>B: 提交 Markdown
    B->>B: 编译公式与图表
    B-->>A: 返回文章预览
    A->>B: 确认发布
    B-->>C: 提供静态阅读页`;
await writeFile(new URL('flow.mmd',asset),flow);await writeFile(new URL('sequence.mmd',asset),sequence);
const goCode=`package main

import "fmt"

// 让内容编译一次，让每次阅读保持轻量。
func main() {
    topics := []string{"技术", "阅读", "生活"}
    for index, topic := range topics {
        fmt.Printf("%02d · %s\\n", index+1, topic)
    }
}`;
const htmlCode=`<article class="post">
  <h1>给想法一个生长的地方</h1>
  <p>这里展示的是 HTML 源码，不会作为页面执行。</p>
</article>`;
const content=`<div class="sample-reading"><div class="sample-reading-main"><a class="back-link" href="index.html">← 返回文章列表</a><header class="sample-heading"><div class="sample-eyebrow">TYPOGRAPHY & CONTENT</div><h1>文章排版样板</h1><p class="sample-deck">从一段文字，到一篇完整的文章。<br>看看标题、代码、公式与图表，在「间隙」里如何呈现。</p><div class="sample-byline"><span class="avatar">狐</span><span>狐不九</span><span>排版演示 · 2026.10.08</span><span>约 6 分钟阅读</span></div></header><article class="sample-prose">
<section id="headings"><div class="section-kicker">01 / TYPOGRAPHY</div><h2>二级标题：给内容一个清晰的层次</h2><p>上方的「文章排版样板」是本页的<strong>一级标题 H1</strong>，也是文章标题。正文以二级标题 H2 划分章节，三级标题 H3 组织章节内的细节。标题有不同的字号、字重与上下留白，让阅读可以自然地停顿。</p><h3 id="heading-three">三级标题：把一件事讲具体</h3><p>中文正文使用系统无衬线字体，保持舒适的行距与适中的行宽。<strong>重点可以加粗</strong>，<em>语气可以用斜体表达</em>，<a href="about.html">链接使用克制的灰色</a>。行内代码如 <code>renderToString()</code> 有单独的底色，帮助读者区分代码与正文。</p><blockquote><p>好的排版，让内容容易被读懂，也让思考有地方停留。</p></blockquote></section>
<section id="code"><div class="section-kicker">02 / CODE</div><h2>代码块：结构与颜色一起说明问题</h2><p>代码保留缩进与换行，标注语言，并以不同颜色区分关键字、字符串和注释。长行可在代码块内横向滚动，不会撑开整篇文章。</p>${code(goCode,'go')}<h3 id="html-source">HTML 示例以源码展示</h3><p>源码中的标签会显示成文字，便于讨论页面结构。</p>${code(htmlCode,'html')}</section>
<section id="math"><div class="section-kicker">03 / MATHEMATICS</div><h2>LaTeX 公式：让数学表达融入正文</h2><p>行内公式随文字流动，例如 ${math('E=mc^2')}，或 ${math('f(x)=x^2+2x+1')}。分数 ${math('\\frac{a}{b}')}、下标 ${math('x_i')} 和希腊字母 ${math('\\theta')} 也保持与正文协调的大小。</p><h3 id="display-math">独立公式与多行对齐</h3><p>较长的表达独立居中展示。下面用均方误差与梯度下降说明求和、下标、分数和多行对齐的效果。</p><div class="math-example">${math(mathSource,true)}</div><details class="source-details"><summary>查看 LaTeX 源码</summary>${code(mathSource,'latex')}</details><p>矩阵与积分也使用相同的排版规则：</p><div class="math-example math-pair"><div>${math(String.raw`\mathbf{A}=\begin{bmatrix}1&2\\3&4\end{bmatrix}`,true)}</div><div>${math(String.raw`\int_0^1 x^2\,\mathrm{d}x=\frac{1}{3}`,true)}</div></div></section>
<section id="mermaid"><div class="section-kicker">04 / DIAGRAMS</div><h2>Mermaid 图：把关系画出来</h2><p>图表来自 Mermaid 源码，编译为 SVG，文字与连线在放大后依然清晰。图形使用与页面一致的浅色背景和灰色强调。</p><h3 id="flowchart">流程图：内容如何变成文章</h3><figure class="diagram-example"><div class="diagram-scroll"><img src="assets/flow.svg" alt="Markdown 经解析校验后分别进行代码高亮、LaTeX 公式与 Mermaid 图编译，最终汇合为静态文章页。"></div><figcaption>图 1 · 从 Markdown 到静态文章页</figcaption></figure><details class="source-details"><summary>查看流程图 Mermaid 源码</summary>${code(flow,'plaintext')}</details><h3 id="sequence-diagram">时序图：预览与发布</h3><figure class="diagram-example"><div class="diagram-scroll"><img src="assets/sequence.svg" alt="作者提交 Markdown，内容编译器编译公式图表并返回预览，作者确认后向读者提供静态页面。"></div><figcaption>图 2 · 作者、编译器与读者之间的交互</figcaption></figure><details class="source-details"><summary>查看时序图 Mermaid 源码</summary>${code(sequence,'plaintext')}</details></section>
<section id="other"><div class="section-kicker">05 / DETAILS</div><h2>列表、表格与提示</h2><p>不同的内容，需要不同的表达方式。列表组织要点，表格帮助比较，提示保留补充信息。</p><ul><li>分类组织文章的主要方向。</li><li>标签连接跨分类的话题。</li><li>专栏把多篇文章组成一条阅读路径。</li></ul><div class="sample-table-scroll"><table class="sample-table"><thead><tr><th>内容类型</th><th>阅读效果</th><th>样板中的位置</th></tr></thead><tbody><tr><td>标题与正文</td><td>层次清晰，留白有节奏</td><td>本文标题及章节</td></tr><tr><td>代码</td><td>语法高亮，可复制</td><td>Go / HTML 示例</td></tr><tr><td>公式</td><td>行内、独立、多行对齐</td><td>KaTeX 静态渲染</td></tr><tr><td>图表</td><td>清晰的矢量图</td><td>Mermaid 流程与时序</td></tr></tbody></table></div><aside class="sample-note"><strong>关于这个样板</strong><p>本页为排版原型。公式已生成 HTML 与 MathML，图表已编译为 SVG；代码高亮也已生成。读者打开页面时，无需再运行公式或图表渲染器。</p></aside></section>
<div class="sample-end"><span>排版服务于内容。</span><p>让每一种表达，都有合适的位置。</p><a href="index.html">回到文章列表 →</a></div></article></div><aside class="sample-toc"><span class="eyebrow">ON THIS PAGE</span><h2>本文目录</h2><nav aria-label="文章目录"><a href="#headings">01 <span>标题与正文</span></a><a class="toc-child" href="#heading-three">三级标题</a><a href="#code">02 <span>代码块</span></a><a class="toc-child" href="#html-source">HTML 源码</a><a href="#math">03 <span>LaTeX 公式</span></a><a class="toc-child" href="#display-math">独立公式</a><a href="#mermaid">04 <span>Mermaid 图</span></a><a class="toc-child" href="#flowchart">流程图</a><a class="toc-child" href="#sequence-diagram">时序图</a><a href="#other">05 <span>列表与表格</span></a></nav><a class="toc-top" href="#sample-top" hidden>返回顶部 ↑</a><div id="copy-status" class="copy-status" role="status" aria-live="polite"></div></aside></div>`;
let staticContent=content;
for(const name of ['flow','sequence']){
  const svg=await readFile(new URL(name+'.svg',asset),'utf8');
  staticContent=staticContent.replace(new RegExp('<img src="assets/'+name+'\\.svg" alt="([^"]*)">'),(_,alt)=>svg.replace('<svg ',`<svg aria-label="${alt}" `).replace(/role="[^"]*"/,'role="img"'));
}
await writeFile(new URL('sample-content.html',import.meta.url),staticContent);
console.log('Generated static math, highlighted code, article HTML, and diagram sources.');
