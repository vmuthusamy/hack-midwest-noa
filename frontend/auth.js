import NextAuth from 'next-auth';
import Google from 'next-auth/providers/google';
import { roleFor, FACILITY_IDS, identityFromGoogleProfile, identityFromSession } from './lib/auth-policy.mjs';

export const { handlers, auth, signIn, signOut } = NextAuth(() => ({
  providers: [Google({ authorization: { params: { scope: 'openid email profile' } } })],
  session: { strategy: 'jwt', maxAge: 8 * 60 * 60 },
  trustHost: true,
  pages: { signIn: '/signin', error: '/signin' },
  callbacks: {
    signIn({ account, profile }) {
      return identityFromGoogleProfile(account, profile) !== null;
    },
    jwt({ token, account, profile }) {
      // Only verified Google callback claims establish identity; ignore client session updates.
      if (account) {
        const identity = identityFromGoogleProfile(account, profile);
        if (!identity) return null;
        token.sub = identity.sub;
        token.email = identity.email;
        token.emailVerified = identity.emailVerified;
        token.name = identity.name;
        delete token.login;
      }
      if (!identityFromSession({ user: { id: token.sub, email: token.email, emailVerified: token.emailVerified } })) return null;
      return token;
    },
    session({ session, token }) {
      session.user = { id: token.sub, name: token.name, email: token.email, emailVerified: token.emailVerified === true, login: token.email, role: roleFor(token.email, process.env, token.sub), facilities: [...FACILITY_IDS] };
      return session;
    },
    redirect({ url, baseUrl }) {
      const origin = process.env.AUTH_URL || baseUrl;
      if (url.startsWith('/') && !url.startsWith('//')) return new URL(url, origin).toString();
      try { if (new URL(url).origin === new URL(origin).origin) return url; } catch {}
      return origin;
    },
  },
}));
