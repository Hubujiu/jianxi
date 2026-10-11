/* One persistent surface per interaction. Timelines are interrupted at their
 * rendered values, so reversing a gesture never swaps or recreates the shape. */
(() => {
  const motion = window.JianxiMotion;
  if (!motion) return;
  const gsap = motion.engine;
  document.documentElement.classList.add('js');
  const $ = (s, r = document) => r.querySelector(s);
  const $$ = (s, r = document) => [...r.querySelectorAll(s)];
  const reduced = motion.reduced;
  const mobile = matchMedia('(max-width: 800px)');
  const spring = motion.ease.spring;
  const duration = () => motion.duration('spring');
  // Navigation owns page choreography; this module mounts content interactions.
  let disposeHover, disposeHeatHover;
  let locked = 0, savedOverflow = '', savedMainOverflow = '';
  function lock(value) { if(value){if(locked++ === 0){savedOverflow=document.body.style.overflow;document.body.style.overflow='hidden';savedMainOverflow=$('.main').style.overflow;if(!mobile.matches)$('.main').style.overflow='hidden'}}else if(locked>0&&--locked===0){document.body.style.overflow=savedOverflow;$('.main').style.overflow=savedMainOverflow} }
  function announce(message) { const el=$('#status'); if(!el)return;el.textContent=message;gsap.killTweensOf(el);gsap.timeline().to(el,{opacity:1,duration:motion.duration('feedback')}).to(el,{opacity:0,duration:motion.duration('fade')},'+='+motion.timing.hold); }
  // A non-modal description stays beside the term and leaves reading scrollable.
  const keywordSurface=document.createElement('div');
  keywordSurface.id='keyword-popover';keywordSurface.className='keyword-popover';keywordSurface.hidden=true;
  keywordSurface.setAttribute('role','dialog');keywordSurface.setAttribute('aria-labelledby','keyword-title');keywordSurface.setAttribute('aria-describedby','keyword-description');
  keywordSurface.innerHTML='<div class="keyword-popover-head"><h2 id="keyword-title"></h2><button type="button" class="keyword-close" aria-label="关闭关键词描述">×</button></div><p id="keyword-description"></p>';
  document.body.append(keywordSurface);
  const keywordTitle=$('h2',keywordSurface),keywordDescription=$('p',keywordSurface),keywordClose=$('button',keywordSurface);
  let keywordTarget=null;
  function closeKeyword(restoreFocus=false){
    if(!keywordTarget)return;
    const source=keywordTarget;keywordTarget=null;
    source.setAttribute('aria-expanded','false');source.removeAttribute('aria-controls');source.removeAttribute('aria-describedby');
    gsap.killTweensOf(keywordSurface);keywordSurface.hidden=true;
    if(restoreFocus&&source.isConnected)source.focus({preventScroll:true});
  }
  function positionKeyword(){
    if(!keywordTarget)return;
    if(!keywordTarget.isConnected){closeKeyword();return}
    const viewport=window.visualViewport;
    const left=viewport?.offsetLeft||0,top=viewport?.offsetTop||0,width=viewport?.width||innerWidth,height=viewport?.height||innerHeight;
    const box=keywordTarget.getBoundingClientRect(),reading=$('.main').getBoundingClientRect();
    if(box.bottom<Math.max(top,reading.top)||box.top>Math.min(top+height,reading.bottom)){closeKeyword(true);return}
    keywordSurface.style.width=Math.min(360,width-24)+'px';
    keywordSurface.style.maxHeight=Math.max(80,height-24)+'px';
    const surfaceHeight=keywordSurface.offsetHeight,surfaceWidth=keywordSurface.offsetWidth;
    const below=box.bottom+10;
    const y=below+surfaceHeight<=top+height-12?below:box.top-surfaceHeight-10;
    keywordSurface.style.left=Math.max(left+12,Math.min(box.left,left+width-surfaceWidth-12))+'px';
    keywordSurface.style.top=Math.max(top+12,Math.min(y,top+height-surfaceHeight-12))+'px';
  }
  function openKeyword(target){
    if(keywordTarget===target){closeKeyword(true);return}
    closeKeyword();closeSearch();if(tocOpen)closeToc();
    keywordTarget=target;keywordTitle.textContent=target.textContent;keywordDescription.textContent=target.dataset.keywordDescription;
    target.setAttribute('aria-expanded','true');target.setAttribute('aria-controls',keywordSurface.id);target.setAttribute('aria-describedby','keyword-description');
    gsap.killTweensOf(keywordSurface);gsap.set(keywordSurface,{clearProps:'opacity,transform'});
    keywordSurface.hidden=false;positionKeyword();
    if(!keywordTarget)return;
    keywordClose.focus({preventScroll:true});
    gsap.fromTo(keywordSurface,{opacity:0,y:4},{opacity:1,y:0,duration:motion.duration('fade'),ease:motion.ease.settle});
  }
  keywordClose.addEventListener('click',()=>closeKeyword(true));
  document.addEventListener('click',event=>{const target=event.target.closest('.prose .keyword-trigger');if(target)openKeyword(target)});
  document.addEventListener('pointerdown',event=>{
    if(keywordTarget&&!keywordSurface.contains(event.target)&&!event.target.closest('.keyword-trigger'))closeKeyword(keywordSurface.contains(document.activeElement));
  });
  document.addEventListener('keydown',event=>{if(keywordTarget&&event.key==='Escape'){event.preventDefault();closeKeyword(true)}});
  document.addEventListener('focusin',event=>{if(keywordTarget&&!keywordSurface.contains(event.target)&&event.target!==keywordTarget)closeKeyword()});
  document.addEventListener('scroll',event=>{if(!keywordSurface.contains(event.target))positionKeyword()},{capture:true,passive:true});
  addEventListener('resize',positionKeyword);
  window.visualViewport?.addEventListener('resize',positionKeyword);
  window.visualViewport?.addEventListener('scroll',positionKeyword);
  new ResizeObserver(positionKeyword).observe(keywordSurface);
  document.addEventListener('jianxi:navigate-start',()=>closeKeyword());
  reduced.addEventListener('change',()=>{if(keywordTarget&&reduced.matches){gsap.killTweensOf(keywordSurface);gsap.set(keywordSurface,{opacity:1,y:0})}});
  // The native modal owns focus containment and makes the reading page inert.
  const viewer=document.createElement('dialog');
  viewer.className='image-viewer';
  viewer.setAttribute('aria-label','图片放大预览');
  viewer.innerHTML='<button class="image-viewer-close" type="button" aria-label="关闭图片预览" autofocus>×</button><figure class="image-viewer-figure"><img class="image-viewer-image" alt=""><figcaption id="image-viewer-caption"></figcaption></figure>';
  document.body.append(viewer);
  const viewerImage=$('img',viewer),viewerCaption=$('figcaption',viewer),viewerFigure=$('figure',viewer);
  let viewedImage=null,viewerClosing=false;
  function imageName(image){
    const label=image.title.trim()||image.alt.trim();
    if(label)return label;
    try{return decodeURIComponent(new URL(image.currentSrc||image.src).pathname.split('/').pop())||'图片'}catch{return '图片'}
  }
  function imageOffset(){
    const source=viewedImage.getBoundingClientRect(),target=viewerImage.getBoundingClientRect();
    return {x:source.left+source.width/2-target.left-target.width/2,y:source.top+source.height/2-target.top-target.height/2,scaleX:source.width/target.width,scaleY:source.height/target.height};
  }
  function finishImageClose(){
    if(!viewer.open)return;
    gsap.killTweensOf([viewer,viewerImage,viewerCaption]);
    viewer.close();lock(false);viewerClosing=false;
    const source=viewedImage;viewedImage=null;
    viewerImage.removeAttribute('src');
    if(source?.isConnected)source.focus({preventScroll:true});
  }
  function closeImage(immediate=false){
    if(!viewer.open)return;
    if(immediate){finishImageClose();return}
    if(viewerClosing)return;
    viewerClosing=true;
    gsap.killTweensOf([viewer,viewerImage,viewerCaption]);
    gsap.to(viewer,{opacity:0,duration:motion.duration('fade'),onComplete:finishImageClose});
    gsap.to(viewerImage,{scale:.96,duration:motion.duration('fade'),ease:motion.ease.settle});
  }
  function openImage(image){
    if(!image.complete||!image.naturalWidth)return;
    closeSearch();if(drawerOpen)closeDrawer();if(tocOpen)closeToc();
    if(viewer.open)finishImageClose();
    viewedImage=image;viewerClosing=false;
    const name=imageName(image);
    viewerImage.width=image.naturalWidth;viewerImage.height=image.naturalHeight;
    viewerImage.src=image.currentSrc||image.src;viewerImage.alt=image.alt||name;
    viewerCaption.textContent=name;
    viewerImage.setAttribute('aria-describedby','image-viewer-caption');
    gsap.set([viewer,viewerImage,viewerCaption],{clearProps:'all'});
    viewer.showModal();lock(true);
    // Start at the in-article geometry, then expand into the viewport.
    gsap.set(viewerImage,{...imageOffset(),transformOrigin:'50% 50%'});
    gsap.set(viewer,{opacity:0});gsap.set(viewerCaption,{opacity:0});
    gsap.to(viewer,{opacity:1,duration:motion.duration('fade')});
    gsap.to(viewerImage,{x:0,y:0,scaleX:1,scaleY:1,duration:motion.duration('pageEnter'),ease:motion.ease.settle});
    gsap.to(viewerCaption,{opacity:1,duration:motion.duration('heading')});
  }
  $('.image-viewer-close',viewer).addEventListener('click',()=>closeImage());
  viewer.addEventListener('cancel',event=>{event.preventDefault();closeImage()});
  viewer.addEventListener('click',event=>{if(event.target===viewer||event.target===viewerImage||event.target===viewerFigure)closeImage()});
  function prepareImages(){
    $$('.prose img').forEach(image=>{
      const figure=image.closest('figure');
      const anchor=image.closest('a');
      const surface=anchor||image;
      if(!figure?.querySelector('figcaption')&&!surface.nextElementSibling?.classList.contains('article-image-caption')){
        const caption=document.createElement('span');
        caption.className='article-image-caption';caption.textContent=imageName(image);
        surface.after(caption);
      }
      if(anchor)return;
      image.classList.add('image-zoom-trigger');image.tabIndex=0;image.setAttribute('role','button');
      image.setAttribute('aria-haspopup','dialog');image.setAttribute('aria-label','放大图片：'+imageName(image));
    });
  }
  document.addEventListener('click',event=>{
    const image=event.target.closest('.image-zoom-trigger');if(image)openImage(image);
  });
  document.addEventListener('keydown',event=>{
    if((event.key==='Enter'||event.key===' ')&&event.target.matches('.image-zoom-trigger')){event.preventDefault();openImage(event.target)}
  });
  document.addEventListener('jianxi:before-swap',()=>closeImage(true));
  addEventListener('resize',()=>{
    if(!viewer.open)return;
    if(viewerClosing){finishImageClose();return}
    gsap.killTweensOf(viewerImage);gsap.set(viewerImage,{x:0,y:0,scaleX:1,scaleY:1});
  });
  reduced.addEventListener('change',()=>{
    if(!viewer.open||!reduced.matches)return;
    if(viewerClosing){finishImageClose();return}
    gsap.killTweensOf([viewer,viewerImage,viewerCaption]);
    gsap.set([viewer,viewerCaption],{opacity:1});gsap.set(viewerImage,{x:0,y:0,scaleX:1,scaleY:1});
  });
  function trap(event,root,close){if(event.key==='Escape'){event.preventDefault();close();return}if(event.key!=='Tab')return;const nodes=$$('a[href],button:not([disabled]),input,select,[tabindex="0"]',root).filter(el=>el.getClientRects().length&&!el.closest('[inert]')&&getComputedStyle(el).visibility!=='hidden');if(!nodes.length)return;const first=nodes[0],last=nodes.at(-1);if(event.shiftKey&&document.activeElement===first){event.preventDefault();last.focus()}else if(!event.shiftKey&&document.activeElement===last){event.preventDefault();first.focus()}}
  const shell=$('#app-shell'), sidebar=$('#sidebar'), main=$('.main');
  const scrollTop=()=>mobile.matches?scrollY:main.scrollTop;
  // Preserve the actual reading entry across next-article links and history.
  function restoreReadingContext(){
    if(document.body.dataset.page!=='article')return;
    try{
      const context=history.state?.readingContext||JSON.parse(sessionStorage.getItem('jianxi-reading-context')||'null');
      sessionStorage.removeItem('jianxi-reading-context');
      if(!context||!/^\/(categories|tags|collections|timeline)\//.test(context.path)||context.path.includes('\\'))return;
      history.replaceState({...history.state,readingContext:context},'');
      const nav=$('.breadcrumbs'),current=$('.breadcrumb-current');
      if(!nav||!current||$$('a',nav).some(link=>link.getAttribute('href')===context.path))return;
      const link=document.createElement('a');link.href=context.path;link.textContent=context.title;
      const slash=document.createElement('span');slash.className='slash';slash.textContent='/';
      nav.insertBefore(link,current);nav.insertBefore(slash,current);
    }catch{}
  }
  let searchOpen=false, drawerOpen=false, tocOpen=false;
  let closeSearch=()=>{};
  let toc=null,tocContent=null,tocTrigger=null,tocAnimation,disposeToc;
  const slot=$('.search-slot');
  if(slot){
    const surface=document.createElement('div');surface.className='search-surface';surface.dataset.state='closed';
    surface.innerHTML=`<button class="surface-trigger" aria-label="搜索文章" aria-haspopup="dialog" aria-expanded="false">${slot.innerHTML}</button>
      <div class="surface-content" inert>
        <div class="surface-head"><div><p class="eyebrow">FIND A THOUGHT</p><h2 id="surface-title">从一个想法，找到另一篇</h2></div>
          <button class="icon-button" data-close-search aria-label="关闭搜索" title="关闭搜索（Esc）"><svg viewBox="0 0 24 24" aria-hidden="true"><use href="/static/icons.svg#close"/></svg></button></div>
        <form class="surface-input" action="/search/"><svg viewBox="0 0 24 24" aria-hidden="true"><use href="/static/icons.svg#search"/></svg>
          <input type="search" name="q" aria-label="搜索关键词" placeholder="搜索标题、正文或标签…" autocomplete="off" maxlength="120"></form>
        <div class="surface-results" aria-live="polite"></div>
        <div class="surface-foot">
          <button class="icon-button" data-close-search aria-label="关闭搜索" title="关闭搜索（Esc）"><svg viewBox="0 0 24 24" aria-hidden="true"><use href="/static/icons.svg#close"/></svg></button>
          <a class="icon-button search-all" href="/search/" aria-label="查看全部搜索结果" title="查看全部搜索结果（Enter）"><svg viewBox="0 0 24 24" aria-hidden="true"><use href="/static/icons.svg#search-page"/></svg></a>
        </div>
      </div>`;
    document.body.append(surface);slot.style.visibility='hidden';slot.tabIndex=-1;slot.setAttribute('aria-hidden','true');
    const trigger=$('.surface-trigger',surface),content=$('.surface-content',surface),input=$('input',surface),results=$('.surface-results',surface);
    const backdrop=document.createElement('div');backdrop.className='search-backdrop';backdrop.hidden=true;document.body.append(backdrop);
    let animation,restoreFocus,request,requestNumber=0;
    function closedBox(){const r=slot.getBoundingClientRect();return {x:r.x,y:r.y,width:r.width,height:r.height,borderRadius:8,backgroundColor:'#ffffff'}}
    function openBox(){const width=Math.min(620,innerWidth-32),height=Math.min(480,innerHeight-64);return {x:(innerWidth-width)/2,y:Math.max(24,(innerHeight-height)*.34),width,height,borderRadius:22,backgroundColor:'#ffffff'}}
    gsap.set(surface,closedBox());
    function setSearch(open){
      if(open===searchOpen)return;if(open&&drawerOpen)closeDrawer();if(open&&tocOpen)closeToc();searchOpen=open;surface.dataset.state=open?'open':'closed';animation?.kill();gsap.killTweensOf(backdrop);
      trigger.setAttribute('aria-expanded',String(open));trigger.inert=open;content.inert=!open;
      if(open){restoreFocus=document.activeElement;shell.inert=true;lock(true);backdrop.hidden=false;surface.setAttribute('role','dialog');surface.setAttribute('aria-modal','true');surface.setAttribute('aria-labelledby','surface-title');gsap.set(content,{visibility:'visible'});input.focus({preventScroll:true});}
      else{shell.inert=false;lock(false);surface.removeAttribute('aria-modal');surface.removeAttribute('role');(restoreFocus?.isConnected?restoreFocus:trigger).focus({preventScroll:true});request?.abort();requestNumber++;}
      animation=gsap.timeline({defaults:{overwrite:'auto'}})
        .to(surface,{...(open?openBox():closedBox()),duration:duration(),ease:spring,boxShadow:open?'0 24px 100px rgba(0,0,0,0.12)':'0 0 0 rgba(0,0,0,0)'},0)
        .to(trigger,{opacity:open?0:1,filter:open?'blur(4px)':'blur(0px)',duration:motion.duration('trigger')},0)
        .to(content,{opacity:open?1:0,filter:open?'blur(0px)':'blur(4px)',duration:motion.duration('fade'),onComplete:()=>{if(!searchOpen)gsap.set(content,{visibility:'hidden'})}},open?motion.duration('lead'):0)
        .to(backdrop,{opacity:open?1:0,duration:motion.duration('fade'),onComplete:()=>{if(!searchOpen)backdrop.hidden=true}},0);
    }
    closeSearch=()=>setSearch(false);
    trigger.addEventListener('click',()=>setSearch(true));$$('[data-close-search]',surface).forEach(button=>button.addEventListener('click',()=>setSearch(false)));backdrop.addEventListener('click',()=>setSearch(false));surface.addEventListener('keydown',e=>{if(searchOpen)trap(e,surface,()=>setSearch(false))});
    document.addEventListener('keydown',e=>{if(((e.metaKey||e.ctrlKey)&&e.key.toLowerCase()==='k')||(e.key==='/'&&!/INPUT|TEXTAREA|SELECT/.test(document.activeElement.tagName))){e.preventDefault();setSearch(!searchOpen)}});
    input.addEventListener('input',async()=>{
      request?.abort();const q=input.value.trim(),ticket=++requestNumber;
      $('.search-all',surface).href='/search/'+(q?'?'+new URLSearchParams({q}):'');
      results.replaceChildren();if(!q)return;
      const p=document.createElement('p');p.className='search-hint';p.textContent='正在查找…';results.append(p);request=new AbortController();
      try{const response=await fetch('/api/search?q='+encodeURIComponent(q),{signal:request.signal});if(!response.ok)throw Error('search');const data=await response.json();if(ticket!==requestNumber)return;results.replaceChildren();for(const post of data.results){const a=document.createElement('a');a.href=post.url;a.append(document.createTextNode(post.title));const small=document.createElement('small');small.textContent=post.snippet;a.append(small);results.append(a)}if(!data.results.length){p.textContent='没有找到相关文章，换个关键词试试。';results.append(p)}}catch(e){if(e.name==='AbortError'||ticket!==requestNumber)return;p.textContent='暂时无法搜索，请稍后重试。';results.replaceChildren(p)}
    });
    const resize=()=>{animation?.kill();gsap.set(surface,searchOpen?openBox():closedBox());gsap.set(content,{opacity:searchOpen?1:0,filter:'none',visibility:searchOpen?'visible':'hidden'});gsap.set(trigger,{opacity:searchOpen?0:1,filter:'none'});gsap.set(backdrop,{opacity:searchOpen?1:0});backdrop.hidden=!searchOpen};
    addEventListener('resize',resize);const alignSearch=()=>{if(!searchOpen)gsap.set(surface,closedBox())};addEventListener('scroll',alignSearch,{passive:true});main.addEventListener('scroll',alignSearch,{passive:true});new ResizeObserver(alignSearch).observe($('.topbar'));reduced.addEventListener('change',resize);
  }
  let drawerAnimation;
  const shade=$('.drawer-shade'),menu=$('[data-menu]');
  function setDrawer(open){if(!mobile.matches||drawerOpen===open)return;drawerOpen=open;drawerAnimation?.kill();sidebar.classList.toggle('drawer-open',open);menu?.setAttribute('aria-expanded',String(open));$('.main').inert=open;sidebar.inert=!open;if(open){lock(true);shade.hidden=false;sidebar.setAttribute('role','dialog');sidebar.setAttribute('aria-modal','true');$('[data-menu-close]').focus()}else{lock(false);sidebar.removeAttribute('role');sidebar.removeAttribute('aria-modal');menu.focus()};drawerAnimation=gsap.timeline().to(sidebar,{x:open?0:'-100%',duration:duration(),ease:spring},0).to(shade,{opacity:open?1:0,duration:motion.duration('fade'),onComplete:()=>{if(!drawerOpen)shade.hidden=true}},0)}
  function closeDrawer(){setDrawer(false)}
  menu?.addEventListener('click',()=>setDrawer(true));$('[data-menu-close]')?.addEventListener('click',closeDrawer);shade?.addEventListener('click',closeDrawer);sidebar?.addEventListener('keydown',e=>{if(drawerOpen)trap(e,sidebar,closeDrawer)});
  const collapse=$('[data-collapse]');let collapsed=false,sideWidth=248;
  // Keep hints outside the sidebar's clipped scrolling and animation surface.
  const sidebarTooltip=document.createElement('div');
  sidebarTooltip.id='sidebar-tooltip';sidebarTooltip.className='sidebar-tooltip';sidebarTooltip.setAttribute('role','tooltip');sidebarTooltip.hidden=true;document.body.append(sidebarTooltip);
  let tooltipTarget;
  function hideSidebarTooltip(){sidebarTooltip.hidden=true;tooltipTarget?.removeAttribute('aria-describedby');tooltipTarget=null}
  function showSidebarTooltip(target){
    hideSidebarTooltip();if(!collapsed||mobile.matches)return;
    tooltipTarget=target;sidebarTooltip.textContent=target.getAttribute('aria-label');sidebarTooltip.hidden=false;target.setAttribute('aria-describedby',sidebarTooltip.id);
    const rect=target.getBoundingClientRect(),hint=sidebarTooltip.getBoundingClientRect();
    sidebarTooltip.style.left=Math.min(rect.right+10,innerWidth-hint.width-8)+'px';
    sidebarTooltip.style.top=Math.max(8,Math.min(rect.top+(rect.height-hint.height)/2,innerHeight-hint.height-8))+'px';
  }
  for(const target of [collapse,...$$('.nav-list .nav',sidebar),$('.profile',sidebar)].filter(Boolean)){
    if(target!==collapse)target.setAttribute('aria-label',$(target.matches('.profile')?'strong':'span',target).textContent.trim());
    target.addEventListener('pointerenter',event=>{if(event.pointerType!=='touch')showSidebarTooltip(target)});
    target.addEventListener('pointerleave',()=>{if(tooltipTarget===target)hideSidebarTooltip()});
    target.addEventListener('focus',()=>{if(target.matches(':focus-visible'))showSidebarTooltip(target)});
    target.addEventListener('blur',()=>{if(tooltipTarget===target)hideSidebarTooltip()});
    target.addEventListener('click',hideSidebarTooltip);
  }
  document.addEventListener('keydown',event=>{if(event.key==='Escape')hideSidebarTooltip()});
  document.addEventListener('pointerdown',hideSidebarTooltip);
  addEventListener('resize',hideSidebarTooltip);
  document.addEventListener('scroll',hideSidebarTooltip,{capture:true,passive:true});
  try{collapsed=localStorage.getItem('jianxi-sidebar-collapsed')==='true';sideWidth=Number(localStorage.getItem('jianxi-sidebar-width'))||248}catch{}
  sideWidth=Math.max(208,Math.min(300,sideWidth));
  function setCollapsed(value,animate=true){hideSidebarTooltip();collapsed=value;document.documentElement.dataset.sidebarCollapsed=String(value);shell.classList.toggle('sidebar-collapsed',value);collapse?.setAttribute('aria-expanded',String(!value));collapse?.setAttribute('aria-label',value?'展开侧栏':'收起侧栏');gsap.to(sidebar,{width:value?64:sideWidth,duration:animate?duration():0,ease:spring,overwrite:true});try{localStorage.setItem('jianxi-sidebar-collapsed',String(value))}catch{}}
  collapse?.addEventListener('click',()=>setCollapsed(!collapsed));if(sidebar&&!mobile.matches)setCollapsed(collapsed,false);
  const grip=$('.sidebar-resizer');let drag;
  grip?.addEventListener('pointerdown',e=>{if(e.button!==0||mobile.matches)return;gsap.killTweensOf(sidebar);drag={x:e.clientX,width:sidebar.getBoundingClientRect().width};grip.setPointerCapture(e.pointerId);document.body.style.userSelect='none'});
  grip?.addEventListener('pointermove',e=>{if(!drag)return;sideWidth=Math.max(208,Math.min(300,drag.width+e.clientX-drag.x));gsap.set(sidebar,{width:sideWidth});grip.setAttribute('aria-valuenow',String(Math.round(sideWidth)))});
  function endDrag(){if(!drag)return;drag=null;document.body.style.userSelect='';try{localStorage.setItem('jianxi-sidebar-width',String(sideWidth))}catch{}}
  grip?.addEventListener('pointerup',endDrag);grip?.addEventListener('pointercancel',endDrag);grip?.addEventListener('lostpointercapture',endDrag);
  grip?.addEventListener('keydown',e=>{if(e.key!=='ArrowLeft'&&e.key!=='ArrowRight')return;e.preventDefault();sideWidth=Math.max(208,Math.min(300,sideWidth+(e.key==='ArrowLeft'?-12:12)));setCollapsed(false);grip.setAttribute('aria-valuenow',String(sideWidth));try{localStorage.setItem('jianxi-sidebar-width',String(sideWidth))}catch{}});
  function setToc(open){if(!toc||!mobile.matches||tocOpen===open)return;tocOpen=open;tocAnimation?.kill();tocTrigger.setAttribute('aria-expanded',String(open));tocTrigger.inert=open;tocContent.inert=!open;if(open){gsap.set(tocContent,{visibility:'visible'});$('.toc-close',toc).focus()}else tocTrigger.focus();tocAnimation=gsap.timeline().to(toc,{width:open?Math.min(340,innerWidth-40):44,height:open?Math.min(110+toc.querySelectorAll('nav a').length*46,innerHeight*.6):44,borderRadius:open?'19px':'22px',backgroundColor:open?'#fff':'#292a28',color:open?'#292a28':'#fff',duration:duration(),ease:spring},0).to(tocContent,{opacity:open?1:0,filter:open?'blur(0px)':'blur(4px)',duration:motion.duration('contentExit'),onComplete:()=>{if(!tocOpen)gsap.set(tocContent,{visibility:'hidden'})}},open?motion.duration('lead'):0).to(tocTrigger,{opacity:open?0:1,duration:motion.duration('micro')},0)}
  function closeToc(){setToc(false)}
  document.addEventListener('pointerdown',e=>{if(tocOpen&&toc&&!toc.contains(e.target))closeToc()});
  function updateMedia(){if(drawerOpen){drawerOpen=false;lock(false);$('.main').inert=false;shade.hidden=true}drawerAnimation?.kill();sidebar?.removeAttribute('role');sidebar?.removeAttribute('aria-modal');if(sidebar){sidebar.inert=mobile.matches;gsap.set(sidebar,{clearProps:'transform'});if(!mobile.matches)setCollapsed(collapsed,false)}tocAnimation?.kill();tocOpen=false;if(toc){gsap.set(toc,{clearProps:'all'});gsap.set(tocContent,{clearProps:'all'});gsap.set(tocTrigger,{clearProps:'all'});tocContent.inert=mobile.matches;tocTrigger.inert=false;tocTrigger.setAttribute('aria-expanded','false')}}
  mobile.addEventListener('change',updateMedia);
  reduced.addEventListener('change',()=>{drawerAnimation?.progress(1);tocAnimation?.progress(1);gsap.killTweensOf(sidebar);if(!mobile.matches&&sidebar)gsap.set(sidebar,{width:collapsed?64:sideWidth})});
  const back=$('.back-top');function updateBack(){if(back)back.hidden=scrollTop()<420}addEventListener('scroll',updateBack,{passive:true});main.addEventListener('scroll',updateBack,{passive:true});
  function initPress(root){
  // Native buttons remain keyboard operable. A small press compression follows
  // genuine pointer input and is never applied to the morphing surfaces.
  $$('a.button,button.button,.catalog-card',root).forEach(el=>{el.addEventListener('pointerdown',e=>{if(e.button===0&&!reduced.matches)gsap.to(el,{scale:.985,duration:motion.duration('fast'),overwrite:true})});for(const event of ['pointerup','pointercancel','pointerleave','blur'])el.addEventListener(event,()=>gsap.to(el,{scale:1,duration:duration(),ease:spring,overwrite:true}))});
  }
  function unmountPage(){
    closeKeyword();
    disposeHeatHover?.();disposeHeatHover=null;
    disposeHover?.();disposeHover=null;
    tocAnimation?.kill();
    disposeToc?.();disposeToc=null;
    gsap.killTweensOf($$('#content *'));
    tocOpen=false;
    toc=tocContent=tocTrigger=null;
  }
  function bindHeatHover(grid){
    const events=new AbortController(),touched=new Set();
    const hover=matchMedia('(hover: hover)');
    let hovered=null;
    const scale=button=>!reduced.matches&&(hovered===button||button.matches(':focus-visible'))?1.08:1;
    function update(event){
      const button=event.target.closest('[data-heat-day]');
      if(!button||!grid.contains(button))return;
      if(event.type.startsWith('pointer')){
        if(event.pointerType==='touch'||!hover.matches||button.contains(event.relatedTarget))return;
        if(event.type==='pointerover')hovered=button;
        else if(hovered===button)hovered=null;
      }
      touched.add(button);
      gsap.to(button,{scale:scale(button),duration:motion.duration('feedback'),ease:motion.ease.standard,overwrite:'auto'});
    }
    for(const name of ['pointerover','pointerout','focusin','focusout'])
      grid.addEventListener(name,update,{signal:events.signal});
    reduced.addEventListener('change',()=>{
      for(const button of touched){gsap.killTweensOf(button,'scale');gsap.set(button,{scale:scale(button)})}
    },{signal:events.signal});
    return ()=>{
      events.abort();
      for(const button of touched){gsap.killTweensOf(button,'scale');gsap.set(button,{clearProps:'transform'})}
      touched.clear();
    };
  }
  function mountPage(){
    $$('.prose .keyword-trigger').forEach(target=>target.removeAttribute('title'));
    prepareImages();
    restoreReadingContext();
    toc=$('.toc');tocContent=$('.toc-content');tocTrigger=$('.toc-toggle');
    tocTrigger?.addEventListener('click',()=>setToc(true));$('.toc-close')?.addEventListener('click',closeToc);toc?.addEventListener('keydown',e=>{if(e.key==='Escape')closeToc()});toc?.addEventListener('click',e=>{if(e.target.closest('a'))closeToc()});
    $$('[data-copy]').forEach(button=>button.addEventListener('click',async()=>{try{await navigator.clipboard.writeText($('pre',button.closest('.code-block')).textContent);announce('代码已复制')}catch{announce('无法访问剪贴板，请手动选择代码')}}));
    if(toc){
      const nav=$('nav',toc),links=$$('a',nav),labels=links.map(link=>$('.toc-label',link));
      const headings=links.map(link=>document.getElementById(decodeURIComponent(link.hash.slice(1))));
      const indicator=document.createElement('span');indicator.className='toc-indicator';indicator.setAttribute('aria-hidden','true');nav.prepend(indicator);
      const events=new AbortController();let current=-1,frame=0,pinned=false;
      const markerBox=()=>({y:links[current].offsetTop+9,height:Math.max(18,links[current].offsetHeight-18)});
      function select(index,animate=true){
        if(index<0||index>=links.length||index===current)return;current=index;
        links.forEach((link,i)=>{
          const active=i===index;link.classList.toggle('active',active);
          if(active)link.setAttribute('aria-current','location');else link.removeAttribute('aria-current');
          gsap.to(labels[i],{scale:active?1.1:1,fontWeight:active?700:450,color:active?'#252523':'#85857e',duration:animate?motion.duration('spring'):0,ease:spring,overwrite:'auto'});
        });
        gsap.to(indicator,{...markerBox(),duration:animate?motion.duration('spring'):0,ease:spring,overwrite:'auto'});
      }
      function update(){
        frame=0;if(pinned||!links.length)return;
        const top=mobile.matches?0:main.getBoundingClientRect().top;
        const threshold=top+Math.min(150,(mobile.matches?innerHeight:main.clientHeight)*.22);
        let index=0;headings.forEach((heading,i)=>{if(heading&&heading.getBoundingClientRect().top<=threshold)index=i});
        const height=mobile.matches?innerHeight:main.clientHeight,total=mobile.matches?document.documentElement.scrollHeight:main.scrollHeight;
        if(total>height+2&&scrollTop()+height>=total-3)index=links.length-1;
        select(index);
      }
      function schedule(){if(!frame)frame=requestAnimationFrame(update)}
      nav.addEventListener('click',e=>{
        const link=e.target.closest('a');
        if(!link||e.button!==0||e.metaKey||e.ctrlKey||e.shiftKey||e.altKey)return;
        const index=links.indexOf(link),heading=headings[index];if(!heading)return;
        e.preventDefault();pinned=true;select(index);
        heading.scrollIntoView({block:'start'});
      },{signal:events.signal});
      function release(){pinned=false}
      addEventListener('wheel',release,{passive:true,signal:events.signal});addEventListener('touchmove',release,{passive:true,signal:events.signal});
      main.addEventListener('pointerdown',e=>{if(!toc.contains(e.target))release()},{signal:events.signal});
      addEventListener('keydown',e=>{if(['ArrowDown','ArrowUp','PageDown','PageUp','Home','End',' '].includes(e.key))release()},{signal:events.signal});
      addEventListener('scroll',schedule,{passive:true,signal:events.signal});main.addEventListener('scroll',schedule,{passive:true,signal:events.signal});
      const align=()=>{if(current>=0){gsap.killTweensOf(indicator);gsap.set(indicator,markerBox())}};
      const resize=new ResizeObserver(align);resize.observe(nav);addEventListener('resize',align,{signal:events.signal});
      reduced.addEventListener('change',()=>{if(current<0)return;gsap.killTweensOf([...labels,indicator]);labels.forEach((label,i)=>gsap.set(label,{scale:i===current?1.1:1,fontWeight:i===current?700:450,color:i===current?'#252523':'#85857e'}));align()},{signal:events.signal});
      const hashIndex=links.findIndex(link=>link.hash===location.hash);select(hashIndex>=0?hashIndex:0,false);
      if(hashIndex>=0&&headings[hashIndex]){
        headings[hashIndex].scrollIntoView({block:'start'});
        history.replaceState(history.state,'',location.pathname+location.search);
      }
      disposeToc=()=>{events.abort();resize.disconnect();cancelAnimationFrame(frame);gsap.killTweensOf([...labels,indicator]);indicator.remove()};
    }
    const heatDays=$$('[data-heat-day]'),dayReset=$('#reset-timeline');
    const heatGrid=$('.heat-grid');
    if(heatGrid)disposeHeatHover=bindHeatHover(heatGrid);
    function filterTimeline(day){
      let visible=0;
      $$('.timeline-entry').forEach(entry=>{entry.hidden=!!day&&entry.dataset.date!==day;if(!entry.hidden)visible++});
      $$('.timeline-month').forEach(month=>{month.hidden=!$$('.timeline-entry:not([hidden])',month).length});
      $$('.timeline-year').forEach(year=>{year.hidden=!$$('.timeline-month:not([hidden])',year).length});
      heatDays.forEach(button=>button.setAttribute('aria-pressed',String(button.dataset.heatDay===day)));
      $('#timeline-filter').hidden=!day;
      if(day)$('#timeline-filter-label').textContent=day.replaceAll('-','.');
      $('#timeline-day-empty').hidden=visible>0||!day;
      $('.timeline-end').hidden=!!day;
    }
    heatDays.forEach(button=>button.addEventListener('click',()=>filterTimeline(button.dataset.heatDay)));
    dayReset?.addEventListener('click',()=>filterTimeline(''));
    if(toc){tocContent.inert=mobile.matches;tocTrigger.inert=false;tocTrigger.setAttribute('aria-expanded','false')}
    updateBack();
    initPress($('#content'));
    disposeHover=motion.bindHover($('#content'),'.entry-content');
  }
  document.addEventListener('jianxi:navigate-start',()=>{closeSearch();if(drawerOpen)closeDrawer();if(tocOpen)closeToc()});
  document.addEventListener('jianxi:before-swap',unmountPage);
  document.addEventListener('jianxi:after-swap',mountPage);
  mountPage();
  initPress($('.topbar'));
  updateMedia();
})();
