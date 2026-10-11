'use client';
import { useEffect, useId, useMemo, useRef, useState } from 'react';
import { SourceLink } from './ui';
import { SEVERITY, alertTimelineModel, timelineClock } from '../lib/alert-timeline.mjs';
import styles from './alert-timeline.module.css';

export function SeverityMeter({ level, label }) {
  return <span className={styles.meter} data-level={level} aria-label={`Severity ${label || 'unknown'}, ${level} of 4`}>
    {SEVERITY.map((_, i) => <i key={i} className={i < level ? styles.on : ''} />)}
  </span>;
}

export function AlertTimeline({ alerts = [], now, mode = 'LIVE', timeZone, operatingHours = '', title = 'Alert timeline', initialWidth = 640 }) {
  const model = useMemo(() => alertTimelineModel(alerts, { now, mode, timeZone, operatingHours }), [alerts, now, mode, timeZone, operatingHours]);
  const [hover, setHover] = useState(null);
  // Draw at the container's real width so labels keep their size in a narrow sidebar.
  const plot = useRef(null);
  const [W, setW] = useState(initialWidth);
  useEffect(() => {
    if (!plot.current || typeof ResizeObserver === 'undefined') return;
    const ro = new ResizeObserver(([entry]) => setW(Math.max(200, Math.round(entry.contentRect.width))));
    ro.observe(plot.current);
    return () => ro.disconnect();
  }, []);
  const hatch = `hatch-${useId().replace(/:/g, '')}`;
  if (!model.lanes.length) return <section className={styles.timeline}><h3 className={styles.title}>{title}</h3><p className={styles.empty}>No local alerts in this evidence.</p></section>;

  const L = 8, R = 8, laneH = 54, top = 26, x = t => L + model.position(t) * (W - L - R);
  const H = top + model.lanes.length * laneH + 8;
  const nowX = model.now != null && model.now >= model.start && model.now <= model.end ? x(model.now) : null;

  return <section className={styles.timeline} aria-labelledby={`${hatch}-t`}>
    <div className={styles.head}>
      <h3 id={`${hatch}-t`} className={styles.title}>{title}</h3>
      <span className={styles.tz}>{model.timeZoneKnown ? `Times in ${model.timeZone}` : 'Times in UTC · facility time zone unknown'}</span>
    </div>
    <div className={styles.plot} ref={plot}>
      <svg viewBox={`0 0 ${W} ${H}`} width="100%" role="img" aria-label={`Timeline of ${model.lanes.length} alerts`}>
        {model.bands.map((b, i) => <rect key={i} x={x(b.start)} y={top - 6} width={x(b.end) - x(b.start)} height={H - top} className={styles.band} />)}
        {model.ticks.filter((_, i) => W >= 420 || i % 2 === 0).map(t => {
          const tx = x(t), anchor = tx < 24 ? 'start' : tx > W - 24 ? 'end' : 'middle';
          const hidden = nowX != null && Math.abs(tx - nowX) < (mode === 'REPLAY' ? 95 : 45);
          return <g key={t}>
            <line x1={tx} x2={tx} y1={top - 6} y2={H} className={styles.grid} />
            {!hidden && <text x={tx} y={12} className={styles.tick} textAnchor={anchor}>{timelineClock(t, model.timeZone)}</text>}
          </g>;
        })}
        {model.lanes.map((l, i) => {
          const y = top + i * laneH, x1 = l.start == null ? L : x(l.start), x2 = l.end == null ? x1 : x(l.end);
          const label = l.event.length > Math.floor((W - 16) / 7) ? `${l.event.slice(0, Math.floor((W - 16) / 7) - 1)}…` : l.event;
          return <g key={l.id} className={styles.lane} data-level={l.level} data-inactive={l.inactive || undefined}
            onMouseEnter={() => setHover(l.id)} onMouseLeave={() => setHover(null)} onFocus={() => setHover(l.id)} onBlur={() => setHover(null)} tabIndex={0}>
            {/* Pattern lives inside the lane so currentColor resolves to this lane's severity color. */}
            <defs><pattern id={`${hatch}-${i}`} width="7" height="7" patternUnits="userSpaceOnUse" patternTransform="rotate(45)">
              <rect width="7" height="7" fill="currentColor" opacity=".18" />
              <line x1="0" y1="0" x2="0" y2="7" stroke="currentColor" strokeWidth="2.5" opacity=".75" />
            </pattern></defs>
            <rect x={0} y={y} width={W} height={laneH} fill="transparent" />
            <text x={Math.min(Math.max(x1, L), Math.max(L, W - 220))} y={y + 14} className={styles.event}>{label}</text>
            {l.start != null && !l.invalidRange && <rect x={x1} y={y + 22} width={Math.max(4, x2 - x1)} height={16} rx={4}
              className={styles.bar} data-fill={l.fill}
              fill={l.fill === 'striped' ? `url(#${hatch}-${i})` : l.fill === 'solid' ? 'currentColor' : 'none'} />}
            {(l.openEnded || l.start == null || l.invalidRange) && <text x={Math.min(x1 + 10, W - 125)} y={y + 35} className={styles.open}>{l.invalidRange ? 'Timing inconsistent' : l.start == null ? 'Start unknown' : 'End unknown'}</text>}
            {l.issued != null && l.start != null && l.issued < l.start && <g>
              <line x1={x(l.issued)} x2={x1} y1={y + 30} y2={y + 30} className={styles.lead} />
              <circle cx={x(l.issued)} cy={y + 30} r={3.5} className={styles.issued} />
            </g>}
          </g>;
        })}
        {nowX != null && <g>
          <line x1={nowX} x2={nowX} y1={top - 8} y2={H} className={styles.now} />
          <text x={nowX > W - 105 ? nowX - 4 : nowX + 4} textAnchor={nowX > W - 105 ? 'end' : 'start'} y={top - 10} className={styles.nowLabel}>{model.clockLabel}</text>
        </g>}
      </svg>
    </div>
    <ul className={styles.rows}>
      {model.lanes.map(l => <li key={l.id} data-active={hover === l.id || undefined} data-inactive={l.inactive || undefined}>
        <SeverityMeter level={l.level} label={l.severity} />
        <span className={styles.rowMain}><b>{l.event}</b><span>{l.summary}{l.status ? ` · ${l.status}` : ''}{l.unknownPlace ? ' · Coverage unknown' : ''}</span></span>
        <span className={styles.rowTime}>{l.startKind}{l.start != null ? ` ${timelineClock(l.start, model.timeZone, true)}` : ''}<br />{l.endKind}{l.end != null ? ` ${timelineClock(l.end, model.timeZone, true)}` : ''}{l.invalidRange ? ' · timing inconsistent' : ''}</span>
        {l.source && <SourceLink href={l.source}>NWS</SourceLink>}
      </li>)}
    </ul>
    <div className={styles.legend}>
      <span><i className={styles.kSolid} />Likely or observed</span>
      <span><i className={styles.kStriped} />Possible</span>
      <span><i className={styles.kOutline} />Unlikely or unknown</span>
      <span><i className={styles.kIssued} />Issued</span>
      {model.bands.length > 0 && <span><i className={styles.kBand} />Example workday</span>}
      <span><i className={styles.kNow} />{model.clockLabel}</span>
    </div>
  </section>;
}
