// Explicit, opt-in migration for the original demo's hash-based taxonomy URLs.
// Custom slugs, article content and IDs are preserved. Uses the normal CAS /
// preview / confirmation / publication workflow, never edits SQLite directly.
import {readFile} from 'node:fs/promises';
import {createHash} from 'node:crypto';
import {resolve,join} from 'node:path';
import {parseArgs} from 'node:util';

const {values}=parseArgs({options:{
  data:{type:'string',default:'data/dev'},
  url:{type:'string',default:'http://127.0.0.1:8080'},
  'token-file':{type:'string'},
  apply:{type:'boolean',default:false},
}});
const origin=new URL(values.url);
if(!['127.0.0.1','localhost','[::1]'].includes(origin.hostname) || origin.protocol!=='http:' || origin.pathname!=='/' || origin.search || origin.hash || origin.username || origin.password)
  throw Error('Demo migration requires a local loopback HTTP server');
const token=(await readFile(values['token-file'] || join(resolve(values.data),'initial-token.txt'),'utf8')).trim();
const call=async (op,input={})=>{
  const response=await fetch(new URL(`/_control/${op}`,origin),{method:'POST',redirect:'error',headers:{Authorization:`Bearer ${token}`,'Content-Type':'application/json'},body:JSON.stringify(input)});
  const result=await response.json();
  if(!response.ok) throw Error(result.error || `Control ${op} failed: ${response.status}`);
  return result;
};
const demo=JSON.parse(await readFile(new URL('../internal/blog/web/demo.json',import.meta.url),'utf8'));
const translations=new Map();
for(const c of demo){
  if(c.category) translations.set(`category:${c.category.name}`,c.category.slug);
  for(const tag of c.tags || []) translations.set(`tag:${tag.name}`,tag.slug);
}
const taxonomy=await call('taxonomy-list');
const status=await call('status');
if(!/^[a-f0-9]{32}$/.test(status.activeRelease)) throw Error('Invalid release ID');
const snapshot=JSON.parse(await readFile(join(resolve(values.data),'releases',status.activeRelease,'snapshot.json'),'utf8'));
if(snapshot.id!==status.activeRelease) throw Error('Local data does not match the target server');
// Plan from the published snapshot, so a retry can publish candidates saved by
// an interrupted run. Never overwrite a candidate with a custom route or name.
const changes=[];
for(const t of Object.values(snapshot.taxonomy)){
  const target=translations.get(`${t.kind}:${t.name}`),candidate=taxonomy[t.id];
  if(!target || t.slug!==`${t.kind}-${createHash('sha256').update(t.name).digest('hex').slice(0,8)}` ||
     !candidate || candidate.deleted || candidate.kind!==t.kind || candidate.name!==t.name ||
     (candidate.slug!==t.slug && candidate.slug!==target)) continue;
  changes.push({taxon:candidate,from:t.slug,to:target});
}
changes.sort((a,b)=>a.taxon.kind.localeCompare(b.taxon.kind)||a.from.localeCompare(b.from));
console.log(JSON.stringify({routes:changes.map(({taxon:t,from,to})=>({name:t.name,from:`/${t.kind==='category'?'categories':'tags'}/${from}/`,to:`/${t.kind==='category'?'categories':'tags'}/${to}/`}))},null,2));
if(!values.apply){console.log('Read-only plan. Use --apply to publish these demo route changes.');}
else if(changes.length){
  // Capture publication base before any writes; concurrent releases cause CAS
  // failure rather than silently applying this plan to a different release.
  for(const {taxon:t,to} of changes)
    if(t.slug!==to) await call('taxonomy',{taxon:{...t,slug:to},expectedRevision:t.revision});
  const change=await call('changeset',{taxonomy:changes.map(({taxon:t})=>t.id),expectedActiveRelease:status.activeRelease});
  const preview=await call('preview',{id:change.id});
  const confirmed=await call('confirm',{previewId:preview.previewId,artifactHash:preview.artifactHash,changeSetHash:change.hash});
  const result=await call('publish',{previewId:preview.previewId,confirmationId:confirmed.confirmationId,changeSetHash:change.hash,expectedActiveRelease:status.activeRelease,idempotencyKey:`demo-routes:${change.id}`});
  console.log(JSON.stringify({published:result.release,changed:changes.length}));
}else{console.log('No legacy demo hash routes found; nothing to change.');}
