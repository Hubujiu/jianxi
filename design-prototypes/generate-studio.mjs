import { mkdir, writeFile } from 'node:fs/promises';
import path from 'node:path';

const icon = '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.4" aria-hidden="true"><circle cx="10.5" cy="10.5" r="6.5"/><path d="m16 16 4.5 4.5"/></svg>';
const arrow = '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.4" aria-hidden="true"><path d="M6 18 18 6M6 6h12v12"/></svg>';

function header(post) {
  return `<header class="studio-header"><a class="identity" href="index.html" aria-label="间隙，返回首页"><span class="brand-brackets" aria-hidden="true"><i></i><i></i></span><span class="brand-name">间隙</span><span class="signature">Jianxi.</span></a><nav class="studio-nav" aria-label="主导航"><a href="index.html" ${post?'':'aria-current="page"'}>文章</a><button data-dialog="categories">分类</button><button data-dialog="tags">标签</button><button data-dialog="about">关于</button></nav><button data-dialog="search" class="studio-search" aria-label="搜索文章">${icon}<span>搜索</span><kbd>/</kbd></button></header>`;
}
function paper() {
  return `<div class="paper-playground"><button class="kinetic-paper" aria-label="展开纸页" aria-pressed="false"><span class="paper-tilt"><span class="paper-stack">${Array.from({length:11},(_,i)=>`<i class="paper-leaf${i===8?' accent-leaf':''}" style="--leaf:${i};--angle:${(i-5)*6}deg" data-angle="${(i-5)*6}"></i>`).join('')}</span></span></button><span class="paper-caption" aria-hidden="true">点一下，让思绪展开 <span>↗</span></span><span id="paper-status" class="visually-hidden" role="status"></span></div>`;
}

function home(posts) {
  return `<main id="main" class="studio-main"><section class="studio-intro" aria-labelledby="intro-title"><div class="intro-copy"><p class="intro-caption">一个人的思考与实践</p><h1 id="intro-title">在字里，<br>也在<span class="gap-word">行间<span class="word-dot">。</span></span></h1><p class="intro-description">写一点技术，留一些观察。<br>给尚未成形的想法，一个慢慢生长的地方。</p></div>${paper()}</section><section id="writing" aria-label="文章列表"><div class="writing-heading"><h2>最近写下的</h2><div class="filters" aria-label="按分类筛选">${['全部','技术','随想','阅读','生活'].map((c,i)=>`<button data-filter="${c}" aria-pressed="${i===0}">${c}</button>`).join('')}</div><span class="filter-count" aria-live="polite">5 篇文章</span></div><p class="active-filter" hidden></p><div class="studio-posts">${posts.map(p=>`<article class="post-row" data-category="${p.category}" data-tags="${p.tags.join(',')}"><div class="studio-meta"><time datetime="${p.date}">${p.date.replaceAll('-','.')}</time><span>${p.category}</span></div><div class="row-content"><h3><a href="${p.slug}.html"><span class="post-title">${p.title}</span><span class="title-arrow" aria-hidden="true">${arrow}</span></a></h3><p>${p.summary}</p><div class="post-tags">${p.tags.map(t=>`<button data-tag="${t}"># ${t}</button>`).join('')}</div></div><span class="row-ink" aria-hidden="true"></span></article>`).join('')}</div></section><section class="closing-note"><span class="closing-brackets" aria-hidden="true">[&nbsp; ]</span><div><h2>未完，待续。</h2><p>有些答案，在下一次记录里。</p></div><button data-dialog="about">关于这个博客 ${arrow}</button></section></main>`;
}
function footer() {
  return `<footer class="studio-footer"><span>© 2026 间隙</span><span>保持好奇，容许留白。</span><a href="../index.html">其他视觉方案 ↗</a><a href="#top">回到顶部 ↑</a></footer>`;
}

export async function generateStudio({root,posts,article,dialogs,galleryHTML}) {
  const folder=path.join(root,'studio');
  await mkdir(folder,{recursive:true});
  function page(post) {
    const main=post?article(post,{id:'studio'}).replace('class="wrap main-content article-main"','class="studio-main article-main"'):home(posts);
    return `<!doctype html><html lang="zh-CN"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><meta name="robots" content="noindex,nofollow"><meta name="description" content="${post?post.summary:'间隙，一个记录技术、阅读与日常思考的地方。'}"><link rel="icon" href="data:,"><title>${post?post.title:'间隙'} · 留白之间</title><link rel="stylesheet" href="../styles.css"><link rel="stylesheet" href="studio.css"><script src="../vendor/gsap.min.js" defer></script><script src="../interactions.js" defer></script><script src="studio-motion.js" defer></script></head><body class="studio${post?' article-page':''}" id="top"><a class="skip-link" href="#main">跳到内容</a><div class="studio-shell">${header(post)}${main}${footer()}</div>${dialogs()}<script type="application/json" id="post-data">${JSON.stringify(posts).replaceAll('<','\\u003c')}</script><noscript><p class="no-script-note">文章可以正常阅读；搜索与分类筛选需要启用 JavaScript。</p></noscript></body></html>`;
  }
  await writeFile(path.join(folder,'index.html'),page());
  for(const p of posts) await writeFile(path.join(folder,`${p.slug}.html`),page(p));
  const latest=`<section class="latest-proposal"><a href="studio/index.html" class="latest-preview"><img src="../output/playwright/studio-desktop.png" alt="留白之间，原创纸页交互与疏朗的中文排版" width="1440" height="1040"></a><div class="latest-copy"><span>从你喜欢的气质，发展自己的表达</span><h2>E / 留白之间</h2><p>暖白与墨色、疏朗的中文排版、一笔陶土红。以「间隙」为线索，设计可以展开的纸页装置、括号标识，以及轻巧的标题和页面过渡。</p><a href="studio/index.html">体验新方案与动效 ↗</a><a href="studio/${posts[0].slug}.html">查看文章阅读页 ↗</a><small>气质参考：<a href="https://cgartlab.com/" target="_blank" rel="noreferrer">CGArtLab</a>；页面构图、标识和交互为独立设计。</small></div></section>`;
  const gallery=galleryHTML.replace('四种风格，同一个记录思考的地方。','新增「留白之间」，沿你喜欢的方向继续打磨。').replace('<div class="proposal-grid">',latest+'<div class="proposal-grid">');
  await writeFile(path.join(root,'index.html'),gallery);
}
