import { riverComparison } from '../lib/case-ui.mjs';
import { MapPin } from 'lucide-react';
import { Badge, Panel, SourceLink } from './ui';
export function GeographyPlot({ alert, facility, replayCase }) {
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
const riverTime = value => value && Number.isFinite(Date.parse(value)) ? new Date(value).toLocaleString('en-US', { year: 'numeric', month: 'short', day: 'numeric', hour: 'numeric', minute: '2-digit', timeZoneName: 'short' }) : 'Not available';
function riverAge(seconds) {
  if (!Number.isFinite(seconds)) return 'Observation age unknown';
  if (seconds < 0) return 'Observation time is ahead of the device clock';
  if (seconds < 60) return 'Less than a minute old';
  const [amount, unit] = seconds < 3600 ? [Math.floor(seconds / 60), 'minute'] : seconds < 86400 ? [Math.floor(seconds / 3600), 'hour'] : [Math.floor(seconds / 86400), 'day'];
  return `${amount} ${unit}${amount === 1 ? '' : 's'} old`;
}
export function RiverContext({ signals, now, mode='LIVE' }) {
  const number = value => Number.isFinite(value) ? new Intl.NumberFormat('en-US', { maximumFractionDigits: 2 }).format(value) : null;
  return <Panel className="river-panel"><div className="panel-header"><div><p className="eyebrow">{mode==='REPLAY'?'RECORDED RIVER OBSERVATION':'SUPPLEMENTAL · LIVE'}</p><h3>Regional river context</h3></div><Badge>{mode==='REPLAY'?'Recorded':'Live'}</Badge></div>{signals.length ? signals.map((signal, index) => {
    const available = signal.state === 'available' && Number.isFinite(signal.value);
    const minor = number(signal.thresholds?.minor);
    const observed = Date.parse(signal.observedAt);
    const age = Number.isFinite(observed) ? (now - observed) / 1000 : signal.observationAgeSeconds;
    return <article className="river-reading" key={`${signal.provider}-${signal.gaugeId}-${index}`}><div className="river-reading-heading"><div><strong>{signal.name || signal.gaugeId || 'Regional gauge'}</strong><p className="microcopy">{signal.provider} · {signal.gaugeId || signal.usgsId || 'Gauge not selected'}</p></div><span className="river-value">{available ? `${number(signal.value)} ${signal.unit || ''}` : 'Unknown'}</span></div><div className="river-facts"><p><span>Observed</span>{riverTime(signal.observedAt)}{mode==='LIVE'&&<small className="river-observation-age">{riverAge(age)}</small>}</p><p><span>Source fetched successfully</span>{riverTime(signal.fetchedAt)}</p><p><span>Minor flood stage at gauge</span>{minor !== null ? `${minor} ${signal.thresholds.unit || signal.unit || ''}` : 'Not published'}</p></div><p className="river-comparison">{riverComparison(signal)}</p><details><summary>Sources & limitations</summary>{signal.attemptedAt&&<p className="microcopy">Last fetch attempted: {riverTime(signal.attemptedAt)}</p>}<div className="river-links"><SourceLink href={signal.source}>Observation source</SourceLink><SourceLink href={signal.metadataSource}>Gauge metadata</SourceLink>{signal.thresholds?.source&&<SourceLink href={signal.thresholds.source}>Flood thresholds</SourceLink>}</div>{signal.approvalStatus&&<p className="microcopy">Observation status: {signal.approvalStatus}</p>}{signal.reportedCategory&&<p className="microcopy">Provider category at this gauge: {signal.reportedCategory}</p>}{signal.relevanceExplanation&&<p className="microcopy">{signal.relevanceExplanation}</p>}{!!signal.gaps?.length&&<p className="microcopy">{signal.gaps.join(' · ')}</p>}</details></article>;
  }) : <p className="muted">No river observation is available for this location.</p>}</Panel>;
}
