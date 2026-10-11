import NextAuth from 'next-auth';
import GitHub from 'next-auth/providers/github';
import { roleFor, FACILITY_IDS } from './lib/auth-policy.mjs';

export const { handlers, auth, signIn, signOut } = NextAuth(() => ({
  providers: [GitHub({ authorization: { params: { scope: 'read:user' } } })],
  session: { strategy: 'jwt', maxAge: 8 * 60 * 60 },
  trustHost: true,
  pages: { signIn: '/signin', error: '/signin' },
  callbacks: {
    jwt({ token, account, profile }) {
      // Only the verified OAuth callback can establish the principal.
      if (account?.provider === 'github' && profile) {
        token.sub = `github:${account.providerAccountId}`;
        token.login = profile.login;
        token.name = profile.name || profile.login;
      }
      return token;
    },
    session({ session, token }) {
      session.user = { id: token.sub, name: token.name, login: token.login, role: roleFor(token.login), facilities: [...FACILITY_IDS] };
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
