import test from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { alertTimelineModel, timelineClock, plainSummary, workdayBands } from '../frontend/lib/alert-timeline.mjs';
const watch = { id: 'w', event: 'Flood Watch', severity: 'Severe', certainty: 'Possible', urgency: 'Future', applicability: 'applicable', geography: 'applicable', lifecycle: 'active', issued: '2026-10-10T08:39:00-04:00', effective: '2026-10-10T08:39:00-04:00', onset: '2026-10-10T08:39:00-04:00', ends: '2026-10-11T08:00:00-04:00', expires: '2026-10-11T08:00:00-04:00' };
const lake = { ...watch, id: 'l', event: 'Lake Wind Advisory', severity: 'Moderate', certainty: 'Likely', urgency: 'Expected', issued: '2026-10-10T15:40:00-04:00', onset: '2026-10-10T15:40:00-04:00', ends: '2026-10-10T23:00:00-04:00', expires: '2026-10-10T23:00:00-04:00' };
const now = Date.parse('2026-10-11T02:20:00Z'), timeZone = 'America/New_York';
test('severity order and certainty fill retain the supplied visual design', () => {
  const { lanes } = alertTimelineModel([lake, watch], { now });
  assert.deepEqual(lanes.map(l => [l.event, l.fill, l.level]), [['Flood Watch', 'striped', 3], ['Lake Wind Advisory', 'solid', 2]]);
  assert.equal(alertTimelineModel([{...watch, certainty: 'Unlikely'}]).lanes[0].fill, 'outline');
  assert.equal(alertTimelineModel([{...watch, certainty: 'Unknown'}]).lanes[0].fill, 'outline');
});
test('only geographic outsiders are excluded; cancelled and expired local messages remain', () => {
  const { lanes } = alertTimelineModel([{...watch, id:'outside', geography:'outside'}, {...watch, id:'expired', applicability:'not-applicable', lifecycle:'expired'}, {...watch, id:'cancel', applicability:'not-applicable', lifecycle:'cancelled'}, {...watch, id:'unknown', geography:'unknown', applicability:'unknown'}], { now });
  assert.equal(lanes.length, 3);
  assert.equal(lanes.find(l => l.id === 'expired').status, 'Expired message');
  assert.equal(lanes.find(l => l.id === 'cancel').status, 'Cancelled message');
  assert.equal(lanes.find(l => l.id === 'unknown').unknownPlace, true);
});
test('source urgency Expected does not claim within an hour or direct safety action', () => {
  assert.equal(plainSummary(lake), 'Moderate · likely · expected');
  assert.equal(plainSummary({}), 'Severity unknown · certainty unknown · urgency unknown');
});
test('message expiry and hazard end remain distinct; expiry never means safe reopening', () => {
  const { lanes } = alertTimelineModel([{...watch, onset:null, ends:null}], { now: Date.parse('2026-10-12T00:00Z') });
  assert.equal(lanes[0].startKind, 'Effective');
  assert.equal(lanes[0].endKind, 'Message expiry');
  assert.equal(lanes[0].status, 'Expired message');
  const expected = alertTimelineModel([{...watch, expires:'2026-10-13T00:00Z'}], {now: Date.parse('2026-10-12T00:00Z')}).lanes[0];
  assert.equal(expected.status, 'Expected end passed');
});
test('missing timestamps retain a row, without inventing a six-hour interval', () => {
  const { lanes } = alertTimelineModel([{id:'missing', event:'Warning', geography:'applicable'}], { now });
  assert.equal(lanes.length, 1); assert.equal(lanes[0].start, null); assert.equal(lanes[0].end, null);
  const open = alertTimelineModel([{...watch, ends:null, expires:null}], {now}).lanes[0];
  assert.equal(open.end, null); assert.equal(open.openEnded, true);
  assert.equal(alertTimelineModel([{...watch, ends:'2020-01-01'}], {now}).lanes[0].invalidRange, true);
});
test('facility IANA zone overrides issuing-office offset and follows DST', () => {
  assert.match(timelineClock('2026-10-11T02:20:00Z', 'America/Chicago'), /9:20 PM CDT/);
  assert.match(timelineClock('2026-11-01T05:30:00Z', timeZone), /1:30 AM EDT/);
  assert.match(timelineClock('2026-11-01T06:30:00Z', timeZone), /1:30 AM EST/);
  const m = alertTimelineModel([watch], {now, timeZone:'wrong/zone'});
  assert.equal(m.timeZone, 'UTC'); assert.equal(m.timeZoneKnown, false);
});
test('replay clock uses the supplied capture timestamp, never wall time', () => {
  const m = alertTimelineModel([watch], {now, mode:'REPLAY', timeZone});
  assert.equal(m.now, now); assert.equal(m.clockLabel, 'Recorded time');
  assert.equal(alertTimelineModel([watch], {mode:'REPLAY'}).now, null);
  assert.equal(alertTimelineModel([watch], {now, mode:'LIVE'}).clockLabel, 'Now');
});
test('workday bands require supplied hours and facility timezone', () => {
  const start = Date.parse('2026-10-11T04:00Z'), end = Date.parse('2026-10-12T03:59Z');
  assert.deepEqual(workdayBands('', start, end, timeZone), []);
  assert.deepEqual(workdayBands('08:00–17:00', start, end, null), []);
  assert.deepEqual(workdayBands('25:00–17:00', start, end, timeZone), []);
  const [band] = workdayBands('08:00–17:00 facility local time (synthetic)', start, end, timeZone);
  assert.equal(new Date(band.start).toISOString(), '2026-10-11T12:00:00.000Z');
  assert.equal(new Date(band.end).toISOString(), '2026-10-11T21:00:00.000Z');
});
test('overnight workday follows autumn DST transition; ambiguous boundaries are omitted', () => {
  const start = Date.parse('2026-11-01T01:00Z'), end = Date.parse('2026-11-01T13:00Z');
  const [band] = workdayBands('22:00–06:00 (synthetic)', start, end, timeZone);
  assert.equal(new Date(band.start).toISOString(), '2026-11-01T02:00:00.000Z');
  assert.equal(new Date(band.end).toISOString(), '2026-11-01T11:00:00.000Z');
  assert.deepEqual(workdayBands('01:30–06:00', start, end, timeZone), []);
});
test('real recorded Batesburg fixture renders actual capture time and warning interval', () => {
  const fixture = JSON.parse(readFileSync(new URL('../go-prototype/internal/weatherdesk/fixtures/batesburg.json', import.meta.url)));
  const a = fixture.feature.properties;
  const m = alertTimelineModel([{...a, issued:a.sent, source:fixture.feature.id, geography:'applicable', applicability:'applicable'}], {now:fixture.capturedAt, mode:'REPLAY', timeZone});
  assert.equal(m.lanes[0].event, 'Flash Flood Warning');
  assert.match(timelineClock(m.now, timeZone), /10:20 PM EDT/);
  assert.match(timelineClock(m.lanes[0].end, timeZone), /11 PM EDT/);
  assert.equal(m.lanes[0].inactive, false); assert.equal(m.lanes[0].endKind, 'Expected end');
});
