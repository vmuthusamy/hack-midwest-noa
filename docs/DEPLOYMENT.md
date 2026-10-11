# Weather Desk deployment

Weather Desk runs on one Akamai Compute instance with Caddy HTTPS, Next.js, Go and a single Strands gRPC worker. Hosted model inference uses the workshop endpoint. Only ports 80 and 443 serve the app; SSH uses a dedicated deployment key. Internal API and gRPC ports are not published.

## GitHub login registration

Register a personal GitHub OAuth app at https://github.com/settings/applications/new.

- Name: Weather Desk
- Homepage: https://172-236-96-42.sslip.io
- Callback: https://172-236-96-42.sslip.io/api/auth/callback/github

Store AUTH_GITHUB_ID and AUTH_GITHUB_SECRET in a private local configuration file. Run ../.venv/bin/python scripts/configure-oauth.py /absolute/path/to/private-oauth.env to update the ignored deployment configuration and recreate only the web container. This script never prints credentials. AUTH_SECRET and WEATHER_DESK_GATEWAY_SECRET are independently generated secrets. AUTH_MANAGER_LOGINS=vmuthusamy and AUTH_MANAGER_IDS=5147665 jointly grant manager authority to that immutable GitHub account; other authenticated users are viewers of the fictional facilities. This is a hackathon workspace role, not a USDA identity assertion.

Never commit runtime .env files, SSH private keys, model credentials or the Akamai Cloud token. The cloud token is used only locally to manage resources; it is not transferred to the application server. The UI contains no provider keys.

## Container launch

From deploy, run docker compose config --quiet, docker compose build and docker compose up -d. Configuration is in deploy/.env on the server with mode 0600. Avoid printing the expanded Compose configuration because it contains secrets. Persistent history is in the history volume; Caddy certificate state has separate persistent volumes. Rebuilds preserve these volumes. Do not use docker compose down -v to update the app.

The server is /opt/weather-desk on Akamai instance 107869915, label weather-desk-hack26, Chicago, 4 GB shared CPU. API-listed cost is $0.036/hour, capped at $24/month before any separately charged options or account credits. Deleting the instance after the hackathon stops its compute charge; merely leaving it running continues billing.

## Verification and mobile

Unauthenticated requests must redirect to sign-in or return an uncached authentication error. Verify the actual GitHub callback, manager identity and viewer restrictions before calling OAuth complete. Run the exact review and simulated-send flow after login, including repeated send attempts and changed evidence.

The responsive website works at the same HTTPS URL on a phone. There is no native mobile client, push-notification service or real SMS/email delivery. Channels and staff are synthetic review data. Missing weather evidence stays unknown; cached or expired warnings never authorize reopening.
