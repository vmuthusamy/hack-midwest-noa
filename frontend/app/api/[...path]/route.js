import { auth } from '../../../auth';
import { forwardRequest } from '../../../lib/gateway.mjs';
export const runtime = 'nodejs';
export const dynamic = 'force-dynamic';
export const maxDuration = 100;
const handle = request => forwardRequest(request, { getSession: auth });
export { handle as GET, handle as POST };
