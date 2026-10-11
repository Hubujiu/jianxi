// Update only two known local demonstration articles through the real MCP.
import {readFile} from 'node:fs/promises';
import {spawn} from 'node:child_process';
import {createInterface} from 'node:readline';
import {parseArgs} from 'node:util';
import {resolve,join} from 'node:path';
import {fileURLToPath} from 'node:url';

const {values}=parseArgs({options:{data:{type:'string',default:'data/dev'},url:{type:'string',default:'http://127.0.0.1:8080'},apply:{type:'boolean',default:false}}});
const base=new URL(values.url);
if(base.protocol!=='http:'||!['127.0.0.1','localhost','[::1]'].includes(base.hostname)||base.pathname!=='/'||base.search||base.hash||base.username||base.password)
  throw Error('This demonstration update requires a local loopback server.');
const token=(await readFile(join(resolve(values.data),'initial-token.txt'),'utf8')).trim();
const child=spawn(fileURLToPath(new URL(process.platform==='win32'?'../bin/jianxi.exe':'../bin/jianxi',import.meta.url)),['mcp'],{env:{...process.env,JIANXI_URL:base.origin,JIANXI_TOKEN:token},stdio:['pipe','pipe','pipe'],windowsHide:true});
const pending=new Map();let serial=0;
child.stderr.resume();
const lines=createInterface({input:child.stdout});
lines.on('line',line=>{
  const message=JSON.parse(line),request=pending.get(message.id);
  if(request){pending.delete(message.id);message.error?request.reject(Error(JSON.stringify(message.error))):request.resolve(message.result)}
});
child.on('exit',()=>{for(const request of pending.values())request.reject(Error('Local MCP stopped'));pending.clear()});
function rpc(method,params){return new Promise((resolve,reject)=>{const id=++serial;pending.set(id,{resolve,reject});child.stdin.write(JSON.stringify({jsonrpc:'2.0',id,method,params})+'\n')})}
async function tool(name,input={}) {
  const response=await rpc('tools/call',{name,arguments:input});
  if(response.isError)throw Error(response.content?.map(c=>c.text||'').join('\n')||`${name} failed`);
  return response.structuredContent.result;
}
const timeout=setTimeout(()=>{child.kill();process.exitCode=1},180000);
try {
  await rpc('initialize',{protocolVersion:'2025-06-18',capabilities:{},clientInfo:{name:'jianxi-local-demo-keywords',version:'1'}});
  child.stdin.write(JSON.stringify({jsonrpc:'2.0',method:'notifications/initialized'})+'\n');
  await tool('blog_content_rules');
  const status=await tool('blog_status');
  const snapshot=JSON.parse(await readFile(join(resolve(values.data),'releases',status.activeRelease,'snapshot.json'),'utf8'));
  if(snapshot.id!==status.activeRelease)throw Error('Data directory does not match the running server');
  const source=await readFile(new URL('../examples/rendering-showcase.md',import.meta.url),'utf8');
  const section=source.match(/### 关键词描述\n[\s\S]*?(?=### 链接与自动链接)/)?.[0];
  if(!section)throw Error('Keyword demonstration section not found');
  const seeds=JSON.parse(await readFile(new URL('../internal/blog/web/demo.json',import.meta.url),'utf8'));
  const plans=[];
  for(const slug of ['rendering-showcase','smaller-software']) {
    const published=Object.values(snapshot.posts).find(r=>r.content.slug===slug);
    if(!published)continue;
    const content={...published.content};
    // Legacy demos may only store IDs. Uploads require explicit route metadata.
    const routeMetadata=id=>{
      const taxon=snapshot.taxonomy[id];
      if(!taxon)throw Error('Demonstration taxonomy not found');
      return {id:taxon.id,name:taxon.name,slug:taxon.slug};
    };
    if(content.categoryId)content.category=routeMetadata(content.categoryId);
    content.tags=(content.tagIds||[]).map(routeMetadata);
    if(slug==='rendering-showcase') {
      if(content.title!=='间隙 · 文章渲染展示')throw Error('Refusing to modify a renamed showcase');
      if(content.markdown.includes('### 关键词描述'))content.markdown=content.markdown.replace(/### 关键词描述\r?\n[\s\S]*?(?=### 链接与自动链接)/,section);
      else if(content.markdown.includes('### 链接与自动链接'))content.markdown=content.markdown.replace('### 链接与自动链接',section+'### 链接与自动链接');
      else throw Error('Showcase insertion point not found');
    } else {
      const seed=seeds.find(c=>c.slug===slug);
      if(content.title!==seed.title)throw Error('Refusing to modify a renamed demonstration article');
      for(const keyword of seed.markdown.matchAll(/\[\[([^|]+)\|([^\]]+)\]\]/g)) {
        if(!content.markdown.includes(keyword[0])) {
          if(!content.markdown.includes(keyword[1]))throw Error(`Demonstration word not found: ${keyword[1]}`);
          content.markdown=content.markdown.replace(keyword[1],keyword[0]);
        }
      }
    }
    if(content.markdown!==published.content.markdown)plans.push(content);
  }
  console.log(JSON.stringify({local:true,changes:plans.map(c=>({title:c.title,url:base.origin+`/posts/${c.slug}/`,keywords:[...c.markdown.matchAll(/\[\[/g)].length}))}));
  if(!values.apply||!plans.length){console.log(plans.length?'Use --apply to update these local demonstration articles.':'Demonstration articles are already up to date.');}
  else {
    const revisions=[];
    for(const content of plans) {
      const copy=await tool('blog_working_copy',{contentId:content.id});
      const published=snapshot.posts[content.id];
      if(copy.expectedDraftRevision&&copy.expectedDraftRevision!==published.revision) {
        const identical=['slug','title','summary','author','markdown','seoTitle','seoDescription','avatarUrl','categoryId'].every(key=>(copy.content[key]||'')===(content[key]||''))&&
          ['tagIds','sources','videos','socialLinks'].every(key=>JSON.stringify(copy.content[key]||[])===JSON.stringify(content[key]||[]));
        if(!identical)throw Error('Demonstration article has an unpublished draft; preserving it.');
        // Resume a previous attempt that already committed this exact change.
        revisions.push(copy.expectedDraftRevision);
        continue;
      }
      const key=`demo-keywords:${copy.id}`;
      const upload=await tool('blog_upload_content',{copyId:copy.id,content,idempotencyKey:key+':upload'});
      if(upload.status!=='parsed')throw Error('Demonstration upload was not parsed');
      const revision=await tool('blog_commit_draft',{uploadId:upload.id,sourceHash:upload.sourceHash,expectedDraftRevision:copy.expectedDraftRevision,epoch:copy.epoch,idempotencyKey:key+':commit'});
      revisions.push(revision.revision);
    }
    const change=await tool('blog_changeset',{revisions,expectedActiveRelease:status.activeRelease});
    const preview=await tool('blog_preview',{id:change.id});
    // --apply is explicit authorization to activate these loopback demo changes.
    const confirmed=await tool('blog_confirm_preview',{previewId:preview.previewId,artifactHash:preview.artifactHash,changeSetHash:change.hash});
    const published=await tool('blog_publish',{previewId:preview.previewId,confirmationId:confirmed.confirmationId,changeSetHash:change.hash,expectedActiveRelease:status.activeRelease,idempotencyKey:'demo-keywords:'+change.id});
    console.log(JSON.stringify({published:published.release,urls:plans.map(c=>base.origin+`/posts/${c.slug}/`)}));
  }
} finally {clearTimeout(timeout);child.stdin.end();child.kill()}
