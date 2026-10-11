import { handlers } from '../../../../auth';
import { authEnvironment } from '../../../../lib/auth-policy.mjs';
export const runtime = 'nodejs';
function checked(method) {
  return request => {
    try { if (authEnvironment().mode !== 'oauth') throw Error('OAuth is disabled'); }
    catch { return Response.json({ error: 'OAuth sign-in is not configured.' }, { status: 503, headers: { 'Cache-Control': 'no-store' } }); }
    return handlers[method](request);
  };
}
export const GET = checked('GET');
export const POST = checked('POST');
