export const FACILITY_IDS = ['SC-DEMO-01', 'FL-DEMO-01', 'CA-DEMO-01'];

// Demo access is an explicit development choice, never a production fallback.
export function authEnvironment(env = process.env) {
  if (env.WEATHER_DESK_AUTH_MODE === 'demo') {
    if (env.NODE_ENV === 'production') throw Error('Demo access is disabled in production');
    return { mode: 'demo' };
  }
  if (env.WEATHER_DESK_AUTH_MODE && env.WEATHER_DESK_AUTH_MODE !== 'oauth') throw Error('Invalid authentication mode');
  if (!env.AUTH_GITHUB_ID || !env.AUTH_GITHUB_SECRET || (env.AUTH_SECRET || '').length < 32 || (env.WEATHER_DESK_GATEWAY_SECRET || '').length < 32 || !env.AUTH_URL) {
    throw Error('OAuth configuration is incomplete');
  }
  const url = new URL(env.AUTH_URL);
  if (url.username || url.password || url.pathname !== '/' || url.search || url.hash) throw Error('AUTH_URL must be a site origin');
  const local = ['localhost', '127.0.0.1', '[::1]'].includes(url.hostname);
  if (url.protocol !== 'https:' && !(env.NODE_ENV !== 'production' && local && url.protocol === 'http:')) throw Error('HTTPS is required');
  return { mode: 'oauth', origin: url.origin };
}
export function roleFor(login, env = process.env) {
  const managers = (env.AUTH_MANAGER_LOGINS || '').split(',').map(value => value.trim().toLowerCase()).filter(Boolean);
  return managers.includes(String(login || '').toLowerCase()) ? 'manager' : 'viewer';
}
export function identityFromSession(session, env = process.env) {
  const user = session?.user;
  if (!user || !/^github:\d+$/.test(user.id || '') || !user.login) return null;
  // Never trust a role posted by the browser or cached in the JWT.
  const allowedIds = (env.AUTH_MANAGER_IDS || '').split(',').map(value => value.trim()).filter(Boolean);
  const candidateRole = roleFor(user.login, env);
  const role = candidateRole === 'manager' && (allowedIds.length === 0 || allowedIds.includes(user.id.slice('github:'.length))) ? 'manager' : 'viewer';
  return { sub: user.id, name: user.name || user.login, login: user.login, role, facilities: [...FACILITY_IDS] };
}
export function demoIdentity() {
  return { sub: 'demo:local', name: 'Demo manager', login: 'demo', role: 'manager', facilities: [...FACILITY_IDS], demo: true };
}
