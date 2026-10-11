import {execFile,spawn} from 'node:child_process';
import {promisify} from 'node:util';
import {createInterface} from 'node:readline';
import {mkdtemp,readFile,rm,chmod} from 'node:fs/promises';
import {tmpdir} from 'node:os';
import {join,resolve,basename,sep} from 'node:path';
import {fileURLToPath} from 'node:url';
import assert from 'node:assert/strict';
import sharp from 'sharp';
const exec=promisify(execFile);
const docker=async(...args)=>(await exec('docker',args,{timeout:120000,maxBuffer:2*1024*1024,windowsHide:true})).stdout.trim();
const name=`jianxi-smoke-${Date.now()}`,compilerName=name+'-compiler';
const image=process.env.JIANXI_TEST_IMAGE||'jianxi:latest';
const compilerImage=process.env.JIANXI_TEST_MCP_IMAGE;
const memory=process.env.JIANXI_TEST_MEMORY||'256m';
const directory=await mkdtemp(join(tmpdir(),'jianxi-docker-smoke-'));
const sleep=ms=>new Promise(resolve=>setTimeout(resolve,ms));
let child,base,token,sequence=0,stderr='',closed=false;
const pending=new Map();
const rpc=(method,params)=>new Promise((resolve,reject)=>{
 const id=++sequence;
 const timer=setTimeout(()=>{pending.delete(id);reject(Error(`MCP timeout: ${method}`));},180000);
 pending.set(id,{resolve:value=>{clearTimeout(timer);resolve(value)},reject:error=>{clearTimeout(timer);reject(error)}});
 child.stdin.write(JSON.stringify({jsonrpc:'2.0',id,method,params})+'\n');
});
const tool=async(name,args={})=>{
 const result=await rpc('tools/call',{name,arguments:args});
 if(result.isError)throw Error(result.content.map(item=>item.text||'').join('\n'));
 return result.structuredContent?.result??JSON.parse(result.content[0].text).result;
};
const status=async()=>{
 const response=await fetch(base+'/_control/status',{method:'POST',headers:{Authorization:`Bearer ${token}`,'Content-Type':'application/json'},body:'{}'});
 assert.equal(response.status,200);return response.json();
};
const ready=async()=>{
 for(let i=0;i<60;i++){try{if((await fetch(base+'/healthz')).ok)return;}catch{}await sleep(500);}
 throw Error('Server did not become ready');
};
const publish=async(revision)=>{
 const current=await tool('blog_status');
 const changes=await tool('blog_changeset',{revisions:[revision],expectedActiveRelease:current.activeRelease});
 const preview=await tool('blog_preview',{id:changes.id});
 const confirmation=await tool('blog_confirm_preview',{previewId:preview.previewId,artifactHash:preview.artifactHash,changeSetHash:preview.changeSetHash});
 return tool('blog_publish',{previewId:preview.previewId,confirmationId:confirmation.confirmationId,changeSetHash:preview.changeSetHash,expectedActiveRelease:current.activeRelease,idempotencyKey:`publish-${sequence}`});
};
try{
 await docker('run','-d','--name',name,'--memory',memory,'--memory-swap',memory,'--cpus','2','-p','127.0.0.1::8080',image,'serve','--demo','--data','/data','--addr','0.0.0.0:8080');
 base=`http://127.0.0.1:${(await docker('port',name,'8080/tcp')).split(':').at(-1)}`;
 await ready();
 await docker('exec',name,'/jianxi','healthcheck');
 for(const binary of ['node','chromium','sh']){
  let available=false;try{await docker('exec',name,binary,'--version');available=true;}catch{}
  assert.equal(available,false,`Server contains ${binary}`);
 }
 const config=JSON.parse(await docker('inspect',name));
 assert.equal(config[0].Config.User,'10001:10001');
 assert.equal(config[0].State.OOMKilled,false);
 await docker('cp',`${name}:/data/initial-token.txt`,join(directory,'token.txt'));
 token=(await readFile(join(directory,'token.txt'),'utf8')).trim();
 await docker('cp',`${name}:/data/admin-route.txt`,join(directory,'route.txt'));
 const route=(await readFile(join(directory,'route.txt'),'utf8')).trim();
 const info=await docker('exec',name,'/jianxi','init-info','--data','/data');
 assert.ok(info.includes(token)&&info.includes(route));
 await docker('exec',name,'/jianxi','init-info','--data','/data','--clear-initial-token');
 assert.ok(!(await docker('exec',name,'/jianxi','init-info','--data','/data')).includes(token));
 const home=await fetch(base);assert.equal(home.status,200);
 const html=await home.text();
 const fonts=[...html.matchAll(/\/static\/fonts\/wenkai\/[^"\s]+\.(?:css|woff2)/g)].map(match=>match[0]);
 assert.ok(fonts.length>=2);
 for(const asset of ['/static/gsap.min.js','/static/katex.min.css',...fonts])assert.equal((await fetch(base+asset)).status,200);
 assert.equal((await fetch(base+route)).status,200);
 assert.equal((await fetch(base+'/admin/')).status,404);
 assert.equal((await fetch(base+'/_control/status',{method:'POST',body:'{}'})).status,401);
 console.log('PASS: scratch server, non-root, no Node/Chromium/shell, public assets and authentication');
 console.log('Idle server memory: '+await docker('stats','--no-stream','--format','{{.MemUsage}}',name));

 if(compilerImage){
  child=spawn('docker',['run','--rm','-i','--name',compilerName,'--network',`container:${name}`,'--memory','512m','--memory-swap','512m','--cpus','2','--shm-size','256m','--security-opt','no-new-privileges:true','-v',`${directory}:/restore-input:ro`,'-e','JIANXI_TOKEN','-e','JIANXI_URL=http://127.0.0.1:8080',compilerImage],{env:{...process.env,JIANXI_TOKEN:token},stdio:['pipe','pipe','pipe'],windowsHide:true});
 }else{
  const executable=fileURLToPath(new URL(process.platform==='win32'?'../bin/jianxi.exe':'../bin/jianxi',import.meta.url));
  const renderer=fileURLToPath(new URL('./compile.mjs',import.meta.url));
  child=spawn(executable,['mcp','--renderer',renderer],{env:{...process.env,JIANXI_URL:base,JIANXI_TOKEN:token},stdio:['pipe','pipe','pipe'],windowsHide:true});
 }
 child.stderr.on('data',bytes=>{stderr=(stderr+bytes).slice(-16384)});
 child.on('error',error=>{for(const task of pending.values())task.reject(error);pending.clear()});
 child.on('exit',()=>{closed=true;for(const task of pending.values())task.reject(Error('MCP exited: '+stderr));pending.clear()});
 createInterface({input:child.stdout}).on('line',line=>{
  let message;try{message=JSON.parse(line)}catch{for(const task of pending.values())task.reject(Error('MCP stdout is not JSON'));return;}
  const task=pending.get(message.id);if(!task)return;pending.delete(message.id);
  message.error?task.reject(Error(JSON.stringify(message.error))):task.resolve(message.result);
 });
 await rpc('initialize',{protocolVersion:'2025-06-18',capabilities:{},clientInfo:{name:'docker-smoke',version:'2'}});
 child.stdin.write(JSON.stringify({jsonrpc:'2.0',method:'notifications/initialized'})+'\n');
 const listed=await rpc('tools/list',{});
 assert.ok(listed.tools.some(tool=>tool.name==='blog_prepare_restore'));
 assert.ok(!listed.tools.find(tool=>tool.name==='blog_upload_content').inputSchema.properties.bundle);
 const media=[];
 for(const format of ['png','jpeg','webp','gif','avif']){
  const bytes=await sharp({create:{width:20,height:12,channels:3,background:'#fff'}}).toFormat(format).toBuffer();
  const asset=await tool('blog_store_asset',{base64:bytes.toString('base64'),alt:'验收图片'});
  assert.equal(asset.mime,'image/'+format);assert.equal(asset.width,20);assert.equal(asset.height,12);media.push(asset.id);
 }
 const copy=await tool('blog_working_copy',{});
 const markdown=(await readFile(new URL('../examples/rendering-showcase.md',import.meta.url),'utf8')).replace(/\/media\/[a-f0-9]{64}/g,`/media/${media[0]}`)+'\n'+media.map(id=>`![验收图片](/media/${id})`).join('\n\n');
 const content={...copy.content,slug:'mcp-compilation-check',title:'本地 MCP 编译验收',markdown};
 const upload=await tool('blog_upload_content',{copyId:copy.id,content,idempotencyKey:'local-compile'});
 assert.equal(upload.status,'parsed');assert.ok(upload.renderBundle);assert.ok(upload.compiled.math);assert.ok(Object.keys(upload.compiled.diagrams).length>=8);
 assert.equal((await tool('blog_parse_upload',{id:upload.id})).status,'parsed');
 const revision=await tool('blog_commit_draft',{uploadId:upload.id,sourceHash:upload.sourceHash,expectedDraftRevision:copy.expectedDraftRevision,epoch:copy.epoch,idempotencyKey:'local-commit'});
 await publish(revision.revision);
 const article=await fetch(base+'/posts/mcp-compilation-check/');assert.equal(article.status,200);
 const body=await article.text();assert.ok(body.includes('katex')&&body.includes('shiki github-light'));
 const diagrams=[...new Set([...body.matchAll(/\/diagrams\/[a-f0-9]{64}\.svg/g)].map(match=>match[0]))];
 assert.ok(diagrams.length>=8);
 for(const path of diagrams){const response=await fetch(base+path);assert.equal(response.status,200);assert.ok((await response.text()).includes('<svg'));}
 const activeBefore=await status();
 const rebuilt=await tool('blog_recompile_revision',{contentId:content.id,revision:revision.revision});
 assert.equal(rebuilt.revision,revision.revision);assert.equal((await status()).activeRelease,activeBefore.activeRelease);
 console.log('PASS: actual stdio MCP, five image formats, showcase rich compilation, preview/confirm/publish and offline recompile');
 const backup=await tool('blog_backup');
 await docker('cp',`${name}:/data/backups/${backup.file}`,join(directory,'backup.zip'));
 // The non-root MCP container has a different UID from the Linux CI runner.
 // Permit traversal and reading only this synthetic test backup; token files stay private.
 if(compilerImage&&process.platform!=='win32'){
  await chmod(directory,0o711);
  await chmod(join(directory,'backup.zip'),0o644);
 }
 const candidate=await tool('blog_prepare_restore',{path:compilerImage?'/restore-input/backup.zip':join(directory,'backup.zip')});
 assert.equal((await status()).activeRelease,activeBefore.activeRelease);
 await tool('blog_restore_backup',{id:candidate.id,artifactHash:candidate.artifactHash});
 const restored=await status();assert.equal(restored.articles,activeBefore.articles);
 assert.equal((await fetch(base+'/posts/mcp-compilation-check/')).status,200);
 console.log('PASS: MCP offline backup rebuild and Node-free server restore');
 if(compilerImage)await docker('stop',compilerName);else child.stdin.end();
 for(let i=0;i<20&&!closed;i++)await sleep(100);
 if(!closed)child.kill();
 const beforeRestart=await status();
 await docker('restart',name);
 base=`http://127.0.0.1:${(await docker('port',name,'8080/tcp')).split(':').at(-1)}`;
 await ready();
 const after=await status();assert.equal(after.activeRelease,beforeRestart.activeRelease);assert.equal(after.articles,beforeRestart.articles);
 assert.equal((await fetch(base+route)).status,200);assert.equal((await fetch(base+'/posts/mcp-compilation-check/')).status,200);
 for(let i=0;i<40;i++){if(await docker('inspect','--format','{{.State.Health.Status}}',name)==='healthy')break;await sleep(1000);}
 assert.equal(await docker('inspect','--format','{{.State.Health.Status}}',name),'healthy');
 assert.equal(await docker('inspect','--format','{{.State.OOMKilled}}',name),'false');
 console.log('Server memory after rich content/restore/restart: '+await docker('stats','--no-stream','--format','{{.MemUsage}}',name));
 console.log('PASS: persistence, stable admin route/token, healthcheck and no server OOM');
}catch(error){
 console.error(await docker('logs',name).catch(()=>''));if(stderr)console.error(stderr);throw error;
}finally{
 if(child&&!closed)child.kill();
 await docker('rm','-f',compilerName).catch(()=>{});
 await docker('rm','-f','-v',name).catch(()=>{});
 for(const task of pending.values())task.reject(Error('Test finished'));pending.clear();
 if(resolve(directory).startsWith(resolve(tmpdir())+sep)&&basename(directory).startsWith('jianxi-docker-smoke-'))await rm(directory,{recursive:true,force:true});
}
