(() => {
  const motion=window.JianxiMotion;if(!motion)return;
  const gsap=motion.engine,reduced=motion.reduced,spring=motion.ease.spring;
  const app=document.querySelector('.admin-app');if(!app)return;
  document.querySelectorAll('form[data-confirm]').forEach(form=>{
    const submit=form.querySelector('button[type="submit"],button[name="action"]');if(!submit)return;
    const surface=document.createElement('div');surface.style.display='inline-block';submit.before(surface);surface.append(submit);
    let confirmed=false,open=false,animation,marker,origin,content,shade,previousOverflow;
    function closedBox(){return marker.getBoundingClientRect()}
    function openedBox(){const width=Math.min(420,innerWidth-36),height=Math.min(260,innerHeight-40);return {x:(innerWidth-width)/2,y:Math.max(20,(innerHeight-height)/2),width,height}}
    function close(){
      if(!open)return;open=false;animation?.kill();app.inert=false;document.body.style.overflow=previousOverflow;content.inert=true;const r=closedBox();
      animation=gsap.timeline().to(surface,{x:r.x,y:r.y,width:r.width,height:r.height,borderRadius:8,backgroundColor:'#292b27',duration:motion.duration('exit'),ease:spring},0).to(content,{opacity:0,filter:'blur(4px)',duration:motion.duration('fast')},0).to(submit,{opacity:1,duration:motion.duration('fade')},0).to(shade,{opacity:0,duration:motion.duration('fade')},0).call(()=>{content.remove();surface.className='';surface.removeAttribute('role');surface.removeAttribute('aria-modal');surface.removeAttribute('aria-labelledby');surface.style.cssText='display:inline-block';submit.style.cssText='';submit.inert=false;marker.before(surface);marker.remove();shade.remove();submit.focus()});
    }
    form.addEventListener('submit',event=>{
      if(confirmed)return;event.preventDefault();if(open||surface.classList.contains('confirmation-surface'))return;open=true;
      origin=surface.getBoundingClientRect();marker=document.createElement('span');marker.style.cssText=`display:inline-block;width:${origin.width}px;height:${origin.height}px`;surface.before(marker);document.body.append(surface);surface.className='confirmation-surface';surface.setAttribute('role','dialog');surface.setAttribute('aria-modal','true');surface.setAttribute('aria-labelledby','confirmation-title');gsap.set(surface,{x:origin.x,y:origin.y,width:origin.width,height:origin.height,left:0,top:0});submit.inert=true;gsap.set(submit,{position:'absolute',inset:0,width:'100%',height:'100%'});
      content=document.createElement('div');content.className='confirmation-content';content.innerHTML='<h2 id="confirmation-title">准备好了，再继续。</h2><p></p><div class="confirmation-actions"><button type="button" class="button" data-cancel>再看一眼</button><button type="button" class="button primary" data-confirm-action>确认继续 ↗</button></div>';content.querySelector('p').textContent=form.dataset.confirm;surface.append(content);shade=document.createElement('div');shade.className='confirmation-backdrop';document.body.append(shade);app.inert=true;previousOverflow=document.body.style.overflow;document.body.style.overflow='hidden';
      const yes=content.querySelector('[data-confirm-action]'),no=content.querySelector('[data-cancel]');no.addEventListener('click',close);yes.addEventListener('click',()=>{confirmed=true;app.inert=false;yes.disabled=true;const hidden=document.createElement('input');hidden.type='hidden';hidden.name=submit.name;hidden.value=submit.value;form.append(hidden);form.requestSubmit()});shade.addEventListener('click',close);gsap.set(content,{visibility:'visible',filter:'blur(4px)'});no.focus();animation=gsap.timeline().to(surface,{...openedBox(),borderRadius:21,backgroundColor:'#fff',color:'#292a28',duration:motion.duration('spring'),ease:spring},0).to(submit,{opacity:0,duration:motion.duration('micro')},0).to(content,{opacity:1,filter:'blur(0px)',duration:motion.duration('fade')},motion.duration('adminLead'));
    });
    surface.addEventListener('keydown',e=>{if(!open)return;const no=content.querySelector('[data-cancel]'),yes=content.querySelector('[data-confirm-action]');if(e.key==='Escape'){e.preventDefault();close()}if(e.key==='Tab'){e.preventDefault();(document.activeElement===no?yes:no).focus()}});
    addEventListener('resize',()=>{if(open){gsap.set(surface,openedBox())}});reduced.addEventListener('change',()=>animation?.progress(1));
  });
})();
