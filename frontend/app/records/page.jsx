import { redirect } from 'next/navigation';
import { currentIdentity } from '../../lib/server-identity';
import DecisionRecords from '../../components/decision-records';
export const dynamic='force-dynamic';
export default async function Page({searchParams}) {
 let identity;try{identity=await currentIdentity();}catch{redirect('/signin');}if(!identity)redirect('/signin');
 const query=await searchParams;
 return <DecisionRecords identity={identity} initialFacility={query.facility}/>;
}
