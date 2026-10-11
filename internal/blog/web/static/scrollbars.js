/* One scrollbar behavior for every scroll container, including dynamically
 * mounted article content. Native scrolling and keyboard controls stay intact. */
(() => {
  document.documentElement.classList.add('scrollbars-ready');
  const timers = new WeakMap();
  document.addEventListener('scroll', event => {
    const scroller = event.target === document ? document.scrollingElement : event.target;
    if (!(scroller instanceof Element)) return;
    scroller.classList.add('is-scrolling');
    clearTimeout(timers.get(scroller));
    timers.set(scroller, setTimeout(() => {
      scroller.classList.remove('is-scrolling');
      timers.delete(scroller);
    }, 1200));
  }, {capture: true, passive: true});
})();
