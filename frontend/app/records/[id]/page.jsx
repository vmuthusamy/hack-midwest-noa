import { redirect, notFound } from 'next/navigation';
import { currentIdentity } from '../../../lib/server-identity';
import DecisionRecords from '../../../components/decision-records';
export const dynamic='force-dynamic';
export default async function Page({params,searchParams}) {
 let identity;try{identity=await currentIdentity();}catch{redirect('/signin');}if(!identity)redirect('/signin');
 const {id}=await params,query=await searchParams;
 if(!/^[A-Za-z0-9_-]{1,128}$/.test(id))notFound();
 return <DecisionRecords identity={identity} recordId={id} initialFacility={query.facility}/>;
}
