'use client';
import { useEffect, useRef, useState } from 'react';
import Link from 'next/link';
import { Activity, ArrowRight, ArrowUpRight, Check, ChevronDown, Clock3, CloudLightning, FileClock, Layers3, MapPin, Moon, Radar, RefreshCw, ShieldCheck, Sun, Users, X } from 'lucide-react';
import { Badge, Button, Panel, SourceLink } from './ui';
import { AccountMenu } from './account-menu';

const facilities = { barnwell: ['Barnwell Field Office', 'South Carolina', 'SC-DEMO-01'], florida: ['Big Bend Field Office', 'Florida', 'FL-DEMO-01'], california: ['Inland Field Office', 'California', 'CA-DEMO-01'] };
const eventLabels = {
  'T+0 · Original instruction approved': 'Original instruction approved',
  'T+5 · Recorded NOAA evidence introduced': 'Weather warning added to the replay',
  'T+6 · Fixture manager approved correction': 'Demo manager approved the correction',
  'T+7 · Simulated send re-check': 'Messages checked before simulated sending',
};
const eventLabel = label => eventLabels[label] || (label.startsWith('Replay case: ') ? caseLabels[label.slice(13)] || label : label.replace(/^T\+\d+\s*·\s*/, '').replace(/Fixture manager/g, 'Demo manager'));
const caseLabels = { applicable: 'Warning covers this facility', outside: 'Warning is outside this facility', missing: 'Facility coverage is unknown', stale: 'Weather evidence needs a refresh', update: 'Relevant evidence has changed', cancel: 'Warning has been cancelled', expired: 'Warning has expired' };
const postureLabels = { normal: 'Normal operations', delay: 'Delay arrivals', 'early-dismissal': 'Early dismissal', telework: 'Telework', closure: 'Close facility' };
const groupLabels = { onsite: 'Already onsite', arriving: 'Not yet onsite', remote: 'Working remotely' };
const displayTime = value => value ? new Date(value).toLocaleString('en-US', { month: 'short', day: 'numeric', hour: 'numeric', minute: '2-digit', timeZoneName: 'short' }) : 'Not available';
async function api(path, body, options = {}) {
  const response = await fetch(path, { method: body ? 'POST' : 'GET', headers: body ? { 'Content-Type': 'application/json' } : {}, body: body ? JSON.stringify(body) : undefined, cache: 'no-store', ...options });
  let data;
  try {
    if (!response.headers.get('content-type')?.includes('application/json')) throw Error('Non-JSON response');
    data = await response.json();
  } catch (error) {
    if (error.name === 'AbortError') throw error;
    throw Error('The service did not respond. Try again.');
  }
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
  </svg><div className="plot-caption"><span><i className="legend-square"/>Official warning polygon</span><span><i className="legend-dot"/>Fictional facility</span></div></div>;
}
function EventTimeline({ events }) {
  return <ol className="timeline">{events.map((event,i)=><li key={`${event.time}-${i}`}><span className="timeline-node"/><div><strong>{eventLabel(event.label)}</strong><span className="microcopy">Recorded {displayTime(event.time)}</span><details><summary>Details</summary><p>{event.detail}</p></details></div></li>)}</ol>;
}
const riverTime = value => value && Number.isFinite(Date.parse(value)) ? new Date(value).toLocaleString('en-US', { year: 'numeric', month: 'short', day: 'numeric', hour: 'numeric', minute: '2-digit', timeZoneName: 'short' }) : 'Not available';
function riverAge(seconds) {
  if (!Number.isFinite(seconds)) return 'Observation age unknown';
  if (seconds < 0) return 'Observation time is ahead of the device clock';
  if (seconds < 60) return 'Less than a minute old';
  const [amount, unit] = seconds < 3600 ? [Math.floor(seconds / 60), 'minute'] : seconds < 86400 ? [Math.floor(seconds / 3600), 'hour'] : [Math.floor(seconds / 86400), 'day'];
  return `${amount} ${unit}${amount === 1 ? '' : 's'} old`;
}
function RiverContext({ signals, now }) {
  const number = value => Number.isFinite(value) ? new Intl.NumberFormat('en-US', { maximumFractionDigits: 2 }).format(value) : null;
  return <Panel className="river-panel"><div className="panel-header"><div><p className="eyebrow">SUPPLEMENTAL · LIVE</p><h3>Regional river context</h3></div><Badge>Facility impact unknown</Badge></div><p className="microcopy">Gauge readings do not establish conditions at this facility or on nearby roads.</p>{signals.length ? signals.map((signal, index) => {
    const available = signal.state === 'available' && Number.isFinite(signal.value);
    const minor = number(signal.thresholds?.minor);
    const observed = Date.parse(signal.observedAt);
    const age = Number.isFinite(observed) ? (now - observed) / 1000 : signal.observationAgeSeconds;
    return <article className="river-reading" key={`${signal.provider}-${signal.gaugeId}-${index}`}><div className="river-reading-heading"><div><strong>{signal.name || signal.gaugeId || 'Regional gauge'}</strong><p className="microcopy">{signal.provider} · {signal.gaugeId || signal.usgsId || 'Gauge not selected'}</p></div><span className="river-value">{available ? `${number(signal.value)} ${signal.unit || ''}` : 'Unknown'}</span></div><div className="river-facts"><p><span>Observed</span>{riverTime(signal.observedAt)}<small className="river-observation-age">{riverAge(age)}</small></p><p><span>Source fetched successfully</span>{riverTime(signal.fetchedAt)}</p><p><span>Minor flood stage at gauge</span>{minor !== null ? `${minor} ${signal.thresholds.unit || signal.unit || ''}` : 'Unavailable'}</p></div><details><summary>Sources & limitations</summary>{signal.attemptedAt&&<p className="microcopy">Last fetch attempted: {riverTime(signal.attemptedAt)}</p>}<div className="river-links"><SourceLink href={signal.source}>Observation source</SourceLink><SourceLink href={signal.metadataSource}>Gauge metadata</SourceLink>{signal.thresholds?.source&&<SourceLink href={signal.thresholds.source}>Flood thresholds</SourceLink>}</div>{signal.approvalStatus&&<p className="microcopy">Observation status: {signal.approvalStatus}</p>}{signal.reportedCategory&&<p className="microcopy">Provider category at this gauge: {signal.reportedCategory}</p>}{signal.relevanceExplanation&&<p className="microcopy">{signal.relevanceExplanation}</p>}{!!signal.gaps?.length&&<p className="microcopy">{signal.gaps.join(' · ')}</p>}</details></article>;
  }) : <p className="muted">No river observation is available for this location.</p>}</Panel>;
}
function ReviewDialog({ replay, busy, canApprove, agentConfigured, onDraft, onClose, onApprove }) {
  const [operatingPosture, setOperatingPosture] = useState(replay.operatingPosture || 'delay');
  const [messages, setMessages] = useState(Object.fromEntries(replay.messages.map(m => [m.group, m.text])));
  const [drafting,setDrafting]=useState(false);
  const [draftError,setDraftError]=useState('');
  const [draftSummary,setDraftSummary]=useState(null);
  async function generateDraft() {
    setDraftError('');setDrafting(true);
    try {const output=await onDraft();setMessages(Object.fromEntries(output.draft_operational_messages.map(m=>[m.group,m.text])));setDraftSummary(output);}
    catch(e){setDraftError(e.message);}
    finally {setDrafting(false);}
  }
  const dialog = useRef(null);
  useEffect(() => { dialog.current.showModal(); const d = dialog.current; return () => d?.close(); }, []);
  return <dialog ref={dialog} className="review-dialog" aria-labelledby="review-title" onCancel={onClose} onClick={e => { if (e.target === dialog.current) onClose(); }}>
    <div className="dialog-heading"><div><p className="eyebrow">Manager review · Simulation</p><h2 id="review-title">Review the correction</h2></div><Button onClick={onClose} aria-label="Close review"><X size={18}/></Button></div>
    <div className="review-context"><span className="eyebrow">REPLACES THE EARLIER INSTRUCTION</span><strong>“Report at 10:00.”</strong><span>{replay.facility.name} · {replay.messages.reduce((n,m)=>n+m.recipients.length,0)} recipients</span></div><p className="muted">Approve the exact messages below. Sending is simulated.</p>
    <div className="posture-field"><label className="field-label" htmlFor="operating-posture">Operating posture</label><select id="operating-posture" value={operatingPosture} disabled={busy||drafting||!canApprove} onChange={e=>setOperatingPosture(e.target.value)}>{Object.entries(postureLabels).map(([value,label])=><option key={value} value={value} disabled={value==='normal'}>{label}{value==='normal'?' · separate review required':''}</option>)}</select><p className="microcopy">Your selection is recorded with these exact messages. Normal operations require a separate review; warning expiry is not an all-clear.</p></div>
    {!!replay.roster.filter(p=>p.essential&&replay.messages.some(m=>m.recipients.includes(p.id))).length&&<details className="essential-review"><summary>Review essential assignments</summary>{replay.roster.filter(p=>p.essential&&replay.messages.some(m=>m.recipients.includes(p.id))).map(p=><p className="microcopy" key={p.id}><strong>{p.name}</strong> · {p.assignment}</p>)}<p className="microcopy">Review these people individually before approval. Essential designation does not override official warning instructions.</p></details>}
    <div className="draft-toolbar"><Badge>{draftSummary?'AI correction · review required':'Example correction · editable'}</Badge><Button disabled={busy||drafting||!agentConfigured||!canApprove} onClick={generateDraft}>{drafting?'Drafting…':'Draft with AI'}</Button></div>
    <p className="microcopy">AI drafts correct the earlier arrival instruction. Check that every message fits your selected posture; closure or early dismissal does not establish a safe route or permission to leave.</p>
    {!agentConfigured&&<p className="microcopy">AI drafting is not connected. You can edit these example messages.</p>}
    {draftError&&<p className="feedback error" role="alert">{draftError}</p>}
    {draftSummary&&<details><summary>Why the AI suggested this · sources</summary><p>{draftSummary.explanation}</p><ul>{draftSummary.changed_facts.map((f,i)=><li key={i}>{f.fact}<span className="mono source-id">{f.source_id}</span></li>)}</ul><p>{draftSummary.evidence_gaps.join(' · ')}</p></details>}
    {replay.messages.map(m => <div className="review-message" key={m.group}><label htmlFor={`message-${m.group}`}><strong>{groupLabels[m.group]}</strong><span className="muted">{m.recipients.length} recipients · {m.recipients.map(id => replay.roster.find(r => r.id === id).name).join(', ')}</span></label><textarea id={`message-${m.group}`} rows={4} value={messages[m.group]} onChange={e => setMessages({ ...messages, [m.group]: e.target.value })}/></div>)}
    <div className="approval-note"><ShieldCheck size={20}/><p>You approve only these messages and recipients. If relevant evidence or text changes, a new review is required.</p></div>
    <div className="dialog-footer"><Button onClick={onClose}>Back to evidence</Button><Button primary disabled={busy || drafting || !canApprove || Object.values(messages).some(t => !t.trim())} onClick={() => onApprove(messages, operatingPosture)}>Approve these messages <ArrowRight size={16}/></Button></div>
  </dialog>;
}
export default function WeatherDesk({ identity }) {
  const canManage = identity.role === 'manager';
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
  const inferenceAbort = useRef(null);
  const approvedDetails = useRef(null);
  const replay = replayData?.replay;
  useEffect(() => { const check=()=>api('/api/status').then(setStatus).catch(()=>setStatus(null));check();const timer=setInterval(check,10000);return ()=>clearInterval(timer); }, []);
  useEffect(() => {
    inferenceAbort.current?.abort();
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
  const postureAssessment = mode === 'REPLAY' ? replayData?.postureAssessment : live?.postureAssessment;
  const postureNeedsRefresh = mode === 'LIVE' && postureAssessment && (!liveFresh || (postureAssessment.recommendedPosture && (!(postureAssessment.triggers || []).length || postureAssessment.triggers.some(trigger => {
    const current = liveAlerts.find(alert => trigger.alertId && alert.id === trigger.alertId);
    return !current || !trigger.source || current.source !== trigger.source || current.lifecycle !== 'active' || current.applicability !== 'applicable' || current.status !== 'Actual';
  }))));
  const canApprove = canManage && replay?.policyDecision?.allow === true;
  async function refresh() {
    const generation=epoch.current;setBusy(true);setError('');setBriefing(null);evidenceIdentity.current=null;
    try { const data=await api(`/api/live?facility=${facility}`);if (generation!==epoch.current) return;setLive(data);setLastRefresh(data.lastSuccessfulRefresh);evidenceIdentity.current=data.lastSuccessfulRefresh; }
    catch(e) { if (generation===epoch.current) {setLive(null);setError(e.message);} }
    finally { if(generation===epoch.current) setBusy(false); }
  }
  async function action(actionName, extras={}) {
    const generation=epoch.current;setBusy(true);setError('');
    try {const data=await api(`/api/replay/action?facility=${facility}`,{action:actionName,revision:replayData.revision,...extras});if(generation!==epoch.current)return;setReplayData(data);setReview(false);setFeedback(actionName==='simulate'?(data.added?`${data.added} correction messages queued in the simulation. Nothing was sent.`:'Re-check complete. No duplicate messages were added.'):actionName==='approve'?'Messages approved. Review is saved; nothing has been sent.':'Recorded scenario updated.');}
    catch(e){if(generation===epoch.current)setError(e.message);}
    finally {if(generation===epoch.current)setBusy(false);}
  }
  async function explain() {
    const generation=epoch.current, timestamp=evidenceIdentity.current,controller=new AbortController();inferenceAbort.current=controller;setBusy(true);setError('');
    try {const data=await api(`/api/briefing?facility=${facility}`,{mode:'LIVE'},{signal:controller.signal});if(generation!==epoch.current || timestamp!==evidenceIdentity.current || data.evidenceTimestamp!==timestamp)return;setBriefing(data.briefing);setStatus(s=>({...s,inferenceVerified:true}));}
    catch(e){if(generation===epoch.current&&e.name!=='AbortError')setError(e.message);}
    finally {if(inferenceAbort.current===controller)inferenceAbort.current=null;if(generation===epoch.current)setBusy(false);}
  }
  async function draftForReview() {
    const generation=epoch.current,revision=replayData.revision,controller=new AbortController();
    inferenceAbort.current=controller;setBusy(true);
    try {
      const data=await api(`/api/briefing?facility=${facility}`,{mode:'REPLAY',revision},{signal:controller.signal});
      if(epoch.current!==generation || data.reviewRevision!==revision) throw Error('The review changed. Open the current package again.');
      setStatus(s=>({...s,inferenceVerified:true}));
      return data.briefing;
    } finally {if(inferenceAbort.current===controller)inferenceAbort.current=null;if(epoch.current===generation)setBusy(false);}
  }
  const briefingCurrent=liveFresh&&(briefing?.source_references||[]).every(id=>{
    const original=live?.alerts.find(a=>a.id===id),current=liveAlerts.find(a=>a.id===id);
    return original&&current&&original.lifecycle===current.lifecycle&&original.applicability===current.applicability;
  });
  const held = replay?.heldRecipients || [];
  const originalDelivered = replay?.roster.filter(p=>p.original==='delivered').length || 0;
  const currentIntents = replay?.approval ? replay.intents.filter(i=>i.approvalBinding===replay.approval.binding).length : 0;
  const affectedCount = replay ? new Set(replay.messages.flatMap(m=>m.recipients)).size : 0;
  const reviewStage = currentIntents ? 'Simulation complete' : replay?.approval ? 'Messages approved' : !canManage ? 'Manager review required' : canApprove ? 'Review needed' : 'Review blocked';
  return <div className={`desk ${theme==='light'?'light':''}`}>
    <aside className="sidebar">
      <Link href="/" className="brand"><span className="brand-symbol"><CloudLightning size={23}/></span><span>weather<span className="brand-secondary">desk</span></span></Link>
      <div className="workspace-label">FACILITY OPERATIONS</div>
      <nav aria-label="Workspace"><a className="nav-item active" href="#overview"><Radar size={18}/>Instruction review</a><a className="nav-item" href="#recipients"><Users size={18}/>People & notices</a><a className="nav-item" href="#history"><FileClock size={18}/>Decision history</a><Link className="nav-item" href="/gallery"><Layers3 size={18}/>Design library</Link><Link className="nav-item" href="/boundaries"><ShieldCheck size={18}/>System boundaries</Link></nav>
      <div className="sidebar-scenarios"><p className="workspace-label">FICTIONAL FACILITIES</p>{Object.entries(facilities).map(([key,info])=><button key={key} onClick={()=>setFacility(key)} className={`facility-item ${facility===key?'selected':''}`}><span className="facility-indicator"/><span>{info[0]}<small>{info[1]}</small></span>{facility===key&&<ChevronDown size={13}/>}</button>)}</div>
      <div className="sidebar-bottom"><div className="session-person"><span className="avatar">{identity.name.slice(0,2).toUpperCase()}</span><div>{identity.name}<small>{identity.demo?'No account signed in':`${identity.role === 'manager'?'Manager':'Viewer'} · GitHub`}</small></div></div><div className="sidebar-foot">NOAA / NWS evidence<br/>Synthetic people · simulated notices</div></div>
    </aside>
    <div className="desk-body"><header className="topbar"><div className="breadcrumb">Operations <span>/</span> {facilities[facility][1]}</div><div className="topbar-tools"><AccountMenu identity={identity}/><Button aria-label={`Switch to ${theme==='dark'?'light':'dark'} theme`} onClick={()=>setTheme(theme==='dark'?'light':'dark')}>{theme==='dark'?<Sun size={17}/>:<Moon size={17}/>}</Button></div></header>
      <main id="overview" className="main-content"><div className="page-heading"><div><p className="eyebrow">EVIDENCE → ACTION</p><h1>{mode==='REPLAY'?'Review the arrival instruction':'Live weather evidence'}<span className="heading-dot">.</span></h1><p className="muted">{facilities[facility][0]} <span className="divider-text">·</span> Fictional facility <span className="divider-text">·</span> {facilities[facility][2]}</p></div><div className="mode-control" aria-label="Data mode">{['REPLAY','LIVE'].map(m=><button key={m} aria-pressed={mode===m} className={mode===m?'selected':''} onClick={()=>setMode(m)}>{m==='REPLAY'?<FileClock size={14}/>:<Activity size={14}/>} {m==='REPLAY'?'Recorded replay':'Live NOAA'}</button>)}</div></div>
        <div className="mobile-facility"><label htmlFor="mobile-facility">Facility</label><select id="mobile-facility" value={facility} onChange={e=>setFacility(e.target.value)}>{Object.entries(facilities).map(([id,value])=><option value={id} key={id}>{value[0]}</option>)}</select></div>
        <div className="mode-banner"><span className={`status-dot ${mode==='LIVE'?'mint':'amber'}`}/><strong>{mode==='REPLAY'?'Recorded NOAA · Replay':'Live mode. Manual refresh.'}</strong><span>{mode==='REPLAY'?`${displayTime(replay?.capturedAt)} · Not live`:'Facility applicability checked at refresh.'}</span>{mode==='LIVE'&&<Button onClick={refresh} disabled={busy}>{busy?<RefreshCw size={14} className="spin"/>:<RefreshCw size={14}/>}Refresh NOAA</Button>}</div>
        {!canManage&&<div className="mode-banner"><ShieldCheck size={16}/><span>Viewer access · You can read evidence and history. A manager must approve changes.</span></div>}{error&&<div className="feedback error" role="alert">{error}</div>}{feedback&&<div className="feedback" role="status">{feedback}</div>}
        {!replay&&<Panel className="loading">Loading the recorded scenario…</Panel>}
        {replay&&<>
          <section className={`official-alert ${alert?'':'no-alert'}`} aria-labelledby="official-heading"><div className="alert-icon"><CloudLightning size={25}/></div><div className="official-content"><div className="alert-heading"><div><p className="eyebrow">{mode==='REPLAY'?'RECORDED OFFICIAL NOAA / NWS WARNING':'OFFICIAL NOAA / NWS EVIDENCE'}</p><h2 id="official-heading">{alert?.event || (live?'No applicable warning returned':'Refresh NOAA to see current evidence')}</h2></div><Badge tone={alert?'amber':''}>{mode==='REPLAY'?'REPLAY':alert?liveLabel:live?(liveFresh?'No applicable alert':'Stale · refresh required'):'Unknown until checked'}</Badge></div><p className="alert-area">{alert?`${alert.issuer} · ${alert.area}`:live?`${relevant.length} applicable alerts. ${unknownCount} alert records have unknown applicability.`:'Live weather is not inferred from the recorded scenario.'}</p>{alert?<><p className="official-instruction">{alert.instruction || 'No instruction text was supplied in this alert. Open the official source for details.'}</p><div className="alert-meta"><span>Issued {displayTime(alert.issued)}</span><span>Expires {displayTime(alert.expires)}</span><SourceLink href={alert.source}>Official source</SourceLink></div><p className="warning-boundary">{mode==='REPLAY'?'Recorded official text.':!liveFresh?'Last received official text. Refresh required; current conditions are unknown.':['expired','cancelled'].includes(alert?.lifecycle)?'Ended warning; no reopening decision.':alert?.lifecycle==='not-yet-effective'?'Not yet effective.':alert?.lifecycle==='unknown'?'Timing or applicability unknown.':'Official instructions display immediately.'} Official instructions bypass approval.</p></>:<p className="warning-boundary">Missing data or the absence of a warning does not establish safe conditions or authorize reopening.</p>}</div></section>
          <div className="evidence-layout">
            <Panel className="decision-panel" aria-labelledby="decision-heading">
              <div className="panel-header"><div><p className="eyebrow">MANAGER DECISION</p><h3 id="decision-heading">{mode==='REPLAY'?'What needs to change?':'Start with the evidence'}</h3></div>{mode==='REPLAY'&&<Badge tone={canApprove&&!replay.approval?'amber':''}>{reviewStage}</Badge>}</div>
              <div className="decision-step"><span className="step-dot"><FileClock size={14}/></span><div><p className="eyebrow">EARLIER INSTRUCTION</p><h4>{mode==='REPLAY'?'“Report at 10:00.”':'No operating instruction connected'}</h4><p className="muted">{mode==='REPLAY'?`${originalDelivered} people received it. ${held.length} queued notices are held.`:'Live mode has no staff notices or prior manager approval.'}</p></div></div>
              <div className="decision-step"><span className="step-dot amber"><CloudLightning size={14}/></span><div><p className="eyebrow">NEW EVIDENCE</p><h4>{mode==='REPLAY'?caseLabels[replay.case]||'Evidence requires review':live?`${relevant.length} applicable alert${relevant.length===1?'':'s'}`:'Current evidence not loaded'}</h4><p className="muted">{mode==='REPLAY'?!canManage?'A manager reviews whether these changed conditions require a correction.':canApprove?'The earlier arrival instruction needs review under the demo policy.':replay.gate==='unknown'?'Resolve missing or stale evidence before approving a correction.':'Existing notices stay held. This change does not authorize reopening.':live?`Last refreshed ${displayTime(lastRefresh)}.`:'Refresh NOAA to check this facility.'}</p></div></div>
              {postureAssessment&&<div className="posture-proposal"><p className="eyebrow">ILLUSTRATIVE POLICY · MANAGER REVIEW</p><strong>{postureNeedsRefresh?'Posture needs refresh':postureAssessment.recommendedPosture?`Suggested: ${postureLabels[postureAssessment.recommendedPosture] || postureAssessment.recommendedPosture}`:'No posture recommendation'}</strong>{postureNeedsRefresh?<p className="microcopy">Supporting weather evidence changed or is no longer current. Refresh and review before using a posture suggestion.</p>:<details><summary>Reasons & evidence gaps</summary><p className="microcopy">{postureAssessment.rationale}</p>{!!postureAssessment.gaps?.length&&<p className="microcopy">{postureAssessment.gaps.join(' · ')}</p>}{!!postureAssessment.whatWouldChange?.length&&<p className="microcopy">What would change this: {postureAssessment.whatWouldChange.join(' · ')}</p>}{postureAssessment.triggers?.map((trigger,i)=><p className="microcopy" key={i}>{trigger.event} · {trigger.criterion} <SourceLink href={trigger.source}>Source</SourceLink></p>)}</details>}</div>}
              <div className="next-action"><p className="eyebrow">{mode==='REPLAY'?(currentIntents?'RECORDED IN THE SIMULATION':replay.approval?'READY FOR THE FINAL CHECK':'YOUR NEXT STEP'):'YOUR NEXT STEP'}</p><p>{mode==='REPLAY'?!canManage?'Read the changed evidence and saved messages. A manager controls approval and simulation.':currentIntents?`${currentIntents} corrections queued. Nothing was sent.`:replay.approval?'The exact messages are approved. Evidence is checked again before queueing.':canApprove?`Review the correction for ${affectedCount} people in two groups.`:'Review the changed evidence before continuing.':canManage?'Read the official warning, then request an evidence explanation.':'Read the official warning and the evidence for this facility.'}</p>
                {mode==='REPLAY'?(currentIntents?<a className="button primary" href="#approved-correction" onClick={()=>{if(approvedDetails.current) approvedDetails.current.open=true;}}>View approved messages<ArrowRight size={16}/></a>:<Button primary disabled={busy||!canManage||(!replay.approval&&!canApprove)} onClick={()=>replay.approval?action('simulate',{binding:replay.approval.binding}):setReview(true)}>{replay.approval?'Re-check & simulate sending':'Review correction'}<ArrowRight size={16}/></Button>):<Button primary disabled={busy||!canManage||!live||!liveFresh||!status?.agentConfigured} onClick={explain}>{busy?'Working…':'Explain this evidence'}<ArrowRight size={16}/></Button>}
                {mode==='REPLAY'&&canManage&&replay.approval&&<div className="secondary-actions"><Button disabled={busy} onClick={()=>setReview(true)}>Edit messages</Button>{currentIntents>0&&<Button disabled={busy} onClick={()=>action('simulate',{binding:replay.approval.binding})}>Re-check simulation</Button>}</div>}
                {mode==='LIVE'&&!status?.agentConfigured&&<p className="microcopy">AI explanation is not connected.</p>}
              </div>
              {mode==='REPLAY'&&<a className="decision-people-link" href="#recipients">See who received the earlier instruction <ArrowRight size={14}/></a>}
            </Panel>
            <Panel className="geography-panel"><div className="panel-header"><div><p className="eyebrow">LOCATION & APPLICABILITY</p><h3>Does it cover this facility?</h3></div><Badge tone={alert?.geography==='applicable'?'mint':''}>{alert?.geography || 'Unknown'}</Badge></div><GeographyPlot alert={alert} facility={currentFacility} replayCase={mode==='REPLAY'?replay.case:null}/><div className="map-evidence"><MapPin size={15}/><span>{alert?.matchMethod || 'No applicable geometry selected'}</span><span className="mono">{currentFacility?.latitude}°, {currentFacility?.longitude}°</span></div><p className="microcopy">Official polygon · Fictional facility · Geographic extent plot</p></Panel>
          </div>
          {mode==='REPLAY'?<Panel id="recipients" className="recipients-panel"><div className="panel-header"><div><p className="eyebrow">RECIPIENTS · SYNTHETIC STAFF</p><h3>Who needs the update?</h3></div><Badge>{replay.roster.length} demo staff</Badge></div><p className="microcopy">Channel routes are planned for the simulation. Essential assignments need individual manager review.</p><div className="group-grid">{['onsite','arriving','remote'].map(group=>{const people=replay.roster.filter(p=>p.group===group);return <div className="group-card" key={group}><div className="group-heading"><span className="group-icon"><Users size={17}/></span><div><h4>{groupLabels[group]}</h4><span className="muted">{people.length} people</span></div><Badge tone={group==='remote'?'':'amber'}>{group==='remote'?'Excluded':replay.approval?(replay.intents.some(i=>i.approvalBinding===replay.approval.binding)?'Queued · simulated':'Approved'):'Needs review'}</Badge></div><p className="list-context">Earlier instruction</p><div className="people-list">{people.map(p=><div className="person-row" key={p.id}><span className="person-avatar">{p.name.slice(0,1)}</span><span>{p.name}{(p.essential||p.channels?.length>0)&&<small className="recipient-routing" title="Planned channels for simulated corrections">{p.essential?'Essential · ':''}{p.channels?.map(channel=>channel==='sms'?'SMS':channel).join(' / ')}</small>}</span><span className={`notice-status ${held.includes(p.id)?'amber-text':''}`}>{p.original==='delivered'?<Check size={12}/>:<Clock3 size={12}/>} {held.includes(p.id)?'Queued · held':p.original==='delivered'?'Delivered · demo':'Not targeted'}</span></div>)}</div><p className="group-foot">{group==='onsite'?'Update their reporting status. This is not an instruction to leave.':group==='arriving'?'Correct the earlier arrival instruction. Held notices stay held.':'No earlier arrival notice, so no correction is targeted.'}</p></div>})}</div>{replay.approval&&<details ref={approvedDetails} id="approved-correction" className="reviewed-package" key={`${replay.approval.binding}-${currentIntents>0}`} open={currentIntents>0?true:undefined}><summary>Approved messages · exact text and recipients</summary><p className="microcopy">Operating posture: {postureLabels[replay.approval.package.parameters.operatingPosture] || 'Earlier approval'}</p>{replay.approval.actorName&&<p className="microcopy">Approved by {replay.approval.actorName} · {displayTime(replay.approval.approvedAt)}</p>}{replay.approval.package.messages.map(m=><article key={m.group}><h4>{groupLabels[m.group]}</h4><p className="microcopy">{m.recipients.map(id=>replay.roster.find(r=>r.id===id)?.name||id).join(', ')}</p><p className="reviewed-text">{m.text}</p></article>)}</details>}<div className="recipient-summary"><span>{replay.intents.length} correction messages queued across this replay</span><span>Simulation only · No messages sent or acknowledged</span></div><details className="notice-stages"><summary>Correction message status</summary><p>These are the corrections, separate from the earlier demo notices above.</p><dl className="notice-state-list"><div><dt>Queued</dt><dd>{replay.intents.length}</dd></div><div><dt>Attempted</dt><dd>0</dd></div><div><dt>Delivered</dt><dd>0</dd></div><div><dt>Acknowledged</dt><dd>0</dd></div></dl><p>Queueing records a simulated request. It does not establish delivery or acknowledgment.</p></details></Panel>:<Panel id="recipients" className="live-list"><div className="panel-header"><div><p className="eyebrow">LIVE FACILITY EVIDENCE</p><h3>Applicable alerts</h3></div><Badge>{relevant.length} returned</Badge></div>{relevant.map(a=><div className="live-alert-row" key={a.id}><div><strong>{a.event}</strong><p className="muted">{a.area} · {a.matchMethod}</p><p className="microcopy">Expires {displayTime(a.expires)}</p><p className="live-instruction">{a.id===alert?.id?'Official instructions shown above.':a.instruction||'No official instruction text supplied.'}</p></div><SourceLink href={a.source}>Source</SourceLink></div>)}{!relevant.length&&<p className="muted">{live?'No applicable alerts returned in this refresh. This does not establish safe conditions.':'Live data has not been refreshed.'}</p>}{live&&<p className="microcopy">Nationwide feed: {live.alerts.length} records checked. {unknownCount} with unknown applicability. Outside-facility alerts do not change this operating instruction.</p>}</Panel>}
          {mode==='LIVE'&&live&&<RiverContext signals={live.riverSignals || []} now={now}/>}
          {briefing&&mode==='LIVE'&&briefingCurrent&&<Panel className="agent-panel"><p className="eyebrow">STRANDS BRIEFING · SNAPSHOT {displayTime(lastRefresh)}</p><h3>Evidence briefing</h3><p className="briefing-summary">{briefing.explanation}</p><details><summary>Facts, sources & gaps</summary><ul>{briefing.changed_facts?.map((f,i)=><li key={i}>{f.fact}<SourceLink href={live.alerts.find(a=>a.id===f.source_id)?.source}>NOAA source</SourceLink></li>)}</ul>{!!briefing.evidence_gaps?.length&&<p className="muted">{briefing.evidence_gaps.join(' · ')}</p>}</details></Panel>}
          {briefing&&mode==='LIVE'&&liveFresh&&!briefingCurrent&&<Panel className="agent-panel"><h3>Briefing needs refresh</h3><p className="muted">A cited alert changed validity. Refresh NOAA before using this explanation.</p></Panel>}
          <div className="bottom-layout"><Panel id="history"><div className="panel-header"><div><p className="eyebrow">DECISION & NOTICE HISTORY</p><h3>{mode==='REPLAY'?'Decision history':'No live decision history connected'}</h3></div><FileClock size={19} className="muted"/></div>{mode==='REPLAY'?<><p className="microcopy">Recent actions in this demo. Earlier records are preserved below.</p><EventTimeline events={replay.history.slice(-4)}/>{replay.history.length>4&&<details className="earlier-history"><summary>Earlier history · {replay.history.length-4} actions</summary><EventTimeline events={replay.history.slice(0,-4)}/></details>}</>:<p className="muted">The replay history is separate. Live mode does not approve instructions or send employee messages.</p>}{mode==='REPLAY'&&!!replay.approvedPackages?.length&&<details className="approval-archive"><summary>{replay.approvedPackages.length} saved approval package{replay.approvedPackages.length===1?'':'s'}</summary>{replay.approvedPackages.map((approval,i)=><article key={i}><p className="microcopy">Approval {i+1} · {approval.binding.slice(0,12)} · {approval.package.facility.name}</p>{approval.actorName&&<p className="microcopy">Approved by {approval.actorName} · {displayTime(approval.approvedAt)}</p>}{approval.package.messages.map(m=><div key={m.group}><strong>{groupLabels[m.group]}</strong><p className="microcopy">{m.recipients.map(id=>replay.roster.find(r=>r.id===id)?.name||id).join(', ')}</p><p className="reviewed-text">{m.text}</p></div>)}</article>)}</details>}<p className="microcopy">Replay history is saved. All sending remains simulated.</p></Panel><Panel className="trace-panel"><p className="eyebrow">EVIDENCE & BOUNDARIES</p><h3>Sources & checks</h3><div className="connection-row"><span className="status-dot mint"/>NOAA / NWS <Badge>Connected</Badge></div><div className="connection-row"><span className="status-dot mint"/>Go + OPA policy <Badge>Enforced</Badge></div><div className="connection-row"><span className="status-dot amber"/>Strands + Akamai <Badge>{status?.inferenceVerified?'Inference connected':status?.agentConfigured?'Configured · ready to test':'Team key needed'}</Badge></div><div className="connection-row"><span className="status-dot"/>Road conditions <Badge>Not connected</Badge></div>{mode==='LIVE'&&<div className="connection-row"><span className="status-dot"/>River observations <Badge>{live?.riverSignals?.some(s=>s.state==='available')?'Regional context':'Unknown'}</Badge></div>}<details><summary>Source details & decision checks</summary><dl><dt>Mode</dt><dd>{mode}</dd><dt>Alert ID</dt><dd className="mono source-id">{alert?.id||'No alert selected'}</dd><dt>Policy</dt><dd>{replay.policyVersion} · illustrative</dd><dt>OPA eligibility</dt><dd>{mode==='REPLAY'?(replay.policyDecision?.allow?'Allow reviewed approval':'Deny approval'):'No live approval gate'}</dd><dt>Policy reasons</dt><dd>{mode==='REPLAY'?(replay.policyDecision?.reasons?.join(' · ')||'Fresh, applicable warning and authorized manager.'):'Official warnings bypass operational approval.'}</dd><dt>Evidence freshness</dt><dd>{mode==='REPLAY'?'Synthetic fixture condition':lastRefresh?displayTime(lastRefresh):'No successful live refresh'}</dd><dt>Approval digest</dt><dd className="mono source-id">{mode==='REPLAY'?replay.approval?.binding||'Not approved':'No live approval'}</dd></dl>{mode==='REPLAY'&&<><label className="field-label" htmlFor="case">Try a different evidence condition</label><select id="case" value={replay.case} disabled={busy||!canManage} onChange={e=>action('case',{case:e.target.value})}>{['applicable','outside','missing','stale','update','cancel','expired'].map(c=><option value={c} key={c}>{caseLabels[c]}</option>)}</select><Button disabled={busy||!canManage} onClick={()=>action('case',{case:'irrelevant'})}>Try an unrelated update</Button></>}</details><Link href="/boundaries" className="source-link boundary-link">Explore system boundaries<ArrowUpRight size={14}/></Link></Panel></div>
        </>}
        <footer className="page-footer"><span>Weather Desk <span className="divider-text">/</span> Hack Midwest experiment</span><span>Evidence → Decision → Record</span></footer>
      </main>
    </div>{review&&replay&&<ReviewDialog replay={replay} busy={busy} canApprove={canApprove} agentConfigured={status?.agentConfigured} onDraft={draftForReview} onClose={()=>{inferenceAbort.current?.abort();setReview(false);setBusy(false);}} onApprove={(messages,operatingPosture)=>action('approve',{messages,operatingPosture})}/>}</div>;
}
