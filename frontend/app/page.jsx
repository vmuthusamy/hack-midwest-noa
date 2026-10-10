'use client';
import { useEffect, useRef, useState } from 'react';
import Link from 'next/link';
import { Activity, ArrowRight, ArrowUpRight, Check, ChevronDown, Clock3, CloudLightning, FileClock, Layers3, MapPin, Moon, Radar, RefreshCw, ShieldCheck, Sun, Users, X } from 'lucide-react';
import { Badge, Button, Panel, SourceLink } from '../components/ui';

const facilities = { barnwell: ['Barnwell Field Office', 'South Carolina', 'SC-DEMO-01'], florida: ['Big Bend Field Office', 'Florida', 'FL-DEMO-01'], california: ['Inland Field Office', 'California', 'CA-DEMO-01'] };
const groupLabels = { onsite: 'Already onsite', arriving: 'Not yet onsite', remote: 'Working remotely' };
const displayTime = value => value ? new Date(value).toLocaleString('en-US', { month: 'short', day: 'numeric', hour: 'numeric', minute: '2-digit', timeZoneName: 'short' }) : 'Not available';
async function api(path, body) {
  const response = await fetch(path, { method: body ? 'POST' : 'GET', headers: body ? { 'Content-Type': 'application/json' } : {}, body: body ? JSON.stringify(body) : undefined, cache: 'no-store' });
  const data = await response.json();
  if (!response.ok) throw Error(data.error || 'Request failed');
  return data;
}
function GeographyPlot({ alert, facility, replayCase }) {
  const geometry = alert?.geometry;
  const polygons = geometry?.type === 'Polygon' ? [geometry.coordinates] : geometry?.type === 'MultiPolygon' ? geometry.coordinates : [];
  const points = polygons.flat(2);
  if (!points.length || replayCase === 'outside' || replayCase === 'missing') return <div className="plot-empty"><MapPin size={32}/><strong>Geography {replayCase === 'outside' ? 'outside this facility' : 'not plotted'}</strong><p>{alert?.matchMethod || 'Select an alert with official polygon geometry.'}</p><span className="mono">{facility?.latitude}, {facility?.longitude}</span></div>;
  const xs = points.map(p => p[0]), ys = points.map(p => p[1]);
  const west = Math.min(...xs, facility.longitude), east = Math.max(...xs, facility.longitude), south = Math.min(...ys, facility.latitude), north = Math.max(...ys, facility.latitude);
  const dx = Math.max(east - west, .01), dy = Math.max(north - south, .01);
  const x = lon => 80 + (lon - west) / dx * 430;
  const y = lat => 260 - (lat - south) / dy * 200;
  return <div className="geography-plot"><svg viewBox="0 0 600 325" role="img" aria-label="Recorded NOAA warning polygon and fictional facility coordinates. Geographic extent plot, not a navigation map.">
    <defs><pattern id="grid" width="40" height="40" patternUnits="userSpaceOnUse"><path d="M 40 0 L 0 0 0 40" fill="none" stroke="currentColor" strokeWidth=".6"/></pattern><linearGradient id="warningFill" x2="1" y2="1"><stop stopColor="#e88649" stopOpacity=".32"/><stop offset="1" stopColor="#e88649" stopOpacity=".10"/></linearGradient></defs>
    <rect width="600" height="325" fill="url(#grid)" className="plot-grid"/>
    {[0,.5,1].map(v => <g key={v} className="plot-coordinates"><text x="15" y={y(south + v * dy) + 4}>{(south + v * dy).toFixed(2)}° N</text><text x={x(west + v * dx) - 24} y="304">{Math.abs(west + v * dx).toFixed(2)}° W</text></g>)}
    {polygons.map((polygon, i) => <path key={i} d={polygon.map(ring => ring.map((p,j) => `${j ? 'L' : 'M'}${x(p[0])},${y(p[1])}`).join(' ') + ' Z').join(' ')} fill="url(#warningFill)" fillRule="evenodd" stroke="#e88649" strokeWidth="2" strokeLinejoin="round"/>)}
    <circle cx={x(facility.longitude)} cy={y(facility.latitude)} r="15" fill="#67d8bc" fillOpacity=".12"/><circle cx={x(facility.longitude)} cy={y(facility.latitude)} r="5" fill="#67d8bc" stroke="var(--plot)" strokeWidth="2"/>
    <text x={x(facility.longitude) + 16} y={y(facility.latitude) - 12} className="plot-facility">Fictional facility</text>
    <text x="530" y="33" className="plot-north">N ↑</text>
  </svg><div className="plot-caption"><span><i className="legend-square"/>Official warning polygon</span><span><i className="legend-dot"/>Facility fixture</span></div></div>;
}
function ReviewDialog({ replay, busy, onClose, onApprove }) {
  const [messages, setMessages] = useState(Object.fromEntries(replay.messages.map(m => [m.group, m.text])));
  const dialog = useRef(null);
  useEffect(() => { dialog.current.showModal(); const d = dialog.current; return () => d?.close(); }, []);
  return <dialog ref={dialog} className="review-dialog" aria-labelledby="review-title" onCancel={onClose} onClick={e => { if (e.target === dialog.current) onClose(); }}>
    <div className="dialog-heading"><div><p className="eyebrow">T+6 · Manager review · Simulation</p><h2 id="review-title">Correct the earlier instruction</h2></div><Button onClick={onClose} aria-label="Close review"><X size={18}/></Button></div>
    <p className="muted">Review the exact text and recipient groups. These editable drafts are fixtures. Official emergency instructions remain separate.</p>
    {replay.messages.map(m => <div className="review-message" key={m.group}><label htmlFor={`message-${m.group}`}><strong>{groupLabels[m.group]}</strong><span className="muted">{m.recipients.length} recipients · {m.recipients.map(id => replay.roster.find(r => r.id === id).name).join(', ')}</span></label><textarea id={`message-${m.group}`} rows={4} value={messages[m.group]} onChange={e => setMessages({ ...messages, [m.group]: e.target.value })}/></div>)}
    <div className="approval-note"><ShieldCheck size={20}/><p>Approval binds this facility, action, recipients, text, policy, and evidence. OPA checks eligibility; Go re-checks the package before simulation.</p></div>
    <div className="dialog-footer"><Button onClick={onClose}>Back to evidence</Button><Button primary disabled={busy || Object.values(messages).some(t => !t.trim())} onClick={() => onApprove(messages)}>Approve correction package <ArrowRight size={16}/></Button></div>
  </dialog>;
}
export default function WeatherDesk() {
  const [facility, setFacility] = useState('barnwell');
  const [mode, setMode] = useState('REPLAY');
  const [replayData, setReplayData] = useState(null);
  const [live, setLive] = useState(null);
  const [lastRefresh, setLastRefresh] = useState(null);
  const [status, setStatus] = useState(null);
  const [briefing, setBriefing] = useState(null);
  const [busy, setBusy] = useState(false);
  const [review, setReview] = useState(false);
  const [feedback, setFeedback] = useState('');
  const [error, setError] = useState('');
  const [theme, setTheme] = useState('dark');
  const [now, setNow] = useState(Date.now());
  useEffect(() => { const timer=setInterval(()=>setNow(Date.now()),1000);return ()=>clearInterval(timer); }, []);
  const epoch = useRef(0);
  const evidenceIdentity = useRef(null);
  const replay = replayData?.replay;
  useEffect(() => { const check=()=>api('/api/status').then(setStatus).catch(()=>setStatus(null));check();const timer=setInterval(check,10000);return ()=>clearInterval(timer); }, []);
  useEffect(() => {
    const generation = ++epoch.current;
    setLive(null);setLastRefresh(null);setBriefing(null);setError('');setFeedback('');setReview(false);setBusy(false);setReplayData(null);evidenceIdentity.current=null;
    api(`/api/replay?facility=${facility}`).then(data => { if (epoch.current === generation) setReplayData(data); }).catch(e => { if (epoch.current === generation) setError(e.message); });
  }, [facility, mode]);
  const liveFresh = live && Number.isFinite(Date.parse(live.lastSuccessfulRefresh)) && now - Date.parse(live.lastSuccessfulRefresh) < 300000 && now >= Date.parse(live.lastSuccessfulRefresh) - 30000;
  const liveAlerts = (live?.alerts || []).map(a => {
    const end=Date.parse(a.expires),start=Date.parse(a.effective);
    const timeKnown=Number.isFinite(end)&&Number.isFinite(start)&&end>start;
    const lifecycle=a.messageType==='Cancel'?'cancelled':!timeKnown?'unknown':now>=end?'expired':now<start?'not-yet-effective':'active';
    const applicability=!liveFresh?'unknown':lifecycle==='unknown'?'unknown':lifecycle!=='active'||a.status!=='Actual'?'not-applicable':a.geography==='applicable'?'applicable':a.geography==='unknown'?'unknown':'not-applicable';
    return {...a,lifecycle,applicability};
  });
  const alert = mode === 'REPLAY' ? replay?.alert : liveAlerts.find(a => a.applicability === 'applicable' && a.lifecycle === 'active') || liveAlerts.find(a => a.geography === 'applicable' && a.status === 'Actual');
  const liveLabel = !liveFresh ? 'Stale · refresh required' : alert?.lifecycle==='expired' ? 'Expired · review required' : alert?.lifecycle==='cancelled' ? 'Cancelled · review required' : alert?.applicability==='applicable' ? 'Applicable at refresh' : 'Applicability unknown';
  const currentFacility = mode === 'REPLAY' ? replay?.facility : live?.facility || replay?.facility;
  const relevant = liveAlerts.filter(a => a.applicability === 'applicable' && a.lifecycle === 'active');
  const unknownCount = liveAlerts.filter(a => a.applicability === 'unknown').length;
  const canApprove = replay?.policyDecision?.allow === true;
  async function refresh() {
    const generation=epoch.current;setBusy(true);setError('');setBriefing(null);evidenceIdentity.current=null;
    try { const data=await api(`/api/live?facility=${facility}`);if (generation!==epoch.current) return;setLive(data);setLastRefresh(data.lastSuccessfulRefresh);evidenceIdentity.current=data.lastSuccessfulRefresh; }
    catch(e) { if (generation===epoch.current) {setLive(null);setError(e.message);} }
    finally { if(generation===epoch.current) setBusy(false); }
  }
  async function action(actionName, extras={}) {
    const generation=epoch.current;setBusy(true);setError('');
    try {const data=await api(`/api/replay/action?facility=${facility}`,{action:actionName,revision:replayData.revision,...extras});if(generation!==epoch.current)return;setReplayData(data);setReview(false);setFeedback(actionName==='simulate'?`${data.added} new simulated intents queued. No delivery or acknowledgment asserted.`:actionName==='approve'?'Correction approved. Re-check evidence before simulation.':'Replay condition updated.');}
    catch(e){if(generation===epoch.current)setError(e.message);}
    finally {if(generation===epoch.current)setBusy(false);}
  }
  async function explain() {
    const generation=epoch.current, timestamp=evidenceIdentity.current;setBusy(true);setError('');
    try {const data=await api(`/api/briefing?facility=${facility}`,{});if(generation!==epoch.current || timestamp!==evidenceIdentity.current || data.evidenceTimestamp!==timestamp)return;setBriefing(data.briefing);}
    catch(e){if(generation===epoch.current)setError(e.message);}
    finally {if(generation===epoch.current)setBusy(false);}
  }
  const held = replay?.heldRecipients || [];
  return <div className={`desk ${theme==='light'?'light':''}`}>
    <aside className="sidebar">
      <Link href="/" className="brand"><span className="brand-symbol"><CloudLightning size={23}/></span><span>weather<span className="brand-secondary">desk</span></span></Link>
      <div className="workspace-label">FACILITY OPERATIONS</div>
      <nav aria-label="Workspace"><a className="nav-item active" href="#overview"><Radar size={18}/>Situation overview</a><a className="nav-item" href="#recipients"><Users size={18}/>People & notices</a><a className="nav-item" href="#history"><FileClock size={18}/>Decision history</a><Link className="nav-item" href="/gallery"><Layers3 size={18}/>Design library</Link><Link className="nav-item" href="/boundaries"><ShieldCheck size={18}/>System boundaries</Link></nav>
      <div className="sidebar-scenarios"><p className="workspace-label">FICTIONAL FACILITIES</p>{Object.entries(facilities).map(([key,info])=><button key={key} onClick={()=>setFacility(key)} className={`facility-item ${facility===key?'selected':''}`}><span className="facility-indicator"/><span>{info[0]}<small>{info[1]}</small></span>{facility===key&&<ChevronDown size={13}/>}</button>)}</div>
      <div className="sidebar-bottom"><div className="session-person"><span className="avatar">DM</span><div>Demo manager<small>No account signed in</small></div></div><div className="sidebar-foot">NOAA / NWS evidence<br/>Synthetic people · simulated notices</div></div>
    </aside>
    <div className="desk-body"><header className="topbar"><div className="breadcrumb">Operations <span>/</span> {facilities[facility][1]}</div><div className="topbar-tools"><span className="prototype-label">DEMO MANAGER · NOT SIGNED IN</span><Button aria-label={`Switch to ${theme==='dark'?'light':'dark'} theme`} onClick={()=>setTheme(theme==='dark'?'light':'dark')}>{theme==='dark'?<Sun size={17}/>:<Moon size={17}/>}</Button></div></header>
      <main id="overview" className="main-content"><div className="page-heading"><div><p className="eyebrow">EVIDENCE → ACTION</p><h1>Situation overview<span className="heading-dot">.</span></h1><p className="muted">{facilities[facility][0]} <span className="divider-text">·</span> Fictional facility <span className="divider-text">·</span> {facilities[facility][2]}</p></div><div className="mode-control" aria-label="Data mode">{['REPLAY','LIVE'].map(m=><button key={m} aria-pressed={mode===m} className={mode===m?'selected':''} onClick={()=>setMode(m)}>{m==='REPLAY'?<FileClock size={14}/>:<Activity size={14}/>} {m==='REPLAY'?'Recorded replay':'Live NOAA'}</button>)}</div></div>
        <div className="mode-banner"><span className={`status-dot ${mode==='LIVE'?'mint':'amber'}`}/><strong>{mode==='REPLAY'?'Recorded NOAA · Replay':'Live mode. Manual refresh.'}</strong><span>{mode==='REPLAY'?`${displayTime(replay?.capturedAt)} · Not live`:'Facility applicability checked at refresh.'}</span>{mode==='LIVE'&&<Button onClick={refresh} disabled={busy}>{busy?<RefreshCw size={14} className="spin"/>:<RefreshCw size={14}/>}Refresh NOAA</Button>}</div>
        {error&&<div className="feedback error" role="alert">{error}</div>}{feedback&&<div className="feedback" role="status">{feedback}</div>}
        {!replay&&<Panel className="loading">Loading the recorded scenario…</Panel>}
        {replay&&<>
          <section className={`official-alert ${alert?'':'no-alert'}`} aria-labelledby="official-heading"><div className="alert-icon"><CloudLightning size={25}/></div><div className="official-content"><div className="alert-heading"><div><p className="eyebrow">{mode==='REPLAY'?'RECORDED OFFICIAL NOAA / NWS WARNING':'OFFICIAL NOAA / NWS EVIDENCE'}</p><h2 id="official-heading">{alert?.event || (live?'No applicable warning returned':'Refresh NOAA to see current evidence')}</h2></div><Badge tone={alert?'amber':''}>{mode==='REPLAY'?'REPLAY':alert?liveLabel:live?(liveFresh?'No applicable alert':'Stale · refresh required'):'Unknown until checked'}</Badge></div><p className="alert-area">{alert?`${alert.issuer} · ${alert.area}`:live?`${relevant.length} applicable alerts. ${unknownCount} alert records have unknown applicability.`:'Live weather is not inferred from the recorded scenario.'}</p>{alert?<><p className="official-instruction">{alert.instruction || 'No instruction text was supplied in this alert. Open the official source for details.'}</p><div className="alert-meta"><span>Issued {displayTime(alert.issued)}</span><span>Expires {displayTime(alert.expires)}</span><SourceLink href={alert.source}>Official source</SourceLink></div><p className="warning-boundary">{mode==='REPLAY'?'Recorded official text.':!liveFresh?'Last received official text. Refresh required; current conditions are unknown.':['expired','cancelled'].includes(alert?.lifecycle)?'Ended warning; no reopening decision.':alert?.lifecycle==='not-yet-effective'?'Not yet effective.':alert?.lifecycle==='unknown'?'Timing or applicability unknown.':'Official instructions display immediately.'} Official instructions bypass approval.</p></>:<p className="warning-boundary">Missing data or the absence of a warning does not establish safe conditions or authorize reopening.</p>}</div></section>
          <div className="evidence-layout"><Panel className="geography-panel"><div className="panel-header"><div><p className="eyebrow">LOCATION & APPLICABILITY</p><h3>Does it cover this facility?</h3></div><Badge tone={alert?.geography==='applicable'?'mint':''}>{alert?.geography || 'Unknown'}</Badge></div><GeographyPlot alert={alert} facility={currentFacility} replayCase={mode==='REPLAY'?replay.case:null}/><div className="map-evidence"><MapPin size={15}/><span>{alert?.matchMethod || 'No applicable geometry selected'}</span><span className="mono">{currentFacility?.latitude}°, {currentFacility?.longitude}°</span></div><p className="microcopy">Official polygon · Fictional facility · Geographic extent plot</p></Panel>
            <Panel className="decision-panel"><div className="panel-header"><div><p className="eyebrow">OPERATING INSTRUCTION</p><h3>{mode==='REPLAY'?'Arrival instruction':'Evidence before a decision'}</h3></div><ShieldCheck size={20} className="mint-text"/></div><div className="decision-step"><span className="step-dot">01</span><div><p className="eyebrow">PREVIOUSLY APPROVED</p><h4>{mode==='REPLAY'?'“Report at 10:00.”':'No prior instruction connected'}</h4><p className="muted">{mode==='REPLAY'?'T+0 · Demo approval · 3 delivered, 2 queued fixtures.':'Live mode has no operating approval or employee notice history.'}</p></div></div><div className="decision-step"><span className="step-dot amber">02</span><div><p className="eyebrow">WHAT CHANGED</p><h4>{mode==='REPLAY'?replay.case==='applicable'?`${replay.alert.event} covers the facility`:replay.case==='irrelevant'?'Facility evidence unchanged':`Replay condition: ${replay.case}`:live?`${relevant.length} applicable alert${relevant.length===1?'':'s'}`:'Current evidence not loaded'}</h4><p className="muted">{mode==='REPLAY'?replay.gate==='review'?'Illustrative policy: review the arrival instruction.':replay.gate==='unknown'?'Missing or stale evidence. Resolve before approval.':'No new hold. Existing holds remain.':live?`Last successful refresh: ${displayTime(lastRefresh)}. Unknown applicability remains explicit.`:'Refresh NOAA to establish current applicability.'}</p></div></div><div className="next-action"><p className="eyebrow">WHAT NEEDS ACTION</p><p>{mode==='REPLAY'?replay.approval?'Approved. Ready to re-check.':canApprove?'Review two group messages.':'Resolve the evidence before approval.':'Review official evidence first.'}</p>{mode==='REPLAY'?<Button primary disabled={busy||(!replay.approval&&!canApprove)} onClick={()=>replay.approval?action('simulate',{binding:replay.approval.binding}):setReview(true)}>{replay.approval?'Re-check & simulate sending':'Review correction'}<ArrowRight size={16}/></Button>:<Button primary disabled={busy||!live||!liveFresh||!status?.agentConfigured} onClick={explain}>{busy?'Working…':'Explain evidence with Strands'}<ArrowRight size={16}/></Button>}{mode==='LIVE'&&!status?.agentConfigured&&<p className="microcopy">Akamai team key needed</p>}</div></Panel></div>
          {mode==='REPLAY'?<Panel id="recipients" className="recipients-panel"><div className="panel-header"><div><p className="eyebrow">PEOPLE & NOTICES · SYNTHETIC ROSTER</p><h3>Who needs a correction?</h3></div><Badge>{replay.roster.length} staff fixtures</Badge></div><div className="group-grid">{['onsite','arriving','remote'].map(group=>{const people=replay.roster.filter(p=>p.group===group);return <div className="group-card" key={group}><div className="group-heading"><span className="group-icon"><Users size={17}/></span><div><h4>{groupLabels[group]}</h4><span className="muted">{people.length} people</span></div><Badge tone={group==='remote'?'':'amber'}>{group==='remote'?'Excluded':replay.approval?(replay.intents.some(i=>i.approvalBinding===replay.approval.binding)?'Intents queued':'Approved'):'Needs review'}</Badge></div><div className="people-list">{people.map(p=><div className="person-row" key={p.id}><span className="person-avatar">{p.name.slice(0,1)}</span><span>{p.name}</span><span className={`notice-status ${held.includes(p.id)?'amber-text':''}`}>{p.original==='delivered'?<Check size={12}/>:<Clock3 size={12}/>} {held.includes(p.id)?'Queued · held':p.original==='delivered'?'Delivered fixture':'Not targeted'}</span></div>)}</div><p className="group-foot">{group==='onsite'?'Reporting update; no direction to leave.':group==='arriving'?'Delivered originals need correction; queued originals stay held.':'No arrival notice. Excluded.'}</p></div>})}</div><div className="recipient-summary"><span>{replay.intents.length} simulated correction intents queued</span><span>Attempted 0 <b>·</b> Delivered 0 <b>·</b> Acknowledged 0</span></div></Panel>:<Panel id="recipients" className="live-list"><div className="panel-header"><div><p className="eyebrow">LIVE FACILITY EVIDENCE</p><h3>Applicable alerts</h3></div><Badge>{relevant.length} returned</Badge></div>{relevant.map(a=><div className="live-alert-row" key={a.id}><div><strong>{a.event}</strong><p className="muted">{a.area} · {a.matchMethod}</p><p className="microcopy">Expires {displayTime(a.expires)}</p><p className="live-instruction">{a.id===alert?.id?'Official instructions shown above.':a.instruction||'No official instruction text supplied.'}</p></div><SourceLink href={a.source}>Source</SourceLink></div>)}{!relevant.length&&<p className="muted">{live?'No applicable alerts returned in this refresh. This does not establish safe conditions.':'Live data has not been refreshed.'}</p>}{live&&<p className="microcopy">Nationwide feed: {live.alerts.length} records checked. {unknownCount} with unknown applicability. Outside-facility alerts do not change this operating instruction.</p>}</Panel>}
          {briefing&&liveFresh&&alert?.lifecycle==='active'&&<Panel className="agent-panel"><p className="eyebrow">STRANDS BRIEFING · SNAPSHOT {displayTime(lastRefresh)}</p><h3>{briefing.explanation}</h3><ul>{briefing.changed_facts?.map((f,i)=><li key={i}>{f.fact}<span className="mono source-id">{f.source_id}</span></li>)}</ul><p className="muted">Evidence gaps: {briefing.evidence_gaps?.join(' · ')}</p></Panel>}
          <div className="bottom-layout"><Panel id="history"><div className="panel-header"><div><p className="eyebrow">DECISION & NOTICE HISTORY</p><h3>{mode==='REPLAY'?'Decision history':'No live decision history connected'}</h3></div><FileClock size={19} className="muted"/></div>{mode==='REPLAY'?<ol className="timeline">{replay.history.map((event,i)=><li key={i}><span className="timeline-node"/><div><strong>{event.label}</strong><span className="microcopy">{displayTime(event.time)}</span><details><summary>Details</summary><p>{event.detail}</p></details></div></li>)}</ol>:<p className="muted">The replay history is separate. Live mode does not approve instructions or send employee messages.</p>}<p className="microcopy">Replay history saved locally. Demo identity; no real delivery.</p></Panel><Panel className="trace-panel"><p className="eyebrow">EVIDENCE & BOUNDARIES</p><h3>Connections</h3><div className="connection-row"><span className="status-dot mint"/>NOAA / NWS <Badge>Connected</Badge></div><div className="connection-row"><span className="status-dot mint"/>Go + OPA policy <Badge>Local</Badge></div><div className="connection-row"><span className="status-dot amber"/>Strands + Akamai <Badge>{status?.agentConfigured?'Configured · unverified':'Team key needed'}</Badge></div><div className="connection-row"><span className="status-dot"/>Road / river signal <Badge>Not connected</Badge></div><details><summary>Source metadata & policy trace</summary><dl><dt>Mode</dt><dd>{mode}</dd><dt>Alert ID</dt><dd className="mono source-id">{alert?.id||'No alert selected'}</dd><dt>Policy</dt><dd>{replay.policyVersion} · illustrative</dd><dt>OPA eligibility</dt><dd>{mode==='REPLAY'?(replay.policyDecision?.allow?'Allow reviewed approval':'Deny approval'):'No live approval gate'}</dd><dt>Policy reasons</dt><dd>{mode==='REPLAY'?(replay.policyDecision?.reasons?.join(' · ')||'Fresh, applicable warning and fixture manager.'):'Official warnings bypass operational approval.'}</dd><dt>Evidence freshness</dt><dd>{mode==='REPLAY'?'Synthetic fixture condition':lastRefresh?displayTime(lastRefresh):'No successful live refresh'}</dd><dt>Approval digest</dt><dd className="mono source-id">{mode==='REPLAY'?replay.approval?.binding||'Not approved':'No live approval'}</dd></dl>{mode==='REPLAY'&&<><label className="field-label" htmlFor="case">Synthetic replay test condition</label><select id="case" value={replay.case} disabled={busy} onChange={e=>action('case',{case:e.target.value})}>{['applicable','outside','missing','stale','update','cancel','expired'].map(c=><option key={c}>{c}</option>)}</select><Button disabled={busy} onClick={()=>action('case',{case:'irrelevant'})}>Test unrelated update</Button></>}</details><Link href="/boundaries" className="source-link boundary-link">Explore system boundaries<ArrowUpRight size={14}/></Link></Panel></div>
        </>}
        <footer className="page-footer"><span>Weather Desk <span className="divider-text">/</span> Hack Midwest experiment</span><span>Evidence → Decision → Record</span></footer>
      </main>
    </div>{review&&replay&&<ReviewDialog replay={replay} busy={busy} onClose={()=>setReview(false)} onApprove={messages=>action('approve',{messages})}/>}</div>;
}
