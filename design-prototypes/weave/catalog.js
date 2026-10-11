const catalogPage = document.body.dataset.page || 'home';
document.querySelectorAll('button[data-about]').forEach(button=>{const link=document.createElement('a');link.href='about.html';link.className=button.className;link.innerHTML=button.innerHTML;if(button.hasAttribute('aria-label'))link.setAttribute('aria-label',button.getAttribute('aria-label'));button.replaceWith(link)});
const escapeHTML = value => String(value).replace(/[&<>"']/g, char => ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[char]));
const categories = [
  { name:'技术', icon:'code', description:'从具体的问题出发，记录独立开发、工具实践与工作方法。' },
  { name:'随想', icon:'feather', description:'留下一些还在成形的想法，关于表达、选择与长期积累。' },
  { name:'阅读', icon:'book', description:'带着问题翻开一本书，把书页之外的发现留在这里。' },
  { name:'生活', icon:'coffee', description:'走慢一点，记录日常里那些细小却值得留下的瞬间。' }
];
const allTags = [...new Set(posts.flatMap(p => p.tags))];
document.querySelector('.sidebar-tags').innerHTML=allTags.map(tag=>`<button class="tag-pill" data-tag="${escapeHTML(tag)}">${escapeHTML(tag)}</button>`).join('');
document.querySelectorAll('.sidebar .nav').forEach(nav=>{
  const label=Array.from(nav.childNodes).filter(node=>node.nodeType===Node.TEXT_NODE).map(node=>node.textContent.trim()).join(' ').trim();
  nav.setAttribute('aria-label',nav.textContent.trim());nav.title=label;
  Array.from(nav.childNodes).filter(node=>node.nodeType===Node.TEXT_NODE&&node.textContent.trim()).forEach(node=>{const span=document.createElement('span');span.className='nav-label';span.textContent=node.textContent;node.replaceWith(span)});
});
const sidebarStorageKey='jianxi-weave-sidebar-collapsed';
function setSidebarCollapsed(collapsed){
  document.querySelector('.app').classList.toggle('sidebar-collapsed',collapsed);
  const expand=document.querySelector('#expand-sidebar');
  expand.disabled=!collapsed;expand.setAttribute('aria-expanded',String(!collapsed));expand.title=collapsed?'展开侧栏':'';
  document.querySelector('#collapse-sidebar').setAttribute('aria-expanded',String(!collapsed));
  try{localStorage.setItem(sidebarStorageKey,String(collapsed))}catch{}
}
try{setSidebarCollapsed(localStorage.getItem(sidebarStorageKey)==='true')}catch{setSidebarCollapsed(false)}
document.querySelector('#collapse-sidebar').addEventListener('click',()=>{if(mobileViewport.matches){setMobileDrawer(false);return}setSidebarCollapsed(true);document.querySelector('#expand-sidebar').focus()});
document.querySelector('#expand-sidebar').addEventListener('click',()=>{setSidebarCollapsed(false);document.querySelector('#collapse-sidebar').focus()});
const mobileViewport=matchMedia('(max-width:800px)'),appShell=document.querySelector('.app'),drawer=document.querySelector('.sidebar'),mainPanel=document.querySelector('.main');
const mobileMenuButton=document.createElement('button');mobileMenuButton.className='iconbutton mobile-menu-trigger';mobileMenuButton.type='button';mobileMenuButton.setAttribute('aria-label','打开导航菜单');mobileMenuButton.setAttribute('aria-controls','blog-sidebar');mobileMenuButton.setAttribute('aria-expanded','false');mobileMenuButton.innerHTML='<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M4 6h16M4 12h16M4 18h16"/></svg>';document.querySelector('.top').prepend(mobileMenuButton);
const drawerBackdrop=document.createElement('button');drawerBackdrop.className='mobile-drawer-backdrop';drawerBackdrop.type='button';drawerBackdrop.tabIndex=-1;drawerBackdrop.setAttribute('aria-label','关闭导航菜单');drawerBackdrop.hidden=true;appShell.append(drawerBackdrop);
let mobileDrawerOpen=false,desktopCollapsedBeforeDrawer=false,bodyOverflowBeforeDrawer='';
function setMobileDrawer(open){
  if(open===mobileDrawerOpen||open&&!mobileViewport.matches)return;
  mobileDrawerOpen=open;mobileMenuButton.setAttribute('aria-expanded',String(open));drawerBackdrop.hidden=!open;
  if(open){
    desktopCollapsedBeforeDrawer=appShell.classList.contains('sidebar-collapsed');bodyOverflowBeforeDrawer=document.body.style.overflow;
    appShell.classList.remove('sidebar-collapsed');appShell.classList.add('mobile-drawer-open');mainPanel.inert=true;document.body.style.overflow='hidden';
    drawer.setAttribute('role','dialog');drawer.setAttribute('aria-modal','true');drawer.setAttribute('aria-label','博客导航菜单');
    document.querySelector('#expand-sidebar').disabled=true;const closeButton=document.querySelector('#collapse-sidebar');closeButton.setAttribute('aria-label','关闭导航菜单');closeButton.innerHTML=icon('close');closeButton.focus();
  }else{
    appShell.classList.remove('mobile-drawer-open');mainPanel.inert=false;document.body.style.overflow=bodyOverflowBeforeDrawer;
    drawer.removeAttribute('role');drawer.removeAttribute('aria-modal');drawer.setAttribute('aria-label','博客导航');
    const closeButton=document.querySelector('#collapse-sidebar');closeButton.setAttribute('aria-label','收起侧栏');closeButton.innerHTML='<svg viewBox="0 0 24 24" aria-hidden="true"><rect x="3" y="4" width="18" height="16" rx="2"/><path d="M9 4v16m7-11-3 3 3 3"/></svg>';
    setSidebarCollapsed(desktopCollapsedBeforeDrawer);if(mobileViewport.matches)mobileMenuButton.focus();
  }
}
mobileMenuButton.addEventListener('click',()=>setMobileDrawer(!mobileDrawerOpen));drawerBackdrop.addEventListener('click',()=>setMobileDrawer(false));
drawer.addEventListener('click',event=>{if(mobileDrawerOpen&&event.target.closest('a,.nav,[data-tag]'))setMobileDrawer(false)},true);
document.addEventListener('keydown',event=>{
  if(!mobileDrawerOpen)return;
  if(event.key==='Escape'){event.preventDefault();setMobileDrawer(false)}
  if(event.key==='Tab'){const controls=[...drawer.querySelectorAll('a[href],button:not([disabled])')].filter(el=>el.getClientRects().length);const first=controls[0],last=controls.at(-1);if(event.shiftKey&&document.activeElement===first){event.preventDefault();last.focus()}else if(!event.shiftKey&&document.activeElement===last){event.preventDefault();first.focus()}}
});
mobileViewport.addEventListener('change',()=>{if(!mobileViewport.matches)setMobileDrawer(false)});
document.querySelectorAll('.search-trigger').forEach(button=>{[...button.childNodes].filter(node=>node.nodeType===Node.TEXT_NODE&&node.textContent.trim()).forEach(node=>{const span=document.createElement('span');span.className='search-label';span.textContent=node.textContent;node.replaceWith(span)})});
document.addEventListener('click',event=>{const tag=event.target.closest('.sidebar-tags [data-tag]');if(tag)document.querySelectorAll('.sidebar-tags [data-tag]').forEach(button=>button.classList.toggle('current',button===tag));else if(event.target.closest('.sidebar .nav,.metric'))document.querySelectorAll('.sidebar-tags [data-tag]').forEach(button=>button.classList.remove('current'))});
const activeTag=catalogPage==='tags'?new URLSearchParams(location.search).get('name'):null;
document.querySelectorAll('.sidebar-tags [data-tag]').forEach(button=>button.classList.toggle('current',button.dataset.tag===activeTag));
const storageKey = 'jianxi-weave-collections-v1';
const seedCollections = [
  {id:'building-small',name:'把想法做成小作品',description:'从开始记录，到动手开发，再到和 AI 一起工作。把零散的实践串成一条线。',articles:['a-place-for-thoughts','smaller-software','working-with-ai']},
  {id:'leave-some-space',name:'给生活留一点空白',description:'读书、散步与观察。关于慢下来以后，生活里那些重新被看见的事。',articles:['reading-with-questions','an-unplanned-afternoon']}
];
let collections = structuredClone(seedCollections);
try {
  const saved = JSON.parse(localStorage.getItem(storageKey));
  if (Array.isArray(saved)) collections = saved.filter(c => c && typeof c.id==='string' && typeof c.name==='string' && typeof c.description==='string' && Array.isArray(c.articles) && c.articles.length >= 2 && c.articles.every(slug=>posts.some(p=>p.slug===slug)));
} catch {}
function updateCollectionCount(){document.querySelectorAll('[data-collection-count]').forEach(el=>el.textContent=collections.length)}
updateCollectionCount();
const mobileNav = document.createElement('nav');
mobileNav.className='mobile-catalog-nav'; mobileNav.setAttribute('aria-label','浏览内容');
mobileNav.innerHTML=[['index.html','首页','home'],['timeline.html','时间线','timeline'],['categories.html','分类','categories'],['tags.html','标签','tags'],['collections.html','专栏','collections'],['about.html','关于','about'],['sample.html','排版样板','sample']].map(([url,label,page])=>`<a href="${url}" class="${catalogPage===page?'active':''}">${label}</a>`).join('');
document.querySelector('.top').after(mobileNav);
const pageNames={categories:'全部分类',tags:'全部标签',collections:'专栏合集',about:'关于我',timeline:'时间线',sample:'排版样板',article:'文章阅读'};
const readingTrailKey='jianxi-weave-pending-reading-trail';
const homeTrail=()=>[{label:'首页',href:'index.html'}];
function validTrail(trail){return Array.isArray(trail)&&trail.length>0&&trail.length<=8&&trail.every(item=>{if(!item||typeof item.label!=='string'||typeof item.href!=='string')return false;try{const url=new URL(item.href,location.href);return url.origin===location.origin&&url.pathname.startsWith('/design-prototypes/weave/')}catch{return false}})}
function rememberReadingTrail(target){
  const url=new URL(target,location.href);
  const trail=[...document.querySelectorAll('.breadcrumbs a')].map(link=>({label:link.querySelector('span')?.textContent||link.textContent,href:link.getAttribute('href')}));
  if(catalogPage==='article')trail.pop();
  try{sessionStorage.setItem(readingTrailKey,JSON.stringify({target:url.pathname,trail:validTrail(trail)?trail:homeTrail(),createdAt:Date.now()}))}catch{}
}
function getReadingTrail(){
  let trail;
  try{const pending=JSON.parse(sessionStorage.getItem(readingTrailKey));if(pending?.target===location.pathname&&Date.now()-pending.createdAt<30000&&validTrail(pending.trail)){trail=pending.trail;sessionStorage.removeItem(readingTrailKey)}}catch{}
  if(!trail&&validTrail(history.state?.weaveReadingTrail))trail=history.state.weaveReadingTrail;
  trail=trail||homeTrail();
  try{history.replaceState({...history.state,weaveReadingTrail:trail},'',location.href)}catch{}
  return trail;
}
document.addEventListener('click',event=>{
  const link=event.target.closest('a[href]');
  if(link){const url=new URL(link.href,location.href);if(url.origin===location.origin&&url.pathname.startsWith('/design-prototypes/weave/')&&posts.some(post=>url.pathname.endsWith('/'+post.slug+'.html')))rememberReadingTrail(url.href)}
  const random=event.target.closest('button[data-random]');if(random){event.stopImmediatePropagation();const target=href(posts[Math.floor(Math.random()*posts.length)]);rememberReadingTrail(target);location.href=target}
},true);
function pageHeader(title,description,eyebrow,button='') {return `<section class="catalog-head"><div><div class="eyebrow">${eyebrow}</div><h1>${escapeHTML(title)}</h1><p>${escapeHTML(description)}</p></div>${button}</section>`}
function catalogFooter(){return '<footer class="page-foot"><span>© 2026 间隙 · Jianxi.</span><span>保持好奇，容许留白。</span><span>所有内容均为演示</span></footer>'}
function postMarkup(p){return `<article class="post"><div class="post-meta"><span class="category">${p.category}</span><span class="meta-dot"></span><time datetime="${p.date.replaceAll('.','-')}">${p.date}</time></div><h3><a href="${href(p)}">${p.title}</a></h3><p>${p.summary}</p><div class="post-tags">${p.tags.map(t=>`<a href="tags.html?name=${encodeURIComponent(t)}"># ${t}</a>`).join('')}</div></article>`}
function updateBreadcrumbs(){
  const trail=catalogPage==='article'?getReadingTrail():homeTrail(),query=new URLSearchParams(location.search);
  if(catalogPage==='article'){trail.push({label:document.querySelector('.sample-heading h1')?.textContent||'文章阅读',href:location.pathname.split('/').pop()})}
  else if(catalogPage!=='home'){
    const base=catalogPage+'.html';trail.push({label:pageNames[catalogPage]||'页面',href:base});
    if((catalogPage==='categories'||catalogPage==='tags')&&query.get('name'))trail.push({label:query.get('name'),href:base+location.search});
    if(catalogPage==='collections'&&query.get('id'))trail.push({label:collections.find(item=>item.id===query.get('id'))?.name||'专栏详情',href:base+location.search});
  }
  document.querySelector('.breadcrumbs ol').innerHTML=trail.map((item,index)=>`<li>${index?'<span class="breadcrumb-slash" aria-hidden="true">/</span>':''}<a href="${escapeHTML(item.href)}" ${index===trail.length-1?'aria-current="page"':''} title="${escapeHTML(item.label)}">${index===0?icon('home'):''}<span>${escapeHTML(item.label)}</span></a></li>`).join('');
}
function setPageContent(content){document.querySelector('.body').innerHTML=content+catalogFooter();document.querySelectorAll('.sidebar .nav').forEach(n=>n.classList.toggle('current',n.dataset.pageLink===catalogPage));updateBreadcrumbs();document.title=pageNames[catalogPage]+' · 间隙'}
function taxonomyDetail(name,tag){const found=posts.filter(p=>tag?p.tags.includes(name):p.category===name);const meta=categories.find(c=>c.name===name);const url=tag?'tags.html':'categories.html';setPageContent(`<a class="back-link" href="${url}">${icon('arrow')}返回${tag?'全部标签':'全部分类'}</a>`+pageHeader(tag?'# '+name:name,tag?'围绕这个话题，看看有哪些相关的想法。':meta?.description||'这个分类还没有文章。',tag?'EXPLORE A TOPIC':'EXPLORE A CATEGORY')+`<div class="detail-layout"><section class="card"><div class="card-top"><h2>${found.length} 篇文章</h2><span class="caption">按发布时间排列</span></div>${found.map(postMarkup).join('')||'<p class="empty">没有找到文章，试试其他分类或标签。</p>'}</section><aside class="card detail-aside"><h2>${tag?'其他标签':'其他分类'}</h2><div class="tag-cloud">${(tag?allTags:categories.map(c=>c.name)).filter(t=>t!==name).map(t=>`<a class="control" href="${url}?name=${encodeURIComponent(t)}">${tag?'# ':''}${t}</a>`).join('')}</div><p>从不同的入口，发现彼此关联的记录。</p><a class="control" href="collections.html">看看专栏合集 ${icon('right')}</a></aside></div>`)}
function renderCategories(){const name=new URLSearchParams(location.search).get('name');if(name)return taxonomyDetail(name,false);setPageContent(pageHeader('全部分类','按内容方向整理文章，从你感兴趣的地方开始。','EXPLORE BY CATEGORY')+`<div class="catalog-toolbar"><span>4 个分类 · 5 篇文章</span><span>每篇文章归属一个主分类</span></div><div class="catalog-grid">${categories.map((c,i)=>{const found=posts.filter(p=>p.category===c.name);return `<a class="card category-card" href="categories.html?name=${encodeURIComponent(c.name)}"><div class="category-card-top"><span class="appmark ${i===0?'accent':''}">${icon(c.icon)}</span><span class="caption">${found.length} 篇文章</span></div><h2>${c.name}</h2><p>${c.description}</p><div class="category-card-bottom"><span>最近更新 · ${found[0]?.date||'暂无文章'}</span><span>浏览文章 ${icon('right')}</span></div></a>`}).join('')}</div><div class="catalog-note"><strong>想连着读一个主题？</strong> 专栏可以把不同分类的文章组织成合集。<a href="collections.html">看看专栏合集 →</a></div>`)}
function renderTags(){const name=new URLSearchParams(location.search).get('name');if(name)return taxonomyDetail(name,true);setPageContent(pageHeader('全部标签','一个关键词，可以连接不同分类里的想法。','FOLLOW A THREAD')+`<div class="catalog-toolbar"><span id="tag-total">${allTags.length} 个标签 · 5 篇文章</span><input class="local-search" id="tag-filter" type="search" placeholder="查找标签…" aria-label="查找标签"></div><div class="tags-grid" id="tag-list"></div><div class="catalog-note">标签连接交叉话题，一篇文章可以关联多个标签。点击任意标签，查看相关的全部文章。</div>`);renderTagCards('');document.querySelector('#tag-filter').addEventListener('input',e=>renderTagCards(e.target.value))}
function renderTagCards(query){const list=allTags.filter(t=>t.toLowerCase().includes(query.trim().toLowerCase()));document.querySelector('#tag-list').innerHTML=list.map(t=>{const found=posts.filter(p=>p.tags.includes(t));return `<a class="card tag-card" href="tags.html?name=${encodeURIComponent(t)}"><span class="appmark">${icon('hash')}</span><span><h2>${t}</h2><small>${found.length} 篇文章 · ${[...new Set(found.map(p=>p.category))].join(' / ')}</small></span>${icon('right')}</a>`}).join('')||'<p class="catalog-empty">没有找到标签，换个关键词试试。</p>';document.querySelector('#tag-total').textContent=query.trim()?`找到 ${list.length} 个标签`:`${allTags.length} 个标签 · 5 篇文章`}
function renderCollections(){const id=new URLSearchParams(location.search).get('id');if(id)return renderCollectionDetail(id);setPageContent(pageHeader('专栏合集','把多篇文章编排在一起，沿着一个主题慢慢读。','CURATED STORIES',`<button class="control primary" data-new-collection>＋ 新增专栏</button>`)+`<div class="catalog-toolbar"><span>${collections.length} 个专栏 · 按阅读顺序编排</span><span>跨分类组织文章</span></div><div class="catalog-grid">${collections.map((c,i)=>`<a class="card collection-card" href="collections.html?id=${encodeURIComponent(c.id)}"><div class="collection-cover ${i%2?'blue':''}"><span class="cover-label">COLLECTION / ${String(i+1).padStart(2,'0')}</span><span class="cover-pages" aria-hidden="true"><i style="--n:-1"></i><i style="--n:0"></i><i style="--n:1"></i></span></div><div class="collection-content"><h2>${escapeHTML(c.name)}</h2><p>${escapeHTML(c.description)}</p><div class="collection-samples">${c.articles.slice(0,3).map((slug,n)=>`<span><b>${String(n+1).padStart(2,'0')}</b>${posts.find(p=>p.slug===slug).title}</span>`).join('')}</div><div class="collection-bottom"><span>${c.articles.length} 篇文章</span><span>进入专栏 ${icon('right')}</span></div></div></a>`).join('')}</div><div class="catalog-note"><strong>一个主题，一条阅读路径。</strong> 同一篇文章可以出现在多个专栏里。新增专栏时，选择文章并调整它们的阅读顺序。</div><p class="prototype-status">原型演示：新增专栏保存在当前浏览器，可刷新查看。</p>`)}
function renderCollectionDetail(id){const c=collections.find(c=>c.id===id);if(!c){setPageContent(pageHeader('专栏不存在','这个专栏可能没有保存在当前浏览器。','COLLECTION')+'<a class="control" href="collections.html">返回全部专栏</a>');return}document.title=c.name+' · 间隙';setPageContent(`<a class="back-link" href="collections.html">${icon('arrow')}返回全部专栏</a>`+pageHeader(c.name,c.description,'A COLLECTION OF IDEAS')+`<div class="detail-layout"><section class="card"><div class="card-top"><h2>专栏目录</h2><span class="caption">${c.articles.length} 篇文章 · 按阅读顺序</span></div>${c.articles.map((slug,i)=>`<div class="collection-post"><span class="ordinal">${String(i+1).padStart(2,'0')}</span>${postMarkup(posts.find(p=>p.slug===slug))}</div>`).join('')}</section><aside class="card detail-aside"><h2>关于这个合集</h2><p>${escapeHTML(c.description)}</p><div class="quiet-rule"></div><span class="caption">涵盖分类</span><div class="tag-cloud">${[...new Set(c.articles.map(slug=>posts.find(p=>p.slug===slug).category))].map(name=>`<a class="control" href="categories.html?name=${encodeURIComponent(name)}">${name}</a>`).join('')}</div><a class="control primary" href="${href(posts.find(p=>p.slug===c.articles[0]))}">从第一篇开始 ${icon('arrow')}</a></aside></div>`);document.title=c.name+' · 间隙'}
function renderAbout(){setPageContent(pageHeader('关于我','认识写下这些文字的人，也认识这个博客。','BEHIND THE WORDS')+`<div class="about-layout"><div class="about-main"><section class="card about-intro"><div class="about-identity"><span class="avatar">狐</span><div><span class="eyebrow">HELLO, I'M HUBUJIU</span><h2>你好，我是狐不九。</h2><p>开发者 · 写作者 · 终身学习者</p></div></div><p class="about-lead">喜欢简单的软件，<br>也喜欢复杂的想法。</p><p>我喜欢从具体的问题出发，动手做一些小作品。在实践中理解工具，在阅读中发现问题，也在日常生活里，留意那些容易被忽略的细节。</p><p>这里是我的个人博客「间隙」。记录技术、阅读与日常思考，给尚未成形的想法，一个慢慢生长的地方。</p><div class="about-signature"><span>保持好奇，容许留白。</span><span>狐不九 / Jianxi.</span></div></section><section class="card about-story"><div class="card-top"><h2>为什么叫「间隙」</h2>${icon('feather')}</div><p>有些想法，需要一点空白才会发生。工作之间、书页之间、日常的片刻停顿，都是重新观察和思考的机会。</p><p>写下来，是为了让这些发现留下来。这里的文字不急着给出答案，而是记录一个人不断学习和尝试的过程。</p><a class="text-button" href="${href(posts[0])}">读读这个博客的开始 ${icon('right')}</a></section></div><aside class="about-aside"><section class="card"><div class="card-top"><h2>我在这里写些什么</h2>${icon('file')}</div>${categories.map((c,i)=>`<a class="recommended" href="categories.html?name=${encodeURIComponent(c.name)}"><span class="appmark ${i===0?'accent':''}">${icon(c.icon)}</span><span><strong>${c.name}</strong><small>${['独立开发与工具实践','还在成形的想法','书页之外的发现','值得留下的日常'][i]}</small></span>${icon('right')}</a>`).join('')}</section><section class="card light about-reading"><div class="card-top"><h2>从这里开始认识我</h2>${icon('book')}</div><p>沿着文章，或者顺着一个合集，慢慢了解我的思考与实践。</p><a class="control" href="index.html">看看最新文章 ${icon('right')}</a><a class="control" href="collections.html">浏览专栏合集 ${icon('right')}</a></section><p class="prototype-status">作者与个人介绍为原型演示内容。</p></aside></div>`)}
if(catalogPage==='categories')renderCategories();if(catalogPage==='tags')renderTags();if(catalogPage==='collections')renderCollections();if(catalogPage==='about')renderAbout();
if(catalogPage==='timeline')renderTimeline();
if(catalogPage==='sample'||catalogPage==='article'){
  Promise.resolve(document.querySelector('.body').innerHTML).then(content=>{
    setPageContent(content);document.querySelector('.body').id='sample-top';document.title=document.querySelector('.sample-heading h1').textContent+' · 间隙';
    if(catalogPage==='article'){const allPosts=document.querySelector('.sidebar [data-filter="全部"]:not([data-nav])');allPosts.classList.add('current')}
    document.querySelectorAll('.copy-code').forEach(button=>button.addEventListener('click',async()=>{try{await navigator.clipboard.writeText(button.closest('.code-example').querySelector('code').textContent);document.querySelector('#copy-status').textContent='代码已复制';button.textContent='已复制';setTimeout(()=>button.textContent='复制代码',1800)}catch{document.querySelector('#copy-status').textContent='无法访问剪贴板，请选中代码后复制。'}}));
    const tocLinks=[...document.querySelectorAll('.sample-toc nav a')];
    const backToTop=document.querySelector('.toc-top');
    const tocPanel=document.querySelector('.sample-toc');tocPanel.id='article-toc';
    const tocOrb=document.createElement('button');tocOrb.className='mobile-toc-orb';tocOrb.type='button';tocOrb.setAttribute('aria-label','展开文章目录');tocOrb.setAttribute('aria-controls','article-toc');tocOrb.setAttribute('aria-expanded','false');tocOrb.innerHTML='<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M9 6h11M9 12h11M9 18h11M4 6h.01M4 12h.01M4 18h.01"/></svg>';document.querySelector('.body').append(tocOrb);
    const tocClose=document.createElement('button');tocClose.type='button';tocClose.className='iconbutton mobile-toc-close';tocClose.setAttribute('aria-label','关闭目录');tocClose.innerHTML=icon('close');tocPanel.prepend(tocClose);
    const articleTitle=document.querySelector('.sample-heading h1');
    const alignTocWithTitle=()=>{if(!mobileViewport.matches)return;const rect=articleTitle.getBoundingClientRect();const top=Math.max(140,Math.min(rect.top+window.scrollY+rect.height/2-22,window.innerHeight-160));document.querySelector('.body').style.setProperty('--mobile-toc-top',top+'px')};
    new ResizeObserver(alignTocWithTitle).observe(articleTitle);window.addEventListener('resize',alignTocWithTitle);alignTocWithTitle();
    const mobileBackToTop=document.createElement('a');mobileBackToTop.className='mobile-reading-top';mobileBackToTop.href='#sample-top';mobileBackToTop.setAttribute('aria-label','返回顶部');mobileBackToTop.innerHTML='<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M12 20V4m-6 6 6-6 6 6"/></svg>';mobileBackToTop.hidden=true;document.querySelector('.body').append(mobileBackToTop);
    let tocOpen=false;
    const toggleMobileToc=open=>{tocOpen=open&&mobileViewport.matches;tocPanel.classList.toggle('mobile-toc-open',tocOpen);tocOrb.setAttribute('aria-expanded',String(tocOpen));tocOrb.setAttribute('aria-label',tocOpen?'收起文章目录':'展开文章目录');if(tocOpen)tocLinks[0]?.focus()};
    tocOrb.addEventListener('click',()=>toggleMobileToc(!tocOpen));tocClose.addEventListener('click',()=>{toggleMobileToc(false);tocOrb.focus()});
    tocLinks.forEach(link=>link.addEventListener('click',()=>{if(mobileViewport.matches){toggleMobileToc(false);document.querySelector(link.hash)?.setAttribute('tabindex','-1');document.querySelector(link.hash)?.focus({preventScroll:true})}}));
    document.addEventListener('click',event=>{if(tocOpen&&!tocPanel.contains(event.target)&&!tocOrb.contains(event.target))toggleMobileToc(false)});
    document.addEventListener('keydown',event=>{if(tocOpen&&event.key==='Escape'){event.preventDefault();toggleMobileToc(false);tocOrb.focus()}});
    mobileViewport.addEventListener('change',()=>toggleMobileToc(false));
    const updateBackToTop=()=>{backToTop.hidden=window.scrollY<240;mobileBackToTop.hidden=backToTop.hidden};
    window.addEventListener('scroll',updateBackToTop,{passive:true});updateBackToTop();
    const observer=new IntersectionObserver(entries=>{const visible=entries.filter(entry=>entry.isIntersecting).sort((a,b)=>a.boundingClientRect.top-b.boundingClientRect.top)[0];if(visible)tocLinks.forEach(link=>{const active=link.hash==='#'+visible.target.id;link.classList.toggle('active',active);if(active)link.setAttribute('aria-current','location');else link.removeAttribute('aria-current')})},{rootMargin:'-10% 0px -60% 0px'});
    tocLinks.forEach(link=>{const section=document.querySelector(link.hash);if(section)observer.observe(section)});
    if(location.hash){document.getElementById(location.hash.slice(1))?.scrollIntoView()}
  }).catch(()=>setPageContent('<p class="empty">样板页暂时无法读取，请通过本机预览服务器打开。</p>'));
}
function renderTimeline(){
  const end=new Date('2026-10-08T00:00:00Z'),start=new Date(end);start.setUTCDate(start.getUTCDate()-364);
  const iso=date=>date.toISOString().slice(0,10);
  const counts=new Map();posts.forEach(p=>{const day=p.date.replaceAll('.','-');counts.set(day,(counts.get(day)||0)+1)});
  const inRange=posts.filter(p=>{const day=p.date.replaceAll('.','-');return day>=iso(start)&&day<=iso(end)});
  const activeDays=new Set(inRange.map(p=>p.date)).size;
  const gridStart=new Date(start);gridStart.setUTCDate(gridStart.getUTCDate()-gridStart.getUTCDay());
  const numCells=Math.ceil((365+start.getUTCDay())/7)*7,columns=numCells/7;
  let cells='',months='',lastMonth=-1;
  for(let i=0;i<numCells;i++){
    const day=new Date(gridStart);day.setUTCDate(day.getUTCDate()+i);const key=iso(day),count=counts.get(key)||0,within=day>=start&&day<=end;
    if(within&&day.getUTCMonth()!==lastMonth){const col=Math.floor(i/7)+1;if(col<=columns-1)months+=`<span style="grid-column:${col}/span 2">${day.getUTCMonth()+1}月</span>`;lastMonth=day.getUTCMonth()}
    const label=`${key} · ${count} 篇文章`;
    cells+=within?`<button class="heat-cell heat-${Math.min(count,4)}" data-heat-day="${key}" aria-label="${label}" title="${label}" aria-pressed="false"></button>`:'<span class="heat-cell heat-padding" aria-hidden="true"></span>';
  }
  setPageContent(pageHeader('时间线','每一次记录，都是思考留下的坐标。','A TRACE OF THOUGHTS')+`<section class="card writing-rhythm" aria-labelledby="rhythm-title"><div class="rhythm-heading"><div><span class="eyebrow">WRITING RHYTHM</span><h2 id="rhythm-title">写作的日常，慢慢累积。</h2></div><span class="rhythm-range">2025.10.09 — 2026.10.08</span></div><div class="rhythm-stats"><div><strong>${inRange.length}<small>篇</small></strong><span>过去一年发布</span></div><div><strong>${activeDays}<small>天</small></strong><span>留下了新记录</span></div><div class="rhythm-message"><span class="rhythm-spark">✦</span><p>不必每天写下什么。<br>值得留下的，终会成为一条线。</p></div></div><div class="heatmap-scroll"><div class="heatmap-layout" style="--heat-columns:${columns}"><div class="heat-months">${months}</div><div class="heat-weekdays"><span>周日</span><span>周三</span><span>周六</span></div><div class="heat-grid" aria-label="过去 365 天每日发文热力图">${cells}</div></div></div><div class="rhythm-foot"><span>每一格代表一天 · 点击查看当天文章</span><div class="heat-legend"><span>少</span>${[0,1,2,3,4].map(level=>`<i class="heat-cell heat-${level}"></i>`).join('')}<span>多</span></div></div></section><section class="timeline-archive" aria-labelledby="archive-title"><div class="archive-heading"><div><h2 id="archive-title">沿着时间，重新遇见。</h2><span class="caption">按首次发布时间倒序 · 共 ${posts.length} 篇文章</span></div><div class="timeline-filter" id="timeline-filter" hidden><span id="timeline-filter-label"></span><button class="text-button" id="reset-timeline">查看全部 ×</button></div></div><div id="timeline-entries"></div><div class="timeline-end"><span></span><p>记录还在继续</p><small>下一个想法，会在这里相遇。</small></div></section>`);
  renderTimelineEntries(posts);
  document.querySelectorAll('[data-heat-day]').forEach(button=>button.addEventListener('click',()=>{
    const day=button.dataset.heatDay;
    document.querySelectorAll('[data-heat-day]').forEach(cell=>cell.setAttribute('aria-pressed',String(cell===button)));
    document.querySelector('#timeline-filter').hidden=false;document.querySelector('#timeline-filter-label').textContent=day.replaceAll('-','.');
    renderTimelineEntries(posts.filter(p=>p.date.replaceAll('.','-')===day));
  }));
  document.querySelector('#reset-timeline').addEventListener('click',()=>{document.querySelector('#timeline-filter').hidden=true;document.querySelectorAll('[data-heat-day]').forEach(cell=>cell.setAttribute('aria-pressed','false'));renderTimelineEntries(posts)});
}
function renderTimelineEntries(items){
  const grouped=new Map();[...items].sort((a,b)=>b.date.localeCompare(a.date)).forEach(p=>{const year=p.date.slice(0,4),month=p.date.slice(5,7);if(!grouped.has(year))grouped.set(year,new Map());const months=grouped.get(year);if(!months.has(month))months.set(month,[]);months.get(month).push(p)});
  document.querySelector('#timeline-entries').innerHTML=[...grouped].map(([year,months])=>`<div class="timeline-year"><div class="year-label"><strong>${year}</strong><span>${[...months.values()].flat().length} 篇记录</span></div><div class="year-track">${[...months].map(([month,articles])=>`<section class="timeline-month"><div class="month-marker"><span class="month-node"></span><h3>${Number(month)} 月</h3><span>${articles.length} 篇</span></div>${articles.map(p=>`<article class="timeline-entry"><div class="entry-date"><span class="entry-node"></span><time datetime="${p.date.replaceAll('.','-')}">${p.date.slice(5).replace('.',' / ')}</time></div><a class="entry-content" href="${href(p)}"><div class="entry-top"><h4>${p.title}</h4>${icon('right')}</div><p>${p.summary}</p><div class="entry-meta"><span class="entry-category">${p.category}</span>${p.tags.map(tag=>`<span># ${tag}</span>`).join('')}</div></a></article>`).join('')}</section>`).join('')}</div></div>`).join('')||'<div class="timeline-empty"><span class="appmark">'+icon('feather')+'</span><p>这一天没有发布文章。</p><small>留白，也是记录的一部分。</small></div>';
}
document.addEventListener('click',e=>{const button=e.target.closest('button');if(!button)return;if(catalogPage!=='home'&&(button.dataset.filter||button.dataset.tag)){e.stopImmediatePropagation();if(button.dataset.tag)location.href='tags.html?name='+encodeURIComponent(button.dataset.tag);else location.href=button.dataset.filter==='全部'?'index.html':'categories.html?name='+encodeURIComponent(button.dataset.filter)}},true);
const createDialog=document.createElement('dialog');createDialog.id='collection-dialog';createDialog.className='collection-dialog';createDialog.setAttribute('aria-labelledby','collection-dialog-title');document.body.append(createDialog);
let chosenArticles=[];
function openCreateDialog(){chosenArticles=[];createDialog.innerHTML=`<form id="collection-form"><div class="dialog-top"><h2 id="collection-dialog-title">新增专栏</h2><button class="iconbutton" type="button" data-cancel-collection aria-label="关闭新增专栏">${icon('close')}</button></div><p class="form-help">把相关文章组成合集，给读者一条清晰的阅读路径。</p><label class="form-field"><span>专栏名称</span><input id="collection-name" name="name" required maxlength="60" placeholder="例如：从想法到小作品" autocomplete="off"></label><label class="form-field"><span>专栏介绍</span><textarea id="collection-description" name="description" maxlength="180" placeholder="这个合集关于什么？"></textarea></label><div class="selection-top"><span>选择文章</span><span class="caption">至少选择 2 篇</span></div><div>${posts.map(p=>`<label class="pick-row"><input type="checkbox" value="${p.slug}" data-pick-article><span>${p.title}<small>${p.category} · ${p.date}</small></span></label>`).join('')}</div><div class="selection-top"><span>阅读顺序</span><span class="caption" id="selection-count">已选 0 篇</span></div><div id="selection-order"><p class="form-help">选中的文章会出现在这里，可上下调整顺序。</p></div><p class="form-error" id="collection-error" role="alert" hidden></p><div class="form-footer"><button type="button" class="control" data-cancel-collection>取消</button><button type="submit" class="control primary">创建专栏</button></div></form>`;createDialog.showModal()}
function renderSelection(){document.querySelector('#selection-count').textContent=`已选 ${chosenArticles.length} 篇`;document.querySelector('#selection-order').innerHTML=chosenArticles.length?chosenArticles.map((slug,i)=>`<div class="selected-row"><span>${String(i+1).padStart(2,'0')}</span><span class="selected-title">${posts.find(p=>p.slug===slug).title}</span><button type="button" data-move-index="${i}" data-direction="-1" aria-label="上移第 ${i+1} 篇" ${i===0?'disabled':''}>↑</button><button type="button" data-move-index="${i}" data-direction="1" aria-label="下移第 ${i+1} 篇" ${i===chosenArticles.length-1?'disabled':''}>↓</button></div>`).join(''):'<p class="form-help">选中的文章会出现在这里，可上下调整顺序。</p>'}
document.addEventListener('change',e=>{if(!e.target.matches('[data-pick-article]'))return;const slug=e.target.value;if(e.target.checked)chosenArticles.push(slug);else chosenArticles=chosenArticles.filter(x=>x!==slug);renderSelection();document.querySelector('#collection-error').hidden=true});
document.addEventListener('click',e=>{const b=e.target.closest('button');if(!b)return;if(b.hasAttribute('data-new-collection'))openCreateDialog();if(b.hasAttribute('data-cancel-collection'))createDialog.close();if(b.dataset.moveIndex!==undefined){const i=Number(b.dataset.moveIndex),next=i+Number(b.dataset.direction);if(next<0||next>=chosenArticles.length)return;[chosenArticles[i],chosenArticles[next]]=[chosenArticles[next],chosenArticles[i]];renderSelection()}});
document.addEventListener('submit',e=>{if(e.target.id!=='collection-form')return;e.preventDefault();const name=document.querySelector('#collection-name').value.trim();const error=document.querySelector('#collection-error');function fail(message){error.textContent=message;error.hidden=false}if(!name)return fail('请填写专栏名称。');if(chosenArticles.length<2)return fail('请至少选择 2 篇文章，组成一个合集。');if(collections.some(c=>c.name===name))return fail('已有同名专栏，请换一个名称。');const c={id:crypto.randomUUID(),name,description:document.querySelector('#collection-description').value.trim(),articles:[...chosenArticles]};try{localStorage.setItem(storageKey,JSON.stringify([...collections,c]))}catch{return fail('当前浏览器无法保存专栏，请检查存储权限后重试。')}collections.push(c);createDialog.close();updateCollectionCount();location.href='collections.html?id='+encodeURIComponent(c.id)});
createDialog.addEventListener('click',e=>{if(e.target===createDialog){const r=createDialog.getBoundingClientRect();if(e.clientX<r.left||e.clientX>r.right||e.clientY<r.top||e.clientY>r.bottom)createDialog.close()}});
