/* Motion enhances an already readable static page. All dependencies are local. */
(() => {
  const paper = document.querySelector('.kinetic-paper');
  const leaves = [...document.querySelectorAll('.paper-leaf')];
  const tilt = document.querySelector('.paper-tilt');
  const status = document.querySelector('#paper-status');
  const reduce = matchMedia('(prefers-reduced-motion: reduce)');
  const gsap = window.gsap;
  let open = false;
  const pose = () => (open ? (innerWidth < 768 ? 10 : 13) : 6);
  const snapPaper = () => {
    if (gsap) { gsap.killTweensOf(leaves); gsap.set(leaves,{rotation:i=>(i-5)*pose()}); }
    else leaves.forEach((leaf,i)=>leaf.style.transform=`rotate(${(i-5)*pose()}deg)`);
  };
  paper?.addEventListener('click', () => {
    open=!open;
    paper.setAttribute('aria-pressed',String(open));
    paper.setAttribute('aria-label',open?'合起纸页':'展开纸页');
    status.textContent=open?'纸页展开了。再点一下，可以合起。':'纸页已合起。';
    if (!gsap || reduce.matches) { snapPaper(); return; }
    gsap.to(leaves,{rotation:i=>(i-5)*pose(),duration:.85,stagger:.025,ease:'back.out(1.15)',overwrite:true});
  });
  reduce.addEventListener('change',()=>{if(reduce.matches && paper)queueMicrotask(snapPaper);});
  if (!gsap) return;
  const motion = gsap.matchMedia();
  motion.add('(prefers-reduced-motion: no-preference)', context => {
    const cleanup = [];
    const bind = (element,event,listener,options) => {
      element.addEventListener(event,listener,options);
      cleanup.push(()=>element.removeEventListener(event,listener,options));
    };
    const watch = nodes => {
      const seen = new WeakSet();
      const observer = new IntersectionObserver(entries => {
        const arriving = entries.filter(e=>e.isIntersecting && !e.target.hidden && !seen.has(e.target)).map(e=>e.target);
        arriving.forEach(el=>{seen.add(el);observer.unobserve(el);});
        if (arriving.length) context.add(()=>gsap.fromTo(arriving,{y:15,opacity:0},{y:0,opacity:1,duration:.72,stagger:.065,ease:'power3.out',clearProps:'transform,opacity',overwrite:true}));
      },{threshold:.08});
      nodes.forEach(node=>observer.observe(node));
      cleanup.push(()=>observer.disconnect());
    };
    gsap.fromTo('.identity',{y:10,opacity:0},{y:0,opacity:1,duration:.85,ease:'power3.out',clearProps:'transform,opacity'});
    const entry = document.querySelector('.studio-intro, .article-heading');
    if(entry) gsap.fromTo(entry,{y:12,opacity:0},{y:0,opacity:1,duration:.8,ease:'power3.out',clearProps:'transform,opacity'});
    watch(document.querySelectorAll('.post-row,.prose section'));
    if(paper) gsap.fromTo('.paper-playground',{y:15,opacity:0},{y:0,opacity:1,duration:.95,ease:'power3.out',clearProps:'transform,opacity'});
    document.querySelectorAll('[data-filter],[data-tag]').forEach(button=>bind(button,'click',()=>{
      const visible=[...document.querySelectorAll('.post-row')].filter(row=>!row.hidden);
      context.add(()=>gsap.fromTo(visible,{y:9,opacity:.35},{y:0,opacity:1,duration:.4,stagger:.035,ease:'power3.out',overwrite:true,clearProps:'transform,opacity'}));
    }));
    const dialogs=[...document.querySelectorAll('dialog')];
    const closing = new WeakSet();
    const close = dialog => {
      if(closing.has(dialog)) return;
      closing.add(dialog);
      context.add(()=>gsap.to(dialog,{y:7,opacity:0,duration:.15,ease:'power2.in',overwrite:true,onComplete:()=>{
        dialog.close();closing.delete(dialog);gsap.set(dialog,{clearProps:'transform,opacity'});
      }}));
    };
    const observer = new MutationObserver(records=>records.forEach(({target})=>{
      if(target.open && !closing.has(target)) context.add(()=>gsap.fromTo(target,{y:16,opacity:0},{y:0,opacity:1,duration:.36,ease:'power3.out',overwrite:true,clearProps:'transform,opacity'}));
    }));
    dialogs.forEach(dialog=>observer.observe(dialog,{attributes:true,attributeFilter:['open']}));
    cleanup.push(()=>observer.disconnect());
    bind(document,'keydown',event=>{
      const open=document.querySelector('dialog[open]');
      if(event.key==='Escape' && open){event.preventDefault();event.stopImmediatePropagation();close(open);}
    },true);
    bind(document,'click',event=>{
      const button=event.target.closest('[data-close]');
      if(button){event.preventDefault();event.stopImmediatePropagation();close(button.closest('dialog'));return;}
      if(event.target instanceof HTMLDialogElement){
        const rect=event.target.getBoundingClientRect();
        if(event.clientX<rect.left||event.clientX>rect.right||event.clientY<rect.top||event.clientY>rect.bottom){event.preventDefault();event.stopImmediatePropagation();close(event.target);}
      }
    },true);
    return ()=>{
      cleanup.forEach(fn=>fn());
      dialogs.forEach(dialog=>{if(closing.has(dialog))dialog.close();});
    };
  });
  motion.add('(hover: hover) and (pointer: fine) and (prefers-reduced-motion: no-preference)', context=>{
    const disposers=[];
    const bind=(element,event,fn)=>{element.addEventListener(event,fn);disposers.push(()=>element.removeEventListener(event,fn));};
    if(paper){
      const rotateX=gsap.quickTo(tilt,'rotationX',{duration:.65,ease:'power3.out'});
      const rotateY=gsap.quickTo(tilt,'rotationY',{duration:.65,ease:'power3.out'});
      let bounds;
      bind(paper,'pointerenter',()=>{bounds=paper.getBoundingClientRect();});
      bind(paper,'pointermove',event=>{
        if(!bounds)return;
        rotateX(-((event.clientY-bounds.top)/bounds.height-.5)*20);
        rotateY(((event.clientX-bounds.left)/bounds.width-.5)*24);
      });
      bind(paper,'pointerleave',()=>{rotateX(0);rotateY(0);bounds=null;});
    }
    document.querySelectorAll('.post-row h3 a').forEach(link=>{
      const title=link.querySelector('.post-title');
      const xTo=gsap.quickTo(title,'x',{duration:.45,ease:'power3.out'});
      bind(link,'pointerenter',()=>xTo(5));bind(link,'pointerleave',()=>xTo(0));
    });
    return ()=>disposers.forEach(dispose=>dispose());
  });
  // Tweens are finite; static documents and bfcache need no perpetual render loop.
})();
