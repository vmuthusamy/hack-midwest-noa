# Hack Midwest submission draft

Prepared for the organizer form shown by the team. Keep this draft local until the team supplies its names, corporate email addresses and video link. No form or message has been sent. The existing repository is the code destination; a second repository is unnecessary.

## Your Full Name

[Team lead's full name — confirm from the team page]

## Email

[Team lead's corporate or university email — confirm from the team page]

## Who is on your team?

[Each team member's full name and corporate or university email — awaiting team page]

## Name of your app

Weather Desk

## What does your app do? Describe in a paragraph or so

Weather Desk helps a facility manager decide what to change when weather disrupts an office's operating plan. It brings official NOAA/NWS alerts together with location checks, facility context and a staff roster, then helps the manager review a delay, telework or closure decision and the exact messages for affected groups. A Strands agent using Akamai-hosted inference explains the evidence and drafts a rationale. Go and OPA enforce the approval rules, recheck the evidence before recording simulated notice intents and prevent duplicate intents. The saved decision record shows what was known, who approved the decision and who still needs a correction. The hackathon app uses real public weather data, fictional facilities and synthetic staff; notification delivery is simulated.

App link: https://weatherdesk.172-236-96-42.sslip.io (redirects to the hosted Google sign-in app).

## Link to 3 minute video demo of your app

[Insert the team's public or unlisted video link after recording. No video has been uploaded.]

### Suggested three minute recording

Show the running app, not slides. Use the deployed feature set unless the team first accepts and deploys the new local monitoring slice.

- **0:00–0:25 — The problem.** “An office has told people when to arrive. The weather changes. A manager needs to know which instruction to correct and who still needs the update.” Show the signed-in manager and facility.
- **0:25–0:55 — The evidence.** Open the clearly labeled recorded Batesburg warning. Show the warning, source time and facility match. Explain that the office and staff are fictional, while the saved warning is real public NWS data. Do not call the replay a current warning.
- **0:55–1:30 — The decision.** Show the policy suggestion and a real generated Strands/Akamai rationale if available. If showing a saved result, say when it was generated. The manager changes the posture to closure, supplies a reason and reviews the exact messages.
- **1:30–2:10 — The people.** Show included and excluded people, their planned channels and separate notification states. Approve, recheck and record simulated intents. Say explicitly that no staff messages are sent in this prototype.
- **2:10–2:40 — Accountability.** Open the decision record to show evidence, approval and original message history. Briefly show real inference telemetry or a traceable validator rejection; avoid presenting policy test results as safety certification.
- **2:40–3:00 — Why continue.** “The next step is monitored regional changes and an in-app assistant that tells the right manager what changed since approval, with sources. Our measure of value is how quickly the correct people receive an approved correction.”

## What challenges are you building for? Check all that apply

Proposed selections, subject to the organizers' eligibility decision:

- **AWS: Build Intelligent AI Agents with Strands Agents SDK.** The Python service uses the actual Strands SDK, a read-only evidence-reference tool and structured output validation. Go invokes it through the gRPC contract. See `briefing.py`, `agent_service/server.py` and `contracts/proto/weatherdesk/v1/briefing.proto`.
- **Akamai Cloud AI Challenge.** The application is hosted on an Akamai Cloud instance and uses the workshop's hosted inference endpoint through the Strands model adapter. See `deploy/`, `docs/DEPLOYMENT.md` and the saved hosted evaluation results. Credits or an SDK installation alone are not the claim.
- **USDA AI Challenge.** The primary scenario is a USDA-style facility manager making and documenting an operating decision with public weather evidence and a synthetic roster. This is a prototype of that workflow, not an official USDA system or an assertion that every challenge requirement is satisfied.

Do not select Flex or DevRev on the basis of the current implementation. No integration or finance use case is demonstrated for those challenges.

## Link to your code repository link

https://github.com/vmuthusamy/hack-midwest-noa

Working branch: `codex/noaa-prototype`. The new local monitoring and submission changes have not been pushed. Before submitting, the team must publish the intended revision and confirm anonymous access to the code. No repository visibility change is authorized by this draft.

## Organizer note for the team to review

We built Weather Desk to explore a practical enterprise operations problem: keeping staff instructions consistent with changing weather evidence. Our hackathon implementation connects Strands Agents SDK and Akamai-hosted inference to a manager-controlled approval workflow, with NOAA/NWS sources, deterministic geographic checks and a retained decision record. We would welcome feedback on the facility-management workflow and the next steps toward an operational pilot. The current app is a working proof of concept with synthetic recipients and simulated delivery.

[If desired, add a team-approved sentence describing the company and its actual use of the relevant technology. Do not imply company sponsorship, deployment or customer usage of Weather Desk without confirmation.]

## Evidence and limits for questions

The collaborator's recorded posture evaluation reports 68/68 matches against authored policy labels after adding Coastal Flood Warning, 68/68 Barnwell location comparisons and 58/58 polygon comparisons across two facilities. Posture checks assume applicable active alerts and fresh data. This is one storm snapshot, with labels written by the evaluator, and the same dataset was used to find and fix the gap. It does not establish zero missed warnings in production. Results are in `docs/evaluation-results/posture-isaias-eval.json`.

The latest live-monitoring slice compiled and displayed public NWS data in a local preview; acceptance and QA remain pending. The existing hosted application and inference evidence are separate from these unpublished changes.
