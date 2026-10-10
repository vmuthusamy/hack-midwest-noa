"use client";
import "./gallery.css";

import { useState } from "react";
import { ArrowRight, ShieldCheck, CloudRain, MapPin, History, Users, Radio, Sun, Moon, ExternalLink, FileCheck2 } from "lucide-react";

// This gallery does not fetch evidence or approve decisions. Its state is visual only.
const periods = [
  { time: "5 PM", probability: 56, temperature: 79, wind: "10 mph S" },
  { time: "6 PM", probability: 49, temperature: 79, wind: "10 mph SSW" },
  { time: "7 PM", probability: 42, temperature: 78, wind: "5 mph S" },
  { time: "8 PM", probability: 62, temperature: 78, wind: "5 mph S" },
];
const sourceLinks = [
  ["NWS geography", "https://www.weather.gov/media/documentation/docs/NWS_Geolocation.pdf"],
  ["NWS alert meanings", "https://www.weather.gov/safety/tornado-ww"],
  ["USWDS alerts", "https://designsystem.digital.gov/components/alert/"],
  ["Accessible charts", "https://designsystem.digital.gov/components/data-visualizations/"],
  ["WAI alerts", "https://www.w3.org/WAI/ARIA/apg/patterns/alert/"],
  ["Grafana history", "https://grafana.com/docs/grafana/latest/alerting/monitor-status/view-alert-state-history/"],
  ["OPA integration", "https://www.openpolicyagent.org/docs/integration"],
];

function Badge({ children, tone = "neutral" }) {
  return <span className={`wg-badge wg-${tone}`}>{children}</span>;
}

function Pattern({ number, title, subtitle, icon: Icon, label, why, data, variants, notes, children }) {
  return <article className="wg-card">
    <header className="wg-card-header">
      <span className="wg-icon"><Icon size={20} aria-hidden="true" /></span>
      <div><p className="wg-kicker">PATTERN {number}</p><h2>{title}</h2></div>
    </header>
    <p className="wg-subtitle">{subtitle}</p>
    <div className="wg-preview"><p className="wg-example-label">{label}</p>{children}</div>
    {notes && <div className="wg-notes">
      <p><strong>Why it helps</strong> {why}</p>
      <p><strong>Needs</strong> {data}</p>
      <p><strong>Variants</strong> {variants}</p>
    </div>}
  </article>;
}

export default function Gallery() {
  const [theme, setTheme] = useState("dark");
  const [notes, setNotes] = useState(true);
  const [geoVariant, setGeoVariant] = useState("polygon");
  return <main className="wg-root" data-theme={theme}>
    <div className="wg-shell">
      <header className="wg-hero">
        <div>
          <p className="wg-kicker">WEATHER DESK / DESIGN INVESTIGATION</p>
          <h1>Evidence first.<br /><span>Decisions that stay explainable.</span></h1>
          <p className="wg-intro">A pattern library for a facility manager who needs to know what changed, who is affected, and what to review.</p>
          <div className="wg-hero-links"><a href="/">← Prototype</a><a href="/boundaries">View the problem boundaries <ArrowRight size={14} aria-hidden="true" /></a></div>
        </div>
        <div className="wg-controls">
          <div className="wg-segment" aria-label="Gallery appearance">
            <button aria-pressed={theme === "dark"} onClick={() => setTheme("dark")}><Moon size={14} aria-hidden="true" />Dark</button>
            <button aria-pressed={theme === "light"} onClick={() => setTheme("light")}><Sun size={14} aria-hidden="true" />Light</button>
          </div>
          <button className="wg-note-toggle" aria-pressed={notes} onClick={() => setNotes(!notes)}>{notes ? "Hide" : "Show"} research notes</button>
        </div>
      </header>

      <div className="wg-disclaimer"><Radio size={17} aria-hidden="true" /><p><strong>Design gallery — no live warnings or sending.</strong> Operational examples are synthetic REPLAY. The forecast card uses a separately labeled recorded NOAA snapshot. Research is public desk research, with usability hypotheses still to test.</p></div>

      <section className="wg-grid" aria-label="Reusable alert and decision patterns">
        <Pattern number="01" title="Official warning first" subtitle="Authority is visible before analysis or approval." icon={Radio} label="SYNTHETIC REPLAY · PLACEHOLDER CONTENT" notes={notes} why="An urgent instruction deserves the most visible space. Separate the official alert from any operating recommendation." data="Alert event + exact text, issuer, source URL, issue/expiry, last successful refresh." variants="Expanded source panel for urgent alerts; slim informational header for non-urgent evidence.">
          <div className="wg-warning">
            <div className="wg-row"><Badge tone="danger">Warning example</Badge><span className="wg-small">Fixture T+5</span></div>
            <h3>Tornado Warning</h3>
            <p>In the app, show the alert’s original official instructions here, immediately and without rewriting.</p>
            <div className="wg-source-row"><span>NWS source · original alert ID</span><ExternalLink size={13} aria-hidden="true" /></div>
          </div>
          <div className="wg-inline-note"><ShieldCheck size={14} aria-hidden="true" />No AI or manager approval before official instructions appear.</div>
        </Pattern>

        <Pattern number="02" title="Why this location?" subtitle="Geography is an explanation, not decoration." icon={MapPin} label="SYNTHETIC GEOMETRY SCHEMATIC · NOT A MAP" notes={notes} why="A statewide alert is not automatically applicable. Show exactly how the facility matched the alert." data="Facility coordinates, official polygon or county/zone identifiers and boundaries, applicability result." variants="Point + polygon footprint; county/zone coverage; explicit unknown when data is missing.">
          <div className="wg-small-segment" aria-label="Applicability examples">
            {["polygon", "zone", "unknown"].map(value => <button key={value} aria-pressed={geoVariant === value} onClick={() => setGeoVariant(value)}>{value}</button>)}
          </div>
          {geoVariant === "polygon" ? <>
            <svg viewBox="0 0 380 140" className="wg-geo" role="img" aria-label="Synthetic schematic: a fictional facility point lies inside a warning shape; geographic position and shape are illustrative only.">
              <defs><pattern id="wg-grid-pattern" width="26" height="26" patternUnits="userSpaceOnUse"><path d="M 26 0 L 0 0 0 26" fill="none" stroke="currentColor" strokeWidth="0.7" opacity="0.18" /></pattern></defs>
              <rect width="380" height="140" fill="url(#wg-grid-pattern)" />
              <path d="M90 40 L221 16 L307 78 L252 127 L139 118 Z" className="wg-warning-shape" />
              <circle cx="200" cy="69" r="13" fill="var(--wg-accent)" opacity="0.18" /><circle cx="200" cy="69" r="5" fill="var(--wg-accent)" />
              <text x="216" y="73" fill="currentColor" fontSize="12">Fictional facility</text>
              <text x="109" y="101" fill="currentColor" fontSize="10">Illustrative warning shape</text>
            </svg><div className="wg-row"><Badge tone="accent">Inside polygon</Badge><span className="wg-small">Basis: point-in-polygon</span></div>
          </> : geoVariant === "zone" ? <div className="wg-location-state"><h3>Coverage by forecast zone</h3><p>Facility zone <code>FLZ028</code> is listed in the recorded Flood Watch.</p><Badge tone="accent">Zone match example</Badge><p className="wg-small">No polygon does not mean irrelevant. County-based watches require county matching.</p></div> : <div className="wg-location-state"><h3>Applicability unknown</h3><p>Missing or unresolved geographic evidence cannot be treated as outside the alert.</p><Badge tone="amber">Needs evidence</Badge><p className="wg-small">Show what is missing. Do not invent a boundary.</p></div>}
        </Pattern>

        <Pattern number="03" title="Show the material change" subtitle="The useful unit is a decision diff, not an alert count." icon={FileCheck2} label="SYNTHETIC REPLAY · ILLUSTRATIVE POLICY" notes={notes} why="Connect a prior operating instruction to new evidence and one next action." data="Prior immutable approval, current relevant facts, source references, policy version and evidence gaps." variants="Side-by-side comparison on desktop; ordered before/changed/action rows on mobile.">
          <div className="wg-diff">
            <div><p className="wg-kicker">PREVIOUSLY APPROVED</p><h3>Report at 10:00</h3><p className="wg-small">Manager approval · T+0</p></div>
            <div><p className="wg-kicker">WHAT CHANGED</p><h3>Warning now applies</h3><p className="wg-small">Illustrative policy holds 2 queued arrival notices.</p></div>
          </div>
          <div className="wg-next"><span>What needs action</span><strong>Review a correction for affected groups</strong><span className="wg-action-example">Review correction <ArrowRight size={14} aria-hidden="true" /></span></div>
          <p className="wg-small wg-margin-top">Operating instruction review is separate from the official warning. Closure does not mean leave the building.</p>
        </Pattern>

        <Pattern number="04" title="People and notice states" subtitle="Queued, delivered and acknowledged mean different things." icon={Users} label="SYNTHETIC ROSTER · SIMULATED HISTORY" notes={notes} why="Show who saw the outdated instruction, who is held, and who was never targeted." data="Group membership, exact message version, distinct notice state events and labeled fixtures." variants="Table for comparison; stacked group cards with the same semantic labels on narrow screens.">
          <div className="wg-table-wrap"><table className="wg-table"><caption>Synthetic correction groups</caption><thead><tr><th scope="col">Group</th><th scope="col">Old notice</th><th scope="col">Correction</th></tr></thead><tbody>
            <tr><th scope="row">Onsite <small>2 staff</small></th><td>1 delivered*<br />1 queued, held</td><td><Badge tone="amber">Draft</Badge></td></tr>
            <tr><th scope="row">Not yet onsite <small>3 staff</small></th><td>2 delivered*<br />1 queued, held</td><td><Badge tone="amber">Draft</Badge></td></tr>
            <tr><th scope="row">Remote <small>1 staff</small></th><td>Not targeted</td><td><Badge>Excluded</Badge></td></tr>
          </tbody></table></div>
          <p className="wg-small wg-margin-top">*Delivery is a fixture. A send intent does not establish delivery or acknowledgement.</p>
        </Pattern>

        <Pattern number="05" title="A reconstructable timeline" subtitle="Preserve originals. Add changes as new events." icon={History} label="SYNTHETIC REPLAY · NO DELIVERY CLAIMS" notes={notes} why="Support audit and recall: what was known, what was approved and what happened next." data="Event IDs, times, actor, evidence/source version, package digest, policy decision and intent IDs." variants="Ordered event list for one facility; state lanes if event volume justifies them.">
          <ol className="wg-timeline">
            <li><time>T+0</time><div><strong>Arrival instruction approved</strong><span>Original decision retained</span></div></li>
            <li><time>T+5</time><div><strong>Applicable warning introduced</strong><span>Official text appears; 2 queued notices held</span></div></li>
            <li><time>T+6</time><div><strong>Correction package reviewed</strong><span>Exact text and synthetic recipients</span></div></li>
            <li><time>T+7</time><div><strong>Re-check before intent</strong><span>Approval + evidence + duplicate prevention</span></div></li>
          </ol>
        </Pattern>

        <Pattern number="06" title="A forecast with provenance" subtitle="Real periods and units, with no invented forecast curve." icon={CloudRain} label="RECORDED NOAA · OCT 10, 2026 · NOT LIVE" notes={notes} why="A simple hourly view connects sourced weather periods to a planned arrival window." data="Original period times, timezone, numeric measure/units, source URL, update availability, fetch time." variants="Discrete bars for precipitation probability; exact-value table. Add thresholds only when policy defines them.">
          <div className="wg-row"><h3>Hourly precipitation probability</h3><span className="wg-small">EDT</span></div>
          <p className="wg-small">Four recorded forecast periods: 42–62% precipitation probability. This is not tornado risk.</p>
          <div className="wg-chart" aria-hidden="true">{periods.map(period => <div className="wg-bar-column" key={period.time}><span>{period.probability}%</span><div className="wg-bar-track"><div className="wg-bar" style={{height: `${period.probability}%`}} /></div><small>{period.time}</small></div>)}</div>
          <div className="wg-sr-only"><table><caption>Recorded NOAA forecast, October 10 2026, EDT</caption><thead><tr><th scope="col">Hour</th><th scope="col">Precipitation probability</th><th scope="col">Temperature</th><th scope="col">Wind</th></tr></thead><tbody>{periods.map(period => <tr key={period.time}><th scope="row">{period.time}</th><td>{period.probability}%</td><td>{period.temperature}°F</td><td>{period.wind}</td></tr>)}</tbody></table></div>
          <div className="wg-provenance"><span>Fetched 21:06 UTC</span><span>Update time unavailable in first-pass extract</span><a href="https://api.weather.gov/gridpoints/TAE/118,69/forecast/hourly">NOAA endpoint <ExternalLink size={11} aria-hidden="true" /></a></div>
        </Pattern>

        <Pattern number="07" title="Explain the policy gate" subtitle="A held action needs an understandable reason." icon={ShieldCheck} label="SYNTHETIC REPLAY · ILLUSTRATIVE OPA RESULT" notes={notes} why="Expose allow/deny and relevant conditions without making the model an authority." data="OPA result + reasons, policy version, evaluated conditions, exact package digest and Go recheck time." variants="Short reason beside the next action; expandable condition-by-condition evidence.">
          <div className="wg-policy"><div className="wg-row"><Badge tone="amber">Intent held</Badge><code>policy: illustrative-v1</code></div><h3>Review the changed package</h3><p>The draft differs from the last approval. Exact message and recipients require manager review.</p><ul><li><span>Evidence applicability</span><strong>Known</strong></li><li><span>Exact approved package</span><strong>Changed</strong></li><li><span>Duplicate intent</span><strong>None recorded</strong></li></ul></div>
          <p className="wg-small wg-margin-top">OPA evaluates policy. Go enforces exact binding and deduplication. One Strands agent explains and drafts; it cannot approve or send.</p>
        </Pattern>
      </section>

      <footer className="wg-footer"><div><p className="wg-kicker">PRIMARY REFERENCES</p><p>Patterns adapted from documented systems. Usability and accessibility of this implementation still need testing.</p></div><nav aria-label="Research sources">{sourceLinks.map(([name, url]) => <a key={name} href={url}>{name}<ExternalLink size={12} aria-hidden="true" /></a>)}</nav></footer>
    </div>
  </main>;
}
