/* Enhance search selects while retaining the native fields for GET and no-JS. */
(() => {
  const motion = window.JianxiMotion;
  if (!motion) return;
  const gsap = motion.engine;
  let controls = [], activeControl, serial = 0;
  const icon = name => `<svg viewBox="0 0 24 24" aria-hidden="true"><use href="/static/icons.svg#${name}"/></svg>`;
  function enhance(select) {
    const original = select.closest('label');
    if (!original) return;
    const id = `search-filter-${++serial}`;
    const caption = original.firstChild.textContent.trim();
    const wrapper = document.createElement('div'); wrapper.className = 'filter-select';
    const label = document.createElement('span'); label.className = 'filter-label'; label.id = id + '-label'; label.textContent = caption;
    const control = document.createElement('div'); control.className = 'filter-control';
    const trigger = document.createElement('button'); trigger.type = 'button'; trigger.className = 'filter-trigger';
    trigger.setAttribute('role', 'combobox'); trigger.setAttribute('aria-haspopup', 'listbox');
    trigger.setAttribute('aria-expanded', 'false'); trigger.setAttribute('aria-controls', id + '-list');
    trigger.setAttribute('aria-labelledby', `${label.id} ${id}-value`);
    const value = document.createElement('span'); value.id = id + '-value';
    trigger.append(value); trigger.insertAdjacentHTML('beforeend', icon('chevron-down'));
    const menu = document.createElement('div'); menu.className = 'filter-menu'; menu.id = id + '-list'; menu.hidden = true;
    menu.setAttribute('role', 'listbox'); menu.setAttribute('aria-labelledby', label.id);
    const options = [...select.options].map((option, index) => {
      const item = document.createElement('div'); item.className = 'filter-option'; item.id = `${id}-option-${index}`;
      item.setAttribute('role', 'option'); item.dataset.index = String(index);
      item.setAttribute('aria-disabled', String(option.disabled));
      const text = document.createElement('span'); text.textContent = option.textContent;
      item.append(text); item.insertAdjacentHTML('beforeend', icon('check')); menu.append(item); return item;
    });
    control.append(trigger, menu); wrapper.append(label, control, select); original.replaceWith(wrapper);
    select.hidden = true;
    const listeners = new AbortController(), signal = listeners.signal;
    let open = false, index = select.selectedIndex, animation, typed = '', typedAt = 0;
    function sync() {
      value.textContent = select.selectedOptions[0]?.textContent || '';
      trigger.disabled = select.disabled;
      options.forEach((item, i) => item.setAttribute('aria-selected', String(i === select.selectedIndex)));
    }
    function highlight(next) {
      index = Math.max(0, Math.min(options.length - 1, next));
      options.forEach((item, i) => item.classList.toggle('active', i === index));
      if (open) {
        trigger.setAttribute('aria-activedescendant', options[index].id);
        options[index].scrollIntoView({block: 'nearest', behavior: 'instant'});
      }
    }
    function settle() {
      animation?.kill(); gsap.set(menu, {opacity: open ? 1 : 0, y: open ? 0 : -5});
      menu.hidden = !open;
    }
    function close(commit = false, immediate = false) {
      if (!open) return;
      if (commit && !select.options[index].disabled) {
        const changed = select.selectedIndex !== index;
        select.selectedIndex = index; sync();
        if (changed) { select.dispatchEvent(new Event('input', {bubbles: true})); select.dispatchEvent(new Event('change', {bubbles: true})); }
      }
      open = false; menu.inert = true;
      trigger.setAttribute('aria-expanded', 'false'); trigger.removeAttribute('aria-activedescendant');
      if (activeControl === api) activeControl = null;
      animation?.kill();
      if (immediate || motion.reduced.matches) { settle(); return; }
      animation = gsap.to(menu, {opacity: 0, y: -5, duration: motion.duration('fast'), ease: motion.ease.standard, onComplete: () => { if (!open) menu.hidden = true; }});
    }
    function show() {
      if (open || select.disabled) return;
      activeControl?.close(false, true); activeControl = api;
      open = true; animation?.kill(); menu.hidden = false; menu.inert = false;
      trigger.setAttribute('aria-expanded', 'true'); highlight(select.selectedIndex);
      if (!menu.style.opacity) gsap.set(menu, {opacity: 0, y: -5});
      animation = gsap.to(menu, {opacity: 1, y: 0, duration: motion.duration('heading'), ease: motion.ease.spring, overwrite: 'auto'});
    }
    trigger.addEventListener('click', () => open ? close() : show(), {signal});
    trigger.addEventListener('blur', () => close(true), {signal});
    trigger.addEventListener('keydown', event => {
      if (event.key === 'Tab') { close(true); return; }
      if (event.key === 'Escape') { if (open) { event.preventDefault(); event.stopPropagation(); close(); } return; }
      if (event.key === 'Enter' || event.key === ' ') { event.preventDefault(); open ? close(true) : show(); return; }
      const steps = {ArrowDown: 1, ArrowUp: -1, PageDown: 10, PageUp: -10};
      if (event.key in steps || event.key === 'Home' || event.key === 'End') {
        event.preventDefault(); const wasOpen = open; show();
        if (event.altKey && event.key === 'ArrowUp' && wasOpen) { close(true); return; }
        let next = event.key === 'Home' ? 0 : event.key === 'End' ? options.length - 1 : wasOpen ? index + steps[event.key] : index;
        const step = steps[event.key] > 0 ? 1 : -1;
        while (next >= 0 && next < options.length && select.options[next].disabled) next += step;
        highlight(next); return;
      }
      if (event.key.length === 1 && !event.ctrlKey && !event.metaKey && !event.altKey) {
        event.preventDefault(); show();
        const now = performance.now(); typed = now - typedAt < 700 ? typed + event.key : event.key; typedAt = now;
        const prefix = [...typed].every(char => char === typed[0]) ? typed[0] : typed;
        for (let offset = 1; offset <= options.length; offset++) {
          const next = (index + offset) % options.length;
          if (!select.options[next].disabled && select.options[next].textContent.trim().toLocaleLowerCase().startsWith(prefix.toLocaleLowerCase())) { highlight(next); break; }
        }
      }
    }, {signal});
    menu.addEventListener('pointerdown', event => event.preventDefault(), {signal});
    menu.addEventListener('pointermove', event => { const item = event.target.closest('.filter-option'); if (item && item.getAttribute('aria-disabled') !== 'true') highlight(+item.dataset.index); }, {signal});
    menu.addEventListener('click', event => {
      const item = event.target.closest('.filter-option'); if (!item || item.getAttribute('aria-disabled') === 'true') return;
      highlight(+item.dataset.index); close(true); trigger.focus({preventScroll: true});
    }, {signal});
    select.addEventListener('change', sync, {signal});
    motion.reduced.addEventListener('change', () => { if (motion.reduced.matches) settle(); }, {signal});
    const api = {close, contains: target => wrapper.contains(target), dispose() { close(false, true); animation?.kill(); listeners.abort(); }};
    sync(); return api;
  }
  function mount() { controls = [...document.querySelectorAll('.search-form .filter-row select')].map(enhance).filter(Boolean); }
  function dispose() { controls.forEach(control => control.dispose()); controls = []; activeControl = null; }
  document.addEventListener('pointerdown', event => { if (activeControl && !activeControl.contains(event.target)) activeControl.close(true); });
  document.addEventListener('jianxi:navigate-start', () => activeControl?.close(false, true));
  document.addEventListener('jianxi:before-swap', dispose);
  document.addEventListener('jianxi:after-swap', mount);
  addEventListener('pagehide', () => activeControl?.close(false, true));
  mount();
})();
