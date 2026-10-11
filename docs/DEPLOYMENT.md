# Weather Desk deployment

Weather Desk runs on one Akamai Compute instance with Caddy HTTPS, Next.js, Go and a single Strands gRPC worker. Hosted model inference uses the workshop endpoint. Only ports 80 and 443 serve the app; SSH uses a dedicated deployment key. Internal API and gRPC ports are not published.

## Free shareable app name

Share https://weatherdesk.172-236-96-42.sslip.io. The free sslip.io hostname resolves to the existing Akamai public IP, 172.236.96.42. Caddy obtains HTTPS for it and redirects visitors to the canonical app origin, preserving the path and query. The address bar then shows the canonical hostname. Google OAuth callbacks and session cookies continue using https://172-236-96-42.sslip.io; the alias does not require another OAuth registration.

This is a free named entry point, not an owned domain or an Akamai DNS zone. No domain purchase or additional compute resource is needed. The deployment only changes the HTTPS edge routing; it does not publish the pending application changes.

## Google login registration

The selected login provider is Google. Register a Web application OAuth client in Google Cloud Console, with only openid, email and profile sign-in scopes.

- App name: Weather Desk
- JavaScript origin: https://172-236-96-42.sslip.io
- Redirect URI: https://172-236-96-42.sslip.io/api/auth/callback/google
- The configured brand remains Testing. With only openid, email and profile, Google’s [basic identity exception](https://support.google.com/cloud/answer/15549945?hl=en) does not require a trusted test-user list. Additional scopes would require reassessing that exception. No saved test-user registration or production brand verification is claimed.

Use a personal Google Cloud project. No billing account or paid Google resource is needed for this sign-in client. Save AUTH_GOOGLE_ID and AUTH_GOOGLE_SECRET in a private local environment file. Include AUTH_MANAGER_EMAILS for the intended verified manager account; optional AUTH_MANAGER_GOOGLE_IDS pins manager access to the immutable Google subject. Missing manager grants leave accounts as viewers; the first person to log in does not become a manager automatically.

Run ../.venv/bin/python scripts/configure-oauth.py /absolute/path/to/private-oauth.env to transfer private runtime configuration and recreate only the web container. The script never prints credentials. AUTH_SECRET and WEATHER_DESK_GATEWAY_SECRET are independent server secrets. Google sign-in uses verified email and immutable google:subject identity; older GitHub session cookies do not grant access.

The Akamai account currently has no configured domains. Akamai DNS Manager hosts DNS for a separately registered domain; it does not sell domain registrations. The temporary HTTPS hostname remains usable while a domain is chosen. See https://techdocs.akamai.com/cloud-computing/docs/getting-started-with-dns-manager.

Never commit runtime .env files, SSH private keys or credentials. The cloud token is used only locally; it is not transferred to the server. The UI contains no provider keys.

## Container launch

From deploy, run docker compose config --quiet, docker compose build and docker compose up -d. Configuration is in deploy/.env on the server with mode 0600. Avoid printing the expanded Compose configuration because it contains secrets. Persistent history is in the history volume; Caddy certificate state has separate persistent volumes. Rebuilds preserve these volumes. Do not use docker compose down -v to update the app.

The server is /opt/weather-desk on Akamai instance 107869915, label weather-desk-hack26, Chicago, 4 GB shared CPU. API-listed cost is $0.036/hour, capped at $24/month before any separately charged options or account credits. Deleting the instance after the hackathon stops its compute charge; merely leaving it running continues billing.

## Verification and mobile

The actual Google account chooser, basic identity consent and callback succeeded on the deployed HTTPS app, showing the intended manager account. Independent cookie-free checks at 2026-10-11T01:45:57Z verified root → /signin (307), Google sign-in (200), protected /api/status and /api/live (401, no-store), Google-only provider discovery, and a null unauthenticated session. Independent QA reviewed the manager/replay screenshot; the coordinating agent additionally read authenticated runtime status. See the [actual callback QA report](evaluation-results/google-live-oauth-qa-summary.json) for attribution and limits. Viewer restrictions and exact review/simulated-send behavior were checked separately with isolated synthetic sessions; no cloud decisions were changed during this read-only callback verification.

The responsive website works at the same HTTPS URL on a phone. There is no native mobile client, push-notification service or real SMS/email delivery. Channels and staff are synthetic review data. Missing weather evidence stays unknown; cached or expired warnings never authorize reopening.
