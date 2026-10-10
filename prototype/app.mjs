import { initialState, advance, setEvidence, policyGate, packageFor } from './model.mjs';
const byId=id=>document.getElementById(id);
let state=initialState();
const recorded=await fetch('../fixtures/noaa-warning-recorded.json').then(r=>r.json());
const liveCheck=await fetch('../fixtures/noaa-live-check.json').then(r=>r.json());
const alert=recorded.feature.properties;
byId('official-instruction').textContent=alert.instruction;
byId('live-check').textContent=`Last verified statewide query: ${new Date(liveCheck.lastSuccessfulRefresh).toLocaleString()} · ${liveCheck.tornadoWarnings} tornado warnings in that response.`;
const sourceRows=[['Source',alert.senderName],['Alert ID',alert.id],['Affected geography',alert.areaDesc],['Original issue',alert.sent],['Original expiry',alert.expires],['Captured successfully',recorded.capturedAt],['Replay timing','Synthetic: T+0 = 09:55; T+5 = 10:00; warning fixture expires 10:30. Original NOAA timestamps above are unchanged.'],['Fictional facility','29.99, -83.44; inside the captured warning polygon.'],['Source link',recorded.feature.id]];
for(const [label,value] of sourceRows) {
  const row=document.createElement('div'), dt=document.createElement('dt'), dd=document.createElement('dd'); dt.className='font-semibold'; dt.textContent=label;
  if(label==='Source link'){const link=document.createElement('a');link.href=value;link.textContent='Open recorded NOAA alert ↗';link.target='_blank';link.rel='noreferrer';link.className='underline';dd.append(link);}else dd.textContent=value;
  row.append(dt,dd); byId('source-details').append(row);
}
const actions=['Approve fictional instruction','Introduce new NOAA evidence','Review group corrections','Approve reviewed corrections','Re-check & simulate queue','Repeat send check'];
function render() {
  byId('replay-panel').hidden=state.mode!=='REPLAY'; byId('live-panel').hidden=state.mode==='REPLAY';
  const gate=policyGate(state.evidence), warningVisible=state.phase>=2 && state.evidence.applicability==='applicable';
  byId('warning').hidden=!warningVisible;
  byId('warning-title').textContent=state.evidence.lifecycle==='active'?'Tornado Warning · recorded replay':`Recorded warning · ${state.evidence.lifecycle} in fixture`;
  byId('timeline').replaceChildren();
  const labels=['T+0 · Approval','T+5 · New evidence','T+6 · Corrections','T+7 · Re-check'];
  const current=state.phase<=1?0:state.phase===2?1:state.phase<=4?2:3;
  labels.forEach((label,i)=> { const li=document.createElement('li'); li.className='timeline-step'; if(i===current) li.setAttribute('aria-current','step'); li.textContent=label; byId('timeline').append(li); });
  byId('prior-status').textContent=state.originalApproved?'Approved by fixture manager · 3 delivered / 2 queued (simulated)':'Ready for fictional manager approval. Delivery fixtures start after approval.';
  const changed=state.phase>=2;
  byId('change-title').textContent=!changed?'Awaiting new evidence':gate.status==='unknown'?'Evidence is unknown':gate.status==='unaffected'?'Warning outside facility':state.evidence.lifecycle!=='active'?'Warning lifecycle changed':'Warning now covers facility';
  byId('change-body').textContent=!changed?'The replay begins before the warning applies.':gate.status==='review'&&state.evidence.lifecycle==='active'?'A warning polygon covers this fictional location. The earlier arrival instruction needs review; 3 recipients already received it.':gate.reason;
  byId('action-title').textContent=state.phase===5?'Simulation recorded':state.phase===4?'Approval ready for re-check':changed?`${state.held.length} queued notices held`:'Approve the starting instruction';
  byId('action-body').textContent=state.phase===5?'5 correction intents are queued in the local experiment. No messages were delivered.':state.phase===4?'Approval is bound to the exact package. Changes to text, recipients, or relevant evidence require review.':'Official instructions display independently. Operational corrections require manager approval.';
  byId('primary').textContent=actions[state.phase];
  byId('primary').disabled=state.phase>=2 && (gate.status!=='review' || state.evidence.lifecycle!=='active' || state.evidence.applicability!=='applicable');
  byId('feedback').textContent=state.feedback;
  renderGroups();
  byId('history').replaceChildren();
  state.history.forEach(text=> { const li=document.createElement('li'); li.className='notice text-sm'; li.textContent=text; byId('history').append(li); });
  if(!state.history.length) byId('history').textContent='No replay decisions recorded yet.';
  byId('trace').textContent=JSON.stringify({ mode:state.mode, policy:state.policyVersion, gate, approval:state.approval?'Exact package approved':'No operational approval', simulatedIntents:state.intents.map(i=>({recipient:i.recipient,state:i.state})), package:packageFor(state) },null,2);
}
function renderGroups() {
  byId('groups').replaceChildren();
  const groupLabels={onsite:'Onsite staff',arriving:'Staff not yet onsite',remote:'Remote staff'};
  for(const group of ['onsite','arriving','remote']) {
    const people=state.roster.filter(r=>r.group===group), row=document.createElement('article'); row.className='border-t border-line pt-5';
    const title=document.createElement('h3'); title.className='font-semibold'; title.textContent=`${groupLabels[group]} · ${people.length}`; row.append(title);
    const status=document.createElement('p'); status.className='text-sm text-dim mt-2'; status.textContent=people.map(r=>`${r.name} (${r.id}): ${!state.originalApproved?'not queued':r.original}${state.held.includes(r.id)?', held by policy':''}`).join(' · '); row.append(status);
    const reason=document.createElement('p'); reason.className='text-sm mt-2'; reason.textContent=group==='remote'?'Excluded from arrival corrections: no original arrival instruction targeted this group. No personal-location weather claim is made.':group==='onsite'?'Clarify the obsolete arrival instruction without directing staff to leave.':'Correct already-delivered notices and replace the held arrival notices.'; row.append(reason);
    if(group!=='remote' && state.phase>=3) {
      const label=document.createElement('label'); label.className='block text-sm font-semibold mt-4'; label.htmlFor=`message-${group}`; label.textContent='Reviewed operational correction · Fixture draft'; row.append(label);
      const area=document.createElement('textarea'); area.id=`message-${group}`; area.className='field mt-2'; area.rows=4; area.value=state.drafts[group];
      area.addEventListener('input',()=> { state.drafts[group]=area.value; if(state.approval) {state.approval=null;state.phase=3;state.feedback='Message edited. Approve the exact new text before simulation.';byId('primary').textContent=actions[3];byId('feedback').textContent=state.feedback;} }); row.append(area);
      if(state.phase===5) {const p=document.createElement('p');p.className='text-sm mt-2 font-semibold';p.textContent='Correction intents: queued (simulation) · Attempted: 0 · Delivered: 0 · Acknowledged: 0';row.append(p);}
    }
    byId('groups').append(row);
  }
}
byId('primary').addEventListener('click',()=> { advance(state); render(); });
byId('case').addEventListener('change',e=> {setEvidence(state,e.target.value);render();});
byId('mode').addEventListener('change',e=> {state.mode=e.target.value==='live'?'LIVE':'REPLAY';render();});
byId('restart').addEventListener('click',()=> {state=initialState();byId('case').value='applicable';render();});
byId('theme').addEventListener('click',()=> {const dark=document.documentElement.dataset.theme!=='dark';document.documentElement.dataset.theme=dark?'dark':'light';byId('theme').textContent=dark?'Light mode':'Dark mode';byId('theme').setAttribute('aria-pressed',String(dark));});
render();
let liveEvidence=null;
let liveValid=false;
let liveBusy=false;
const liveNode=(tag,text,classes='')=>{const el=document.createElement(tag);el.textContent=text;el.className=classes;return el;};
byId('refresh-live').addEventListener('click',async()=>{
  if(liveBusy)return;liveBusy=true;byId('refresh-live').disabled=true;byId('brief-live').disabled=true;liveValid=false;byId('agent-panel').hidden=true;
  byId('live-status').textContent='Refreshing NOAA. Previously displayed evidence is last known until refresh succeeds.';
  try {
    const res=await fetch('/api/evidence'),data=await res.json();if(!res.ok)throw Error(data.error||'Refresh failed');
    liveEvidence=data;liveValid=true;
    const applicable=data.alerts.filter(a=>a.applicability==='applicable'), unknown=data.alerts.filter(a=>a.applicability==='unknown');
    byId('live-title').textContent=applicable.length?`${applicable.length} alert(s) apply to this facility`:'No applicable polygon alerts returned';
    byId('live-status').textContent=`Last successful refresh: ${new Date(data.lastSuccessfulRefresh).toLocaleString()} · ${data.alerts.length} statewide alerts checked · ${unknown.length} applicability unknown. Absence of an alert is not an all-clear. ${data.agentConfigured?'Strands endpoint configured.':'AI briefing needs AI_BASE_URL, AI_API_KEY, and AI_MODEL_ID in .env.'}`;
    byId('brief-live').disabled=!data.agentConfigured;
    byId('live-alerts').replaceChildren();
    const uncertainList=document.createElement('details');uncertainList.className='space-y-4';uncertainList.append(liveNode('summary',`${unknown.length} alert(s) have unknown facility applicability · Review missing evidence`,'cursor-pointer font-semibold'));
    const visible=[...applicable,...unknown];
    for(const a of visible) {
      const card=document.createElement('article');card.className=a.applicability==='applicable'?'panel bg-warning text-warning-ink':'panel';
      card.append(liveNode('p',a.applicability==='applicable'?'Applies to facility · Official NOAA evidence':'Applicability unknown · Missing evidence','eyebrow'));
      card.append(liveNode('h3',a.event,'text-xl font-semibold mt-2'));
      card.append(liveNode('p',`${a.issuer} · ${a.area}`,'text-sm mt-2'));
      card.append(liveNode('blockquote',a.instruction||'Official instruction field is missing.','font-semibold whitespace-pre-line mt-4'));
      const details=document.createElement('details');details.className='mt-3';details.append(liveNode('summary','Source and timestamps','cursor-pointer'));
      details.append(liveNode('pre',JSON.stringify({id:a.id,source:a.source,issued:a.issued,effective:a.effective,expires:a.expires,geography:a.geography,lifecycle:a.lifecycle,severity:a.severity,certainty:a.certainty,urgency:a.urgency},null,2),'text-xs mt-2'));card.append(details);
      if(a.applicability==='applicable'||a.event==='Tornado Warning')byId('live-alerts').append(card);else uncertainList.append(card);
    }
    if(unknown.length)byId('live-alerts').append(uncertainList);
    const gap=document.createElement('div');gap.className='notice text-sm';gap.textContent=data.gaps.join(' ');byId('live-alerts').append(gap);
  } catch(err) { byId('live-title').textContent='Current conditions are unknown';byId('live-status').textContent=`${err.message} Any displayed alert is last known, not current.`; }
  finally{liveBusy=false;byId('refresh-live').disabled=false;}
});
byId('brief-live').addEventListener('click',async()=>{
  if(!liveValid||!liveEvidence)return;
  byId('brief-live').disabled=true;byId('refresh-live').disabled=true;
  byId('live-status').textContent='Drafting an evidence briefing with one Strands agent. Official NOAA instructions remain visible.';
  try {
    const res=await fetch('/api/briefing',{method:'POST'}),data=await res.json();if(!res.ok)throw Error(data.error||'Briefing failed');
    byId('agent-panel').hidden=false;byId('agent-explanation').textContent=data.briefing.explanation;byId('agent-output').textContent=JSON.stringify(data,null,2);
    byId('live-status').textContent=`AI draft for human review · Based on NOAA refresh ${data.evidenceRefresh}. No operational decision or send occurred.`;
  }catch(err){byId('live-status').textContent=err.message;}
  finally{byId('refresh-live').disabled=false;byId('brief-live').disabled=!liveValid||!liveEvidence.agentConfigured;}
});
setInterval(()=>{if(liveEvidence && Date.now()-Date.parse(liveEvidence.lastSuccessfulRefresh)>300000 && liveValid){liveValid=false;byId('brief-live').disabled=true;byId('live-status').textContent='Last NOAA refresh is over five minutes old. Current conditions are unknown; refresh before briefing.';}},15000);
