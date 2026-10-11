'use client';
import { useEffect, useId, useMemo, useRef, useState } from 'react';
import 'maplibre-gl/dist/maplibre-gl.css';
import { extentProjection, threatMapData } from '../lib/threat-map-geometry.mjs';
import styles from './threat-map.module.css';

// MapLibre examples: https://maplibre.org/maplibre-gl-js/docs/examples/add-a-geojson-polygon/
// Public basemap is optional. The official evidence and SVG extent never need it.
export function ThreatMap({ alerts=[], facility={}, signals=[], mode='REPLAY', capturedAt=null }) {
  const host=useRef(null);
  const [basemap,setBasemap]=useState('loading');
  // Parent clocks may rerender each second. Rebuild only when plotted facts change.
  const plottedFacts=JSON.stringify({alerts:alerts.map(a=>({id:a.id,event:a.event,geometry:a.geometry,geography:a.geography})),facility:{longitude:facility.longitude,latitude:facility.latitude},signals:signals.map(s=>({longitude:s.longitude,latitude:s.latitude,name:s.name,gaugeId:s.gaugeId,usgsId:s.usgsId}))});
  const data=useMemo(()=>threatMapData(alerts,facility,signals),[plottedFacts]);
  const projection=useMemo(()=>extentProjection(data.bounds),[data.bounds]);
  const gridId=`threat-grid-${useId().replace(/:/g,'')}`;
  useEffect(()=>{
    let map, observer, cancelled=false, failed=false;
    setBasemap('loading');
    const fallback=()=>{
      if (cancelled || failed) return;
      failed=true;
      setBasemap('unavailable');
      clearTimeout(timer);
      observer?.disconnect();
      map?.remove();
      map=null;
    };
    const timer=setTimeout(fallback,8000);
    if (!data.point || !data.bounds) { fallback(); return()=>{cancelled=true;clearTimeout(timer);}; }
    import('maplibre-gl').then(maplibre=>{
      if (cancelled || failed) return;
      const gl=maplibre.default || maplibre;
      // Next emits a hashed, same-origin worker asset in webpack and Turbopack.
      // https://maplibre.org/maplibre-gl-js/docs/#esm
      gl.setWorkerUrl(new URL('maplibre-gl/dist/maplibre-gl-worker.mjs', import.meta.url).toString());
      map=new gl.Map({container:host.current,style:'https://demotiles.maplibre.org/style.json',center:data.point,zoom:7,attributionControl:true,renderWorldCopies:false});
      map.scrollZoom.disable();
      map.dragRotate.disable();
      map.touchZoomRotate.disableRotation();
      map.addControl(new gl.NavigationControl({showCompass:false}),'top-right');
      map.addControl(new gl.ScaleControl({unit:'imperial'}),'bottom-left');
      map.on('error',fallback);
      map.once('load',()=>{
        if (cancelled || !map) return;
        try {
          map.addSource('official-alerts',{type:'geojson',data:{type:'FeatureCollection',features:data.polygons}});
          map.addLayer({id:'warning-fill',type:'fill',source:'official-alerts',paint:{'fill-color':['case',['get','isWarning'],'#e47b45','#f1be54'],'fill-opacity':.24}});
          map.addLayer({id:'warning-line',type:'line',source:'official-alerts',paint:{'line-color':['case',['get','isWarning'],'#bf5e2c','#b28a34'],'line-width':2.5}});
          const points=[{type:'Feature',properties:{role:'facility'},geometry:{type:'Point',coordinates:data.point}},...data.gauges.map(g=>({...g,properties:{...g.properties,role:'gauge'}}))];
          map.addSource('locations',{type:'geojson',data:{type:'FeatureCollection',features:points}});
          map.addLayer({id:'location-pins',type:'circle',source:'locations',paint:{'circle-color':['case',['==',['get','role'],'facility'],'#127e68','#245ba8'],'circle-radius':['case',['==',['get','role'],'facility'],8,6],'circle-stroke-color':'#ffffff','circle-stroke-width':2}});
          map.fitBounds(data.bounds,{padding:42,maxZoom:11,duration:0});
          map.getCanvas().setAttribute('aria-label','Official alert geography, fictional facility and regional gauges.');
          observer=new ResizeObserver(()=>map?.resize());
          observer.observe(host.current);
          clearTimeout(timer);
          setBasemap('ready');
        } catch { fallback(); }
      });
    }).catch(fallback);
    return()=>{cancelled=true;clearTimeout(timer);observer?.disconnect();map?.remove();};
  },[data]);
  const warningLabel=data.relation.label==='Unknown' ? data.zoneMatch?'Zone match':'Not mapped' : data.relation.label;
  const capture=typeof capturedAt==='string'&&Number.isFinite(Date.parse(capturedAt))?new Date(capturedAt).toISOString().replace('T',' ').replace('.000Z',' UTC'):null;
  const pin=point=>projection?.(point)||[0,0];
  return <section className={styles.map} aria-label="Threat map">
    <header className={styles.header}><div><p>{mode==='REPLAY'?'Recorded geography':'Live alert geography'}</p><h3>Threat map</h3></div><strong className={styles.status} data-relation={warningLabel}>{warningLabel}{data.hasWatch&&warningLabel!=='Inside'?' / Watch':''}</strong></header>
    <div className={styles.viewport}>
      <div ref={host} className={styles.basemap} style={{visibility:basemap==='ready'?'visible':'hidden'}}/>
      {basemap!=='ready' && <svg className={styles.extent} viewBox="0 0 640 330" role="img" aria-label="Official alert polygon, fictional facility and known regional gauge coordinates; extent diagram without a basemap.">
        <defs><pattern id={gridId} width="40" height="40" patternUnits="userSpaceOnUse"><path d="M40 0H0V40" fill="none" stroke="currentColor" strokeWidth=".6"/></pattern></defs>
        <rect width="640" height="330" fill={`url(#${gridId})`} opacity=".2"/>
        {data.polygons.map((polygon,i)=><path key={i} d={polygon.geometry.coordinates.map(p=>p.map(ring=>ring.map((point,j)=>`${j?'L':'M'}${pin(point).join(',')}`).join(' ')+'Z').join(' ')).join(' ')} fill={polygon.properties.isWarning?'#e47b45':'#f1be54'} fillOpacity=".22" stroke={polygon.properties.isWarning?'#d8733c':'#ba9036'} strokeWidth="2" fillRule="evenodd"><title>{polygon.properties.event} · official polygon</title></path>)}
        {data.gauges.map((g,i)=>{const [x,y]=pin(g.geometry.coordinates);return <g key={i}><rect x={x-6} y={y-6} width="12" height="12" fill="#4487d9" stroke="white" strokeWidth="2"/><title>{g.properties.name} · regional context only</title><text x={x+12} y={y+4} className={styles.label}>Gauge {i+1}</text></g>;})}
        {data.point&&<g><circle cx={pin(data.point)[0]} cy={pin(data.point)[1]} r="14" fill="#39b49a" fillOpacity=".16"/><circle cx={pin(data.point)[0]} cy={pin(data.point)[1]} r="7" fill="#127e68" stroke="white" strokeWidth="2"/><text x={pin(data.point)[0]+13} y={pin(data.point)[1]-12} className={styles.label}>Facility</text></g>}
        <text x="604" y="28" className={styles.label}>N ↑</text>
      </svg>}
      {!data.point&&<p className={styles.empty}>Facility coordinates unavailable.</p>}
      <span className={styles.mapState}>{basemap==='ready'?'MapLibre basemap':basemap==='loading'?'Loading basemap · extent shown':'Basemap unavailable · extent shown'}</span>
    </div>
    <div className={styles.legend}><span><i className={styles.warning}/>Official alert polygon</span><span><i className={styles.facility}/>Fictional facility</span>{data.gauges.length>0&&<span><i className={styles.gauge}/>Regional gauge</span>}</div>
    {warningLabel==='Near'&&<p className={styles.note}>About {data.relation.distanceMiles.toFixed(1)} mi from the warning boundary. Near does not mean applicable.</p>}
    {warningLabel==='Outside'&&<p className={styles.note}>Facility is outside the displayed warning polygon{data.hasWatch?'; the applicable watch is separate.':'.'}</p>}
    {data.zoneMatch&&!data.polygons.length&&<p className={styles.note}>Official zone membership is available; no warning polygon was supplied.</p>}
    {data.gauges.map((g,i)=><p className={styles.gaugeNote} key={i}>Gauge {i+1}: {g.properties.name} · regional context; facility impact unknown.</p>)}
    {!data.gauges.length&&<p className={styles.gaugeNote}>No gauge coordinates supplied for this case.</p>}
    <footer className={styles.footer}>{capture?`${mode==='REPLAY'?'Recorded':'Refreshed'} ${capture}`:'Source capture time unavailable'}<span>Geography display · operating decisions use server checks</span></footer>
  </section>;
}
