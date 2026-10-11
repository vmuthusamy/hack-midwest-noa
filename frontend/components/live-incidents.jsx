'use client';
import {useEffect,useState} from 'react';
import Link from 'next/link';
import {WorkspaceHeader} from './workspace-header';
import {Badge,Panel,SourceLink} from './ui';
import {FACILITIES,api} from '../lib/case-ui.mjs';
import styles from './live-incidents.module.css';

const REGIONS='AL:Alabama|AK:Alaska|AZ:Arizona|AR:Arkansas|CA:California|CO:Colorado|CT:Connecticut|DE:Delaware|DC:District of Columbia|FL:Florida|GA:Georgia|HI:Hawaii|ID:Idaho|IL:Illinois|IN:Indiana|IA:Iowa|KS:Kansas|KY:Kentucky|LA:Louisiana|ME:Maine|MD:Maryland|MA:Massachusetts|MI:Michigan|MN:Minnesota|MS:Mississippi|MO:Missouri|MT:Montana|NE:Nebraska|NV:Nevada|NH:New Hampshire|NJ:New Jersey|NM:New Mexico|NY:New York|NC:North Carolina|ND:North Dakota|OH:Ohio|OK:Oklahoma|OR:Oregon|PA:Pennsylvania|RI:Rhode Island|SC:South Carolina|SD:South Dakota|TN:Tennessee|TX:Texas|UT:Utah|VT:Vermont|VA:Virginia|WA:Washington|WV:West Virginia|WI:Wisconsin|WY:Wyoming|AS:American Samoa|GU:Guam|MP:Northern Mariana Islands|PR:Puerto Rico|VI:US Virgin Islands'.split('|').map(x=>x.split(':'));
const FACILITY_REGIONS={batesburg:'SC',barnwell:'SC',florida:'FL',california:'CA',montana:'MT'};
function stamp(value){if(!value)return 'Not available';const d=new Date(value);return Number.isFinite(+d)?d.toLocaleString(undefined,{month:'short',day:'numeric',hour:'numeric',minute:'2-digit',timeZoneName:'short'}):'Not available';}
function age(value,now){const seconds=Math.max(0,Math.floor((now-new Date(value))/1000));return Number.isFinite(seconds)?seconds<60?`${seconds}s ago`:`${Math.floor(seconds/60)}m ago`:'Not checked yet';}
export default function LiveIncidents({identity}) {
 const [facility,setFacility]=useState('batesburg'),[region,setRegion]=useState('SC'),[data,setData]=useState(null),[error,setError]=useState(''),[now,setNow]=useState(Date.now()),[scope,setScope]=useState('region'),[limit,setLimit]=useState(20),[theme,setTheme]=useState('dark');
 useEffect(()=>{const timer=setInterval(()=>setNow(Date.now()),1000);return()=>clearInterval(timer)},[]);
 useEffect(()=>{
  let active=true,timer;const controller=new AbortController();setData(null);setError('');setLimit(20);
  async function refresh(){if(document.visibilityState==='hidden'){timer=setTimeout(refresh,15000);return;}
   try{const result=await api(`/api/incidents?facility=${facility}&region=${region}`,undefined,controller.signal);if(active){setData(result);setError('');}}
   catch(e){if(active&&e.name!=='AbortError')setError(e.message);}
   finally{if(active)timer=setTimeout(refresh,15000);}
  }
  refresh();return()=>{active=false;clearTimeout(timer);controller.abort();};
 },[facility,region]);
 const overdue=!!error||data?.health!=='current'||now-new Date(data?.lastSuccessfulRefresh)>90000;
 const alerts=data?.alerts||[];
 const shown=scope==='facility'?alerts.filter(a=>a.geography==='applicable'):alerts;
 const applicable=alerts.filter(a=>a.applicability==='applicable').length;
 const regionName=REGIONS.find(([id])=>id===region)?.[1]||(region==='ALL'?'United States':'Marine / unmapped');
 return <div className={`desk ${theme==='light'?'light':''}`}><WorkspaceHeader identity={identity} theme={theme} onTheme={()=>setTheme(theme==='dark'?'light':'dark')}/><main className={styles.page}>
  <div className={styles.heading}><div><Badge tone={overdue?'warn':'good'}>{!data&&!error?'LIVE · checking':overdue?'LIVE · check overdue':'LIVE'}</Badge><h1>{regionName} weather</h1><p>NWS feed · updated every minute</p></div><Link className="button" href={`/?facility=${facility}`}>Open manager case</Link></div>
  <div className={styles.filters}><label>Region<select value={region} onChange={e=>{setRegion(e.target.value);setScope('region');setLimit(20)}}><option value="ALL">All states and territories</option>{REGIONS.map(([id,name])=><option key={id} value={id}>{name}{data?.regionCounts?.[id]?` (${data.regionCounts[id]})`:''}</option>)}<option value="OTHER">Marine / unmapped</option></select></label><label>Fictional office<select value={facility} onChange={e=>{setFacility(e.target.value);setRegion(FACILITY_REGIONS[e.target.value]);setLimit(20)}}>{Object.entries(FACILITIES).map(([id,info])=><option value={id} key={id}>{info[0]}</option>)}</select></label></div>
  {(error||data?.error)&&<p role="alert" className={styles.warning}>{error||data.error}. Current conditions are unknown until checking succeeds.</p>}
  {data?.persistenceError&&<p role="alert" className={styles.warning}>{data.persistenceError}</p>}
  <div className={styles.stats}><div><strong>{data?alerts.length:'—'}</strong><span>{overdue?'Alerts in last snapshot':'Regional alerts'}</span></div><div><strong>{data?applicable:'—'}</strong><span>{overdue?'Last matched to office':'Office matches'}</span></div><div><strong>{age(data?.lastSuccessfulRefresh,now)}</strong><span>Last checked</span></div></div>
  <Panel className={styles.alerts}><div className={styles.sectionHead}><h2>Alerts near your operation</h2><label className={styles.toggle}><input type="checkbox" checked={scope==='facility'} onChange={e=>{setScope(e.target.checked?'facility':'region');setLimit(20)}}/> Office matches only</label></div>
   <p className={styles.note}>A regional alert may not cover this office. Times use your device’s timezone.</p>
   {!data&&!error&&<p role="status">Loading the shared NWS snapshot…</p>}
   {data&&shown.length===0&&<p>{overdue?'No matching alerts in the last snapshot. Current conditions are unknown.':'No active alerts found in this view. This does not establish safe conditions.'}</p>}
   {shown.slice(0,limit).map(a=><article className={styles.alert} key={a.id}><div className={styles.alertTitle}><h3>{a.event}</h3><Badge tone={a.geography==='applicable'?'warn':''}>{a.geography==='applicable'?'Covers office':a.geography==='outside'?'Regional only':'Coverage unknown'}</Badge></div><p className={styles.area}>{a.area}</p><p className={styles.note}>{a.lifecycle==='unknown'?'Timing unknown · ':a.lifecycle==='not-yet-effective'?'Upcoming · ':''}Issued {stamp(a.issued)} · Message expires {stamp(a.expires)}</p>{a.geography==='applicable'&&a.instruction&&<p className={styles.instruction}>{a.instruction}</p>}<details><summary>Areas, source and timing</summary><p>{a.area}</p><p>{a.severity} · {a.certainty} · {a.urgency}</p><p>Office match: {a.matchMethod}</p><p>NWS status: {a.status} · {a.messageType}</p><SourceLink href={a.source}>Official NWS alert</SourceLink><p className={styles.id}>{a.id}</p></details></article>)}
   {shown.length>limit&&<button className="button" onClick={()=>setLimit(limit+20)}>Show 20 more ({shown.length-limit} remaining)</button>}
  </Panel>
  <Panel><h2>Recent changes in {regionName}</h2><p className={styles.note}>Observed since {stamp(data?.startedAt)}. Latest {Math.min(100,data?.historyTotal||0)} of {data?.historyTotal||0} retained changes in this region.</p>
   <div className={styles.history}>{data?.history?.map((entry,i)=><div className={styles.historyRow} key={`${entry.id}-${entry.observedAt}-${i}`}><time>{stamp(entry.observedAt)}</time><div><strong>{entry.event}</strong><p>{entry.kind}</p><details><summary>Area and source</summary><p>{entry.area}</p><p>NWS issued {stamp(entry.issued)} · Message expires {stamp(entry.expires)}</p><SourceLink href={entry.id}>NWS source</SourceLink></details></div></div>)}</div>
   {data&&!data.history?.length&&<p>No observed changes recorded for this region yet.</p>}
  </Panel>
  <details className={styles.method}><summary>How updates and history work</summary><p>One server collector checks the national NWS feed every 60 seconds. All regions share that snapshot; this page reads the cache every 15 seconds while visible. Conditional requests reuse unchanged feeds. Failed checks back off to at most five minutes. Last successful check: {stamp(data?.lastSuccessfulRefresh)}. NWS feed timestamp: {stamp(data?.feedUpdated)}.</p><p>Changes are retained for seven days, capped at 5,000 nationally. {data?.prunedEvents||0} older events have been removed. Initial snapshot means already present when monitoring began. “Left active feed” does not establish cancellation, ended danger or safe reopening. This is observed history, not a complete NWS archive.</p><p>Publication delays and polling gaps are possible. Official warnings are shown without waiting for AI. Monitoring runs no inference or evaluations and does not change an approved decision. Open the manager case and explicitly check live weather to review a new operating decision.</p></details>
 </main></div>;
}
