(() => {
  const posts = JSON.parse(document.querySelector('#post-data').textContent);
  const input = document.querySelector('#search-input');
  const dialogIDs = { search: 'search', categories: 'categories-dialog', tags: 'tags-dialog', about: 'about-dialog' };
  document.querySelectorAll('[data-dialog]').forEach(button => button.addEventListener('click', () => {
    const dialog = document.getElementById(dialogIDs[button.dataset.dialog]);
    dialog.showModal();
    if (button.dataset.dialog === 'search') input.focus();
  }));
  document.querySelectorAll('dialog').forEach(dialog => {
    dialog.querySelector('[data-close]').addEventListener('click', () => dialog.close());
    dialog.addEventListener('keydown', event => {
      if (event.key === 'Escape') { event.preventDefault(); event.stopPropagation(); dialog.close(); }
    });
    dialog.addEventListener('click', event => { if (event.target === dialog) {
      const r = dialog.getBoundingClientRect();
      if (event.clientX < r.left || event.clientX > r.right || event.clientY < r.top || event.clientY > r.bottom) dialog.close();
    } });
  });
  document.addEventListener('keydown', event => {
    if (event.key === '/' && !event.ctrlKey && !event.metaKey && !event.altKey && !event.target.closest('input,textarea,[contenteditable="true"]') && !document.querySelector('dialog[open]')) {
      event.preventDefault(); document.getElementById('search').showModal(); input.focus();
    }
  });
  function search() {
    const terms = input.value.trim().toLocaleLowerCase().split(/\s+/).filter(Boolean);
    const results = document.getElementById('search-results');
    results.replaceChildren();
    if (!terms.length) { document.getElementById('search-status').textContent = '输入关键词，找回一个想法。'; return; }
    const matches = posts.filter(p => terms.every(t => [p.title,p.summary,p.category,...p.tags,...p.headings,...p.paragraphs.flat(),p.quote].join(' ').toLocaleLowerCase().includes(t)));
    document.getElementById('search-status').textContent = matches.length ? `找到 ${matches.length} 篇文章` : '没有找到相关文章，试试其他关键词。';
    for (const p of matches) {
      const a = document.createElement('a'); a.href = `${p.slug}.html`; a.className = 'search-result';
      const meta = document.createElement('small'); meta.textContent = `${p.category} / ${p.date}`;
      const title = document.createElement('strong'); title.textContent = p.title;
      const snippet = document.createElement('p'); snippet.textContent = [...p.paragraphs.flat(), p.summary].find(t=>terms.some(term=>t.toLocaleLowerCase().includes(term))) || p.summary;
      a.append(meta,title,snippet); results.append(a);
    }
  }
  input.addEventListener('input',search);
  document.getElementById('search-form').addEventListener('submit',event=>{event.preventDefault();search();});
  const rows = [...document.querySelectorAll('.post-row')];
  function filter(category = '全部', tag = '', updateURL = true) {
    if (!rows.length) { location.href = `index.html?tag=${encodeURIComponent(tag)}#writing`; return; }
    let count = 0;
    for (const row of rows) { row.hidden = !((category === '全部' || row.dataset.category === category) && (!tag || row.dataset.tags.split(',').includes(tag))); if(!row.hidden) count++; }
    document.querySelectorAll('[data-filter]').forEach(b=>b.setAttribute('aria-pressed',String(b.dataset.filter===category && !tag)));
    document.querySelector('.filter-count').textContent = `${count} 篇文章`;
    const note = document.querySelector('.active-filter');
    note.hidden = !tag && count > 0; note.textContent = count === 0 ? '没有匹配的文章，点击「全部」重新浏览。' : `正在查看标签：${tag}。点击「全部」清除筛选。`;
    if(updateURL) {
      const url = new URL(location.href); url.search = '';
      if(tag) url.searchParams.set('tag',tag); else if(category!=='全部') url.searchParams.set('category',category);
      try { history.replaceState(null,'',url); } catch { /* Some browsers restrict file:// history changes; filtering still works. */ }
    }
  }
  document.querySelectorAll('[data-filter]').forEach(b=>b.addEventListener('click',()=>filter(b.dataset.filter)));
  document.querySelectorAll('[data-tag]').forEach(b=>b.addEventListener('click',()=>{filter('全部',b.dataset.tag);document.getElementById('writing').scrollIntoView();}));
  const params = new URLSearchParams(location.search);
  if(rows.length) filter(params.get('category') || '全部',params.get('tag') || '',false);
  const sections = document.querySelectorAll('.prose section[id]');
  if(sections.length && 'IntersectionObserver' in window) {
    const observer = new IntersectionObserver(entries => { entries.forEach(entry => { if(entry.isIntersecting) document.querySelectorAll('.toc nav a').forEach(a=>{
      if(a.hash===`#${entry.target.id}`) a.setAttribute('aria-current','location'); else a.removeAttribute('aria-current');
    }); }); },{rootMargin:'-10% 0px -65% 0px'});
    sections.forEach(s=>observer.observe(s));
  }
})();
