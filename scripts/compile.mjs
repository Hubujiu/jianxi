import {readFile,writeFile,mkdtemp,rm} from 'node:fs/promises';
import {tmpdir} from 'node:os';
import {join,resolve} from 'node:path';
import {fileURLToPath} from 'node:url';
import {execFile} from 'node:child_process';
import {promisify} from 'node:util';
const exec=promisify(execFile);
let input='';for await(const chunk of process.stdin){input+=chunk;if(input.length>16*1024*1024)throw Error('RENDER_LIMIT_EXCEEDED');}
const request=JSON.parse(input);
try {
 if(request.kind==='image'){
  const {default:sharp}=await import('sharp');
  const data=Buffer.from(request.source,'base64');
  const meta=await sharp(data,{limitInputPixels:40_000_000,animated:true}).metadata();
  if(!['png','jpeg','webp','gif','heif'].includes(meta.format))throw Error('不支持的图片类型');
  if(meta.format==='heif'&&meta.compression!=='av1')throw Error('只接受 AVIF，不接受 HEIC');
  const format=meta.format==='heif'?'avif':meta.format;
  if(!meta.width||!meta.height||meta.width*(meta.pageHeight||meta.height)>40_000_000)throw Error('图片像素超出限制');
  console.log(JSON.stringify({format,width:meta.width,height:meta.pageHeight||meta.height}));
 } else if(request.kind==='code'){
  const {createHighlighter,bundledLanguages,bundledLanguagesAlias}=await import('shiki');
  const blocks=request.blocks;
  if(!Array.isArray(blocks)||blocks.length>10000)throw Error('代码块数量超出限制');
  const language=raw=>{
   const name=String(raw||'').trim().toLowerCase();
   return Object.hasOwn(bundledLanguages,name)||Object.hasOwn(bundledLanguagesAlias,name)?name:'text';
  };
  const langs=[...new Set(blocks.map(block=>language(block.language)))].filter(lang=>lang!=='text');
  const highlighter=await createHighlighter({themes:['github-light'],langs});
  try{
   const html=blocks.map(block=>highlighter.codeToHtml(block.source,{lang:language(block.language),theme:'github-light'}));
   console.log(JSON.stringify({html}));
  }finally{highlighter.dispose();}
 } else if(request.kind==='math'){
  const {default:katex}=await import('katex');
  if(request.source.length>8192)throw Error('公式超过 8 KiB');
  const html=katex.renderToString(request.source,{displayMode:request.display,throwOnError:true,trust:false,strict:'error',maxExpand:100,maxSize:20,output:'htmlAndMathml'});
  console.log(JSON.stringify({html}));
 } else if(request.kind==='mermaid'){
  const source=request.source;
  if(source.length>16384)throw Error('图表超过 16 KiB');
  if(!/^\s*(flowchart|graph|sequenceDiagram|classDiagram|stateDiagram(?:-v2)?|erDiagram|gantt|pie|timeline)\b/.test(source))throw Error('UNSUPPORTED_MERMAID_TYPE');
  if(/%%\{|^---|\bclick\b|https?:|javascript:|data:|<|>|@\{|\blink\s|\bclassDef\b|\bstyle\b/im.test(source.replaceAll('-->','').replaceAll('->>','').replaceAll('->','').replaceAll('-->>','')))throw Error('图表不允许配置覆盖、HTML、回调、外部资源或自定义样式');
  const dir=await mkdtemp(join(tmpdir(),'jianxi-render-'));
  try{
   const inFile=join(dir,'input.mmd'),outFile=join(dir,'output.svg');
   await writeFile(inFile,source);
   await writeFile(join(dir,'config.json'),JSON.stringify({securityLevel:'strict',htmlLabels:false,theme:'base',themeVariables:{fontFamily:'sans-serif',primaryColor:'#f3f2ef',primaryTextColor:'#222222',primaryBorderColor:'#999999',lineColor:'#777777',secondaryColor:'#ffffff',tertiaryColor:'#e8e7e3'},flowchart:{htmlLabels:false},maxEdges:150,maxTextSize:16384}));
   const cli=fileURLToPath(new URL('../node_modules/@mermaid-js/mermaid-cli/src/cli.js',import.meta.url));
   const args=[cli,'-i',inFile,'-o',outFile,'-c',join(dir,'config.json'),'-b','transparent','-q'];
   if(process.env.PUPPETEER_EXECUTABLE_PATH){await writeFile(join(dir,'puppeteer.json'),JSON.stringify({executablePath:process.env.PUPPETEER_EXECUTABLE_PATH,...(process.env.JIANXI_CHROMIUM_HEADLESS_SHELL==='1'?{headless:'shell'}:{}),args:process.env.JIANXI_CHROMIUM_NO_SANDBOX==='1'?['--no-sandbox']:[]}));args.push('-p',join(dir,'puppeteer.json'));}
   await exec(process.execPath,args,{timeout:25000,maxBuffer:1024*1024,windowsHide:true});
   const svg=await readFile(outFile,'utf8');
   if(/<script|<foreignObject|<iframe|\son[a-z]+\s*=|(?:href|src)\s*=\s*["'](?:https?:|\/\/|data:|javascript:)/i.test(svg))throw Error('不安全的图表输出');
   console.log(JSON.stringify({svg}));
  }finally{await rm(dir,{recursive:true,force:true});}
 }else throw Error('未知编译类型');
}catch(error){console.log(JSON.stringify({error:String(error.message).slice(0,1800)}));process.exitCode=1;}
