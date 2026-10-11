// Pure presentation model. Geographic applicability and approval remain server decisions.
export const SEVERITY = ['Minor', 'Moderate', 'Severe', 'Extreme'];
const CERTAINTY_WORD = { Observed: 'observed', Likely: 'likely', Possible: 'possible', Unlikely: 'unlikely' };
const URGENCY_WORD = { Immediate: 'immediate', Expected: 'expected', Future: 'future', Past: 'past' };
const CERTAINTY_FILL = { Observed: 'solid', Likely: 'solid', Possible: 'striped', Unlikely: 'outline' };
const HOUR = 3600e3, DAY = 24 * HOUR;
const ms = value => { const t = typeof value === 'number' ? value : Date.parse(value || ''); return Number.isFinite(t) ? t : null; };
export function validTimeZone(value) {
  if (!value) return null;
  try { new Intl.DateTimeFormat('en-US', { timeZone: value }).format(0); return value; } catch { return null; }
}
export function timelineClock(value, timeZone, withDay = false) {
  const t = ms(value);
  if (t == null) return 'Unknown';
  return new Intl.DateTimeFormat('en-US', { timeZone: validTimeZone(timeZone) || 'UTC', ...(withDay ? { weekday: 'short' } : {}), hour: 'numeric', minute: '2-digit', timeZoneName: 'short' }).format(t).replace(':00', '');
}
export function severityLevel(value) { return SEVERITY.indexOf(value) + 1; }
export function plainSummary(a) {
  return [SEVERITY.includes(a.severity) ? a.severity : 'Severity unknown', CERTAINTY_WORD[a.certainty] || 'certainty unknown', URGENCY_WORD[a.urgency] || 'urgency unknown'].join(' · ');
}
function localParts(t, timeZone) {
  return Object.fromEntries(new Intl.DateTimeFormat('en-CA', { timeZone, year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit', hourCycle: 'h23' }).formatToParts(t).filter(p => p.type !== 'literal').map(p => [p.type, Number(p.value)]));
}
// Resolve a facility wall-clock boundary against IANA rules. Ambiguous/nonexistent
// DST boundaries are omitted rather than selecting an undocumented operating time.
function wallTime(day, minutes, timeZone) {
  const d = new Date(day), target = { year: d.getUTCFullYear(), month: d.getUTCMonth() + 1, day: d.getUTCDate(), hour: Math.floor(minutes / 60), minute: minutes % 60 };
  const nominal = day + minutes * 60000, offsets = new Set();
  for (const delta of [-DAY, 0, DAY]) {
    const t = nominal + delta, p = localParts(t, timeZone);
    offsets.add(Date.UTC(p.year, p.month - 1, p.day, p.hour, p.minute) - t);
  }
  const matches = [...offsets].map(offset => nominal - offset).filter(t => {
    const p = localParts(t, timeZone); return Object.keys(target).every(k => p[k] === target[k]);
  });
  return matches.length === 1 ? matches[0] : null;
}
export function workdayBands(hours, start, end, timeZone) {
  const zone = validTimeZone(timeZone);
  const match = /^(\d{1,2}):(\d\d)\s*[–-]\s*(\d{1,2}):(\d\d)(?:\s|$)/.exec(hours || '');
  if (!match || !zone || start == null || end == null) return [];
  const [oh, om, ch, cm] = match.slice(1).map(Number);
  if (oh > 23 || ch > 23 || om > 59 || cm > 59) return [];
  const open = oh * 60 + om, close = ch * 60 + cm;
  if (open === close) return [];
  const bands = [], p = localParts(start, zone);
  for (let day = Date.UTC(p.year, p.month - 1, p.day) - DAY; day <= end + DAY; day += DAY) {
    const s = wallTime(day, open, zone), e = wallTime(day + (close < open ? DAY : 0), close, zone);
    if (s != null && e != null && e > start && s < end) bands.push({ start: Math.max(s, start), end: Math.min(e, end) });
  }
  return bands;
}
export function alertTimelineModel(alerts = [], { now, mode = 'LIVE', timeZone, operatingHours = '' } = {}) {
  const clock = ms(now), zone = validTimeZone(timeZone);
  const lanes = alerts.filter(a => a && a.geography !== 'outside').map((a, i) => {
    const onset = ms(a.onset), effective = ms(a.effective), issued = ms(a.issued), ends = ms(a.ends), expires = ms(a.expires);
    const start = onset ?? effective ?? issued, end = ends ?? expires;
    const cancelled = a.lifecycle === 'cancelled' || a.messageType === 'Cancel';
    const expired = a.lifecycle === 'expired' || (clock != null && expires != null && expires <= clock);
    const endPassed = clock != null && ends != null && ends <= clock;
    const status = cancelled ? 'Cancelled message' : expired ? 'Expired message' : endPassed ? 'Expected end passed' : '';
    return { id: a.id || `alert-${i}`, event: a.event || 'Unnamed alert', source: a.source,
      severity: a.severity, level: severityLevel(a.severity), fill: CERTAINTY_FILL[a.certainty] || 'outline', summary: plainSummary(a),
      issued, start, end, startKind: onset != null ? 'Onset' : effective != null ? 'Effective' : issued != null ? 'Issued' : 'Start unknown',
      endKind: ends != null ? 'Expected end' : expires != null ? 'Message expiry' : 'End unknown',
      invalidRange: start != null && end != null && end < start, openEnded: end == null,
      inactive: Boolean(status), status, unknownPlace: a.geography !== 'applicable' && a.geography !== 'inside',
    };
  }).sort((x, y) => Number(x.inactive) - Number(y.inactive) || y.level - x.level || (x.start ?? Infinity) - (y.start ?? Infinity));
  const times = lanes.flatMap(l => [l.start, l.end, l.issued].filter(t => t != null));
  if (clock != null) times.push(clock);
  let start = times.length ? Math.min(...times) : null, end = times.length ? Math.max(...times) : null;
  const ticks = [];
  if (start != null) {
    const pad = Math.max(HOUR / 4, (end - start) * .04); start -= pad; end += pad;
    const step = ([.25, .5, 1, 2, 3, 6, 12, 24].find(s => (end - start) / HOUR / s <= 6) || 24) * HOUR;
    for (let t = Math.ceil(start / step) * step; t <= end; t += step) ticks.push(t);
  }
  return { lanes, start, end, ticks, now: clock, timeZone: zone || 'UTC', timeZoneKnown: Boolean(zone), clockLabel: mode === 'REPLAY' ? 'Recorded time' : 'Now',
    bands: workdayBands(operatingHours, start, end, zone), position: t => start == null || t == null ? null : Math.max(0, Math.min(1, (t - start) / (end - start))) };
}
