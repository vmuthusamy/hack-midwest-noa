import {readFile} from 'node:fs/promises';
import {initialState,advance,approvePackage,setEvidence} from '../prototype/model.mjs';
const fixtures=JSON.parse(await readFile(new URL('../fixtures/change-cases.json',import.meta.url)));
let missed=0,material=0,unnecessary=0,unaffected=0;
for(const f of fixtures){
 const s=initialState();advance(s);advance(s);advance(s);approvePackage(s);setEvidence(s,f.case);
 if(f.material){material++;if(Boolean(s.approval)!==f.expectedApprovalValid)missed++;}
 else {unaffected++;if(Boolean(s.approval)!==f.expectedApprovalValid)unnecessary++;}
}
// Separate independent outside-warning sample: a warning never affecting this site.
const outside=initialState();setEvidence(outside,'outside');advance(outside);advance(outside);
unaffected++;if(outside.held.length)unnecessary++;
console.log(JSON.stringify({scope:'Deterministic fixture samples only; no model evaluation',missedMaterialChanges:{count:missed,samples:material},unnecessaryNewHoldsOrInvalidations:{count:unnecessary,samples:unaffected},actualAgentQualitySamples:0},null,2));
if(missed||unnecessary)process.exitCode=1;
