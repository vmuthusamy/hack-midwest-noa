import { auth } from '../auth';
import { authEnvironment, demoIdentity, identityFromSession } from './auth-policy.mjs';

export async function currentIdentity() {
  const configuration = authEnvironment();
  return configuration.mode === 'demo' ? demoIdentity() : identityFromSession(await auth());
}
