// Independent coherent-case gateway QA. All sessions and backends are fixtures.
import test from 'node:test';
import assert from 'node:assert/strict';
import {createHash,createHmac} from 'node:crypto';
import {forwardRequest} from '../frontend/lib/gateway.mjs';
const env={NODE_ENV:'production',AUTH_GOOGLE_ID:'fixture-client',AUTH_GOOGLE_SECRET:'fixture-secret',AUTH_SECRET:'a'.repeat(32),WEATHER_DESK_GATEWAY_SECRET:'b'.repeat(32),AUTH_URL:'https://case-qa.example',WEATHER_DESK_API_URL:'http://fixture-backend.example',AUTH_MANAGER_EMAILS:'manager@example.test'};
const request=(path,method='GET',body,headers={})=>new Request(env.AUTH_URL+path,{method,headers:{...(method==='POST'?{'Origin':env.AUTH_URL,'Content-Type':'application/json'}:{}),...headers},...(body===undefined?{}:{body:typeof body==='string'?body:JSON.stringify(body)})});
function fixture(role='manager',changes={}){const calls=[];return{calls,deps:{env:{...env,...changes},getSession:async()=>({user:{id:'google:qaSubject',name:'Synthetic case QA',email:role+'@example.test',emailVerified:true,role:'manager',facilities:['FORGED']}}),fetchBackend:async(url,options)=>{calls.push({url:String(url),...options});return Response.json({fixture:true})}}};}
const writes=['/api/case?facility=montana','/api/case/action?facility=barnwell','/api/case/briefing?facility=florida'];
const reads=['/api/case?facility=montana','/api/records?facility=barnwell','/api/records/qa_record-123?facility=florida','/api/metrics?facility=california'];

test('Case reads and printable records admit verified viewers with server-derived scope',async()=>{
 const {calls,deps}=fixture('viewer');for(const path of reads)assert.equal((await forwardRequest(request(path),deps)).status,200,path);
 assert.equal(calls.length,reads.length);for(const call of calls){const principal=JSON.parse(Buffer.from(call.headers['X-Weather-Principal'],'base64url'));assert.equal(principal.role,'viewer');assert(principal.facilities.includes('MT-DEMO-01'));assert(!principal.facilities.includes('FORGED'));assert.equal(call.cache,'no-store');}
});
test('All case creation, draft, approval, queue and AI writes require manager',async()=>{
 const {calls,deps}=fixture('viewer');for(const path of writes)for(const action of ['choose','approve','simulate'])assert.equal((await forwardRequest(request(path,'POST',{action,revision:'fixture'}),deps)).status,403,path);assert.equal(calls.length,0);
});
test('Case writes reject absent or cross-site origin and missing OAuth configuration',async()=>{
 for(const path of writes)for(const origin of ['', 'https://different.example','null']){const {calls,deps}=fixture();assert.equal((await forwardRequest(request(path,'POST',{}, {'Origin':origin}),deps)).status,403);assert.equal(calls.length,0);}
 for(const key of ['AUTH_GOOGLE_ID','AUTH_GOOGLE_SECRET','AUTH_SECRET','WEATHER_DESK_GATEWAY_SECRET']){const {calls,deps}=fixture('manager',{[key]:''});assert.equal((await forwardRequest(request('/api/case?facility=barnwell','POST',{mode:'REPLAY'}),deps)).status,503);assert.equal(calls.length,0);}
});
test('New routes reject wrong methods, unsupported facility and malformed record IDs',async()=>{
 for(const [path,method,status] of [['/api/case','DELETE',405],['/api/case/action','GET',405],['/api/case/briefing','GET',405],['/api/records','POST',405],['/api/records/qa-id','POST',405],['/api/metrics','POST',405],['/api/records/a%2Fb','GET',404],['/api/records/a.b','GET',404],['/api/records/'+'x'.repeat(129),'GET',404],['/api/case?facility=unlisted','GET',404]]){const {calls,deps}=fixture();assert.equal((await forwardRequest(request(path,method,method==='POST'?{}:undefined),deps)).status,status,path);assert.equal(calls.length,0);}
});
test('Case approval forwarding preserves reviewed bytes and signs exact request',async()=>{
 const {calls,deps}=fixture();const path='/api/case/action?facility=montana';const body='{"action":"choose","revision":"exact","operatingPosture":"closure","rationale":"Reviewed rationale","effectiveAt":"2026-10-10T22:00:00Z","messages":{"qa-person":"Exact individual message"}}';
 assert.equal((await forwardRequest(request(path,'POST',body,{'X-Weather-Principal':'forged','Cookie':'fixture-browser-cookie'}),deps)).status,200);
 const call=calls[0],encoded=call.headers['X-Weather-Principal'],principal=JSON.parse(Buffer.from(encoded,'base64url'));assert.equal(call.body.toString(),body);assert.equal(principal.path,path);assert.equal(principal.method,'POST');assert.equal(principal.body_sha256,createHash('sha256').update(body).digest('hex'));assert.equal(call.headers['X-Weather-Signature'],createHmac('sha256',env.WEATHER_DESK_GATEWAY_SECRET).update(encoded).digest('hex'));assert.equal(principal.sub,'google:qaSubject');assert.equal(principal.role,'manager');assert.equal(call.headers.Cookie,undefined);assert.equal(call.redirect,'error');
});
test('Case APIs and records reject unauthenticated or legacy-provider sessions before backend',async()=>{
 for(const session of [null,{user:{id:'github:legacy',email:'manager@example.test',emailVerified:true}},{user:{id:'google:qaSubject',email:'manager@example.test',emailVerified:false}}])for(const path of [...reads,...writes]){const {calls,deps}=fixture();const method=writes.includes(path)?'POST':'GET';assert.equal((await forwardRequest(request(path,method,method==='POST'?{}:undefined),{...deps,getSession:async()=>session})).status,401);assert.equal(calls.length,0);}
});
test('Case write size/type and printable backend failures remain fail-closed',async()=>{
 for(const [body,headers,status] of [['{}',{'Content-Type':'text/plain'},415],['x'.repeat(32769),{},413]]){const {calls,deps}=fixture();assert.equal((await forwardRequest(request('/api/case/action','POST',body,headers),deps)).status,status);assert.equal(calls.length,0);}
 const {deps}=fixture();const response=await forwardRequest(request('/api/records/qa-id'),{...deps,fetchBackend:async()=>new Response('<private fixture html>',{status:500,headers:{'Content-Type':'text/html'}})});assert.equal(response.status,502);assert.equal(response.headers.get('Cache-Control'),'no-store');assert(!(await response.text()).includes('private fixture'));
});
