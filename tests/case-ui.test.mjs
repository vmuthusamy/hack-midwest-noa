import test from 'node:test';
import assert from 'node:assert/strict';
import {projectEvidence,riverComparison,effectiveTimestamp,localInput,facilityTime,deliverySummary,notificationChannels,notificationStatus,revisionConflictDraft} from '../frontend/lib/case-ui.mjs';
const now=Date.parse('2026-10-11T02:00:00Z');
const alert={id:'test-warning',source:'https://api.weather.gov/alerts/test-warning',status:'Actual',messageType:'Alert',geography:'applicable',applicability:'applicable',lifecycle:'active',effective:'2026-10-11T01:00:00Z',expires:'2026-10-11T02:00:30Z',ends:'2026-10-12T00:00:00Z'};
function evidence(overrides={}){return{lastSuccessfulRefresh:'2026-10-11T01:59:50Z',alerts:[alert],postureAssessment:{recommendedPosture:'telework',triggers:[{alertId:alert.id,source:alert.source}]},...overrides};}
test('Current live evidence preserves the server suggestion without choosing an operation',()=>{const p=projectEvidence(evidence(),'LIVE',now);assert.equal(p.fresh,true);assert.equal(p.suggestionCurrent,true);assert.equal(p.assessment.recommendedPosture,'telework');});
test('Message expiry invalidates approval freshness even while refresh is recent and hazard end is later',()=>{const p=projectEvidence(evidence(),'LIVE',now+31000);assert.equal(p.fresh,false);assert.equal(p.suggestionCurrent,false);assert.equal(p.alerts[0].lifecycle,'expired');assert.equal(p.alerts[0].ends,alert.ends);assert.notEqual(p.assessment.recommendedPosture,'normal');});
test('Missing or mismatched suggestion source suppresses the suggestion',()=>{for(const trigger of [{alertId:'missing',source:alert.source},{alertId:alert.id,source:'https://api.weather.gov/alerts/different'},{alertId:alert.id}])assert.equal(projectEvidence(evidence({postureAssessment:{recommendedPosture:'telework',triggers:[trigger]}}),'LIVE',now).suggestionCurrent,false);});
test('Five-minute stale evidence and unknown capture time cannot authorize review',()=>{for(const stamp of ['2026-10-11T01:54:00Z','invalid']){const p=projectEvidence(evidence({lastSuccessfulRefresh:stamp}),'LIVE',now);assert.equal(p.fresh,false);assert.equal(p.suggestionCurrent,false);}});
test('Replay preserves recorded lifecycle without borrowing the real clock',()=>{const p=projectEvidence(evidence(),'REPLAY',now+86400000);assert.equal(p.fresh,true);assert.equal(p.alerts[0].lifecycle,'active');assert.equal(p.suggestionCurrent,true);});
test('Planning-watch null suggestion remains null and does not become normal operations',()=>{const p=projectEvidence(evidence({postureAssessment:{status:'planning-review',recommendedPosture:null,triggers:[]}}),'LIVE',now);assert.equal(p.suggestionCurrent,true);assert.equal(p.assessment.recommendedPosture,null);});
test('Cancelled or changed applicability requires a new live review',()=>{const p=projectEvidence(evidence({alerts:[{...alert,messageType:'Cancel'}]}),'LIVE',now);assert.equal(p.fresh,false);assert.equal(p.suggestionCurrent,false);assert.equal(p.alerts[0].lifecycle,'cancelled');});

test('River comparisons use only the backend verified same-gauge difference',()=>{assert.equal(riverComparison({value:0,thresholds:{minor:19.5,unit:'ft'},minorStageDifference:null}),'Threshold comparison unavailable');assert.equal(riverComparison({unit:'ft',minorStageDifference:0}),'At minor flood stage at this gauge');assert.match(riverComparison({unit:'ft',minorStageDifference:-6.56}),/^6.56 ft below/);assert.match(riverComparison({unit:'ft',minorStageDifference:1.2}),/^1.2 ft above/);});

test('Untouched local time retains exact source seconds for recorded rationale binding',()=>{const original='2026-10-11T02:20:18Z';assert.equal(effectiveTimestamp(localInput(original),original),'2026-10-11T02:20:18.000Z');assert.equal(effectiveTimestamp('invalid',original),null);assert.notEqual(effectiveTimestamp(localInput('2026-10-11T03:00:00Z'),original),new Date(original).toISOString());});
test('South Carolina expiry uses facility timezone even when browser timezone differs',()=>{assert.match(facilityTime('2026-10-11T12:00:00Z','barnwell'),/8:00 AM EDT/);});

test('Delivery display uses only the current exact approval and recipient, preserving queued intents',()=>{const intent={approvalBinding:'new',recipient:'S02',simulated:true};const events=[{...intent,state:'attempted',channel:'sms'},{...intent,state:'unreachable',channel:'sms'},{...intent,state:'next-channel',channel:'email'},{...intent,state:'delivered',channel:'email'},{...intent,approvalBinding:'old',state:'acknowledged',channel:'sms'}];const result=deliverySummary(events,[intent],'new','S02');assert.equal(result.label,'Delivered via email (simulated)');assert.equal(result.fallback,true);assert.equal(result.trace.length,4);assert.equal(deliverySummary(events,[intent],'new','S01'),null);assert.equal(deliverySummary([], [intent],'new','S02').label,'Queued (simulated)');assert.equal(deliverySummary(events,[intent],null,'S02'),null);assert.equal(intent.state,undefined);});

test('Notification channels stay separate from exact-bound simulated progress and exclusions',()=>{
 const person={id:'S01',included:true,channels:['sms','email']};
 const intent={approvalBinding:'reviewed',recipient:person.id,simulated:true};
 assert.equal(notificationChannels(person),'SMS + email');
 assert.equal(notificationStatus(person,[],[],'reviewed').label,'Not attempted');
 assert.equal(notificationStatus(person,[],[intent],'reviewed').label,'Queued (simulated)');
 for(const [state,label] of [['attempted','Attempted'],['delivered','Delivered'],['acknowledged','Acknowledged']]){
  const events=[{...intent,state,channel:'sms'}];
  assert.equal(notificationStatus(person,events,[intent],'reviewed').label,`${label} via SMS (simulated)`);
  assert.equal(notificationStatus(person,events,[intent],'different').label,'Not attempted');
  assert.equal(notificationStatus({...person,included:false},events,[intent],'reviewed').label,'Not targeted');
 }
 assert.equal(notificationStatus(person,[{...intent,state:'delivered',simulated:false,channel:'sms'}],[], 'reviewed').label,'Not attempted');
 assert.equal(notificationChannels({...person,included:false}),'Not targeted');
});
test('Revision recovery keeps the local decision and isolates message notes from a changed roster',()=>{
 const active={mode:'LIVE',revision:'old',facility:{id:'SC-DEMO-02'},recipients:[{id:'S01',name:'Old recipient',included:true,text:'Old message'},{id:'S02',name:'Unchanged recipient',included:true,text:'Unchanged'}]};
 const latest={...active,revision:'new',evidence:{lastSuccessfulRefresh:'new evidence'},recipients:[{id:'S01',name:'New recipient',included:true,text:'New reviewed message'}]};
 const draft={posture:'closure',rationale:'Unsaved manager rationale',effective:'2026-10-11T09:00',overrideReason:'Local power unavailable'};
 const error={status:409,data:{conflict:'revision',case:latest}};
 const result=revisionConflictDraft(error,active,'LIVE',draft,{S01:'My edited message',S02:'Unchanged'});
 assert.deepEqual(result.draft,draft);
 assert.equal(result.latest,latest);
 assert.deepEqual(result.notes,[{id:'S01',name:'Old recipient',text:'My edited message'}]);
 assert.equal(latest.recipients[0].text,'New reviewed message');
 assert.equal(result.messages,undefined);
 for(const changed of [{...error,status:403},{status:409,data:{case:latest}},{status:409,data:{conflict:'revision',case:{...latest,mode:'REPLAY'}}},{status:409,data:{conflict:'revision',case:{...latest,facility:{id:'OTHER'}}}},{status:409,data:{conflict:'revision',case:null}}])assert.equal(revisionConflictDraft(changed,active,'LIVE',draft,{}),null);
});
