import test from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { listedPlaces, parseBulletin } from '../frontend/lib/nws-bulletin.mjs';

const southeast = JSON.parse(readFileSync(new URL('../fixtures/isaias-2026-10-10/alerts-southeast-2026-10-11T0220Z.json', import.meta.url)));
const barnwell = JSON.parse(readFileSync(new URL('../fixtures/isaias-2026-10-10/alerts-point-barnwell-2026-10-11T0220Z.json', import.meta.url)));
const ffw = southeast.features.find(f => f.properties.event === 'Flash Flood Warning' && f.properties.areaDesc.includes('Aiken')).properties;
const watch = barnwell.features.find(f => f.properties.event === 'Flood Watch').properties;

test('warning-style text: product id, area list, observation, tags, locations', () => {
  const b = parseBulletin(ffw.description);
  assert.equal(b.productId, 'FFWCAE');
  const area = b.sections.find(s => s.kind === 'list');
  assert.equal(area.heading, 'Flash Flood Warning for');
  assert.equal(area.items.length, 4);
  assert.match(b.sections.find(s => s.bullet && /law enforcement/.test(s.text)).text, /^At 945 PM EDT, local law enforcement reported/);
  assert.equal(b.tags.source, 'Law enforcement reported.');
  assert.match(b.tags.impact, /^Flash flooding of small creeks/);
  const loc = b.sections.find(s => s.kind === 'locations');
  assert.ok(loc.places.includes('Batesburg-Leesville'));
  assert.ok(loc.places.includes('Shull Island'));
  assert.match(loc.note, /Interstate 20/);
});

test('WHAT/WHERE/WHEN/IMPACTS sections keep labels and sub-items', () => {
  const b = parseBulletin(watch.description);
  assert.deepEqual(b.sections.map(s => s.label), ['What', 'Where', 'When', 'Impacts', 'Additional details']);
  assert.equal(b.sections.at(-1).items.length, 2);
  assert.match(b.sections[0].text, /^Flash flooding caused by excessive rainfall continues to be possible\.$/);
});

test('no words are invented: every parsed string appears in the original after unwrapping', () => {
  for (const p of [ffw, watch]) {
    const flat = p.description.replace(/\s+/g, ' ');
    const b = parseBulletin(p.description);
    const strings = b.sections.flatMap(s => [s.text, s.heading, s.note, ...(s.items || []), ...(s.places || [])]).concat(Object.values(b.tags)).filter(Boolean);
    for (const s of strings) assert.ok(flat.includes(s), s);
  }
});

test('facility place match is exact', () => {
  const b = parseBulletin(ffw.description);
  assert.deepEqual(listedPlaces(b, ['batesburg-leesville']), ['Batesburg-Leesville']);
  assert.deepEqual(listedPlaces(b, ['Batesburg']), []);
  assert.deepEqual(parseBulletin('').sections, []);
});
