# Weather Desk: UX investigation and pattern library

Research date: October 10, 2026. Public desk research only; no user interviews or usability studies were performed. Sources inform design choices, not a claim that the prototype is best in class, policy compliant, or safety certified.

## What real applications already do

This comparison addresses the implemented manager screen, not just a generic component checklist. Four public products were reviewed through their official documentation, illustrated tutorials and datasheets. No paid product account, private customer configuration or live end-to-end product trial was used. These are reference applications; “best in class” is not a measured ranking.

| Real application | Concrete documented workflow / screen | What Weather Desk adapts | Required data and present limit |
|---|---|---|---|
| **Everbridge Visual Command Center + Mass Notification** | VCC combines assets, risks and context, then connects to incident communications. Its public product page shows the command-center map. The Mass Notification datasheet describes guided templates, reporting and follow-up notices. | An applicable warning leads to the affected facility and people, then a reviewed correction. Group targeting and follow-up messaging already exist in established products. | Facility/asset coordinates, sourced hazard, selected contacts, exact text, notice history. Our people and delivery records are simulated; no real notification channel. |
| **PagerDuty incident details + Timeline** | Incident status, assignee and actions are separate from a timestamped Timeline of actions and notifications. Timeline filters narrow the events. Narrative Builder links evidence to important event markers. | Put the current instruction and next action above a concise decision history; preserve approval, hold, review and simulated queue events. | Actor, timestamp, previous decision, evidence reference, notice events. A staff acknowledgement must never mean the weather warning is resolved. |
| **ArcGIS Dashboards: Redlands Emergency Dashboard** | Esri's illustrated tutorial shows a map, open-incident list, details and type chart. Selecting an incident highlights it on the map; chart selection filters map/list; a facility selector focuses the map. | One selected facility and evidence item should drive the location panel, change explanation and affected groups consistently. Keep the list when the map cannot explain chronology. | Official geometry, facility identity, incident IDs and shared selection. Our extent plot is a geographic explanation, not a basemap, routing tool or full GIS dashboard. |
| **NWS weather.gov point forecast / Hourly Weather Graph** | The official guide takes a chosen location to a point forecast, then an hourly graph with selectable weather variables and periods. | Show the selected facility's actual hourly values with units, local time and source/update information; chart only the relevant measure. | Original forecast periods, probability/temperature/wind units, timezone and freshness. Our gallery chart is a recorded snapshot; current manager screen does not yet ingest forecasts. |

Primary screen/workflow references: [Everbridge VCC illustrated product page](https://www.everbridge.com/products/visual-command-center/), [Everbridge notification datasheet, dated 2021](https://www.everbridge.com/wp-content/uploads/DS-Mass_Notification.pdf), [PagerDuty Timeline documentation and screenshots](https://support.pagerduty.com/main/docs/incidents#incident-timeline), [PagerDuty Narrative Builder](https://support.pagerduty.com/main/docs/narrative-builder), [Esri illustrated emergency dashboard tutorial](https://learn.arcgis.com/en/projects/monitor-real-time-emergencies/), [NWS illustrated hourly-forecast guide](https://www.weather.gov/dlh/GetyourHourlyWeatherForecastfromNWS).

The Everbridge support-console page was also indexed by search, but direct access redirected to its support login. Product findings above therefore rely on the accessible public product page and datasheet; they are not a claim of an authenticated console inspection.

### What this means for the current prototype

Already implemented: a selected fictional facility; recorded/live separation; original official instructions; a geographic applicability explanation; previous instruction and changed evidence; selected group messages for review; separate simulated notice stages; a decision history; an exact-package check before recording a simulated send intent. These match the problem diagram's evidence → checks → one drafting agent → manager → people/history flow. Official warnings follow its independent path.

Our specific experiment asks **“Who received the instruction that this new evidence makes outdated?”** A before/after decision panel and correction package make that task the main screen. This is a design hypothesis, not evidence that Everbridge, PagerDuty or Esri lack equivalent capabilities. Alert maps, targeted messaging, AI summaries and timelines alone are not a demonstrated differentiator.

Remaining product gaps: real signed-in manager and facility authorization, real roster integration, real delivery/acknowledgement, live forecast ingestion, supplementary road/river/power evidence, and actual Strands inference with the organizer's Akamai key. The current demo manager is not an authenticated user. No completed usability study or safety certification is claimed.

### Gallery mapping

The Design Library now identifies the real reference on each pattern and states what is adapted: NWS → authoritative warning/forecast; Everbridge → asset relevance and group follow-up; PagerDuty → evidence-backed event history; ArcGIS → synchronized facility/list/map context. The policy card is explicitly our implementation boundary, not a screenshot or feature claim about those products. Technical details stay in research notes. User-facing replay labels use “Start” and “5 minutes later,” not developer timeline notation.

## Audience and task

Primary audience: a USDA-style facility manager responsible for an operating instruction and its correction. The user needs to answer: **What applies here? What changed? Who received the earlier instruction? What must I review now?** Secondary audience: an auditor reconstructing the decision. Staff receive group-specific operational notices; they are not expected to use a weather analyst console.

This persona is a working hypothesis. OPM's current procedures distinguish operating announcements and employee work arrangements, reinforcing the need for role-specific communications. Their DC-specific guidance is context, not this fictional Florida facility's policy. The prototype uses an explicitly illustrative policy and synthetic roster. [OPM procedures](https://www.opm.gov/policy-data-oversight/pay-leave/reference-materials/handbooks/dcdismissal/)

## Evidence that changes the implementation

1. **Location must be resolved before ranking an alert.** NWS distinguishes polygon warnings from county/zone products. Forecast zones and counties are not interchangeable. A forecast-zone-only query can omit tornado and severe-thunderstorm warnings and watches. Use a point query, or a county query followed by precise applicability checks. A null polygon does not mean the alert is irrelevant. Resolve the facility's county and forecast zone; display the applicable basis. [NWS geolocation guide](https://www.weather.gov/media/documentation/docs/NWS_Geolocation.pdf)
2. **Watch, warning, and emergency have distinct meanings.** Use the official event name. A Tornado Warning may be radar indicated; it is not automatically confirmation of a tornado on the ground. Preserve description and instructions from the specific alert. [NWS tornado alert definitions](https://www.weather.gov/safety/tornado-ww)
3. **Urgency needs words and accessible semantics.** USWDS recommends naming the alert type and choosing the announcement role to match urgency. WAI advises that alerts should not steal keyboard focus or disappear automatically. Announce newly material emergency changes once; refresh timestamps belong in a polite status region. [USWDS alert](https://designsystem.digital.gov/components/alert/), [WAI alert pattern](https://www.w3.org/WAI/ARIA/apg/patterns/alert/)
4. **State and notification history are separate records.** Grafana separates alert state changes from notification history. PagerDuty exposes timestamped incident actions. Borrow the traceability pattern, not their resolve/acknowledge semantics: acknowledging a staff notice here does not resolve a weather warning. [Grafana history](https://grafana.com/docs/grafana/latest/alerting/monitor-status/view-alert-state-history/), [PagerDuty incident timeline](https://support.pagerduty.com/main/docs/incidents)
5. **Charts need one clear question and equivalent text.** USWDS recommends familiar chart types, explicit conclusions, labels and underlying data without hover requirements. Colors must not carry meaning alone. [USWDS visualization guidance](https://designsystem.digital.gov/components/data-visualizations/), [WAI use of color](https://www.w3.org/WAI/WCAG22/Understanding/use-of-color.html)

## Screen hierarchy and product mapping

The existing `docs/problem-map.html` is the boundary model for this investigation. Its NOAA, facility/people and supplementary-signal nodes map to provenance and applicability cards; deterministic checks map to an explicit policy explanation; the one Strands node maps to a cited draft explanation; the manager node maps to exact package review; recipient-package/history nodes map to the state matrix and append-only timeline. Official emergency instructions remain on the diagram's bypass path and never wait for the model or operational approval. The supplementary-signal node is still visibly unconnected.

The new Go/OPA direction refines the deterministic-checks node: embedded OPA/Rego evaluates allow/deny and reasons; Go performs location/time normalization, exact approval binding, transaction/dedup checks and event recording. The model is read-only. A policy panel displays the version and failed conditions, not a claim of real USDA authorization. [OPA integration](https://www.openpolicyagent.org/docs/integration), [OPA decision logs](https://www.openpolicyagent.org/docs/management-decision-logs)

| Position | User question | Component and required data |
|---|---|---|
| Top left | Which facility and mode? | Facility identity/location; LIVE or REPLAY; recorded/fetch timestamp; fictional facility badge |
| Top, full width | What official instruction applies? | Official event, issuer, source, exact instructions, affected area, issue/expiry, applicability basis |
| Middle left | What did I approve? | Immutable prior decision text, recipients, policy version, approval timestamp |
| Middle right | What changed and needs action? | Material fact diff, cited evidence, gaps, deterministic hold reason; one Review correction action |
| Lower workspace | Who needs which correction? | Group rows; exact message; separate approved/queued/attempted/delivered/acknowledged states |
| Bottom | How can I reconstruct this? | Timestamped evidence, approval, re-check and simulated intent events; expandable technical trace |

Dark is the initial presentation requested by the user. A light variant uses the same hierarchy and semantic tokens. Tailwind and reusable React components provide consistency; using them does not confer the USWDS component's accessibility test results. Do not copy a government identity banner onto a personal hackathon app.

## Seven-pattern gallery

`frontend/app/gallery/page.jsx` and its `gallery.css` are a reviewable Next.js pattern gallery, with theme and annotation toggles. All operational examples are labeled synthetic REPLAY. NOAA forecast values are separately labeled a recorded snapshot.

| Pattern | Why it fits | Variants | Data contract / guardrail |
|---|---|---|---|
| 1. Official warning banner | Gives the authoritative instruction highest prominence | Expanded instruction panel; slim source/header when no urgent instruction exists | Exact alert text, ID URL, issuer, issue/expiry, last fetch. No approval button on emergency instructions. Gallery contains placeholder copy, not fabricated NWS instructions. |
| 2. Applicability footprint | Explains why statewide evidence does or does not apply here | Polygon+point map; zone/county coverage list; unknown panel | Facility coordinates; Polygon/MultiPolygon with holes; county/zone IDs and boundaries; explicit applicability state. Gallery schematic is not a real map. Resolve official geometry before rendering a geographic map. |
| 3. Previously approved / changed / next action | Makes the differentiation visible | Side-by-side cards on desktop; stacked change rows on mobile | Approval snapshot, relevant evidence conditions, material change list, gap list. Irrelevant refreshes should not trigger reapproval. No unsupported closure recommendation. |
| 4. Recipient-state matrix | Shows the correction audience without conflating delivery stages | Semantic table; stacked group cards on narrow screens | Synthetic recipient IDs/group membership; previous notice and correction state; exact message versions. Delivery and acknowledgement fixtures must be explicit. [USWDS tables](https://designsystem.digital.gov/components/table/) |
| 5. Decision event timeline | Supports reconstruction and shows what happened after approval | Compact ordered list; state lanes when volume justifies it | Immutable events with event ID/time/actor/source and approved package digest. A simulated send intent is not delivered. Original approval and notice remain. [Grafana state timeline](https://grafana.com/docs/grafana/latest/visualizations/panels-visualizations/visualizations/state-timeline/) |
| 6. Hourly forecast + provenance strip | Relates actual NOAA periods to the arrival window | Simple bars for precipitation probability; compact table | Forecast periods, timezone, units, issuance/update availability, last fetch. Gallery's 56/49/42/62% are recorded precipitation probabilities, not confidence or tornado risk scores. No interpolated hazard prediction or reopening inference. |
| 7. Policy explanation | Makes a held action understandable and reviewable | Short reason beside next action; expandable evaluated conditions | OPA decision ID, policy version, allow/deny, named conditions, Go package digest/recheck timestamp. Show missing data as unknown. A local prototype's illustrative rule is not an agency mandate. |

### Reusable snippets

```jsx
// Exact official text; material emergency updates can be announced once.
<section aria-labelledby="official-heading">
  <h2 id="official-heading">{alert.event}</h2>
  <p className="whitespace-pre-wrap">{alert.instruction}</p>
  <a href={alert.source}>Official NWS source</a>
</section>
<p role="status">Last successful refresh: {formattedFetchTime}</p>
```

```jsx
// Delivery is a fact from a labeled fixture, never inferred from an intent.
<table>
  <caption>Synthetic correction recipients</caption>
  <thead><tr><th scope="col">Group</th><th scope="col">Old notice</th>
    <th scope="col">Correction</th></tr></thead>
  <tbody>{groups.map(group => <tr key={group.id}>
    <th scope="row">{group.name}</th><td>{group.previousState}</td>
    <td>{group.correctionState}</td></tr>)}</tbody>
</table>
```

## Design hypotheses and review tasks

- Hypothesis: a before/after change panel helps managers identify outdated instructions faster than an alert list. Test both variants with the same applicable warning fixture; ask what changed and who needs a correction.
- Hypothesis: a group matrix makes mixed delivery history clearer than a single sent count. Ask participants to distinguish queued, attempted, delivered and acknowledged records.
- Test an out-of-area polygon, null-geometry zone watch, missing/stale data, cancellation and irrelevant update. Ask whether each requires review; expiration must not be interpreted as safe reopening.
- Ask the manager to approve an exact package, then change recipients or message text and attempt another intent. Changed package must require review; repeated unchanged intent must be prevented.
- Verify keyboard flow, visible focus, 200% zoom, narrow viewport, both themes and screen-reader announcement behavior. These checks remain to be conducted on the integrated page; this research makes no conformance claim.

Acceptance metrics: task completion and errors with sample counts; missed material changes and unnecessary holds. Do not invent scores, interviews or confidence percentages. Supplementary road/power/river evidence is still a product gap; unknown must remain visible.

## Integrated gallery verification

After the real-product update, the production Next.js build passed. A headless browser checked all seven product-reference annotations, dark/light switching, research-note expansion, and the geographic-coverage variant. No page JavaScript errors occurred; the 390px mobile view had no page overflow and contained no developer-style elapsed-time notation. Keyboard, screen-reader and user-task testing remain separate work; these checks are not a usability validation or accessibility conformance claim.
