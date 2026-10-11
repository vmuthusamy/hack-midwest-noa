export const OUTCOMES=[['accepted','Accepted'],['rejected','Rejected'],['timeout','Timed out'],['cancelled','Cancelled'],['providerError','Provider error']];
export function bucketTotal(bucket){return OUTCOMES.every(([key])=>Number.isFinite(bucket?.[key])&&bucket[key]>=0)?OUTCOMES.reduce((sum,[key])=>sum+bucket[key],0):null;}
export function isPercentileBin(bin,value){return Number.isFinite(value)&&Number.isFinite(bin?.minMs)&&value>=bin.minMs&&(bin.maxMs===null||(Number.isFinite(bin.maxMs)&&value<bin.maxMs));}
export function measuredDuration(value){return Number.isFinite(value)?value<1000?`${Number(value.toFixed(2))} ms`:`${(value/1000).toFixed(1)} s`:'Not recorded';}
export function histogramRange(bin){return bin.maxMs===null?`${bin.minMs/1000}+ s`:`${bin.minMs/1000}–${bin.maxMs/1000} s`;}
