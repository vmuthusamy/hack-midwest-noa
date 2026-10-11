import { createHash, createHmac, randomUUID } from 'node:crypto';
import { authEnvironment, identityFromSession, demoIdentity } from './auth-policy.mjs';

const routes = new Map([
  ['/api/status', 'GET'], ['/api/replay', 'GET'], ['/api/live', 'GET'],
  ['/api/replay/action', 'POST'], ['/api/briefing', 'POST'],
  ['/api/case', ['GET', 'POST']], ['/api/case/action', 'POST'], ['/api/case/briefing', 'POST'], ['/api/records', 'GET'], ['/api/metrics', 'GET'],
]);
const facilities = new Set(['barnwell', 'florida', 'california', 'montana', 'batesburg']);
const json = (status, error) => Response.json({ error }, { status, headers: { 'Cache-Control': 'no-store' } });

export function signedHeaders(identity, method, path, body, secret, now = Date.now(), nonce = randomUUID()) {
  const principal = { sub: identity.sub, name: identity.name, login: identity.login, role: identity.role, facilities: identity.facilities, exp: Math.floor(now / 1000) + 60, method, path, body_sha256: createHash('sha256').update(body).digest('hex'), nonce };
  const encoded = Buffer.from(JSON.stringify(principal), 'utf8').toString('base64url');
  return { 'X-Weather-Principal': encoded, 'X-Weather-Signature': createHmac('sha256', secret).update(encoded).digest('hex') };
}
async function readBody(request) {
  if (!request.body) return Buffer.alloc(0);
  const reader = request.body.getReader();
  const chunks = [];
  let total = 0;
  try {
    for (;;) {
      const { done, value } = await reader.read();
      if (done) break;
      total += value.byteLength;
      if (total > 32768) { await reader.cancel(); throw Error('Body too large'); }
      chunks.push(Buffer.from(value));
    }
  } finally { reader.releaseLock(); }
  return Buffer.concat(chunks);
}

// Dependencies are injectable only by server tests, never by an HTTP request.
export async function forwardRequest(request, { getSession, fetchBackend = fetch, env = process.env } = {}) {
  let configuration;
  try { configuration = authEnvironment(env); } catch { return json(503, 'Sign-in is not configured. Contact the workspace owner.'); }
  let identity;
  try { identity = configuration.mode === 'demo' ? demoIdentity() : identityFromSession(await getSession(), env); }
  catch { return json(401, 'Your session could not be verified. Sign in again.'); }
  if (!identity) return json(401, 'Sign in to use Weather Desk.');
  const url = new URL(request.url);
  const allowed = routes.get(url.pathname) || (/^\/api\/records\/[A-Za-z0-9_-]{1,128}$/.test(url.pathname) ? 'GET' : null);
  if (!allowed) return json(404, 'Route not found');
  if (!(Array.isArray(allowed) ? allowed : [allowed]).includes(request.method)) return json(405, 'Method not allowed');
  const scenario = url.searchParams.get('facility');
  if (scenario && !facilities.has(scenario)) return json(404, 'Unknown facility');
  if (request.method === 'POST') {
    const expectedOrigin = configuration.origin || url.origin;
    if (request.headers.get('origin') !== expectedOrigin) return json(403, 'Request origin is not allowed.');
    if (identity.role !== 'manager') return json(403, 'Manager access is required for this action.');
    if (!request.headers.get('content-type')?.toLowerCase().startsWith('application/json')) return json(415, 'JSON is required');
  }
  let body;
  try { body = request.method === 'POST' ? await readBody(request) : Buffer.alloc(0); }
  catch { return json(413, 'The request is too large.'); }
  const path = url.pathname + url.search;
  let backend;
  try {
    backend = new URL(env.WEATHER_DESK_API_URL || (configuration.mode === 'demo' ? 'http://127.0.0.1:8087' : ''));
    if (!['http:', 'https:'].includes(backend.protocol) || backend.username || backend.password || backend.pathname !== '/' || backend.search || backend.hash) throw Error('Invalid backend');
  } catch { return json(503, 'The workspace service is not configured.'); }
  const headers = { 'Accept': 'application/json' };
  if (configuration.mode === 'oauth') Object.assign(headers, signedHeaders(identity, request.method, path, body, env.WEATHER_DESK_GATEWAY_SECRET));
  if (request.method === 'POST') { headers['Content-Type'] = 'application/json'; headers.Origin = configuration.origin || url.origin; }
  try {
    const response = await fetchBackend(new URL(path, backend), {
      method: request.method, headers, body: request.method === 'POST' ? body : undefined,
      cache: 'no-store', redirect: 'error', signal: AbortSignal.any([request.signal, AbortSignal.timeout(95000)]),
    });
    if (!response.headers.get('content-type')?.includes('application/json')) return json(502, 'The service did not respond. Try again.');
    return new Response(response.body, { status: response.status, headers: { 'Content-Type': 'application/json', 'Cache-Control': 'no-store' } });
  } catch (error) {
    return json(error.name === 'TimeoutError' ? 504 : request.signal.aborted ? 408 : 502, 'The service did not respond. Try again.');
  }
}
