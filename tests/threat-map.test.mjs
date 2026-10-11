import test from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { extentProjection, polygonGeometry, polygonRelation, threatMapData } from '../frontend/lib/threat-map-geometry.mjs';
const rectangle=(west,south,east,north)=>[[west,south],[east,south],[east,north],[west,north],[west,south]];
const square={type:'Polygon',coordinates:[rectangle(-82,33,-81,34)]};
test('inside, boundary, near and outside are display relationships, not applicability',()=>{
  assert.deepEqual(polygonRelation(square,[-81.5,33.5]),{label:'Inside',distanceMiles:0});
  assert.equal(polygonRelation(square,[-81,33.5]).label,'Inside');
  const near=polygonRelation(square,[-80.9,33.5]);
  assert.equal(near.label,'Near');
  assert.ok(near.distanceMiles>5&&near.distanceMiles<6);
  assert.equal(polygonRelation(square,[-80.8,33.5]).label,'Outside');
  assert.equal(Object.hasOwn(near,'applicable'),false);
});
test('polygon holes and separated MultiPolygons preserve their real geography',()=>{
  const holes={type:'Polygon',coordinates:[rectangle(-82,33,-81,34),rectangle(-81.7,33.3,-81.3,33.7)]};
  assert.equal(polygonRelation(holes,[-81.5,33.5]).label,'Outside');
  assert.equal(polygonRelation(holes,[-81.8,33.5]).label,'Inside');
  const multi={type:'MultiPolygon',coordinates:[square.coordinates,[rectangle(-80,33,-79,34)]]};
  assert.equal(polygonRelation(multi,[-79.5,33.5]).label,'Inside');
  assert.equal(polygonRelation(multi,[-80.5,33.5]).label,'Outside');
});
test('unknown or malformed geometry never becomes a warning polygon',()=>{
  for (const geometry of [null,{type:'Point',coordinates:[-81,33]},{type:'Polygon',coordinates:[]},{type:'Polygon',coordinates:[[[0,0],[1,0],[1,1],[0,1]]]},{type:'Polygon',coordinates:[rectangle(179,33,-179,34)]},{type:'Polygon',coordinates:[rectangle(-82,90,-81,91)]}]) {
    assert.equal(polygonGeometry(geometry),null);
    assert.deepEqual(polygonRelation(geometry,[-81,33]),{label:'Unknown',distanceMiles:null});
  }
  const zone=threatMapData([{event:'Winter Storm Warning',geography:'applicable',geometry:null}],{latitude:47.5,longitude:-111.3});
  assert.equal(zone.zoneMatch,true);
  assert.equal(zone.polygons.length,0);
  assert.equal(zone.relation.label,'Unknown');
});
test('map aggregates warnings, ignores invented gauge coordinates and bounds actual points',()=>{
  const alerts=[{id:'far',event:'Flood Warning',geometry:square,geography:'outside'},{id:'inside',event:'Tornado Warning',geometry:{type:'Polygon',coordinates:[rectangle(-80,33,-79,34)]},geography:'applicable'}];
  const data=threatMapData(alerts,{longitude:-79.5,latitude:33.5},[{gaugeId:'missing',longitude:null,latitude:null},{gaugeId:'known',longitude:-81.2,latitude:33.4}]);
  assert.equal(data.relation.label,'Inside');
  assert.equal(data.gauges.length,1);
  assert.equal(data.gauges[0].properties.gaugeId,'known');
  const project=extentProjection(data.bounds);
  for (const point of [[-79.5,33.5],[-81.2,33.4],[-82,33],[-79,34]]) {
    const [x,y]=project(point);
    assert.ok(x>=0&&x<=640&&y>=0&&y<=330);
  }
  assert.equal(extentProjection(null),null);
});
test('saved official warning places Batesburg inside and Barnwell outside',()=>{
  const captured=JSON.parse(readFileSync(new URL('../fixtures/isaias-2026-10-10/alerts-southeast-2026-10-11T0220Z.json',import.meta.url),'utf8'));
  const warning=captured.features.find(a=>a.properties.id==='urn:oid:2.49.0.1.840.0.be5613bdb2f88e91a29b7e41456b681f8ab18db1.001.1');
  assert.ok(warning,'actual recorded warning ID exists');
  assert.equal(polygonRelation(warning.geometry,[-81.54,33.91]).label,'Inside');
  assert.equal(polygonRelation(warning.geometry,[-81.35,33.28]).label,'Outside');
  const barnwell=threatMapData([{...warning.properties,geometry:warning.geometry},{event:'Flood Watch',geography:'applicable'}],{longitude:-81.35,latitude:33.28});
  assert.equal(barnwell.hasWatch,true);
  assert.equal(barnwell.relation.label,'Outside');
});
