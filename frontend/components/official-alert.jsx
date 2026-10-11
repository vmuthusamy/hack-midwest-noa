import { MapPin, ShieldAlert } from 'lucide-react';
import { SourceLink } from './ui';
import { SeverityMeter } from './alert-timeline';
import { severityLevel, timelineClock } from '../lib/alert-timeline.mjs';
import { listedPlaces, parseBulletin } from '../lib/nws-bulletin.mjs';
import styles from './official-alert.module.css';

// The official NWS alert, rendered in-app. Text is NWS's own, regrouped by
// parseBulletin; nothing here is summarized or generated.
export function OfficialAlert({ alert, timeZone, placeNames = [] }) {
  if (!alert) return null;
  const b = parseBulletin(alert.description);
  const matches = listedPlaces(b, placeNames);
  const level = severityLevel(alert.severity);
  const rest = b.sections;
  const tags = [['Hazard', b.tags.hazard], ['Source', b.tags.source], ['Impact', b.tags.impact]].filter(([, v]) => v);

  const steps = (alert.instruction || '').split(/\n\s*\n/).map(t => t.replace(/\s+/g, ' ').trim()).filter(Boolean);

  // At a glance: what, until when, how bad, does it name us, what to do. Everything else is one click away.
  return <article className={styles.card} data-level={level} aria-labelledby={`alert-${alert.id}`}>
    <header className={styles.head}>
      <div className={styles.titleRow}>
        <ShieldAlert size={20} aria-hidden="true" />
        <h3 id={`alert-${alert.id}`}>{alert.event}</h3>
        <SeverityMeter level={level} label={alert.severity} /><span className={styles.sev}>{alert.severity || 'Unknown'}</span>
      </div>
      <p className={styles.when}>Message expires {timelineClock(alert.expires, timeZone, true)}</p>
    </header>

    {matches.length > 0 && <p className={styles.match}><MapPin size={16} aria-hidden="true" /><b>{matches.join(', ')}</b> named</p>}

    <p className={`official-instruction ${styles.do}`}>{steps[0] || 'No instructions in this message. Open the source.'}</p>

    <details className={styles.bulletin}>
      <summary>Full NWS text</summary>
      {alert.ends && <p>Expected hazard end {timelineClock(alert.ends, timeZone, true)}</p>}
      <p className={styles.meta}>{[alert.certainty, alert.urgency, b.tags.source?.replace(/\.$/, ''), alert.messageType === 'Update' && 'Updated', `Issued ${timelineClock(alert.issued, timeZone, true)} by ${alert.issuer || 'NWS'}`].filter(Boolean).join(' · ')}</p>
      {steps.slice(1).map((t, i) => <p key={i} className="official-instruction">{t}</p>)}
      {tags.filter(([k]) => k !== 'Source').map(([k, v]) => <p key={k}><b>{k}:</b> {v}</p>)}
      {rest.map((s, i) => {
        if (s.kind === 'text') return <p key={i}>{s.text}</p>;
        if (s.kind === 'labeled') return <section key={i} className={styles.section}><h4>{s.label}</h4>{s.items ? <ul>{s.items.map((t, j) => <li key={j}>{t}</li>)}</ul> : <p>{s.text}</p>}</section>;
        if (s.kind === 'locations') return <section key={i} className={styles.section}><h4>{s.heading}</h4><ul className={styles.places}>{s.places.map(p => <li key={p} data-match={matches.includes(p) || undefined}>{p}</li>)}</ul>{s.note && <p className={styles.note}>{s.note}</p>}</section>;
        if (s.kind === 'list') return <section key={i} className={styles.section}><h4>{s.heading}</h4><ul>{s.items.map((t, j) => <li key={j}>{t}</li>)}</ul></section>;
        return null;
      })}
      {alert.source && <p className={styles.foot}><SourceLink href={alert.source}>NWS source</SourceLink>{b.productId ? ` · ${b.productId}` : ''}</p>}
    </details>
  </article>;
}
