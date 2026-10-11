import {mkdir,copyFile,cp,readFile,writeFile} from 'node:fs/promises';
import {createHash} from 'node:crypto';
const out=new URL('../internal/blog/web/static/',import.meta.url);
await mkdir(out,{recursive:true});
for (const [src,dest] of [
 ['gsap/dist/gsap.min.js','gsap.min.js'],
 ['katex/dist/katex.min.css','katex.min.css'],
 ['katex/LICENSE','KATEX-LICENSE.txt'],
 ['gsap/README.md','GSAP-README.txt'],
]) await copyFile(new URL('../node_modules/'+src,import.meta.url),new URL(dest,out));
await cp(new URL('../node_modules/katex/dist/fonts/',import.meta.url),new URL('fonts/',out),{recursive:true});

// Preserve the upstream frequency-based Unicode subsets. Only copy the two
// proportional weights used by the UI; do not ship Light or Mono variants.
const fontPackage=new URL('../node_modules/lxgw-wenkai-webfont/',import.meta.url);
const fontOut=new URL('fonts/wenkai/',out);
await mkdir(fontOut,{recursive:true});
const hash=bytes=>createHash('sha256').update(bytes).digest('hex').slice(0,16);
let fontCSS='',preload='',count=0,total=0;
for(const weight of ['regular','bold']){
 const source=await readFile(new URL(`lxgwwenkai-${weight}.css`,fontPackage),'utf8');
 const replacements=new Map();
 for(const match of source.matchAll(/url\('\.\/files\/([^']+\.woff2)'\)/g)){
  const file=match[1];
  const bytes=await readFile(new URL(`files/${file}`,fontPackage));
  const dest=`${hash(bytes)}.woff2`;
  await writeFile(new URL(dest,fontOut),bytes);
  replacements.set(file,dest);
  count++;total+=bytes.length;
  // One small, guaranteed-use Latin / punctuation / common-character subset.
  // The remaining subsets are discovered together via CSS, without JS queues.
  if(weight==='regular' && file==='lxgwwenkai-regular-subset-119.woff2')
   preload=`/static/fonts/wenkai/${dest}`;
 }
 if(!replacements.size) throw new Error(`No WenKai ${weight} subsets found`);
 fontCSS+=source.replace(/url\('\.\/files\/([^']+\.woff2)'\)/g,(_,file)=>`url('./${replacements.get(file)}')`)+'\n';
}
if(!preload) throw new Error('WenKai critical subset is missing');
const cssName=`${hash(fontCSS)}.css`;
await writeFile(new URL(cssName,fontOut),fontCSS);
await copyFile(new URL('OFL.txt',fontPackage),new URL('WENKAI-OFL.txt',out));
await copyFile(new URL('LICENSE',fontPackage),new URL('WENKAI-WEBFONT-LICENSE.txt',out));
await writeFile(new URL('../font-head.html',out),`{{define "font-head"}}<link rel="preload" href="${preload}" as="font" type="font/woff2" crossorigin><link rel="stylesheet" href="/static/fonts/wenkai/${cssName}">{{end}}\n`);
console.log(`Self-hosted WenKai (${count} subsets, ${(total/1024/1024).toFixed(2)} MiB on disk), GSAP and KaTeX assets ready.`);
