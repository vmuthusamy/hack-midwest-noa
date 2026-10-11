import Link from 'next/link';
import { redirect } from 'next/navigation';
import { CloudLightning, LogIn, ShieldCheck } from 'lucide-react';
import { signIn } from '../../auth';
import { currentIdentity } from '../../lib/server-identity';
import { authEnvironment } from '../../lib/auth-policy.mjs';
export const dynamic = 'force-dynamic';
export default async function SignInPage({ searchParams }) {
  let ready = false;
  try { ready = authEnvironment().mode === 'oauth'; if (await currentIdentity()) redirect('/'); }
  catch (error) { if (error?.digest?.startsWith('NEXT_REDIRECT')) throw error; }
  const params = await searchParams;
  return <main className="desk signin-page"><section className="signin-card"><Link href="/" className="brand"><CloudLightning size={25}/><span>weather<span className="brand-secondary">desk</span></span></Link><p className="eyebrow">FACILITY OPERATIONS</p><h1>Sign in to Weather Desk</h1><p className="muted">Review NOAA evidence and keep a clear record of operating instructions.</p>{params.error&&<p className="feedback error" role="alert">Sign-in did not complete. Please try again.</p>}{ready?<form action={async()=>{'use server';authEnvironment();await signIn('github',{redirectTo:'/'});}}><button type="submit" className="button primary"><LogIn size={18}/>Continue with GitHub</button></form>:<p className="feedback error" role="status">Sign-in setup is incomplete. The workspace owner needs to finish connecting GitHub.</p>}<div className="signin-note"><ShieldCheck size={18}/><p>Manager access is assigned by the workspace owner. Other signed-in users can view evidence.</p></div><p className="microcopy">Fictional facilities and staff · Sending remains simulated</p></section></main>;
}
