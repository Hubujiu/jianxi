import { readFile, writeFile } from 'node:fs/promises';
const source = await readFile(new URL('index.html', import.meta.url), 'utf8');
function readingPage(content,title,page,slug){
  let html=source.replace('<body>',`<body data-page="${page}">`).replace('<title>间隙 · 博客首页原型</title>',`<title>${title} · 间隙</title>`);
  html=html.replace('</head>','<link rel="stylesheet" href="sample.css"></head>');
  const start=html.indexOf(' <div class="body">'),end=html.indexOf('</main>',start);
  html=html.slice(0,start)+' <div class="body" id="sample-top">'+content+'</div>\n'+html.slice(end);
  html=html.replace(/<nav class="breadcrumbs"[\s\S]*?<\/nav>/,`<nav class="breadcrumbs" aria-label="面包屑"><ol><li><a href="index.html"><svg viewBox="0 0 24 24" aria-hidden="true"><path d="m3 10 9-7 9 7v10a1 1 0 0 1-1 1h-5v-8H9v8H4a1 1 0 0 1-1-1Z"/></svg><span>首页</span></a></li><li><span class="breadcrumb-slash" aria-hidden="true">/</span><a href="${slug}.html" aria-current="page"><span>${title}</span></a></li></ol></nav>`);
  return html.replace('showPosts();','if(document.querySelector("#posts"))showPosts();');
}
for (const [page, title] of [['categories','全部分类'], ['tags','全部标签'], ['collections','专栏合集'], ['about','关于我'], ['timeline','时间线'], ['sample','文章排版样板']]) {
  let html = source.replace('<body>', `<body data-page="${page}">`).replace('<title>间隙 · 博客首页原型</title>', `<title>${title} · 间隙</title>`);
  if(page==='sample'){
    html=html.replace('</head>','<link rel="stylesheet" href="assets/katex.min.css"><link rel="stylesheet" href="sample.css"></head>');
    const content=await readFile(new URL('sample-content.html',import.meta.url),'utf8');
    const start=html.indexOf(' <div class="body">'),end=html.indexOf('</main>',start);
    html=html.slice(0,start)+' <div class="body" id="sample-top">'+content+'</div>\n'+html.slice(end);
    html=html.replace('showPosts();','if(document.querySelector("#posts"))showPosts();');
  }
  await writeFile(new URL(`${page}.html`, import.meta.url), html);
}
const originalHTML=await readFile(new URL('../studio/index.html',import.meta.url),'utf8');
const articleData=JSON.parse(originalHTML.match(/<script[^>]*id="post-data"[^>]*>([\s\S]*?)<\/script>/)[1]);
const esc=value=>String(value).replace(/[&<>"']/g,char=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[char]));
for(const [index,p] of articleData.entries()){
  const next=articleData[(index+1)%articleData.length],date=p.date.replaceAll('-','.');
  const content=`<div class="sample-reading"><div class="sample-reading-main"><a class="back-link" href="index.html">← 返回文章列表</a><header class="sample-heading"><a class="sample-eyebrow article-category" href="categories.html?name=${encodeURIComponent(p.category)}">${esc(p.category)}</a><h1>${esc(p.title)}</h1><p class="sample-deck">${esc(p.summary)}</p><div class="sample-byline"><span class="avatar">狐</span><span>狐不九</span><span>发布于 <time datetime="${p.date}">${date}</time></span><span>更新于 <time datetime="${p.date}">${date}</time></span></div></header><article class="sample-prose"><p class="article-lead">${esc(p.paragraphs[0][0])}</p>${p.headings.map((heading,i)=>`<section id="section-${i+1}"><div class="section-kicker">${String(i+1).padStart(2,'0')} / NOTES</div><h2>${esc(heading)}</h2>${p.paragraphs[i].filter((_,j)=>i!==0||j!==0).map(paragraph=>`<p>${esc(paragraph)}</p>`).join('')}${i===0?`<blockquote><p>${esc(p.quote)}</p></blockquote>`:''}${i===1?'<aside class="sample-note"><strong>留给自己的笔记</strong><p>先做一件具体的小事，再从真实的反馈里寻找下一步。</p></aside>':''}</section>`).join('')}<div class="sample-end"><span>写到这里，先告一段落。</span><div class="article-tags">${p.tags.map(tag=>`<a href="tags.html?name=${encodeURIComponent(tag)}"># ${esc(tag)}</a>`).join('')}</div><a class="article-next" href="${next.slug}.html"><small>接着读</small><strong>${esc(next.title)}</strong><span aria-hidden="true">↗</span></a></div></article></div><aside class="sample-toc"><span class="eyebrow">ON THIS PAGE</span><h2>本文目录</h2><nav aria-label="文章目录">${p.headings.map((heading,i)=>`<a href="#section-${i+1}">${String(i+1).padStart(2,'0')} <span>${esc(heading)}</span></a>`).join('')}</nav><a class="toc-top" href="#sample-top" hidden>返回顶部 ↑</a><div id="copy-status" class="copy-status" role="status" aria-live="polite"></div></aside></div>`;
  await writeFile(new URL(p.slug+'.html',import.meta.url),readingPage(content,esc(p.title),'article',p.slug));
}
