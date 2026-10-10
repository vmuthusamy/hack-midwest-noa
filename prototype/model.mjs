// Local experiment logic. These gates only create simulated send intents.
export const facility = { id: 'FL-DEMO-01', name: 'Big Bend Field Office', latitude: 29.99, longitude: -83.44 };
export const policyVersion = 'illustrative-v1';
export const roster = [
  { id:'S01', name:'Alex', group:'onsite', original:'delivered' },
  { id:'S02', name:'Jordan', group:'onsite', original:'queued' },
  { id:'S03', name:'Morgan', group:'arriving', original:'delivered' },
  { id:'S04', name:'Casey', group:'arriving', original:'delivered' },
  { id:'S05', name:'Taylor', group:'arriving', original:'queued' },
  { id:'S06', name:'Sam', group:'remote', original:'not targeted' },
];
export const fixtureDrafts = {
  onsite: 'The earlier “Report at 10:00” operating instruction is superseded. Your onsite reporting status is under review. This message does not direct you to leave the building. The official NWS instructions are displayed separately.',
  arriving: 'The earlier “Report at 10:00” operating instruction is superseded. The facility manager has paused the arrival instruction pending a further operational update. This is not a reopening notice. The official NWS instructions are displayed separately.',
};
export function evidenceFor(kind='applicable') {
  // This is synthetic timing and a test geography result, never live evidence.
  const e = { mode:'REPLAY', applicability:'applicable', event:'Tornado Warning', evidenceId:'fixture:noaa-derived-warning', materialVersion:'v1', status:'Actual', lifecycle:'active', fresh:true, instructionsVersion:'recorded-original' };
  if(kind==='outside') e.applicability='outside';
  if(kind==='missing') e.applicability='unknown';
  if(kind==='stale') e.fresh=false;
  if(kind==='update') { e.materialVersion='v2'; e.instructionsVersion='synthetic-update'; }
  if(kind==='cancel') { e.lifecycle='cancelled'; e.materialVersion='cancel-v2'; }
  if(kind==='expired') { e.lifecycle='expired'; e.materialVersion='expired-v2'; }
  if(kind==='irrelevant') e.unrelatedStatewideUpdate='unrelated-alert';
  return e;
}
export function policyGate(e) {
  if (!e || e.applicability==='unknown' || !e.fresh) return { status:'unknown', hold:true, reason:'Evidence incomplete or stale; review required. No all-clear.' };
  if (e.lifecycle!=='active') return { status:'review', hold:true, reason:'Warning changed or ended. Expiration/cancellation does not authorize reopening.' };
  if (e.applicability==='outside') return { status:'unaffected', hold:false, reason:'This warning is outside the facility. No new hold from this evidence.' };
  return { status:'review', hold:true, reason:'Applicable warning; illustrative policy holds queued arrival notices and requests operational review.' };
}
function canonical(v) {
  if (Array.isArray(v)) return v.map(canonical);
  if (v && typeof v==='object') return Object.fromEntries(Object.keys(v).sort().map(k=>[k,canonical(v[k])]));
  return v;
}
export function relevantConditions(e) {
  return Object.fromEntries(['mode','applicability','event','evidenceId','materialVersion','status','lifecycle','fresh','instructionsVersion'].map(k=>[k,e[k]]));
}
export function packageFor(state) {
  return { facility:state.facility, action:'revise-arrival-instruction', parameters:{ previous:'Report at 10:00', replacement:'Arrival instruction paused pending manager update' }, policyVersion:state.policyVersion, evidence:relevantConditions(state.evidence), messages:['onsite','arriving'].map(group=>({ group, recipients:state.roster.filter(r=>r.group===group).map(r=>r.id).sort(), text:state.drafts[group] })) };
}
export const fingerprint = value => JSON.stringify(canonical(value));
export function initialState() {
  return { phase:0, mode:'REPLAY', facility:{...facility}, policyVersion, evidence:evidenceFor(), roster:structuredClone(roster), drafts:{...fixtureDrafts}, approval:null, originalApproved:false, held:[], intents:[], history:[], feedback:'Start by approving the fictional arrival instruction.' };
}
export function setEvidence(state,kind) {
  state.evidence=evidenceFor(kind);
  if (state.approval && state.approval.binding!==fingerprint(packageFor(state))) {
    state.approval=null; state.phase=Math.min(state.phase,3);
    state.history.push('Operational approval invalidated: relevant evidence changed.');
    state.feedback='Evidence changed. Review the package again before any simulated sending.';
  } else if(kind==='irrelevant') state.feedback='Unrelated statewide update: operational approval remains valid.';
  if(state.phase>=2) holdQueued(state);
}
export function holdQueued(state) {
  if(policyGate(state.evidence).hold) {
    state.held=[...new Set([...state.held,...state.roster.filter(r=>r.original==='queued').map(r=>r.id)])];
  }
}
export function approvePackage(state) {
  if(state.mode!=='REPLAY' || state.phase<3 || !state.originalApproved) return { ok:false, reason:'A reviewed replay package is required.' };
  if(policyGate(state.evidence).status!=='review' || state.evidence.lifecycle!=='active') return { ok:false, reason:'The baseline correction needs fresh, applicable active evidence. Resolve the evidence change first.' };
  if(state.evidence.applicability!=='applicable') return { ok:false, reason:'Facility applicability is not established.' };
  if(Object.values(state.drafts).some(t=>!t.trim())) return { ok:false, reason:'Every targeted group needs a reviewed message.' };
  const pkg=packageFor(state);
  state.approval={ manager:'fixture-manager', binding:fingerprint(pkg), package:structuredClone(pkg), state:'approved' };
  state.phase=4; state.history.push('T+6 · Fixture manager approved the exact recipients, messages, policy, and evidence conditions.');
  return { ok:true };
}
export function recheckAndSimulate(state) {
  if(state.mode!=='REPLAY' || !state.approval) return { ok:false, reason:'Manager approval is required.' };
  if(state.approval.binding!==fingerprint(packageFor(state))) return { ok:false, reason:'Approved package changed; review again.' };
  if(!state.evidence.fresh || state.evidence.applicability!=='applicable' || state.evidence.lifecycle!=='active') return { ok:false, reason:'Current evidence does not satisfy the approved conditions.' };
  let added=0;
  for(const msg of state.approval.package.messages) for(const recipient of msg.recipients) {
    const key=fingerprint({ binding:state.approval.binding, recipient });
    if(!state.intents.some(i=>i.key===key)) { state.intents.push({ key, recipient, message:msg.text, state:'queued', simulated:true }); added++; }
  }
  state.phase=5;
  if(added) state.history.push(`T+7 · Re-check passed; ${added} simulated send intents queued. No delivery or acknowledgment asserted.`);
  return { ok:true, added };
}
export function advance(state) {
  if(state.phase===0) {
    state.originalApproved=true; state.phase=1;
    state.history.push('T+0 · Manager approved “Report at 10:00.” Delivery fixtures: 3 delivered, 2 queued; 1 remote staff member excluded.');
    state.feedback='Original notice states are simulated fixtures. Continue to the new evidence.';
  } else if(state.phase===1) {
    state.phase=2; holdQueued(state);
    state.history.push(`T+5 · Evidence applied. ${state.held.length} queued arrival notices held by illustrative policy. Official instructions are independent.`);
    state.feedback=policyGate(state.evidence).reason;
  } else if(state.phase===2) { state.phase=3; state.feedback='Review group recipients and each correction. These drafts are fixtures, not AI output.'; }
  else if(state.phase===3) { const result=approvePackage(state); state.feedback=result.ok?'Operational correction approved. Re-check current evidence before simulation.':result.reason; }
  else { const result=recheckAndSimulate(state); state.feedback=result.ok?`${result.added} new simulated intents queued. Delivery and acknowledgment remain unasserted.`:result.reason; }
}
