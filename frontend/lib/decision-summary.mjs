import { workdayBands, validTimeZone, timelineClock } from './alert-timeline.mjs';
import { projectEvidence } from './case-ui.mjs';

const instant = value => { const n=Date.parse(value || ''); return Number.isFinite(n)?n:null; };
const duration = ms => { const m=Math.floor(ms/60000); return m<60?`${m} min`:`${Math.floor(m/60)} h${m%60?` ${m%60} min`:''}`; };

// Presentation only. These labels never decide applicability, approval or sending.
export function decisionSummary(current, timeZone, wallClock=Date.now()) {
 const evidence=current?.evidence || {}, clock=current?.mode==='REPLAY'?instant(evidence.lastSuccessfulRefresh):wallClock;
 const projection=projectEvidence(evidence,current?.mode,clock);
 const triggers=evidence.postureAssessment?.triggers || [];
 const relevant=projection.alerts.filter(a=>a.geography!=='outside');
 const alert=relevant.find(a=>triggers.some(t=>t.matched&&t.alertId===a.id)) || relevant.find(a=>a.applicability==='applicable') || relevant[0];
 const refreshed=instant(evidence.lastSuccessfulRefresh), expires=instant(alert?.expires);
 const complete=clock!=null && refreshed!=null && clock>=refreshed-30000 && clock-refreshed<300000 && projection.fresh && alert?.geography==='applicable' && alert?.applicability==='applicable' && alert?.lifecycle==='active' && alert?.status==='Actual' && expires!=null && expires>clock && /^https:\/\/api\.weather\.gov\//.test(alert?.source||'');
 const confidence=!complete?'Low':['Observed','Likely'].includes(alert.certainty)?'High':alert.certainty==='Possible'?'Medium':'Low';
 const confidenceReason=!complete?'Missing, stale or inapplicable alert evidence.':`NWS certainty: ${alert.certainty}; refresh is within five minutes${current.mode==='REPLAY'?' of the recorded clock':''}.`;
 let overlap='Unknown', overlapReason='A facility-local schedule and applicable alert timing are needed.';
 const start=instant(alert?.onset)||instant(alert?.effective), end=instant(alert?.ends)||expires;
 const zone=validTimeZone(timeZone), hours=current?.operatingHours||'';
 if(complete&&zone&&/^\d{1,2}:\d\d\s*[–-]\s*\d{1,2}:\d\d/.test(hours)&&start!=null&&end!=null&&end>start) {
   const horizon=clock+24*3600000, from=Math.max(clock,start), to=Math.min(horizon,end);
   const bands=workdayBands(hours,clock,horizon,zone);
   if(bands.length) {
     const milliseconds=to>from?bands.reduce((sum,b)=>sum+Math.max(0,Math.min(to,b.end)-Math.max(from,b.start)),0):0;
     overlap=duration(milliseconds);
     overlapReason=`Next 24 hours, using ${hours}. ${alert.ends?'Expected hazard window':'Message window; hazard end is unknown'}. Zero overlap does not establish safe reopening.`;
   }
 }
 return {alert,confidence,confidenceReason,overlap,overlapReason,expiresIn:complete?duration(expires-clock):'Unknown',recorded:current?.mode==='REPLAY',clock};
}

export function plainOperatingRationale(current,timeZone,wallClock=Date.now()) {
 const s=decisionSummary(current,timeZone,wallClock), a=s.alert;
 if(!a || s.confidence==='Low')return 'Weather evidence needs review. Confirm the current warning and facility conditions before approving an operating change.';
 const name=current.facility?.name||'this facility';
 const weather=`${s.recorded?'Recorded ':''}${a.event} covers ${name}; message expires ${timelineClock(a.expires,timeZone,true)}.`;
 const power=current.evidence?.powerSignal;
 const powerText=power?.status==='yes'?` Power reported out (${power.source || 'manager report'}, ${timelineClock(power.observedAt,timeZone,true)}).`:'';
 const action={closure:'close the facility',telework:'telework for eligible staff',delay:'delay arrivals'}[current.operatingPosture] || 'review operating arrangements';
 return `${weather}${powerText} Proposed: ${action} from ${timelineClock(current.effectiveAt,timeZone,true)}. Review essential assignments separately.`;
}
