// Display-only geometry. Server applicability and policy decisions remain authoritative.
const EARTH_MILES = 3958.7613;
const rad = degrees => degrees * Math.PI / 180;
export function validCoordinate(point) {
  return Array.isArray(point) && point.length >= 2 && Number.isFinite(point[0]) && Number.isFinite(point[1]) && Math.abs(point[0]) <= 180 && Math.abs(point[1]) <= 85;
}
export function facilityPoint(facility) {
  const point = [facility?.longitude, facility?.latitude];
  return validCoordinate(point) ? point : null;
}
export function polygonGeometry(geometry) {
  const polygons = geometry?.type === 'Polygon' ? [geometry.coordinates] : geometry?.type === 'MultiPolygon' ? geometry.coordinates : null;
  if (!Array.isArray(polygons) || !polygons.length) return null;
  let count = 0;
  for (const polygon of polygons) {
    if (!Array.isArray(polygon) || !polygon.length) return null;
    for (const ring of polygon) {
      if (!Array.isArray(ring) || ring.length < 4 || !ring.every(validCoordinate)) return null;
      count += ring.length;
      if (count > 10000 || ring[0][0] !== ring.at(-1)[0] || ring[0][1] !== ring.at(-1)[1]) return null;
      if (Math.max(...ring.map(p=>p[0]))-Math.min(...ring.map(p=>p[0])) > 180) return null;
    }
  }
  return { type: 'MultiPolygon', coordinates: polygons };
}
function onSegment(point, a, b) {
  const cross = (point[0]-a[0])*(b[1]-a[1])-(point[1]-a[1])*(b[0]-a[0]);
  return Math.abs(cross) <= 1e-10 && point[0] >= Math.min(a[0],b[0])-1e-10 && point[0] <= Math.max(a[0],b[0])+1e-10 && point[1] >= Math.min(a[1],b[1])-1e-10 && point[1] <= Math.max(a[1],b[1])+1e-10;
}
function inRing(point, ring) {
  let inside = false;
  for (let i=0, j=ring.length-1; i<ring.length; j=i++) {
    const a=ring[i], b=ring[j];
    if (onSegment(point,a,b)) return true;
    if ((a[1]>point[1]) !== (b[1]>point[1]) && point[0] < (b[0]-a[0])*(point[1]-a[1])/(b[1]-a[1])+a[0]) inside=!inside;
  }
  return inside;
}
function angularDistance(a,b) {
  const dlat=rad(b[1]-a[1]), dlon=rad(b[0]-a[0]);
  const h=Math.sin(dlat/2)**2+Math.cos(rad(a[1]))*Math.cos(rad(b[1]))*Math.sin(dlon/2)**2;
  return 2*Math.asin(Math.sqrt(Math.max(0,Math.min(1,h))));
}
function bearing(a,b) {
  const lat1=rad(a[1]), lat2=rad(b[1]), dlon=rad(b[0]-a[0]);
  return Math.atan2(Math.sin(dlon)*Math.cos(lat2), Math.cos(lat1)*Math.sin(lat2)-Math.sin(lat1)*Math.cos(lat2)*Math.cos(dlon));
}
function segmentMiles(point,a,b) {
  const segment=angularDistance(a,b), distance=angularDistance(a,point);
  if (!segment) return distance*EARTH_MILES;
  const difference=bearing(a,point)-bearing(a,b);
  const along=Math.atan2(Math.sin(distance)*Math.cos(difference),Math.cos(distance));
  if (along<0 || along>segment) return Math.min(distance,angularDistance(b,point))*EARTH_MILES;
  return Math.abs(Math.asin(Math.max(-1,Math.min(1,Math.sin(distance)*Math.sin(difference)))))*EARTH_MILES;
}
export function polygonRelation(geometry, point) {
  const normalized=polygonGeometry(geometry);
  if (!normalized || !validCoordinate(point)) return { label:'Unknown', distanceMiles:null };
  if (normalized.coordinates.some(polygon=>inRing(point,polygon[0]) && !polygon.slice(1).some(ring=>inRing(point,ring)))) return { label:'Inside', distanceMiles:0 };
  let nearest=Infinity;
  for (const polygon of normalized.coordinates) for (const ring of polygon) for (let i=1;i<ring.length;i++) nearest=Math.min(nearest,segmentMiles(point,ring[i-1],ring[i]));
  return { label:nearest<=10?'Near':'Outside', distanceMiles:nearest };
}
export function threatMapData(alerts=[], facility={}, signals=[]) {
  const point=facilityPoint(facility);
  let vertexBudget=10000;
  const polygons=alerts.slice(0,64).flatMap(alert=>{
    const geometry=polygonGeometry(alert?.geometry);
    const count=geometry?.coordinates.flat(2).length||0;
    if (!geometry || count>vertexBudget) return [];
    vertexBudget-=count;
    return [{ type:'Feature', properties:{id:alert.id||'',event:alert.event||'Official alert',isWarning:/warning/i.test(alert.event||'')}, geometry }];
  });
  const gauges=signals.slice(0,32).flatMap(signal=>{
    const coordinates=[signal?.longitude,signal?.latitude];
    return validCoordinate(coordinates)?[{ type:'Feature', properties:{name:signal.name||signal.gaugeId||'Regional gauge',gaugeId:signal.gaugeId||signal.usgsId||''}, geometry:{type:'Point',coordinates} }]:[];
  });
  const relations=polygons.filter(p=>p.properties.isWarning).map(p=>polygonRelation(p.geometry,point));
  const relation=relations.find(r=>r.label==='Inside') || relations.sort((a,b)=>(a.distanceMiles??Infinity)-(b.distanceMiles??Infinity))[0] || {label:'Unknown',distanceMiles:null};
  const zoneMatch=alerts.some(a=>a.geography==='applicable' && !polygonGeometry(a.geometry));
  const hasWatch=alerts.some(a=>/watch/i.test(a.event||'') && a.geography!=='outside');
  const positions=[...polygons.flatMap(p=>p.geometry.coordinates.flat(2)), ...gauges.map(g=>g.geometry.coordinates), ...(point?[point]:[])];
  let bounds=null;
  if (positions.length) {
    const west=Math.min(...positions.map(p=>p[0])),east=Math.max(...positions.map(p=>p[0]));
    const south=Math.min(...positions.map(p=>p[1])),north=Math.max(...positions.map(p=>p[1]));
    const padX=Math.max((east-west)*.12,.025),padY=Math.max((north-south)*.12,.025);
    bounds=[[Math.max(-180,west-padX),Math.max(-85,south-padY)],[Math.min(180,east+padX),Math.min(85,north+padY)]];
  }
  return { point,polygons,gauges,bounds,relation,zoneMatch,hasWatch };
}
export function extentProjection(bounds,width=640,height=330) {
  if (!bounds) return null;
  const mercator=latitude=>Math.log(Math.tan(Math.PI/4+rad(latitude)/2));
  const [[west,south],[east,north]]=bounds, dx=rad(east-west),dy=mercator(north)-mercator(south);
  const scale=Math.min((width-72)/dx,(height-72)/dy);
  const left=(width-dx*scale)/2,top=(height-dy*scale)/2;
  return point=>[left+rad(point[0]-west)*scale,top+(mercator(north)-mercator(point[1]))*scale];
}
