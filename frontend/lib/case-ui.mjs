export const FACILITIES = { batesburg: ['Batesburg Field Office', 'South Carolina', 'SC-DEMO-02'], barnwell: ['Barnwell Field Office', 'South Carolina', 'SC-DEMO-01'], florida: ['Big Bend Field Office', 'Florida', 'FL-DEMO-01'], california: ['Inland Field Office', 'California', 'CA-DEMO-01'], montana: ['Great Falls Field Office', 'Montana', 'MT-DEMO-01'] };
export const POSTURES = { normal: 'Normal operations', delay: 'Delay arrivals', 'early-dismissal': 'Early dismissal', telework: 'Telework', closure: 'Close facility' };
export const GROUPS = { onsite: 'Already onsite', arriving: 'Not yet onsite', remote: 'Working remotely' };
export const dateTime = value => value && Number.isFinite(Date.parse(value)) ? new Date(value).toLocaleString('en-US', { year:'numeric',month:'short',day:'numeric',hour:'numeric',minute:'2-digit',timeZoneName:'short' }) : 'Not recorded';
export function localInput(value) {
  const date = new Date(value);
  if (!Number.isFinite(date.getTime())) return '';
  return new Date(date.getTime()-date.getTimezoneOffset()*60000).toISOString().slice(0,16);
}
export function projectEvidence(evidence, mode, now) {
  const fetched = Date.parse(evidence?.lastSuccessfulRefresh);
  const fresh = mode === 'REPLAY' || (Number.isFinite(fetched) && now-fetched<300000 && now>=fetched-30000);
  const alerts = (evidence?.alerts || []).map(alert => {
    if (mode === 'REPLAY') return alert;
    const start=Date.parse(alert.effective), end=Date.parse(alert.expires);
    const known=Number.isFinite(start)&&Number.isFinite(end)&&end>start;
    const lifecycle=alert.messageType==='Cancel'?'cancelled':!known?'unknown':now>=end?'expired':now<start?'not-yet-effective':'active';
    const applicability=!fresh||lifecycle==='unknown'?'unknown':lifecycle!=='active'||alert.status!=='Actual'?'not-applicable':alert.geography==='applicable'?'applicable':alert.geography==='unknown'?'unknown':'not-applicable';
    return {...alert,lifecycle,applicability};
  });
  const reviewCurrent=fresh && (mode==='REPLAY' || alerts.every((alert,index)=>{const captured=evidence.alerts[index];return captured.geography==='outside'||(alert.lifecycle===captured.lifecycle&&alert.applicability===captured.applicability);}));
  const assessment=evidence?.postureAssessment;
  const triggers=assessment?.triggers || [];
  const suggestionCurrent=reviewCurrent && (!assessment?.recommendedPosture || (triggers.length>0&&triggers.every(trigger=>alerts.some(alert=>alert.id===trigger.alertId&&alert.source===trigger.source&&trigger.source&&alert.lifecycle==='active'&&alert.applicability==='applicable'&&alert.status==='Actual'))));
  return {fresh:reviewCurrent,alerts,assessment,suggestionCurrent};
}
export async function api(path, body, signal) {
  const response=await fetch(path,{method:body===undefined?'GET':'POST',headers:body===undefined?{}:{'Content-Type':'application/json'},body:body===undefined?undefined:JSON.stringify(body),cache:'no-store',signal});
  let data;
  try { if(!response.headers.get('content-type')?.includes('application/json'))throw Error();data=await response.json(); }
  catch(error){if(error.name==='AbortError')throw error;throw Error('The service did not respond. Try again.');}
  if(!response.ok){const error=Error(data.error||'The request could not be completed.');error.status=response.status;error.data=data;throw error;}
  return data;
}

export function riverComparison(signal) {
  const difference=signal?.minorStageDifference;
  if (!Number.isFinite(difference)) return 'Threshold comparison unavailable';
  if (difference===0) return 'At minor flood stage at this gauge';
  const amount=new Intl.NumberFormat('en-US',{maximumFractionDigits:2}).format(Math.abs(difference));
  return `${amount} ${signal.thresholds?.unit||signal.unit||''} ${difference>0?'above':'below'} minor flood stage at this gauge`;
}

export const FACILITY_TIMEZONES={batesburg:'America/New_York',barnwell:'America/New_York',florida:'America/New_York',california:'America/Los_Angeles',montana:'America/Denver'};
export function facilityTime(value,facility,short=false){return value&&Number.isFinite(Date.parse(value))?new Date(value).toLocaleString('en-US',{...(short?{}:{month:'short',day:'numeric'}),hour:'numeric',minute:'2-digit',timeZone:FACILITY_TIMEZONES[facility]||'America/New_York',timeZoneName:'short'}):'Not recorded';}
export function effectiveTimestamp(input,original){if(input===localInput(original)&&Number.isFinite(Date.parse(original)))return new Date(original).toISOString();const parsed=new Date(input);return Number.isFinite(parsed.getTime())?parsed.toISOString():null;}
export function deliverySummary(events,intents,binding,recipient){
 if(!binding)return null;
 const trace=(events||[]).filter(event=>event.approvalBinding===binding&&event.recipient===recipient&&event.simulated===true);
 const latest=trace.at(-1);
 if(!latest)return (intents||[]).some(intent=>intent.approvalBinding===binding&&intent.recipient===recipient&&intent.simulated===true)?{label:'Queued (simulated)',fallback:false,trace:[]}:null;
 const names={attempted:'Attempted',delivered:'Delivered',acknowledged:'Acknowledged',unreachable:'Unreachable','next-channel':'Next channel'};
 return {label:`${names[latest.state]||latest.state} via ${latest.channel==='sms'?'SMS':latest.channel} (simulated)`,fallback:trace.some(event=>event.state==='unreachable')&&trace.some(event=>event.state==='next-channel'),trace};
}

// Planned channels describe the route. Only exact-bound events describe progress.
export function notificationChannels(person) {
 return person.included ? person.channels?.map(channel=>channel==='sms'?'SMS':channel).join(' + ') || 'No channel' : 'Not targeted';
}
export function notificationStatus(person,events,intents,binding) {
 if (!person.included) return {label:'Not targeted',fallback:false,trace:[]};
 return deliverySummary(events,intents,binding,person.id) || {label:'Not attempted',fallback:false,trace:[]};
}
// A conflict installs new evidence, but keeps the manager's unsaved decision.
// Message edits are notes only: a changed roster must be reviewed from scratch.
export function revisionConflictDraft(error,active,mode,draft,messages) {
 const latest=error.data?.case;
 if(error.status!==409 || error.data?.conflict!=='revision' || !active?.facility?.id || !latest?.revision || latest.mode!==mode || latest.facility?.id!==active.facility.id) return null;
 const notes=(active.recipients||[]).filter(person=>person.included && messages[person.id]!==undefined && messages[person.id]!==person.text).map(person=>({id:person.id,name:person.name,text:messages[person.id]}));
 return {latest,draft:{posture:draft.posture,rationale:draft.rationale,effective:draft.effective,overrideReason:draft.overrideReason},notes};
}
