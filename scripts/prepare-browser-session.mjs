// Local development helper. Never prints credentials; output belongs under data/.
import {readFile,writeFile} from 'node:fs/promises';
const token=(await readFile(new URL('../data/dev/initial-token.txt',import.meta.url),'utf8')).trim();
const response=await fetch('http://127.0.0.1:8080/admin/login',{method:'POST',redirect:'manual',headers:{'Content-Type':'application/x-www-form-urlencoded','Origin':'http://127.0.0.1:8080'},body:new URLSearchParams({token})});
if(response.status!==303)throw Error('Local login failed');
const value=response.headers.getSetCookie()[0].match(/^jianxi_session=([^;]+)/)[1];
await writeFile(new URL('../data/dev/browser-state.json',import.meta.url),JSON.stringify({cookies:[{name:'jianxi_session',value,domain:'127.0.0.1',path:'/admin',expires:Math.floor(Date.now()/1000)+3600,httpOnly:true,secure:false,sameSite:'Lax'}],origins:[]}),{mode:0o600});
console.log('Private browser session prepared.');
