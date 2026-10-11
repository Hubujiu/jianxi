/* Public pages remain complete HTML documents. Enhance ordinary same-origin
 * navigation by retaining the shell and fetching only the next page's content. */
(() => {
  const motion = window.JianxiMotion;
  if (!motion || document.body.dataset.prefix) return;
  const gsap = motion.engine;
  const sidebar = document.querySelector('#sidebar');
  const content = document.querySelector('#content');
  const main = document.querySelector('.main');
  if (!sidebar || !content || !main || !window.AbortController) return;
  const mobile = matchMedia('(max-width: 800px)');
  const reduced = motion.reduced;
  const spring = motion.ease.spring;
  const pageSequence = motion.createPageSequence(content, document.querySelector('.topbar'));
  const normalize = path => path.replace(/\/$/, '') || '/';
  const publicURL = url => url.origin === location.origin &&
    !/^\/(admin|_control|api|media|static)(\/|$)/.test(url.pathname);
  const indicator = document.createElement('div');
  indicator.className = 'nav-indicator';
  indicator.setAttribute('aria-hidden', 'true');
  sidebar.prepend(indicator);
  const status = document.createElement('div');
  status.className = 'sr-only';
  status.setAttribute('role', 'status');
  status.setAttribute('aria-live', 'polite');
  document.body.append(status);
  let activeLink, indicatorTween, indicatorTarget, request, ticket = 0;
  let displayedURL = new URL(location.href);
  let navigating = false;

  function selectedLink(url, context = history.state?.readingContext) {
    const links = [...sidebar.querySelectorAll('a.nav')];
    const exact = path => links.find(link => normalize(new URL(link.href).pathname) === normalize(path));
    const direct = exact(url.pathname);
    if (direct) return direct;
    let path = url.pathname;
    if (path.startsWith('/posts/') && context?.path) path = context.path;
    const contextual = exact(path);
    if (contextual) return contextual;
    const section = path.match(/^\/(categories|tags|collections|timeline)\//)?.[1];
    return exact(section ? `/${section}/` : '/');
  }

  function alignIndicator(animate = false) {
    if (!activeLink) return;
    const target = activeLink.getBoundingClientRect();
    const box = sidebar.getBoundingClientRect();
    const scroller = activeLink.closest('.sidebar-extra');
    const clip = scroller?.getBoundingClientRect();
    const visible = target.height > 0 && (!clip || (target.top >= clip.top && target.bottom <= clip.bottom));
    if (!visible) { indicator.hidden = true; return; }
    indicator.hidden = false;
    const position = {x: target.left - box.left, y: target.top - box.top, width: target.width, height: target.height};
    if (animate && !reduced.matches && indicatorTween?.isActive() &&
        Object.keys(position).every(key => Math.abs(position[key] - indicatorTarget?.[key]) < .01)) return;
    indicatorTarget = position;
    if (animate && !reduced.matches) {
      indicatorTween = gsap.to(indicator, {...position, duration: motion.duration('spring'), ease: spring, overwrite: true});
    } else {
      indicatorTween?.kill();
      gsap.set(indicator, position);
    }
  }

  function select(url, animate, context) {
    activeLink = selectedLink(url, context);
    sidebar.querySelectorAll('a.nav').forEach(link => {
      const current = link === activeLink;
      link.classList.toggle('current', current);
      if (current) link.setAttribute('aria-current', normalize(new URL(link.href).pathname) === normalize(url.pathname) ? 'page' : 'location');
      else link.removeAttribute('aria-current');
    });
    alignIndicator(animate);
    sidebar.classList.add('has-nav-indicator');
  }
  select(displayedURL, false);
  new ResizeObserver(() => alignIndicator(false)).observe(sidebar);
  sidebar.querySelector('.sidebar-extra')?.addEventListener('scroll', () => alignIndicator(false), {passive: true});
  addEventListener('resize', () => alignIndicator(false));
  reduced.addEventListener('change', () => alignIndicator(false));

  function saveScroll() {
    if (navigating) return;
    history.replaceState({...history.state, jianxi: true, scroll: mobile.matches ? {x: scrollX, y: scrollY} : {x: main.scrollLeft, y: main.scrollTop}}, '', location.href);
  }
  history.scrollRestoration = 'manual';
  saveScroll();
  let scrollFrame;
  function queueSaveScroll() {
    if (scrollFrame) return;
    scrollFrame = requestAnimationFrame(() => { scrollFrame = null; saveScroll(); });
  }
  addEventListener('scroll', queueSaveScroll, {passive: true});
  main.addEventListener('scroll', queueSaveScroll, {passive: true});

  function syncSidebar(next) {
    // Reconcile actual publication changes without rebuilding an unchanged nav.
    sidebar.querySelectorAll('.nav-list a').forEach(link => {
      const source = [...next.querySelectorAll('.nav-list a')].find(item => item.getAttribute('href') === link.getAttribute('href'));
      const counter = link.querySelector('small');
      if (source && counter) counter.textContent = source.querySelector('small')?.textContent || '';
    });
    const signature = element => {
      const clone = element.cloneNode(true);
      clone.querySelectorAll('.current,[aria-current]').forEach(link => { link.classList.remove('current'); link.removeAttribute('aria-current'); });
      return clone.innerHTML;
    };
    for (const selector of ['.sidebar-extra', '.brand-name', '.profile']) {
      const current = sidebar.querySelector(selector), source = next.querySelector(selector);
      if (current && source && signature(current) !== signature(source)) {
        const scroll = current.scrollTop;
        current.innerHTML = source.innerHTML;
        current.scrollTop = scroll;
      }
    }
  }

  function syncHead(next) {
    document.title = next.title;
    for (const selector of ['meta[name="description"]', 'meta[name="robots"]', 'link[rel="canonical"]', 'script[type="application/ld+json"]']) {
      const old = document.head.querySelector(selector), source = next.head.querySelector(selector);
      if (source) { if (old) old.replaceWith(source.cloneNode(true)); else document.head.append(source.cloneNode(true)); }
      else old?.remove();
    }
  }

  function syncBreadcrumbs(next) {
    const current = document.querySelector('.breadcrumbs');
    const incoming = next.querySelector('.breadcrumbs');
    if (!current || !incoming || current.innerHTML === incoming.innerHTML) return;
    const home = current.firstElementChild;
    const nextHome = incoming.firstElementChild;
    if (!home?.matches('a') || !nextHome?.matches('a') || !home.isEqualNode(nextHome)) {
      current.replaceChildren(...incoming.childNodes);
      return;
    }
    // Keep the persistent home link mounted while the page-specific crumbs change.
    [...current.childNodes].filter(node => node !== home).forEach(node => node.remove());
    current.append(...[...incoming.childNodes].filter(node => node !== nextHome));
  }

  async function ensureStyles(next, signal) {
    for (const source of next.querySelectorAll('head link[rel="stylesheet"]')) {
      const url = new URL(source.getAttribute('href'), location.origin);
      if (url.origin !== location.origin || [...document.styleSheets].some(sheet => sheet.href === url.href)) continue;
      await new Promise((resolve, reject) => {
        if (signal.aborted) { reject(new DOMException('Aborted', 'AbortError')); return; }
        let link = [...document.head.querySelectorAll('link[rel="stylesheet"]')].find(link => link.href === url.href);
        const created = !link;
        if (!link) { link = source.cloneNode(true); link.href = url.href; }
        const cleanup = () => { signal.removeEventListener('abort', abort); link.removeEventListener('load', done); link.removeEventListener('error', failed); };
        const done = () => { cleanup(); resolve(); };
        const failed = () => { cleanup(); if (created) link.remove(); reject(Error('Page stylesheet failed')); };
        const abort = () => { cleanup(); if (created) link.remove(); reject(new DOMException('Aborted', 'AbortError')); };
        link.addEventListener('load', done, {once: true}); link.addEventListener('error', failed, {once: true});
        signal.addEventListener('abort', abort, {once: true});
        if (created) document.head.append(link);
      });
    }
  }

  function scrollToDestination(url, position) {
    if (url.hash) {
      let id;
      try { id = decodeURIComponent(url.hash.slice(1)); } catch { id = url.hash.slice(1); }
      const target = document.getElementById(id);
      if (target) { target.scrollIntoView(); return; }
    }
    (mobile.matches ? window : main).scrollTo({left: position?.x || 0, top: position?.y || 0, behavior: 'instant'});
  }

  async function navigate(url, options = {}) {
    const sequence = ++ticket;
    request?.abort();
    request = new AbortController();
    const signal = request.signal;
    if (!options.history) saveScroll();
    navigating = true;
    content.setAttribute('aria-busy', 'true');
    document.dispatchEvent(new CustomEvent('jianxi:navigate-start'));
    select(url, !options.history, options.context);
    pageSequence.begin();
    try {
      const response = await fetch(url.href, {signal, headers: {Accept: 'text/html'}});
      const destination = new URL(response.url);
      destination.hash = url.hash;
      if (!publicURL(destination) || !response.headers.get('content-type')?.includes('text/html')) throw Error('Native destination');
      const next = new DOMParser().parseFromString(await response.text(), 'text/html');
      const incoming = next.querySelector('#content');
      if (!incoming || !next.querySelector('#sidebar') || next.body.dataset.prefix) throw Error('Native document');
      await ensureStyles(next, signal);
      if (sequence !== ticket) return;
      // Fetch and stylesheet loading started at the click. Only retire the old
      // readable page once its replacement is ready to render in this frame.
      await pageSequence.leave();
      if (signal.aborted || sequence !== ticket) return;
      pageSequence.finish();
      document.dispatchEvent(new CustomEvent('jianxi:before-swap'));
      if (!options.history) history.pushState({jianxi: true, scroll: {x: 0, y: 0}, readingContext: options.context || null}, '', destination.href);
      syncHead(next);
      syncSidebar(next.querySelector('#sidebar'));
      document.body.dataset.page = next.body.dataset.page;
      document.body.dataset.prefix = next.body.dataset.prefix || '';
      content.replaceChildren(...incoming.childNodes);
      syncBreadcrumbs(next);
      displayedURL = destination;
      scrollToDestination(destination, options.history ? history.state?.scroll : null);
      document.dispatchEvent(new CustomEvent('jianxi:after-swap', {detail: {history: !!options.history}}));
      select(destination, !options.history, history.state?.readingContext);
      status.textContent = document.title;
      if (options.keyboard && !sidebar.contains(document.activeElement)) content.focus({preventScroll: true});
      pageSequence.enter({history: !!options.history, anchor: !!destination.hash});
    } catch (error) {
      if (signal.aborted || sequence !== ticket) return;
      pageSequence.finish();
      location.assign(url.href);
    } finally {
      if (sequence === ticket) { navigating = false; content.removeAttribute('aria-busy'); saveScroll(); }
    }
  }

  document.addEventListener('click', event => {
    if (event.defaultPrevented || event.button !== 0 || event.metaKey || event.ctrlKey || event.shiftKey || event.altKey) return;
    const link = event.target.closest('a[href]');
    if (!link || link.hasAttribute('download') || (link.target && link.target !== '_self') || link.rel.split(/\s+/).includes('external')) return;
    const url = new URL(link.href);
    if (!publicURL(url)) return;
    if (url.pathname === displayedURL.pathname && url.search === displayedURL.search) {
      if (navigating) { ticket++; request?.abort(); navigating = false; content.removeAttribute('aria-busy'); pageSequence.cancel(); select(displayedURL, true); }
      if (url.hash) { pageSequence.finish(); return; }
      event.preventDefault();
      select(displayedURL, true);
      return;
    }
    event.preventDefault();
    let context = null;
    if (link.closest('.post, .timeline-entry, .next-article')) {
      context = document.body.dataset.page === 'article' ? history.state?.readingContext :
        {path: displayedURL.pathname, title: document.querySelector('.page-heading h1')?.textContent || '首页'};
    }
    navigate(url, {keyboard: event.detail === 0, context});
  });

  document.addEventListener('submit', event => {
    const form = event.target;
    if (event.defaultPrevented || form.method.toLowerCase() !== 'get') return;
    const url = new URL(form.action);
    if (!publicURL(url) || url.pathname !== '/search/') return;
    event.preventDefault();
    url.search = new URLSearchParams(new FormData(form)).toString();
    navigate(url, {keyboard: true});
  });

  addEventListener('popstate', () => {
    const url = new URL(location.href);
    if (url.pathname === displayedURL.pathname && url.search === displayedURL.search) {
      if (navigating) { ticket++; request?.abort(); navigating = false; content.removeAttribute('aria-busy'); pageSequence.cancel(); }
      pageSequence.finish();
      displayedURL = url;
      scrollToDestination(url, history.state?.scroll);
      select(url, false, history.state?.readingContext);
      return;
    }
    navigate(url, {history: true, context: history.state?.readingContext});
  });
  pageSequence.enter({initial: true, anchor: !!location.hash});
})();
