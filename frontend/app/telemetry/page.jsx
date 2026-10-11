import { redirect } from 'next/navigation';
import { currentIdentity } from '../../lib/server-identity';
import InferenceTelemetry from '../../components/inference-telemetry';
export const dynamic='force-dynamic';
export default async function Page() {
 let identity;try{identity=await currentIdentity();}catch{redirect('/signin');}if(!identity)redirect('/signin');
 return <InferenceTelemetry identity={identity}/>;
}
