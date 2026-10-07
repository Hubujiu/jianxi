import { readFile,writeFile } from 'node:fs/promises';
const dir=new URL('../../design-prototypes/vendor/',import.meta.url);
const generated=await readFile(new URL('../../design-prototypes/studio/index.html',import.meta.url),'utf8');
const allText=generated.replace(/<script[\s\S]*?<\/script>/g,'').replace(/<[^>]+>/g,'');
const data=JSON.parse(generated.match(/<script type="application\/json" id="post-data">([\s\S]*?)<\/script>/)[1]);
const chars=[...new Set(allText+JSON.stringify(data)+'返回所有文章本文目录回到顶部发布于更新于写到这里先告一段落接着读留给自己的笔记慢慢生长搜索文章输入标题正文或标签中的关键词')].join('');
async function font(family,text,name){
  const url='https://fonts.googleapis.com/css2?family='+encodeURIComponent(family)+'&display=swap&text='+encodeURIComponent(text);
  const response=await fetch(url,{headers:{'User-Agent':'Mozilla/5.0 Chrome/131.0.0.0 Safari/537.36'}});
  if(!response.ok)throw new Error(`${name}: CSS ${response.status}`);
  const css=await response.text();
  const source=css.match(/url\(([^)]+)\)/)?.[1];
  if(!source)throw new Error(`${name}: Missing source`);
  const file=await fetch(source);
  if(!file.ok)throw new Error(`${name}: font ${file.status}`);
  await writeFile(new URL(name,dir),Buffer.from(await file.arrayBuffer()));
  console.log(name+' downloaded');
}
await font('Noto Serif SC:wght@400',chars,'jianxi-serif.woff2');
await font('Cormorant Garamond:ital,wght@1,500','Jianxi.0123456789/- ','jianxi-signature.woff2');
for(const [folder,name] of [['notoserifsc','NOTO-SERIF-OFL.txt'],['cormorantgaramond','CORMORANT-OFL.txt']]){
  const result=await fetch(`https://raw.githubusercontent.com/google/fonts/main/ofl/${folder}/OFL.txt`);
  if(!result.ok)throw new Error('License retrieval failed: '+folder);
  await writeFile(new URL(name,dir),await result.text());
}
