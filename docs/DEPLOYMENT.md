# Weather Desk deployment

## Current status: retired October 11, 2026

The hackathon instance `107869915` was deleted after stopping its containers and saving a private local archive of decision history, inference telemetry and runtime configuration. A follow-up API lookup returned 404 and the instance inventory returned zero. A later recheck found the automatically generated recovery image `private/41809535`; it was also deleted. The private-image inventory then returned zero. No block volumes, NodeBalancers, Kubernetes clusters, databases, object-storage buckets or allocated IPs remain. Object Storage is disabled; Managed and Longview subscriptions are inactive.

The free DuckDNS name is retained with its IPv4 record set to `0.0.0.0`, removing its link to the released address. The public app is offline. Local app processes were stopped and inference keys cleared from active runtime environment files. The installed Strands SDK is an inert library; this does not revoke the organizers' shared inference key or control other team members' usage.

The account reported a $0.00 balance and $0.58 unbilled usage at teardown. Earlier usage may still appear on a final invoice; promotional-credit treatment was not verified. The Akamai account remains open with no resources found in the checked billable service categories.

The sections below describe the former deployment and restoration configuration. Provision a new instance and explicitly restore credentials before using deployment scripts again.

Weather Desk runs on one Akamai Compute instance with Caddy HTTPS, Next.js, Go and a single Strands gRPC worker. Hosted model inference uses the workshop endpoint. Only ports 80 and 443 serve the app; SSH uses a dedicated deployment key. Internal API and gRPC ports are not published.

## Free shareable app name

Share https://weatherdesk-hackmidwest.duckdns.org. DuckDNS resolves this free hostname to the existing Akamai public IP, 172.236.96.42. Caddy serves the app over HTTPS at this canonical origin. Both previous sslip.io addresses redirect here, preserving the path and query. Google sign-in returns to the named hostname.

DNS is hosted by DuckDNS; application compute remains on Akamai. No domain purchase or additional compute resource is required. This hostname update reused existing application images and did not deploy the new monitoring slice.

## Google login registration

The selected login provider is Google. Register a Web application OAuth client in Google Cloud Console, with only openid, email and profile sign-in scopes.

- App name: Weather Desk
- JavaScript origin: https://weatherdesk-hackmidwest.duckdns.org
- Redirect URI: https://weatherdesk-hackmidwest.duckdns.org/api/auth/callback/google
- The former sslip.io origin and callback remain registered for rollback; current runtime uses the DuckDNS origin.
- The configured brand remains Testing. With only openid, email and profile, Google’s [basic identity exception](https://support.google.com/cloud/answer/15549945?hl=en) does not require a trusted test-user list. Additional scopes would require reassessing that exception. No saved test-user registration or production brand verification is claimed.

Use a personal Google Cloud project. No billing account or paid Google resource is needed for this sign-in client. Save AUTH_GOOGLE_ID and AUTH_GOOGLE_SECRET in a private local environment file. Include AUTH_MANAGER_EMAILS for the intended verified manager account; optional AUTH_MANAGER_GOOGLE_IDS pins manager access to the immutable Google subject. Missing manager grants leave accounts as viewers; the first person to log in does not become a manager automatically.

Run ../.venv/bin/python scripts/configure-oauth.py /absolute/path/to/private-oauth.env to transfer private runtime configuration and recreate only the web container. The script never prints credentials. AUTH_SECRET and WEATHER_DESK_GATEWAY_SECRET are independent server secrets. Google sign-in uses verified email and immutable google:subject identity; older GitHub session cookies do not grant access.

The Akamai account currently has no configured domains. Akamai DNS Manager hosts DNS for a separately registered domain; it does not sell domain registrations. The free DuckDNS hostname is the current canonical address. See https://techdocs.akamai.com/cloud-computing/docs/getting-started-with-dns-manager.

Never commit runtime .env files, SSH private keys or credentials. The cloud token is used only locally; it is not transferred to the server. The UI contains no provider keys.

## Changing the canonical hostname

Set `PUBLIC_HOST` to the hostname and `AUTH_URL` to its HTTPS origin in the private deployment environment. Register the matching Google origin and callback first. Recreate `api`, `web` and `edge` with `docker compose up -d --no-deps --no-build --force-recreate api web edge` to apply the environment while reusing deployed images. Recreating the edge also refreshes its file-mounted Caddyfile. Preserve all named volumes. Do not use a full deployment build for a hostname-only change when pending application changes have not been accepted.

## Container launch

From deploy, run docker compose config --quiet, docker compose build and docker compose up -d. Configuration is in deploy/.env on the server with mode 0600. Avoid printing the expanded Compose configuration because it contains secrets. Persistent history is in the history volume; Caddy certificate state has separate persistent volumes. Rebuilds preserve these volumes. Do not use docker compose down -v to update the app.

The server is /opt/weather-desk on Akamai instance 107869915, label weather-desk-hack26, Chicago, 4 GB shared CPU. API-listed cost is $0.036/hour, capped at $24/month before any separately charged options or account credits. Deleting the instance after the hackathon stops its compute charge; merely leaving it running continues billing.

## Verification and mobile

On October 11, 2026, the named hostname resolved to the Akamai server, passed normal TLS verification and returned unauthenticated visitors to `/signin`. Both old HTTPS origins returned 302 redirects to the new hostname. Google sign-in completed in the browser and displayed the intended manager at the new address. This was a hostname/sign-in check; flow acceptance and broader QA remain separate.


The actual Google account chooser, basic identity consent and callback succeeded on the deployed HTTPS app, showing the intended manager account. Independent cookie-free checks at 2026-10-11T01:45:57Z verified root → /signin (307), Google sign-in (200), protected /api/status and /api/live (401, no-store), Google-only provider discovery, and a null unauthenticated session. Independent QA reviewed the manager/replay screenshot; the coordinating agent additionally read authenticated runtime status. See the [actual callback QA report](evaluation-results/google-live-oauth-qa-summary.json) for attribution and limits. Viewer restrictions and exact review/simulated-send behavior were checked separately with isolated synthetic sessions; no cloud decisions were changed during this read-only callback verification.

The responsive website works at the same HTTPS URL on a phone. There is no native mobile client, push-notification service or real SMS/email delivery. Channels and staff are synthetic review data. Missing weather evidence stays unknown; cached or expired warnings never authorize reopening.
