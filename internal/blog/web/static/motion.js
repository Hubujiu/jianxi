/* Shared motion policy. GSAP owns playback; components own their DOM and state. */
(() => {
  const engine = window.gsap;
  if (!engine) return;
  const reduced = matchMedia('(prefers-reduced-motion: reduce)');
  const timing = Object.freeze({
    micro: .1, fast: .12, trigger: .13, feedback: .15, contentExit: .16, fade: .2,
    heading: .28, pageEnter: .34, pageLead: .045, pageStagger: .035,
    loadingLead: .18, loadingSweep: 1.1,
    exit: .42, spring: .48, lead: .08, adminLead: .07, hold: 1.8
  });
  const ease = Object.freeze({spring: 'jianxi.spring', settle: 'power3.out', standard: 'power1.out'});
  engine.registerEase(ease.spring, t => 1 - Math.exp(-9 * t) * (Math.cos(7 * t) + 9 / 7 * Math.sin(7 * t)));
  function duration(name) {
    if (!(name in timing)) throw Error(`Unknown motion timing: ${name}`);
    return reduced.matches ? 0 : timing[name];
  }

  function bindHover(root, selector) {
    const touched = new Set();
    const rest = {backgroundColor: 'transparent', borderColor: 'transparent'};
    const active = {backgroundColor: '#fcfcfc', borderColor: '#eeeeee'};
    const state = el => el.matches(':hover,:focus-visible') ? active : rest;
    function update(event) {
      const el = event.target.closest(selector);
      if (!el || !root.contains(el) || (event.type.startsWith('pointer') && el.contains(event.relatedTarget))) return;
      touched.add(el);
      engine.to(el, {...state(el), duration: duration('feedback'), ease: ease.standard, overwrite: 'auto'});
    }
    function settle() {
      if (!reduced.matches) return;
      for (const el of touched) {
        engine.killTweensOf(el, 'backgroundColor,borderColor');
        if (el.isConnected) engine.set(el, state(el));
      }
    }
    const events = ['pointerover', 'pointerout', 'focusin', 'focusout'];
    events.forEach(name => root.addEventListener(name, update));
    reduced.addEventListener('change', settle);
    return () => {
      events.forEach(name => root.removeEventListener(name, update));
      reduced.removeEventListener('change', settle);
      for (const el of touched) {
        engine.killTweensOf(el, 'backgroundColor,borderColor');
        el.style.removeProperty('background-color');
        el.style.removeProperty('border-color');
      }
      touched.clear();
    };
  }
  function createPageSequence(root, topbar) {
    // Animate layout blocks, never their nested controls or the persistent shell.
    const selector = '.listing > .page-heading, .listing > .search-form, .catalog-grid, .post-list,' +
      ' .writing-rhythm, .archive-heading, .timeline-year, .listing > .empty-state,' +
      ' .listing > .pagination, .reading-main > .back-link, .article-heading, .prose, .page-foot';
    const touched = new Set();
    const progress = document.createElement('span');
    progress.className = 'page-progress';
    progress.setAttribute('aria-hidden', 'true');
    const track = document.createElement('span');
    track.className = 'page-progress-track';
    track.setAttribute('aria-hidden', 'true');
    track.append(progress);
    topbar?.append(track);
    let animation, loading, resolveExit;
    function blocks() {
      return [...root.querySelectorAll(selector)].filter(el => {
        const box = el.getBoundingClientRect();
        return box.width > 0 && box.height > 0 && box.bottom > 0 && box.top < innerHeight;
      }).slice(0, 6);
    }
    function stop() {
      animation?.kill(); animation = null;
      // A superseded request must also release its exit await, then check ticket.
      const resolve = resolveExit; resolveExit = null; resolve?.();
    }
    function stopLoading() {
      loading?.kill(); loading = null;
      engine.killTweensOf(progress);
      engine.set(progress, {opacity: 0});
    }
    function finish() {
      stop(); stopLoading();
      if (touched.size) engine.set([...touched], {clearProps: 'opacity,transform'});
      touched.clear();
      root.inert = false;
      delete root.dataset.transition;
    }
    function begin() {
      stop(); stopLoading();
      root.inert = false;
      root.dataset.transition = 'loading';
      if (reduced.matches) return;
      const targets = blocks(); targets.forEach(el => touched.add(el));
      animation = engine.timeline().to(targets, {
        opacity: .72, y: -4, duration: duration('feedback'), ease: ease.standard, overwrite: 'auto'
      }, 0);
      // An indeterminate line appears only when the operation takes long enough.
      // Its playback and delay are owned by the same GSAP timeline, not a timer.
      loading = engine.timeline().set(progress, {opacity: 0, xPercent: -100}, 0)
        .to(progress, {opacity: .65, duration: duration('fast')}, duration('loadingLead'))
        .to(progress, {xPercent: 400, duration: duration('loadingSweep'), ease: 'none', repeat: -1}, duration('loadingLead'));
    }
    function leave() {
      stop();
      root.dataset.transition = 'leaving';
      root.inert = true;
      if (reduced.matches) return Promise.resolve();
      const targets = blocks(); targets.forEach(el => touched.add(el));
      return new Promise(resolve => {
        resolveExit = resolve;
        animation = engine.timeline({onComplete: () => {
          resolveExit = null; resolve();
        }}).to(targets, {opacity: 0, y: -8, duration: duration('contentExit'), ease: ease.standard}, 0);
      });
    }
    function enter({history: restoring = false, anchor = false, initial = false} = {}) {
      finish();
      if (reduced.matches) return;
      const targets = blocks();
      if (!targets.length) return;
      targets.forEach(el => touched.add(el));
      root.dataset.transition = 'entering';
      const heading = targets.filter(el => el.matches('.page-heading,.article-heading,.back-link'));
      const body = targets.filter(el => !heading.includes(el));
      const stationary = restoring || anchor;
      // Slow deferred scripts must not hide server HTML that has already painted.
      const painted = initial && performance.getEntriesByType('paint').some(entry => entry.name === 'first-contentful-paint');
      engine.set(targets, {opacity: painted ? 1 : 0, y: stationary ? 0 : initial ? 8 : 12});
      animation = engine.timeline({defaults: {ease: ease.settle}, onComplete: finish})
        .addLabel('heading', 0).addLabel('body', stationary ? 0 : duration('pageLead'));
      if (heading.length) animation.to(heading, {opacity: 1, y: 0, duration: duration(stationary ? 'fade' : 'heading')}, 'heading');
      if (body.length) animation.to(body, {
        opacity: 1, y: 0, duration: duration(stationary ? 'fade' : 'pageEnter'),
        stagger: {amount: stationary ? 0 : timing.pageStagger * Math.min(body.length - 1, 3)}
      }, 'body');
    }
    function cancel() {
      stop(); stopLoading(); root.inert = false;
      if (reduced.matches || !touched.size) { finish(); return; }
      root.dataset.transition = 'entering';
      animation = engine.timeline({onComplete: finish}).to([...touched], {
        opacity: 1, y: 0, duration: duration('fade'), ease: ease.settle
      }, 0);
    }
    reduced.addEventListener('change', () => { if (reduced.matches) finish(); });
    addEventListener('pagehide', finish);
    addEventListener('hashchange', finish);
    root.addEventListener('focusin', event => { if (event.target !== root) finish(); });
    return Object.freeze({begin, leave, enter, cancel, finish});
  }
  window.JianxiMotion = Object.freeze({engine, reduced, timing, ease, duration, bindHover, createPageSequence});
})();
