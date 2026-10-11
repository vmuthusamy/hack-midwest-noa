# Demo brief: Hurricane Isaias

Goal: a 3-minute demo built on a real storm. Ship by 05:00 CT. Demos start at 09:00 CT.
Rules: no invented data. Label every value as **Live**, **Recorded <time>**, or **Example**. Don't break existing tests.

## Story (the order the screens appear)
1. "Isaias hit Destin, FL on Oct 9: at least 4 dead, hundreds of thousands without power."
2. 4am. A facility in SC is inside a real NWS Flash Flood Warning. Local law enforcement reported the flooding.
3. The system recommends **Telework**, shows the threshold it crossed and the evidence behind it, and the manager approves.
4. Who gets notified and why, with a phone preview of the message.
5. The decision record: what was known, and when.

## Recorded data (already saved, use it)
`fixtures/isaias-2026-10-10/` was captured at 2026-10-11T02:20Z from the live APIs.
- `alerts-southeast-*.json`: 68 NWS alerts (SC/GA/NC/FL/AL), including Flash Flood Warning `urn:oid:2.49.0.1.840.0.be5613bdb2f88e91a29b7e41456b681f8ab18db1.001.1` (Aiken/Edgefield/Lexington/Saluda; Severe/Likely/Immediate; expires 23:00 EDT Oct 10).
- `alerts-point-barnwell-*.json`: Barnwell has a **Flood Watch** (until 08:00 EDT) and a Lake Wind Advisory. Barnwell is **outside** the warning polygon.
- `points-barnwell-*.json`, `nwps-srbs1-*.json`, `usgs-02175200-*.json`: zone lookup and gauge data for Barnwell.

## Tasks, in priority order

**1. Isaias recorded case** (follow the `case-recordings/` pattern)
- New fictional facility: `SC-DEMO-02` "Batesburg Field Office", 33.91, −81.54. Verified inside the warning polygon.
- Replay that warning: the posture should be **telework** using the existing flood rule.
- Keep Barnwell as the "Flood Watch → planning review" example.
- Done when: the case loads offline, the posture is telework, and the trigger cites the alert ID.

**2. First screen: put the status on top**
- Replace the "One situation. One clear decision." hero with a status bar: facility · hazard · expires · a "staff leave in X" countdown · 5-step posture ladder with the recommended step highlighted.
- Layout: evidence on the left, recommendation card on the right (about ⅓ of the width), people and message below, decision record at the bottom. One column on phones.
- One demo banner only: "Demo · fictional facility and staff · nothing is sent." Remove the other caveats from the panels.
- Done when: the posture is readable without scrolling at 1440×900 and at 390px.

**3. Recommendation card**
- Show: posture · "Crossed:" criterion · rationale · Approve · Change (with a required reason, saved in the record) · **Copy rationale** (clipboard).
- Done when: overriding to Closure saves the reason, and the reason appears in `/records/:id`.

**4. Threat map** (MapLibre, `https://demotiles.maplibre.org/style.json`, no key)
- Warning polygon, facility pin, river gauge marker.
- If the facility is outside the polygon but within 10 mi, label it "Near". If it's inside, label it "Inside".
- Done when: Batesburg shows Inside and Barnwell shows Outside / Watch.

**5. People and message preview**
- One reason per person, e.g. "Telework-eligible → telework", "Essential → report", "Already remote → skipped".
- A phone-style SMS preview for each group. Lead with the action, and end with "Next update 6:00 AM."
- Delivery states: queued → sent → delivered → confirmed, plus **unreachable → next channel**. Label all of it Simulated.

**6. Power outage input** (the brief lists it as a supplementary signal)
- Manager enters it: "Facility power out: yes/no/unknown", with source and time. It shows up in the evidence ledger and the record.
- Rule: power out, plus an active warning → suggest **closure**.
- Done when: the value appears with its source and time, and the posture changes from telework to closure.

**7. Telemetry page polish**
- Tiles: calls · accepted · caught by validator · median latency.
- A "Caught" list showing real validator rejections (`docs/evaluation-results/model-*.json`).

## Alert timeline (integrated)
`frontend/components/alert-timeline.jsx` + `.module.css` + `frontend/lib/alert-timeline.mjs`; tests in `tests/alert-timeline.test.mjs` (10 passing).
- Integrated in the evidence panel with explicit `mode`, facility IANA `timeZone`, `clockMs`, and the configured synthetic `operatingHours`. Source IDs and gaps remain expandable.
- In replay, pass the replay clock as `now`, not `Date.now()`.
- It encodes severity as color + a 4-bar meter, certainty as solid vs striped, and urgency as plain words. It shades the workday and shows a "Now" line. It drops outside alerts, keeps unknown coverage (flagged), and keeps expired and cancelled local messages with explicit lifecycle labels. Message expiry is distinct from the expected hazard end.

## Fix the deployed screen first (from a review of the live site, 21:30 CT)
- **Above the fold** currently shows a slogan plus a full-width NWS box. The decision isn't visible. Show facility, hazard, expiry and posture at the top.
- **Lake Wind Advisory** (advice for boaters) has the same weight as the Flood Watch. Show only hazards that affect operations; put the rest behind "Other alerts (1)".
- **The stepper hides the work.** You only see one step at a time, and the last step leaves a large empty area under a short card. Drop the stepper: show the decision, the people and the record on one page in reading order.
- **Repeated caveats.** "Threshold comparison unavailable" appears twice, and the river card has 3 caveat lines and the footer 2 more. Keep one line each.
- **Small gray text.** Labels are about 11px gray on near-black. Use at least 13px for anything a user needs to read.

## Copy: write like the facility manager talks
Use the facility name, a time and a number in every heading. No taglines, no paired fragments ("One X. One Y."), and no abstract nouns like situation, evidence or instruction.

| Now | Use |
|---|---|
| One situation. One clear decision. | *(delete; the status bar is the heading)* |
| Review the evidence, choose the instruction, and keep the record. | *(delete)* |
| Refresh into new case | Check weather now |
| Planning watch · manager review | Flood Watch until 7 AM: plan, don't close yet |
| Why this decision fits the evidence | Reason (saved in the record) |
| The record is complete. | Telework approved by Venki M at 9:24 PM |
| Flood Watch covers Barnwell Field Office. Review telework and reporting arrangements… no closure is inferred. | Flood Watch for Barnwell until 7 AM, no warning yet. Telework-eligible staff work from home; essential staff report as scheduled. Next check 5 AM. |
| Gauge readings do not establish conditions at this facility or on nearby roads. | River gauge near Barnwell: 6.46 ft. No flood stage is published for it. |
| Official emergency instructions apply independently of this operating decision. | NWS instructions always come first. |

## Demo safety
- Real calls take **30–43 s**. Pre-run the Strands draft for the Isaias case and show the cached result, labelled with its time. Do a live call only as an optional extra.
- Keep the app working if NWS is unreachable: the recorded case must not depend on the network.

## Reference
Pattern gallery with visuals, data needs and snippets: https://claude.ai/artifact/WdkvBz2PBXofyK8SUEHSix

## Notification channels and official alert cards

The supplied channel-icons and official-alert components are integrated in the manager case. Channel chips describe the planned order; status icons use the existing exact-approval notification events, never the preview's sample values. Both recipient tables share these components.

The official alert card shows NWS instructions immediately, severity/certainty/urgency, source attribution and exact listed-place matches. The full bulletin is expandable and retains observation paragraphs. Message expiry and expected hazard end are labeled separately in the facility timezone. Matching a place name is display context only; geographic applicability remains a server check.

## Phone flow pending user acceptance

Crisis response view: official alert and source instruction paragraph, operating choice, one review action. Full NWS text, map/timeline, rationale and local power inputs open on demand. Channel icons continue to describe planned routes separately from recorded simulated status. Preserve the user's existing approval and show its actual posture at the top. Pause further QA until the user accepts this flow.

## Decision detail additions · pending flow acceptance

- Work overlap: remaining alert window intersected with supplied facility-local example hours over the next 24 hours. Uses the replay capture clock in REPLAY. Missing timing/schedule or stale applicability is Unknown. Batesburg's recorded warning overlaps 0 minutes of 08:00–17:00 work; no commute hours are invented. Message expiry remains distinct from expected hazard end.
- Evidence confidence: High for fresh, applicable active Actual NWS evidence with Observed/Likely certainty; Medium for Possible; Low otherwise. This transparent presentation rule does not authorize action or assess safety. Replay freshness is explicitly relative to its capture clock.
- Plain rationale: default untouched drafts use short sourced sentences with facility, message expiry, selected posture and effective time. Power reports retain source/time. Saved approvals and custom rationales are preserved; Use concise rationale explicitly prepares a revision. No invented Sunday closure or supervisor contact instructions.
- Calculation explanations stay collapsed. Deployment build completed; additional QA remains paused pending user acceptance.
