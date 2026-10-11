import {readFile} from 'node:fs/promises';
import {spawn} from 'node:child_process';
import {createInterface} from 'node:readline';
import {fileURLToPath} from 'node:url';
import {parseArgs} from 'node:util';
import {resolve,join} from 'node:path';
const {values}=parseArgs({options:{data:{type:'string',default:'data/dev'},url:{type:'string',default:'http://127.0.0.1:8080'}}});
const token=(await readFile(join(resolve(values.data),'initial-token.txt'),'utf8')).trim();
const executable=new URL(process.platform==='win32'?'../bin/jianxi.exe':'../bin/jianxi',import.meta.url);
const child=spawn(fileURLToPath(executable),['mcp'],{env:{...process.env,JIANXI_URL:values.url,JIANXI_TOKEN:token},stdio:['pipe','pipe','pipe'],windowsHide:true});
const pending=new Map();let id=0;let stderr='';child.stderr.on('data',b=>stderr+=b);
const lines=createInterface({input:child.stdout});lines.on('line',line=>{let msg;try{msg=JSON.parse(line)}catch{throw Error('MCP stdout contained non-JSON output')};const task=pending.get(msg.id);if(task){pending.delete(msg.id);msg.error?task.reject(Error(JSON.stringify(msg.error))):task.resolve(msg.result)}});
function call(method,params){return new Promise((resolve,reject)=>{const requestId=++id;pending.set(requestId,{resolve,reject});child.stdin.write(JSON.stringify({jsonrpc:'2.0',id:requestId,method,params})+'\n')})}
const timeout=setTimeout(()=>{child.kill();throw Error('MCP check timed out')},15000);
try{
 const init=await call('initialize',{protocolVersion:'2025-06-18',capabilities:{},clientInfo:{name:'jianxi-smoke-check',version:'1'}});
 child.stdin.write(JSON.stringify({jsonrpc:'2.0',method:'notifications/initialized'})+'\n');
 const tools=await call('tools/list',{});if(tools.tools.length<20)throw Error('Missing tools');
 const result=await call('tools/call',{name:'blog_status',arguments:{}});if(result.isError)throw Error('Status tool failed');
 const rules=await call('tools/call',{name:'blog_content_rules',arguments:{}});if(rules.isError)throw Error('Content rules tool failed');
 const contract=rules.structuredContent?.result;
 if(!Array.isArray(contract?.syntax)||contract.syntax.length<27)throw Error('Missing syntax rules');
 for(const name of ['headings','code_blocks','tables','images','footnotes','keywords','escaping','inline_math','block_math','math_fences','mermaid_flowchart','mermaid_sequence','mermaid_class','mermaid_state','mermaid_er','mermaid_gantt','mermaid_pie','mermaid_timeline'])
  if(!contract.syntax.some(s=>s.name===name&&s.description&&s.example))throw Error(`Missing syntax description: ${name}`);
 if(!tools.tools.find(t=>t.name==='blog_upload_content')?.description.includes('blog_content_rules'))throw Error('Upload tool does not reference syntax rules');
 console.log(JSON.stringify({protocol:init.protocolVersion,tools:tools.tools.length,syntaxes:contract.syntax.length,status:'passed'}));
}finally{clearTimeout(timeout);child.stdin.end();child.kill()}
