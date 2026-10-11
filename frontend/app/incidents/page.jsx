import { redirect } from 'next/navigation';
import { currentIdentity } from '../../lib/server-identity';
import LiveIncidents from '../../components/live-incidents';
export const dynamic='force-dynamic';
export default async function Page() {
 let identity;try {identity=await currentIdentity();}catch {redirect('/signin');}
 if(!identity)redirect('/signin');
 return <LiveIncidents identity={identity}/>;
}
