export const FACILITY_IDS = ['SC-DEMO-01', 'FL-DEMO-01', 'CA-DEMO-01'];

// Demo access is an explicit development choice, never a production fallback.
export function authEnvironment(env = process.env) {
  if (env.WEATHER_DESK_AUTH_MODE === 'demo') {
    if (env.NODE_ENV === 'production') throw Error('Demo access is disabled in production');
    return { mode: 'demo' };
  }
  if (env.WEATHER_DESK_AUTH_MODE && env.WEATHER_DESK_AUTH_MODE !== 'oauth') throw Error('Invalid authentication mode');
  if (!env.AUTH_GOOGLE_ID || !env.AUTH_GOOGLE_SECRET || (env.AUTH_SECRET || '').length < 32 || (env.WEATHER_DESK_GATEWAY_SECRET || '').length < 32 || !env.AUTH_URL) {
    throw Error('OAuth configuration is incomplete');
  }
  const url = new URL(env.AUTH_URL);
  if (url.username || url.password || url.pathname !== '/' || url.search || url.hash) throw Error('AUTH_URL must be a site origin');
  const local = ['localhost', '127.0.0.1', '[::1]'].includes(url.hostname);
  if (url.protocol !== 'https:' && !(env.NODE_ENV !== 'production' && local && url.protocol === 'http:')) throw Error('HTTPS is required');
  return { mode: 'oauth', origin: url.origin };
}
const validGoogleSubject = value => /^google:[A-Za-z0-9_-]{1,255}$/.test(value || '');
const emailAddress = value => typeof value === 'string' && /^[^\s@]+@[^\s@]+$/.test(value.trim()) ? value.trim().toLowerCase() : null;
export function identityFromGoogleProfile(account, profile) {
  if (account?.provider !== 'google' || profile?.email_verified !== true || typeof profile.sub !== 'string' || account.providerAccountId !== profile.sub) return null;
  const sub = `google:${account.providerAccountId}`, email = emailAddress(profile.email);
  if (!validGoogleSubject(sub) || !email) return null;
  return { sub, email, emailVerified: true, name: profile.name || email };
}
export function roleFor(email, env = process.env, subject) {
  const managers = (env.AUTH_MANAGER_EMAILS || '').split(',').map(value => value.trim().toLowerCase()).filter(Boolean);
  const allowedIds = (env.AUTH_MANAGER_GOOGLE_IDS || '').split(',').map(value => value.trim()).filter(Boolean);
  const listedEmail = managers.includes(emailAddress(email));
  const pinnedSubject = allowedIds.length === 0 || (validGoogleSubject(subject) && allowedIds.includes(subject.slice('google:'.length)));
  return listedEmail && pinnedSubject ? 'manager' : 'viewer';
}
export function identityFromSession(session, env = process.env) {
  const user = session?.user, email = emailAddress(user?.email);
  // Old provider cookies, unverified email, and client-supplied roles grant no authority.
  if (!user || !validGoogleSubject(user.id) || user.emailVerified !== true || !email) return null;
  return { sub: user.id, name: user.name || email, email, login: email, role: roleFor(email, env, user.id), facilities: [...FACILITY_IDS] };
}
export function demoIdentity() {
  return { sub: 'demo:local', name: 'Demo manager', login: 'demo', role: 'manager', facilities: [...FACILITY_IDS], demo: true };
}
